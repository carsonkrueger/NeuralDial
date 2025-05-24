package models

import (
	"time"

	"github.com/carsonkrueger/main/tools"
)

type WebSocketOptions struct {
	PongDeadline       *time.Duration
	PongInterval       *time.Duration
	CloseOnHandleError bool
	// if allowed is empty, all message types are allowed
	AllowedMessageTypes []int
}

func (opts *WebSocketOptions) HandleDefaults() {
	if opts == nil {
		opts = &WebSocketOptions{}
	}
	if opts.PongInterval == nil {
		opts.PongInterval = tools.Ptr(10 * time.Second)
	}
}
