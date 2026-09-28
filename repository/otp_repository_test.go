package repository

import (
	"context"
	"dailzo/config"
	"dailzo/db"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testDB connects using the same config the running server uses (DB_HOST /
// DB_PORT / etc, same as `go run main.go`). There's no separate test
// database in this project yet, so these run as integration tests against
// whatever Postgres the app itself points at — skipped automatically if it
// isn't reachable, so `go test ./...` still passes in an environment with no
// DB configured (e.g. a fresh checkout with nothing running yet).
func testDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	cfg := config.LoadConfig()
	db.ConnectDatabase(cfg)
	if db.DB == nil {
		t.Skip("no reachable database — skipping repository integration test")
	}
	return db.DB
}

// seedTestUser inserts a throwaway user row so otps.user_id's FK is satisfied,
// and returns its id.
func seedTestUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	id := GetIdToRecord("TESTUSR")
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, username, email, password, user_type, created_on, last_updated_on)
		 VALUES ($1, $2, $3, 'x', 'customer', $4, $4)`,
		id, id, id+"@test.local", time.Now())
	if err != nil {
		t.Fatalf("seed test user: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	})
	return id
}

func TestGenerateOTP_IsSixDigits(t *testing.T) {
	r := NewOTPRepository(nil)
	for i := 0; i < 20; i++ {
		code := r.GenerateOTP()
		if len(code) != 6 {
			t.Fatalf("GenerateOTP() = %q, want a 6-digit string", code)
		}
		for _, c := range code {
			if c < '0' || c > '9' {
				t.Fatalf("GenerateOTP() = %q, contains a non-digit", code)
			}
		}
	}
}

func TestCreateOTP_RateLimitsAfterMax(t *testing.T) {
	pool := testDB(t)
	r := NewOTPRepository(pool)
	userID := seedTestUser(t, pool)
	ctx := context.Background()

	for i := 0; i < otpSendMax; i++ {
		if _, err := r.CreateOTP(ctx, userID, "mobile"); err != nil {
			t.Fatalf("CreateOTP #%d: unexpected error: %v", i+1, err)
		}
	}

	_, err := r.CreateOTP(ctx, userID, "mobile")
	if !errors.Is(err, ErrOTPRateLimited) {
		t.Fatalf("CreateOTP after %d sends: got err=%v, want ErrOTPRateLimited", otpSendMax, err)
	}
}

func TestVerifyOTP_LocksAfterMaxAttempts(t *testing.T) {
	pool := testDB(t)
	r := NewOTPRepository(pool)
	userID := seedTestUser(t, pool)
	ctx := context.Background()

	if _, err := r.CreateOTP(ctx, userID, "mobile"); err != nil {
		t.Fatalf("CreateOTP: unexpected error: %v", err)
	}

	for i := 0; i < otpVerifyAttempt; i++ {
		valid, err := r.VerifyOTP(ctx, userID, "000000") // deliberately wrong
		if err != nil {
			t.Fatalf("VerifyOTP attempt %d: unexpected error: %v", i+1, err)
		}
		if valid {
			t.Fatalf("VerifyOTP attempt %d: matched a code that was never issued", i+1)
		}
	}

	_, err := r.VerifyOTP(ctx, userID, "000000")
	if !errors.Is(err, ErrOTPLocked) {
		t.Fatalf("VerifyOTP after %d wrong attempts: got err=%v, want ErrOTPLocked", otpVerifyAttempt, err)
	}
}

func TestVerifyOTP_AcceptsCorrectCode(t *testing.T) {
	pool := testDB(t)
	r := NewOTPRepository(pool)
	userID := seedTestUser(t, pool)
	ctx := context.Background()

	otp, err := r.CreateOTP(ctx, userID, "mobile")
	if err != nil {
		t.Fatalf("CreateOTP: unexpected error: %v", err)
	}

	valid, err := r.VerifyOTP(ctx, userID, otp.OTPCode)
	if err != nil {
		t.Fatalf("VerifyOTP: unexpected error: %v", err)
	}
	if !valid {
		t.Fatal("VerifyOTP: correct code was rejected")
	}

	// A used OTP must not verify again.
	valid, err = r.VerifyOTP(ctx, userID, otp.OTPCode)
	if err == nil && valid {
		t.Fatal("VerifyOTP: an already-used OTP verified a second time")
	}
}
