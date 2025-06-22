package services

import (
	gctx "context"
	"io"
	"slices"

	"github.com/carsonkrueger/main/context"
	"github.com/gorilla/websocket"
)

type webSocketService struct {
	context.ServiceContext
}

func NewWebSocketService(ctx context.ServiceContext) *webSocketService {
	return &webSocketService{
		ctx,
	}
}

func (ws *webSocketService) StartStreamingResponseSocket(conn *websocket.Conn, handler context.StreamingSocketHandler) {
	opts := handler.Options()
	opts.HandleDefaults()

	ctx, cancel := gctx.WithCancel(gctx.Background())
	ctx = context.WithCancel(ctx, cancel)
	// incoming := make(chan []byte, 10)
	// outgoing := make(chan models.StreamResponse, 10)
	incomingR, incomingW := io.Pipe()
	outgoingR, outgoingW := io.Pipe()
	buf := make([]byte, 1024*100) // 100 KB

	// conn.SetReadDeadline(time.Now().Add(opts.KeepAliveDuration))
	// conn.SetPongHandler(func(string) error {
	// 	return conn.SetReadDeadline(time.Now().Add(opts.KeepAliveDuration))
	// })

	// // PING PONG Handler
	// go func() {
	// 	ticker := time.NewTicker(*opts.PongInterval)
	// 	defer ticker.Stop()
	// 	for {
	// 		select {
	// 		case <-ticker.C:
	// 			_ = conn.SetWriteDeadline(time.Now().Add(*opts.PongDeadline))
	// 			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
	// 				cancel()
	// 				return
	// 			}
	// 		case <-ctx.Done():
	// 			return
	// 		}
	// 	}
	// }()

	// Reader goroutine
	go func() {
		defer incomingW.Close()
		for {
			select {
			default:
				msgType, msg, err := conn.ReadMessage()
				if err != nil {
					cancel()
					return
				}
				if len(opts.AllowedMessageTypes) > 0 && !slices.Contains(opts.AllowedMessageTypes, msgType) {
					cancel()
					return
				}
				_, err = incomingW.Write(msg)
				if err != nil {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// writer goroutine
	go func() {
		for {
			select {
			default:
				_, err := outgoingR.Read(buf)
				if err != nil {
					cancel()
					return
				}
				err = conn.WriteMessage(websocket.BinaryMessage, buf)
				if err != nil {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	handler.HandleRequestWithStreaming(ctx, incomingR, outgoingW)
}
