package private

import (
	"net/http"
	"sync"
	"time"

	"github.com/carsonkrueger/main/builders"
	"github.com/carsonkrueger/main/context"
	"github.com/carsonkrueger/main/models"
	"github.com/carsonkrueger/main/tools"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	WebTextGet = "WebTextGet"
	WebTextWS  = "WebTextWS"
)

type webText struct {
	context.AppContext
}

func NewWebText(ctx context.AppContext) *webText {
	return &webText{
		AppContext: ctx,
	}
}

func (um webText) Path() string {
	return "/web_text"
}

func (um *webText) PrivateRoute(b *builders.PrivateRouteBuilder) {
	b.NewHandle().Register(builders.GET, "/", um.webTextGet).SetPermissionName(WebTextGet).Build()
	b.NewHandle().Register(builders.GET, "/ws", um.textWebSocket).SetPermissionName(WebTextWS).Build()
}

func (r *webText) webTextGet(res http.ResponseWriter, req *http.Request) {
	lgr := r.Lgr("webTextGet")
	lgr.Info("Called")
}

func (r *webText) textWebSocket(res http.ResponseWriter, req *http.Request) {
	lgr := r.Lgr("textWebSocket")
	lgr.Info("Called")
	ctx := req.Context()
	mutex := sync.Mutex{}
	llmStreamingModel := models.LLMStreamingModel{}

	done := make(chan int)
	defer close(done)

	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		tools.HandleError(req, res, lgr, err, 500, "Error setting up websocket")
		return
	}
	defer conn.Close()

	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(3 * time.Minute))
	})

	go func() {
	outer:
		for {
			select {
			case <-done:
				return
			default:
				lgr.Info("Ping")
				mutex.Lock()
				err := conn.WriteMessage(websocket.PingMessage, nil)
				mutex.Unlock()
				if err != nil {
					lgr.Warn("No pong, closing connection...")
					done <- 1
					break outer
				}
				lgr.Info("Pong")
				time.Sleep(6 * time.Second)
			}
		}
	}()

outer:
	for {
		select {
		case <-done:
			break
		default:
			lgr.Info("Reading...")
			msgType, reqBytes, err := conn.ReadMessage()
			if err != nil {
				lgr.Warn("No messages, closing connection...", zap.Error(err))
				done <- 1
				break outer
			}

			if msgType != websocket.TextMessage {
				lgr.Warn("Invalid websocket message type")
				closeMsg := websocket.FormatCloseMessage(websocket.CloseUnsupportedData, "Unsupported message type")
				mutex.Lock()
				conn.WriteMessage(websocket.CloseMessage, closeMsg)
				mutex.Unlock()
				done <- 1
				break outer
			}

			msg := string(reqBytes)

			llmRes, err := r.SM().LLMService().Generate(ctx, &llmStreamingModel, msg)
			lgr.Info("Generate", zap.Any("Msg History:", llmStreamingModel.Messages()))
			if err != nil {
				lgr.Error("Could not generate LLM response", zap.Error(err))
				continue
			}

			mutex.Lock()
			err = conn.WriteMessage(websocket.BinaryMessage, []byte(llmRes))
			mutex.Unlock()
			if err != nil {
				lgr.Error("Could not write message to connection", zap.Error(err))
				continue
			}
		}
	}

	lgr.Info("Leaving...")
}
