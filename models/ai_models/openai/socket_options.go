package openai

import (
	"time"

	"github.com/carsonkrueger/main/models"
	"github.com/carsonkrueger/main/tools"
	"github.com/gorilla/websocket"
)

func NewTextOptions() models.WebSocketOptions {
	return models.WebSocketOptions{
		PongDeadline:        tools.Ptr(10 * time.Minute),
		PongInterval:        tools.Ptr(10 * time.Second),
		AllowedMessageTypes: []int{websocket.TextMessage},
	}
}
