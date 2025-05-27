package services

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/carsonkrueger/main/models"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type WebSocketService interface {
	StartSocket(conn *websocket.Conn, handler WebSocketHandler, opts *models.WebSocketOptions)
	StartStreamingResponseSocket(conn *websocket.Conn, handler WebSocketHandler, opts *models.WebSocketOptions)
}

type webSocketService struct {
	ServiceContext
}

func NewWebSocketService(ctx ServiceContext) *webSocketService {
	return &webSocketService{
		ctx,
	}
}

type StreamResponse struct {
	MsgType *int
	Done    bool
	// id of data if any
	ID   *string
	Data []byte
}

type WebSocketHandler interface {
	HandleRequest(ctx context.Context, msgType int, req []byte) (*int, []byte, error)
	HandleRequestWithStreaming(ctx context.Context, msgType int, req []byte, out chan<- StreamResponse) error
	HandleClose()
}

func (ws *webSocketService) StartSocket(conn *websocket.Conn, handler WebSocketHandler, opts *models.WebSocketOptions) {
	opts.HandleDefaults()

	lgr := ws.Lgr("StartSocket")
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
			ctx := context.Background()
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

func (ws *webSocketService) StartStreamingResponseSocket(conn *websocket.Conn, handler WebSocketHandler, opts *models.WebSocketOptions) {
	opts.HandleDefaults()

	lgr := ws.Lgr("StartSocket")
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
			ctx := context.Background()
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

			ch := make(chan StreamResponse)
			go func() {
				handler.HandleRequestWithStreaming(ctx, msgType, reqBytes, ch)
				if err != nil {
					lgr.Error("Could not handle request with streaming", zap.Error(err))
					if opts.CloseOnHandleError {
						done <- true
					}
				}
			}()

			for v := range ch {
				if v.MsgType == nil || v.Data == nil {
					if v.Done {
						break
					}
					continue
				}

				mutex.Lock()
				err = conn.WriteMessage(*v.MsgType, v.Data)
				mutex.Unlock()
				if err != nil {
					lgr.Error("Could not write message to connection", zap.Error(err))
					continue
				}
				if v.Done {
					break
				}
			}
			close(ch)
		}
	}

	handler.HandleClose()
}
