package services

import (
	"context"
	"errors"
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
	Err     error
	// id of data if any
	ID   *string
	Data []byte
}

type WebSocketHandler interface {
	HandleRequest(ctx context.Context, msgType int, req []byte) (*int, []byte, error)
	HandleRequestWithStreaming(ctx context.Context, req []byte, out chan<- StreamResponse)
	PreprocessRequest(ctx context.Context, req []byte)
	IsHandling() bool
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

	incoming := make(chan []byte)
	outgoing := make(chan StreamResponse)
	done := make(chan error)
	defer close(incoming)
	defer close(outgoing)
	defer close(done)

	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(*opts.PongDeadline))
	})

	// Reader goroutine
	go func() {
		defer close(done)
		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				done <- errors.New("error reading message")
				break
			}
			if len(opts.AllowedMessageTypes) > 0 && !slices.Contains(opts.AllowedMessageTypes, msgType) {
				done <- errors.New("invalid websocket message type")
				break
			}
			incoming <- msg
		}
	}()

	// Writer goroutine
	go func() {
		for {
			select {
			case v := <-outgoing:
				if v.Err != nil {
					done <- v.Err
				}
				if v.MsgType == nil || v.Data == nil {
					if v.Done {
						break
					}
					continue
				}
				err := conn.WriteMessage(*v.MsgType, v.Data)
				if err != nil {
					done <- errors.New("error writing message")
					return
				}
			case err := <-done:
				if err != nil {
					lgr.Error("Error with websocket", zap.Error(err))
				}
				return
			}
		}
	}()

	// Main loop: Process client messages and optionally send server responses
	for {
		select {
		case msg := <-incoming:
			ctx := context.Background()

			handler.PreprocessRequest(ctx, msg)
			if handler.IsHandling() {
				continue
			}

			go func() {
				ch := make(chan StreamResponse)
				go handler.HandleRequestWithStreaming(ctx, msg, ch)
				defer close(ch)
				for v := range ch {
					if v.Err != nil {
						if opts.CloseOnHandleError {
							done <- errors.New("error handling stream request")
							break
						}
						continue
					}
					if v.MsgType == nil || v.Data == nil {
						if v.Done {
							break
						}
						continue
					}
					outgoing <- v
					if v.Done {
						break
					}
				}
			}()
		case err := <-done:
			if err != nil {
				lgr.Error("error with websocket", zap.Error(err))
			}
			break
		}
	}
}
