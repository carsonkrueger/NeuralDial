package services

import (
	"context"
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
	llmService := ws.SM().LLMService()
	agent, memoryBuffer, err := llmService.NewConversationalAgent(nil)
	if err != nil {
		lgr.Error("Failed to create agent", zap.Error(err))
		return
	}

	done := make(chan bool)
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
					done <- true
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
			ctx := context.Background()

			msgType, reqBytes, err := conn.ReadMessage()
			if err != nil {
				lgr.Warn("No messages, closing connection...", zap.Error(err))
				done <- true
				break outer
			}

			if msgType != websocket.TextMessage {
				lgr.Warn("Invalid websocket message type")
				closeMsg := websocket.FormatCloseMessage(websocket.CloseUnsupportedData, "Unsupported message type")
				mutex.Lock()
				conn.WriteMessage(websocket.CloseMessage, closeMsg)
				mutex.Unlock()
				done <- true
				break outer
			}

			msg := string(reqBytes)

			res, err := llmService.Generate(ctx, agent, memoryBuffer, msg)
			if err != nil {
				lgr.Error("Could not generate LLM response", zap.Error(err))
				continue
			}
			resBytes := []byte(res)

			// resBytes, err := r.SM().VoiceService().TextToSpeech(llmRes)
			// if err != nil {
			// 	lgr.Error("Could not generate Text To Speech response", zap.Error(err))
			// 	continue
			// }

			mutex.Lock()
			err = conn.WriteMessage(websocket.BinaryMessage, resBytes)
			mutex.Unlock()
			if err != nil {
				lgr.Error("Could not write message to connection", zap.Error(err))
				continue
			}
		}
	}
}
