package models

import "time"

// Order represents an order record in the database
type Order struct {
	ID               string    `json:"id"`                 // Order ID
	UserID           string    `json:"user_id"`            // User ID who placed the order
	RestaurantID     string    `json:"restaurant_id"`      // Restaurant ID
	Status           string    `json:"status"`             // Order status (e.g., Pending, Completed)
	TotalAmount      float64   `json:"total_amount"`       // Total order amount
	OrderDate        time.Time `json:"order_date"`         // Date and time when the order was placed
	DeliveryPersonID string    `json:"delivery_person_id"` // Delivery person ID
	CreatedOn        time.Time `json:"created_on"`         // Timestamp when the order was created
	LastUpdatedOn    time.Time `json:"last_updated_on"`    // Timestamp when the order was last updated
	CreatedBy        string    `json:"created_by"`         // ID of the user who created the order
	LastModifiedBy   string    `json:"last_modified_by"`   // ID of the user who last modified the order
}

type OrderDisplay struct {
    OrderID            string    `json:"orderId"`
    HotelName          string    `json:"hotelName"`
    HotelAddress       string    `json:"hotelAddress"`
    DeliveryAddressName string   `json:"deliveryAddressName"`
    DeliveryAddress    string    `json:"deliveryAddress"`
    OrderAmt           string    `json:"orderAmt"`
    OrderItems         []Item    `json:"orderItems"`
    OrderTime          string    `json:"orderTime"`
    OrderStatus        string    `json:"orderStatus"`
    DeliveryRating     int       `json:"deliveryRating"`
    FoodRating         int       `json:"foodRating"`
}

type Item struct {
    ItemName  string `json:"itemName"`
    Quantity  int    `json:"quantity"`
    ItemPrice int    `json:"itemPrice"`
}
