package services

import (
	gctx "context"
	"encoding/json"

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

func (ws *webSocketService) StartStreamingResponseSocket(conn *websocket.Conn, handler context.StreamingSocketHandler) {
	lgr := ws.Lgr("StartStreamingResponseSocket")
	ctx, cancel := gctx.WithCancel(gctx.Background())
	ctx = context.WithCancel(ctx, cancel)
	incoming := make(chan []byte)
	outgoing := make(chan models.StreamingResponse[models.StreamingResponseBody])

	// Reader goroutine
	go func() {
		for {
			select {
			default:
				_, msg, err := conn.ReadMessage()
				if err != nil {
					cancel()
					lgr.Warn("Closing connection - failed read")
					return
				}
				lgr.Debug("Received msg", zap.Int("size", len(msg)))
				incoming <- msg
			case <-ctx.Done():
				lgr.Info("ws service: reader done")
				return
			}
		}
	}()

	// writer goroutine
	go func() {
		defer close(outgoing)
		for {
			select {
			case res := <-outgoing:
				bytes, err := json.Marshal(res.Data)
				if err != nil {
					lgr.Warn("Closing connection - failed marshal")
					return
				}
				lgr.Debug("Sending msg", zap.Int("size", len(bytes)))
				err = conn.WriteMessage(res.Type, bytes)
				if err != nil {
					lgr.Warn("Closing connection - failed write")
					return
				}
			case <-ctx.Done():
				lgr.Info("ws service: writer done")
				return
			}
		}
	}()

	// main loop
	handler.HandleRequestWithStreaming(ctx, incoming, outgoing)
}
