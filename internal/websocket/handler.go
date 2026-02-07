package websocket

import (
	"log"

	"github.com/gofiber/websocket/v2"
)

// TrackDeliveryHandler creates a WebSocket handler for delivery tracking
func TrackDeliveryHandler(hub *Hub) func(*websocket.Conn) {
	return func(c *websocket.Conn) {
		orderID := c.Params("orderId")
		if orderID == "" {
			log.Println("WebSocket: Missing order ID")
			c.Close()
			return
		}

		// Register client
		hub.Register(orderID, c)
		log.Printf("WebSocket: Client connected for order %s", orderID)

		defer func() {
			hub.Unregister(orderID, c)
			c.Close()
			log.Printf("WebSocket: Client disconnected for order %s", orderID)
		}()

		// Send initial connection confirmation
		if err := c.WriteJSON(map[string]interface{}{
			"type":     "connected",
			"order_id": orderID,
			"message":  "Connected to delivery tracking",
		}); err != nil {
			log.Printf("WebSocket: Failed to send connection confirmation: %v", err)
			return
		}

		// Keep connection alive and listen for client messages (ping/pong)
		for {
			messageType, msg, err := c.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					log.Printf("WebSocket: Connection closed normally for order %s", orderID)
				} else {
					log.Printf("WebSocket: Read error for order %s: %v", orderID, err)
				}
				break
			}

			// Handle ping messages
			if messageType == websocket.PingMessage {
				if err := c.WriteMessage(websocket.PongMessage, nil); err != nil {
					log.Printf("WebSocket: Failed to send pong: %v", err)
					break
				}
			}

			// Echo back text messages for testing (optional)
			if messageType == websocket.TextMessage {
				log.Printf("WebSocket: Received message from client: %s", string(msg))
			}
		}
	}
}

// UpgradeConfig returns the WebSocket upgrade configuration
func UpgradeConfig() websocket.Config {
	return websocket.Config{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		Origins:         []string{"*"}, // Configure for production
	}
}
