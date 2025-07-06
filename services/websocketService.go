package services

import (
	gctx "context"
	"encoding/json"

	"github.com/carsonkrueger/main/context"
	"github.com/carsonkrueger/main/models"
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
	incoming := make(chan []byte)
	outgoing := make(chan models.StreamingResponse[json.Marshaler])

	// Reader goroutine
	go func() {
		for {
			select {
			default:
				_, msg, err := conn.ReadMessage()
				if err != nil {
					cancel()
					return
				}
				incoming <- msg
			case <-ctx.Done():
				lgr.Info("websocketService: context done")
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
				bytes, err := res.Data.MarshalJSON()
				if err != nil {
					return
				}
				err = conn.WriteMessage(res.Type, bytes)
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
	handler.HandleRequestWithStreaming(ctx, incoming, outgoing)
}
