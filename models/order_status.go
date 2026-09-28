package models

// ValidOrderStatuses is the single allow-list for order.status writes. Both
// the customer-facing PATCH /orders/{id}/status and the vendor-facing PATCH
// /vendor/orders/:id/status must go through IsValidOrderStatus so neither path
// can write a status the other doesn't recognize.
var ValidOrderStatuses = map[string]bool{
	"pending":    true,
	"confirmed":  true,
	"preparing":  true,
	"ready":      true,
	"picked_up":  true,
	"in_transit": true,
	"delivered":  true,
	"cancelled":  true,
}

func IsValidOrderStatus(status string) bool {
	return ValidOrderStatuses[status]
}
