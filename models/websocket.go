package models

import (
	"encoding/json"
	"time"

	"github.com/carsonkrueger/main/tools"
)

type WebSocketOptions struct {
	KeepAliveDuration  time.Duration
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
	if opts.PongDeadline == nil {
		opts.PongDeadline = tools.Ptr(3 * time.Second)
	}
	if opts.KeepAliveDuration == 0 {
		opts.KeepAliveDuration = 10 * time.Minute
	}
	if opts.PongInterval == nil {
		opts.PongInterval = tools.Ptr(10 * time.Second)
	}
}

type StreamingResponse[T json.Marshaler] struct {
	Type int
	Data T
}

type StreamingReader <-chan []byte
type StreamingWriter[T json.Marshaler] chan<- StreamingResponse[T]

type StreamingResponseBodyType string

const (
	SR_AGENT_START StreamingResponseBodyType = "agent_start"
	SR_AGENT_SPEAK StreamingResponseBodyType = "agent_speak"
)

type StreamingResponseBody struct {
	Type StreamingResponseBodyType
	Data []byte
}

func (s StreamingResponseBody) MarshalJSON() ([]byte, error) {
	return json.Marshal(s)
}
