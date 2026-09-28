package payments

import (
	"context"
	"dailzo/config"
	"dailzo/models"
	"dailzo/repository"
	"time"
)

// NextRun computes the next reconciliation time after `from` for a cadence.
func NextRun(from time.Time, cadence string, hour int) time.Time {
	base := time.Date(from.Year(), from.Month(), from.Day(), hour, 0, 0, 0, from.Location())
	if !base.After(from) {
		base = base.Add(24 * time.Hour)
	}
	switch cadence {
	case "weekly":
		return base.AddDate(0, 0, 7)
	case "monthly":
		return base.AddDate(0, 1, 0)
	default: // daily
		return base
	}
}

// periodStart returns the window start for the cadence ending at `end`.
func periodStart(end time.Time, cadence string) time.Time {
	switch cadence {
	case "weekly":
		return end.AddDate(0, 0, -7)
	case "monthly":
		return end.AddDate(0, -1, 0)
	default:
		return end.AddDate(0, 0, -1)
	}
}

// StartReconciliationScheduler runs reconciliation on the configured cadence
// (daily|weekly|monthly). It ticks once a minute and fires when now >= next_run.
func StartReconciliationScheduler(ctx context.Context, repo *repository.PaymentV2Repository) {
	log := config.SetupLogger()
	tick := time.NewTicker(1 * time.Minute)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-tick.C:
			cfg, err := repo.GetReconConfig(ctx)
			if err != nil || !cfg.Enabled {
				continue
			}
			if cfg.NextRunOn == nil {
				next := NextRun(now, cfg.Cadence, cfg.RunAtHour)
				_ = repo.TouchReconConfigRun(ctx, now, next)
				continue
			}
			if now.Before(*cfg.NextRunOn) {
				continue
			}
			end := now
			start := periodStart(end, cfg.Cadence)
			run, rerr := repo.RunReconciliation(ctx, cfg.Cadence, start, end)
			if rerr != nil {
				log.Error().Err(rerr).Msg("reconciliation run failed")
			} else {
				log.Info().Str("run", run.ID).Float64("discrepancy", run.DiscrepancyTotal).
					Int("unmatched", run.UnmatchedCount).Msg("reconciliation completed")
			}
			_ = repo.TouchReconConfigRun(ctx, now, NextRun(now, cfg.Cadence, cfg.RunAtHour))
		}
	}
}

// StartStatusPollBackstop periodically reconciles payments stuck in a
// non-terminal processing/pending_3ds state by polling the gateway (#8). This is
// the safety net for lost/never-delivered webhooks.
func StartStatusPollBackstop(ctx context.Context, repo *repository.PaymentV2Repository, gateways map[string]Gateway, interval, stuckAfter time.Duration) {
	log := config.SetupLogger()
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			stuck, err := repo.ListStuck(ctx, stuckAfter)
			if err != nil {
				continue
			}
			for _, p := range stuck {
				gw := gateways[p.Gateway]
				if gw == nil || p.GatewayTxnID == "" {
					continue
				}
				status, perr := gw.Poll(ctx, p.GatewayTxnID)
				if perr != nil {
					continue
				}
				switch status {
				case models.StatusCaptured:
					_, _ = repo.Transition(ctx, p.ID, models.StatusCaptured, repository.TransitionOpts{
						EventType: models.EntryCapture, CaptureAmount: p.Amount,
						LedgerEntry: models.EntryCapture, LedgerDir: models.DirCredit,
					})
					log.Info().Str("payment", p.ID).Msg("poll backstop: converged to captured")
				case models.StatusFailed:
					_, _ = repo.Transition(ctx, p.ID, models.StatusFailed, repository.TransitionOpts{EventType: models.EntryAdjustment})
					log.Info().Str("payment", p.ID).Msg("poll backstop: converged to failed")
				}
			}
		}
	}
}
