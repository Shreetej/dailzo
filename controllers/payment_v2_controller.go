package controllers

import (
	"dailzo/models"
	"dailzo/payments"
	"dailzo/pkg/response"
	"dailzo/repository"
	"time"

	"github.com/gofiber/fiber/v2"
)

// PaymentV2Controller implements the payments-redesign HTTP surface.
// See docs/PAYMENTS_REDESIGN.md.
type PaymentV2Controller struct {
	repo     *repository.PaymentV2Repository
	gateways map[string]payments.Gateway
	def      string // default gateway name for prepaid
}

func NewPaymentV2Controller(repo *repository.PaymentV2Repository, gws map[string]payments.Gateway, def string) *PaymentV2Controller {
	return &PaymentV2Controller{repo: repo, gateways: gws, def: def}
}

type createPaymentReq struct {
	OrderID              string  `json:"order_id"`
	Method               string  `json:"method"`   // cod|mada|visa|...
	Amount               float64 `json:"amount"`   // exact order total (immediate full capture)
	Currency             string  `json:"currency"` // default SAR
	Token                string  `json:"token"`    // SDK payment token (never a PAN)
	ReturnURL            string  `json:"return_url"`
	ClientIdempotencyKey string  `json:"client_idempotency_key"`
}

// CreatePayment creates + (for prepaid) immediately captures the exact amount.
// Requires an Idempotency-Key header (or client_idempotency_key in body, #9).
func (c *PaymentV2Controller) CreatePayment(ctx *fiber.Ctx) error {
	var req createPaymentReq
	if err := ctx.BodyParser(&req); err != nil {
		return response.Error(ctx, fiber.StatusBadRequest, "BAD_REQUEST", "invalid body")
	}
	if req.Amount <= 0 || req.Method == "" {
		return response.Error(ctx, fiber.StatusBadRequest, "BAD_REQUEST", "amount and method required")
	}
	if req.Currency == "" {
		req.Currency = "SAR"
	}
	idemKey := ctx.Get("Idempotency-Key")
	if idemKey == "" {
		idemKey = req.ClientIdempotencyKey
	}
	userID, _ := ctx.Locals("user_id").(string)

	// Idempotency: replay the same result for a repeated key.
	if idemKey != "" {
		if existing, err := c.repo.GetByIdempotencyKey(ctx.Context(), idemKey); err == nil && existing != nil {
			return response.Success(ctx, existing)
		}
	}

	// COD — no gateway; order is created elsewhere, payment awaits delivery.
	if req.Method == models.MethodCOD {
		p, err := c.repo.Create(ctx.Context(), models.PaymentV2{
			OrderID: req.OrderID, UserID: userID, Method: models.MethodCOD,
			Gateway: models.GatewayCOD, Currency: req.Currency, Amount: req.Amount,
			Status: models.StatusCreated, IsCOD: true, IdempotencyKey: idemKey,
			ClientIdempotencyKey: req.ClientIdempotencyKey,
		})
		if err != nil {
			return response.Error(ctx, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
		}
		out, err := c.repo.Transition(ctx.Context(), p.ID, models.StatusCODPending,
			repository.TransitionOpts{EventType: "authorize"})
		if err != nil {
			return response.Error(ctx, fiber.StatusInternalServerError, "COD_FAILED", err.Error())
		}
		return response.Success(ctx, out)
	}

	// Prepaid — immediate Sale via the gateway.
	gw := c.gateways[c.def]
	if gw == nil {
		return response.Error(ctx, fiber.StatusServiceUnavailable, "NO_GATEWAY", "no payment gateway configured")
	}
	p, err := c.repo.Create(ctx.Context(), models.PaymentV2{
		OrderID: req.OrderID, UserID: userID, Method: req.Method, Gateway: gw.Name(),
		Currency: req.Currency, Amount: req.Amount, Status: models.StatusCreated,
		IdempotencyKey: idemKey, ClientIdempotencyKey: req.ClientIdempotencyKey,
	})
	if err != nil {
		return response.Error(ctx, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	if _, err := c.repo.Transition(ctx.Context(), p.ID, models.StatusProcessing,
		repository.TransitionOpts{EventType: "authorize"}); err != nil {
		return response.Error(ctx, fiber.StatusInternalServerError, "STATE_FAILED", err.Error())
	}

	res, err := gw.Sale(ctx.Context(), payments.SaleRequest{
		Amount: req.Amount, Currency: req.Currency, Method: req.Method, Token: req.Token,
		OrderID: p.ID, UserID: userID, ReturnURL: req.ReturnURL,
	})
	if err != nil {
		_, _ = c.repo.Transition(ctx.Context(), p.ID, models.StatusFailed,
			repository.TransitionOpts{EventType: "capture"})
		return response.Error(ctx, fiber.StatusBadGateway, "GATEWAY_ERROR", err.Error())
	}

	switch res.Status {
	case models.StatusCaptured:
		out, err := c.repo.Transition(ctx.Context(), p.ID, models.StatusCaptured, repository.TransitionOpts{
			EventType: models.EntryCapture, GatewayTxnID: res.TxnID, CaptureAmount: req.Amount,
			LedgerEntry: models.EntryCapture, LedgerDir: models.DirCredit,
		})
		if err != nil {
			return response.Error(ctx, fiber.StatusInternalServerError, "CAPTURE_STATE_FAILED", err.Error())
		}
		c.generateInvoice(ctx, out)
		return response.Success(ctx, out)
	case models.StatusPending3DS:
		out, _ := c.repo.Transition(ctx.Context(), p.ID, models.StatusPending3DS,
			repository.TransitionOpts{EventType: "3ds", GatewayTxnID: res.TxnID})
		return response.SuccessWithMessage(ctx, res.ThreeDSRedirect, out)
	default:
		out, _ := c.repo.Transition(ctx.Context(), p.ID, models.StatusFailed,
			repository.TransitionOpts{EventType: "capture", GatewayTxnID: res.TxnID})
		return response.SuccessWithMessage(ctx, "payment_failed", out)
	}
}

// generateInvoice creates a draft ZATCA-scaffold invoice for a captured/collected
// payment (idempotent). VAT is split at the KSA rate; the Phase-1 TLV QR + PIH
// chain are populated. Phase-2 cryptographic stamp + clearance are the TODO.
func (c *PaymentV2Controller) generateInvoice(ctx *fiber.Ctx, p models.PaymentV2) {
	subtotal, vat := payments.VATBreakdown(p.Amount, models.VATRate)
	prevHash, _ := c.repo.LastInvoiceHash(ctx.Context())
	invNo := "INV-" + p.ID
	uuid := payments.NewUUID()
	hash := payments.InvoiceHash(invNo, prevHash, p.Amount, p.LastUpdatedOn)
	qr := payments.BuildPhase1QR(p.LastUpdatedOn, p.Amount, vat)
	_, _ = c.repo.CreateInvoice(ctx.Context(), models.Invoice{
		OrderID: p.OrderID, PaymentID: p.ID, InvoiceNumber: invNo, ZatcaUUID: uuid,
		ZatcaPIH: hash, QRBase64: qr, Status: "draft", Subtotal: subtotal,
		VATAmount: vat, Total: p.Amount, Currency: p.Currency,
	})
}

// GetOrderInvoices returns invoices for an order.
func (c *PaymentV2Controller) GetOrderInvoices(ctx *fiber.Ctx) error {
	invs, err := c.repo.GetInvoicesByOrder(ctx.Context(), ctx.Params("orderId"))
	if err != nil {
		return response.Error(ctx, fiber.StatusInternalServerError, "INVOICE_FAILED", err.Error())
	}
	return response.Success(ctx, invs)
}

// GetPayment returns a payment by id, or (when ?order_id= is set) the list of
// payments for an order.
func (c *PaymentV2Controller) GetPayment(ctx *fiber.Ctx) error {
	p, err := c.repo.GetByID(ctx.Context(), ctx.Params("id"))
	if err != nil || p == nil {
		return response.Error(ctx, fiber.StatusNotFound, "NOT_FOUND", "payment not found")
	}
	return response.Success(ctx, p)
}

// ListPayments returns payments for an order (?order_id=).
func (c *PaymentV2Controller) ListPayments(ctx *fiber.Ctx) error {
	orderID := ctx.Query("order_id")
	if orderID == "" {
		return response.Error(ctx, fiber.StatusBadRequest, "BAD_REQUEST", "order_id required")
	}
	list, err := c.repo.ListByOrder(ctx.Context(), orderID)
	if err != nil {
		return response.Error(ctx, fiber.StatusInternalServerError, "LIST_FAILED", err.Error())
	}
	return response.Success(ctx, list)
}

// CODCollect marks driver cash collection (delivery app).
func (c *PaymentV2Controller) CODCollect(ctx *fiber.Ctx) error {
	driverID, _ := ctx.Locals("user_id").(string)
	p, err := c.repo.SetCODCollected(ctx.Context(), ctx.Params("id"), driverID)
	if err != nil {
		return response.Error(ctx, fiber.StatusBadRequest, "COD_COLLECT_FAILED", err.Error())
	}
	c.generateInvoice(ctx, p)
	return response.Success(ctx, p)
}

// CODReconcile marks a collected COD payment reconciled (ops/finance).
func (c *PaymentV2Controller) CODReconcile(ctx *fiber.Ctx) error {
	p, err := c.repo.SetCODReconciled(ctx.Context(), ctx.Params("id"))
	if err != nil {
		return response.Error(ctx, fiber.StatusBadRequest, "COD_RECONCILE_FAILED", err.Error())
	}
	return response.Success(ctx, p)
}

type refundReq struct {
	Amount float64 `json:"amount"`
	Reason string  `json:"reason"`
}

// Refund issues a full/partial refund (the only way to reduce a charge —
// we never raise a second/higher charge).
func (c *PaymentV2Controller) Refund(ctx *fiber.Ctx) error {
	var req refundReq
	if err := ctx.BodyParser(&req); err != nil {
		return response.Error(ctx, fiber.StatusBadRequest, "BAD_REQUEST", "invalid body")
	}
	p, err := c.repo.GetByID(ctx.Context(), ctx.Params("id"))
	if err != nil || p == nil {
		return response.Error(ctx, fiber.StatusNotFound, "NOT_FOUND", "payment not found")
	}
	if req.Amount <= 0 || req.Amount > (p.AmountCaptured-p.AmountRefunded) {
		return response.Error(ctx, fiber.StatusBadRequest, "BAD_AMOUNT", "refund exceeds captured balance")
	}
	// Gateway refund for prepaid (COD refunds are handled operationally / cash).
	if !p.IsCOD {
		gw := c.gateways[p.Gateway]
		if gw != nil {
			if _, rerr := gw.Refund(ctx.Context(), p.GatewayTxnID, req.Amount); rerr != nil {
				return response.Error(ctx, fiber.StatusBadGateway, "REFUND_FAILED", rerr.Error())
			}
		}
	}
	newRefunded := p.AmountRefunded + req.Amount
	toStatus := models.StatusPartiallyRefunded
	if newRefunded >= p.AmountCaptured {
		toStatus = models.StatusRefunded
	}
	out, err := c.repo.Transition(ctx.Context(), p.ID, toStatus, repository.TransitionOpts{
		EventType: models.EntryRefund, RefundAmount: req.Amount,
		LedgerEntry: models.EntryRefund, LedgerDir: models.DirDebit,
	})
	if err != nil {
		return response.Error(ctx, fiber.StatusInternalServerError, "REFUND_STATE_FAILED", err.Error())
	}
	return response.Success(ctx, out)
}

// Webhook is the provider callback (POST /payments/webhook/:gateway) — PUBLIC
// (no JWT; providers call it). It verifies the signature, dedupes by
// gateway_event_id (#8 idempotency), and applies the result via the FSM. This
// is also the canonical 3DS-resume path (pending_3ds → captured). Out-of-order
// or already-applied events are acknowledged with 200 and no-op'd rather than
// erroring, so providers don't retry forever.
func (c *PaymentV2Controller) Webhook(ctx *fiber.Ctx) error {
	gwName := ctx.Params("gateway")
	gw := c.gateways[gwName]
	if gw == nil {
		return response.Error(ctx, fiber.StatusNotFound, "NO_GATEWAY", "unknown gateway")
	}
	sig := ctx.Get("X-Signature")
	ev, err := gw.VerifyWebhook(sig, ctx.Body())
	if err != nil {
		return response.Error(ctx, fiber.StatusBadRequest, "BAD_SIGNATURE", err.Error())
	}

	// Idempotency: a re-delivered event is acknowledged without reprocessing.
	if seen, _ := c.repo.WebhookEventSeen(ctx.Context(), ev.GatewayEventID); seen {
		return response.SuccessWithMessage(ctx, "duplicate_ignored", fiber.Map{"event_id": ev.GatewayEventID})
	}

	p, err := c.repo.GetByGatewayTxn(ctx.Context(), ev.TxnID)
	if err != nil || p == nil {
		return response.Error(ctx, fiber.StatusNotFound, "UNKNOWN_TXN", "no payment for txn")
	}

	var toStatus string
	opts := repository.TransitionOpts{
		EventType: "webhook", GatewayEventID: ev.GatewayEventID, Signature: true,
	}
	switch ev.Status {
	case models.StatusCaptured:
		toStatus = models.StatusCaptured
		opts.CaptureAmount = p.Amount
		opts.LedgerEntry = models.EntryCapture
		opts.LedgerDir = models.DirCredit
	case models.StatusFailed:
		toStatus = models.StatusFailed
	case models.StatusRefunded:
		toStatus = models.StatusRefunded
		opts.RefundAmount = p.Amount - p.AmountRefunded
		opts.LedgerEntry = models.EntryRefund
		opts.LedgerDir = models.DirDebit
	default:
		return response.SuccessWithMessage(ctx, "ignored_status", fiber.Map{"status": ev.Status})
	}

	// FSM guard: if the event can't legally apply (e.g. capture after refund,
	// or a late duplicate), acknowledge without changing state.
	if !models.CanTransition(p.Status, toStatus) {
		return response.SuccessWithMessage(ctx, "out_of_order_ignored",
			fiber.Map{"current": p.Status, "event": toStatus})
	}
	out, terr := c.repo.Transition(ctx.Context(), p.ID, toStatus, opts)
	if terr != nil {
		return response.Error(ctx, fiber.StatusInternalServerError, "APPLY_FAILED", terr.Error())
	}
	if toStatus == models.StatusCaptured {
		c.generateInvoice(ctx, out)
	}
	return response.Success(ctx, out)
}

// GetReconConfig returns the reconciliation cadence config.
func (c *PaymentV2Controller) GetReconConfig(ctx *fiber.Ctx) error {
	cfg, err := c.repo.GetReconConfig(ctx.Context())
	if err != nil {
		return response.Error(ctx, fiber.StatusInternalServerError, "CONFIG_FAILED", err.Error())
	}
	return response.Success(ctx, cfg)
}

type reconConfigReq struct {
	Cadence   string `json:"cadence"` // daily|weekly|monthly
	RunAtHour int    `json:"run_at_hour"`
	Timezone  string `json:"timezone"`
	Enabled   *bool  `json:"enabled"`
}

// UpdateReconConfig sets the reconciliation cadence (daily|weekly|monthly).
func (c *PaymentV2Controller) UpdateReconConfig(ctx *fiber.Ctx) error {
	var req reconConfigReq
	if err := ctx.BodyParser(&req); err != nil {
		return response.Error(ctx, fiber.StatusBadRequest, "BAD_REQUEST", "invalid body")
	}
	cur, _ := c.repo.GetReconConfig(ctx.Context())
	if req.Cadence == "" {
		req.Cadence = cur.Cadence
	}
	if req.Timezone == "" {
		req.Timezone = cur.Timezone
	}
	enabled := cur.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	cfg, err := c.repo.UpdateReconConfig(ctx.Context(), req.Cadence, req.RunAtHour, req.Timezone, enabled)
	if err != nil {
		return response.Error(ctx, fiber.StatusBadRequest, "CONFIG_FAILED", err.Error())
	}
	return response.Success(ctx, cfg)
}

// ListReconRuns returns recent reconciliation runs.
func (c *PaymentV2Controller) ListReconRuns(ctx *fiber.Ctx) error {
	runs, err := c.repo.ListReconRuns(ctx.Context(), 50)
	if err != nil {
		return response.Error(ctx, fiber.StatusInternalServerError, "RUNS_FAILED", err.Error())
	}
	return response.Success(ctx, runs)
}

// RunReconNow triggers an immediate reconciliation for the current cadence
// window (ops/testing).
func (c *PaymentV2Controller) RunReconNow(ctx *fiber.Ctx) error {
	cfg, _ := c.repo.GetReconConfig(ctx.Context())
	end := time.Now()
	start := end.AddDate(0, 0, -1)
	switch cfg.Cadence {
	case "weekly":
		start = end.AddDate(0, 0, -7)
	case "monthly":
		start = end.AddDate(0, -1, 0)
	}
	run, err := c.repo.RunReconciliation(ctx.Context(), cfg.Cadence, start, end)
	if err != nil {
		return response.Error(ctx, fiber.StatusInternalServerError, "RECON_FAILED", err.Error())
	}
	return response.Success(ctx, run)
}
