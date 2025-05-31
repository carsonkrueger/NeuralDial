package services

import (
	gctx "context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/carsonkrueger/main/context"
	"github.com/carsonkrueger/main/models"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type webSocketService struct {
	context.ServiceContext
}

func NewWebSocketService(ctx context.ServiceContext) *webSocketService {
	return &webSocketService{
		ctx,
	}
}

func (ws *webSocketService) StartSocket(conn *websocket.Conn, handler context.WebSocketHandler, opts *models.WebSocketOptions) {
	opts.HandleDefaults()

	lgr := ws.Lgr("StartSocket")
	ctx, cancel := gctx.WithCancel(gctx.Background())
	ctx = context.WithCancel(ctx, cancel)
	mutex := sync.Mutex{}
	done := make(chan bool)
	defer close(done)

	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(*opts.PongDeadline))
	})

	go func() {
	outer:
		for {
			select {
			case <-done:
				return
			default:
				mutex.Lock()
				err := conn.WriteMessage(websocket.PingMessage, nil)
				mutex.Unlock()
				if err != nil {
					lgr.Warn("No pong")
					done <- true
					break outer
				}
				time.Sleep(*opts.PongInterval)
			}
		}
	}()

outer:
	for {
		select {
		case <-done:
			break
		default:
			msgType, reqBytes, err := conn.ReadMessage()
			if err != nil {
				lgr.Warn("No messages, closing connection...", zap.Error(err))
				done <- true
				break outer
			}

			if len(opts.AllowedMessageTypes) > 0 && !slices.Contains(opts.AllowedMessageTypes, msgType) {
				lgr.Warn(fmt.Sprintf("Invalid websocket message type: %d", msgType))
				closeMsg := websocket.FormatCloseMessage(websocket.CloseUnsupportedData, "Unsupported message type")
				mutex.Lock()
				conn.WriteMessage(websocket.CloseMessage, closeMsg)
				mutex.Unlock()
				done <- true
				break outer
			}

			resType, resBytes, err := handler.HandleRequest(ctx, msgType, reqBytes)
			if err != nil {
				lgr.Error("Could not handle request", zap.Error(err))
				if opts.CloseOnHandleError {
					done <- true
					break outer
				}
				continue
			}

			if resType == nil || resBytes == nil {
				continue
			}

			mutex.Lock()
			err = conn.WriteMessage(*resType, resBytes)
			mutex.Unlock()
			if err != nil {
				lgr.Error("Could not write message to connection", zap.Error(err))
				continue
			}
		}
	}

	handler.HandleClose()
}

func (ws *webSocketService) StartStreamingResponseSocket(conn *websocket.Conn, handler context.WebSocketHandler, opts *models.WebSocketOptions) {
	opts.HandleDefaults()

	ctx, cancel := gctx.WithCancel(gctx.Background())
	ctx = context.WithCancel(ctx, cancel)
	incoming := make(chan []byte)
	var writeMutex sync.Mutex

	conn.SetReadDeadline(time.Now().Add(opts.KeepAliveDuration))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(opts.KeepAliveDuration))
	})

	// PING PONG Handler
	go func() {
		ticker := time.NewTicker(*opts.PongInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(*opts.PongDeadline))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					cancel()
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Reader goroutine
	go func() {
		for {
			select {
			default:
				msgType, msg, err := conn.ReadMessage() // 1 reader - no mutex
				if err != nil {
					cancel()
					return
				}
				if len(opts.AllowedMessageTypes) > 0 && !slices.Contains(opts.AllowedMessageTypes, msgType) {
					cancel()
					return
				}
				incoming <- msg
			case <-ctx.Done():
				return
			}
		}
	}()

	// Main loop: Process client messages and optionally send server responses
outer:
	for {
		select {
		case msg := <-incoming:
			ch := make(chan models.StreamResponse)
			go func() {
				for v := range ch {
					if v.Err != nil {
						if opts.CloseOnHandleError {
							cancel()
							return
						}
						continue
					}
					if v.MsgType == nil || v.Data == nil {
						if v.Done {
							return
						}
						continue
					}
					writeMutex.Lock()
					err := conn.WriteMessage(*v.MsgType, v.Data)
					writeMutex.Unlock()
					if err != nil {
						cancel()
						return
					}
					if v.Done {
						return
					}
				}
			}()
			go func() {
				defer close(ch)
				handler.HandleRequestWithStreaming(ctx, msg, ch)
			}()
		case <-ctx.Done():
			break outer
		}
	}
}
