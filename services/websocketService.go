package services

import (
	gctx "context"
	"io"

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
	lgr := ws.Lgr("StartStreamingResponseSocket")
	ctx, cancel := gctx.WithCancel(gctx.Background())
	ctx = context.WithCancel(ctx, cancel)
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
				_, msg, err := conn.ReadMessage()
				if err != nil {
					cancel()
					return
				}
				_, err = incomingW.Write(msg)
				if err != nil {
					return
				}
			case <-ctx.Done():
				lgr.Info("websocketService: context done")
				return
			}
		}
	}()

	// writer goroutine
	go func() {
		for {
			select {
			default:
				n, err := outgoingR.Read(buf)
				if err != nil {
					cancel()
					return
				}
				err = conn.WriteMessage(websocket.BinaryMessage, buf[:n])
				if err != nil {
					return
				}
			case <-ctx.Done():
				lgr.Info("websocketService: context done")
				return
			}
		}
	}()

	// main loop
	handler.HandleRequestWithStreaming(ctx, incomingR, outgoingW)
}
