package utils

import "testing"

// TestGenerateAndParseJWT_Roundtrip covers the core auth primitive: a token
// this package signs must be parseable by itself, and must carry both the
// user id (sub) and role (user_type) claims — RoleMiddleware/AdminOnly/etc
// silently allow nothing if user_type is missing (this was a real bug once,
// per the comment in GenerateJWT).
func TestGenerateAndParseJWT_Roundtrip(t *testing.T) {
	token, err := GenerateJWT("USR123", "customer")
	if err != nil {
		t.Fatalf("GenerateJWT returned error: %v", err)
	}
	if token == "" {
		t.Fatal("GenerateJWT returned an empty token")
	}

	claims, err := ParseJWT(token)
	if err != nil {
		t.Fatalf("ParseJWT returned error on a token this package just signed: %v", err)
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub != "USR123" {
		t.Errorf("claims[sub] = %v, want %q", claims["sub"], "USR123")
	}
	userType, ok := claims["user_type"].(string)
	if !ok || userType != "customer" {
		t.Errorf("claims[user_type] = %v, want %q", claims["user_type"], "customer")
	}
}

func TestParseJWT_RejectsGarbage(t *testing.T) {
	if _, err := ParseJWT("not-a-real-token"); err == nil {
		t.Fatal("ParseJWT accepted a malformed token")
	}
}

func TestParseJWT_RejectsTamperedToken(t *testing.T) {
	token, err := GenerateJWT("USR123", "customer")
	if err != nil {
		t.Fatalf("GenerateJWT returned error: %v", err)
	}
	// Flip the last character of the signature — must not still verify.
	tampered := token[:len(token)-1] + "x"
	if tampered == token {
		t.Skip("token ended in 'x' already, cannot construct a tampered variant this way")
	}
	if _, err := ParseJWT(tampered); err == nil {
		t.Fatal("ParseJWT accepted a token with a tampered signature")
	}
}
