package payments

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"time"
)

// ZATCA e-invoicing helpers (KSA). This implements the Phase-1 (simplified
// invoice) TLV QR correctly; the Phase-2 cryptographic stamp + clearance/
// reporting API integration is the tracked TODO (docs/PAYMENTS_REDESIGN.md §7).

// SellerName / SellerVAT come from env (fall back to placeholders for dev).
func sellerName() string {
	if v := os.Getenv("ZATCA_SELLER_NAME"); v != "" {
		return v
	}
	return "Dailzo"
}

func sellerVAT() string {
	if v := os.Getenv("ZATCA_VAT_NUMBER"); v != "" {
		return v
	}
	return "300000000000003" // placeholder 15-digit KSA VAT number
}

// VATBreakdown splits a VAT-inclusive total into subtotal + VAT at the KSA rate.
func VATBreakdown(total, rate float64) (subtotal, vat float64) {
	round := func(f float64) float64 { return math.Round(f*100) / 100 }
	vat = round(total * rate / (1 + rate))
	subtotal = round(total - vat)
	return
}

// tlv appends one TLV field (tag, length, value) to buf.
func tlv(buf []byte, tag byte, value string) []byte {
	v := []byte(value)
	buf = append(buf, tag, byte(len(v)))
	return append(buf, v...)
}

// BuildPhase1QR returns the base64-encoded ZATCA Phase-1 TLV QR payload with the
// five mandatory tags: seller name, VAT number, timestamp, invoice total, VAT total.
func BuildPhase1QR(timestamp time.Time, total, vat float64) string {
	var buf []byte
	buf = tlv(buf, 1, sellerName())
	buf = tlv(buf, 2, sellerVAT())
	buf = tlv(buf, 3, timestamp.UTC().Format(time.RFC3339))
	buf = tlv(buf, 4, fmt.Sprintf("%.2f", total))
	buf = tlv(buf, 5, fmt.Sprintf("%.2f", vat))
	return base64.StdEncoding.EncodeToString(buf)
}

// InvoiceHash computes a SHA-256 hex hash of the core invoice fields. Used to
// seed the PIH (previous-invoice-hash) chain. NOTE: the real ZATCA hash is over
// the canonicalized UBL XML — this is a scaffold placeholder.
func InvoiceHash(invoiceNumber, prevHash string, total float64, ts time.Time) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%.2f|%s", invoiceNumber, prevHash, total, ts.UTC().Format(time.RFC3339))))
	return hex.EncodeToString(h[:])
}

// NewUUID returns a random RFC-4122 v4 UUID string (no external dep).
func NewUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
