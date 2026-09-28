package repository

import (
	"context"
	"testing"
)

// TestCreateRegistration_OwnershipEnforced covers the fix for the IDOR where
// PUT /restaurant/:id/payment etc had no auth at all: a second user must not
// be able to update (or silently take over) a registration id created by
// someone else.
func TestCreateRegistration_OwnershipEnforced(t *testing.T) {
	pool := testDB(t)
	r := NewRegistrationRepository(pool)
	ctx := context.Background()

	ownerID := "TESTUSR_owner"
	otherID := "TESTUSR_other"

	created, err := r.CreateRegistration(ctx, ownerID, map[string]interface{}{"name": "Test Diner"})
	if err != nil {
		t.Fatalf("CreateRegistration: unexpected error: %v", err)
	}
	regID, _ := created["id"].(string)
	if regID == "" {
		t.Fatal("CreateRegistration did not return an id")
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM restaurant_registrations WHERE id = $1`, regID)
	})

	// Owner can read and update their own registration.
	if _, err := r.GetRegistration(ctx, regID, ownerID); err != nil {
		t.Fatalf("GetRegistration by owner: unexpected error: %v", err)
	}
	if _, err := r.UpdateRegistration(ctx, regID, ownerID, map[string]interface{}{"step": 2}); err != nil {
		t.Fatalf("UpdateRegistration by owner: unexpected error: %v", err)
	}

	// A different user must be refused on both read and write.
	if _, err := r.GetRegistration(ctx, regID, otherID); err == nil {
		t.Fatal("GetRegistration succeeded for a non-owner — ownership check is not enforced")
	}
	if _, err := r.UpdateRegistration(ctx, regID, otherID, map[string]interface{}{"step": 99}); err == nil {
		t.Fatal("UpdateRegistration succeeded for a non-owner — ownership check is not enforced")
	}

	// Re-using the same id as a different user must not silently reassign it.
	if _, err := r.CreateRegistration(ctx, otherID, map[string]interface{}{"id": regID, "name": "Hijacked"}); err == nil {
		t.Fatal("CreateRegistration let a different user take over an existing registration id")
	}

	owned, err := r.IsRegistrationOwner(ctx, regID, ownerID)
	if err != nil {
		t.Fatalf("IsRegistrationOwner: unexpected error: %v", err)
	}
	if !owned {
		t.Fatal("IsRegistrationOwner reported false for the actual owner")
	}
	owned, err = r.IsRegistrationOwner(ctx, regID, otherID)
	if err != nil {
		t.Fatalf("IsRegistrationOwner: unexpected error: %v", err)
	}
	if owned {
		t.Fatal("IsRegistrationOwner reported true for a non-owner")
	}
}
