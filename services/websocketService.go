package services

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type WebSocketService interface {
	StartConversation(conn *websocket.Conn)
}

type webSocketService struct {
	ServiceContext
}

func NewWebSocketService(ctx ServiceContext) *webSocketService {
	return &webSocketService{
		ctx,
	}
}

func (ws *webSocketService) StartConversation(conn *websocket.Conn) {
	lgr := ws.Lgr("StartConversation")
	mutex := sync.Mutex{}
	// llmStreamingModel := models.LLMStreamingModel{}

	done := make(chan int)
	defer close(done)

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

			if msgType != websocket.BinaryMessage {
				lgr.Warn("Invalid websocket message type")
				closeMsg := websocket.FormatCloseMessage(websocket.CloseUnsupportedData, "Unsupported message type")
				mutex.Lock()
				conn.WriteMessage(websocket.CloseMessage, closeMsg)
				mutex.Unlock()
				done <- 1
				break outer
			}

			// msg := string(reqBytes)

			// llmRes, err := r.SM().LLMService().Generate(ctx, &llmStreamingModel, msg)
			// lgr.Info("Generate", zap.Any("Msg History:", llmStreamingModel.Messages()))
			// if err != nil {
			// 	lgr.Error("Could not generate LLM response", zap.Error(err))
			// 	continue
			// }

			// resBytes, err := r.SM().VoiceService().TextToSpeech(llmRes)
			// if err != nil {
			// 	lgr.Error("Could not generate Text To Speech response", zap.Error(err))
			// 	continue
			// }

			mutex.Lock()
			err = conn.WriteMessage(websocket.BinaryMessage, reqBytes)
			mutex.Unlock()
			if err != nil {
				lgr.Error("Could not write message to connection", zap.Error(err))
				continue
			}
		}
	}
}
