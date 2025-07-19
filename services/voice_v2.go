package services

import (
	gctx "context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/carsonkrueger/main/context"
	"github.com/carsonkrueger/main/models"

	msginterfaces "github.com/deepgram/deepgram-go-sdk/v3/pkg/api/agent/v1/websocket/interfaces"
	client "github.com/deepgram/deepgram-go-sdk/v3/pkg/client/agent"
	"github.com/deepgram/deepgram-go-sdk/v3/pkg/client/interfaces"
)

func NewDeepgramHandler(log io.Writer) DeepgramHandler {
	return DeepgramHandler{
		binaryChan:                   make(chan *[]byte),
		openChan:                     make(chan *msginterfaces.OpenResponse),
		welcomeResponse:              make(chan *msginterfaces.WelcomeResponse),
		conversationTextResponse:     make(chan *msginterfaces.ConversationTextResponse),
		userStartedSpeakingResponse:  make(chan *msginterfaces.UserStartedSpeakingResponse),
		agentThinkingResponse:        make(chan *msginterfaces.AgentThinkingResponse),
		functionCallRequestResponse:  make(chan *msginterfaces.FunctionCallRequestResponse),
		agentStartedSpeakingResponse: make(chan *msginterfaces.AgentStartedSpeakingResponse),
		agentAudioDoneResponse:       make(chan *msginterfaces.AgentAudioDoneResponse),
		closeChan:                    make(chan *msginterfaces.CloseResponse),
		errorChan:                    make(chan *msginterfaces.ErrorResponse),
		unhandledChan:                make(chan *[]byte),
		injectionRefusedResponse:     make(chan *msginterfaces.InjectionRefusedResponse),
		keepAliveResponse:            make(chan *msginterfaces.KeepAlive),
		settingsAppliedResponse:      make(chan *msginterfaces.SettingsAppliedResponse),
		log:                          log,
	}
}

type DeepgramHandler struct {
	binaryChan                   chan *[]byte
	openChan                     chan *msginterfaces.OpenResponse
	welcomeResponse              chan *msginterfaces.WelcomeResponse
	conversationTextResponse     chan *msginterfaces.ConversationTextResponse
	userStartedSpeakingResponse  chan *msginterfaces.UserStartedSpeakingResponse
	agentThinkingResponse        chan *msginterfaces.AgentThinkingResponse
	functionCallRequestResponse  chan *msginterfaces.FunctionCallRequestResponse
	agentStartedSpeakingResponse chan *msginterfaces.AgentStartedSpeakingResponse
	agentAudioDoneResponse       chan *msginterfaces.AgentAudioDoneResponse
	closeChan                    chan *msginterfaces.CloseResponse
	errorChan                    chan *msginterfaces.ErrorResponse
	unhandledChan                chan *[]byte
	injectionRefusedResponse     chan *msginterfaces.InjectionRefusedResponse
	keepAliveResponse            chan *msginterfaces.KeepAlive
	settingsAppliedResponse      chan *msginterfaces.SettingsAppliedResponse
	log                          io.Writer
}

func (dch DeepgramHandler) GetBinary() []*chan *[]byte {
	return []*chan *[]byte{&dch.binaryChan}
}

func (dch DeepgramHandler) GetOpen() []*chan *msginterfaces.OpenResponse {
	return []*chan *msginterfaces.OpenResponse{&dch.openChan}
}

func (dch DeepgramHandler) GetWelcome() []*chan *msginterfaces.WelcomeResponse {
	return []*chan *msginterfaces.WelcomeResponse{&dch.welcomeResponse}
}

func (dch DeepgramHandler) GetConversationText() []*chan *msginterfaces.ConversationTextResponse {
	return []*chan *msginterfaces.ConversationTextResponse{&dch.conversationTextResponse}
}

func (dch DeepgramHandler) GetUserStartedSpeaking() []*chan *msginterfaces.UserStartedSpeakingResponse {
	return []*chan *msginterfaces.UserStartedSpeakingResponse{&dch.userStartedSpeakingResponse}
}

func (dch DeepgramHandler) GetAgentThinking() []*chan *msginterfaces.AgentThinkingResponse {
	return []*chan *msginterfaces.AgentThinkingResponse{&dch.agentThinkingResponse}
}

func (dch DeepgramHandler) GetAgentStartedSpeaking() []*chan *msginterfaces.AgentStartedSpeakingResponse {
	return []*chan *msginterfaces.AgentStartedSpeakingResponse{&dch.agentStartedSpeakingResponse}
}

func (dch DeepgramHandler) GetAgentAudioDone() []*chan *msginterfaces.AgentAudioDoneResponse {
	return []*chan *msginterfaces.AgentAudioDoneResponse{&dch.agentAudioDoneResponse}
}

func (dch DeepgramHandler) GetClose() []*chan *msginterfaces.CloseResponse {
	return []*chan *msginterfaces.CloseResponse{&dch.closeChan}
}

func (dch DeepgramHandler) GetError() []*chan *msginterfaces.ErrorResponse {
	return []*chan *msginterfaces.ErrorResponse{&dch.errorChan}
}

func (dch DeepgramHandler) GetUnhandled() []*chan *[]byte {
	ch := dch.unhandledChan
	return []*chan *[]byte{&ch}
}

func (dch DeepgramHandler) GetInjectionRefused() []*chan *msginterfaces.InjectionRefusedResponse {
	return []*chan *msginterfaces.InjectionRefusedResponse{&dch.injectionRefusedResponse}
}

func (dch DeepgramHandler) GetKeepAlive() []*chan *msginterfaces.KeepAlive {
	return []*chan *msginterfaces.KeepAlive{&dch.keepAliveResponse}
}

func (dch DeepgramHandler) GetFunctionCallRequest() []*chan *msginterfaces.FunctionCallRequestResponse {
	return []*chan *msginterfaces.FunctionCallRequestResponse{&dch.functionCallRequestResponse}
}

func (dch DeepgramHandler) GetSettingsApplied() []*chan *msginterfaces.SettingsAppliedResponse {
	return []*chan *msginterfaces.SettingsAppliedResponse{&dch.settingsAppliedResponse}
}

type voiceV2 struct {
	dgWS     *client.WSChannel
	callback DeepgramHandler
	context.ServiceContext
}

func NewVoiceV2(ctx gctx.Context, svcCtx context.ServiceContext, dgApiKey string, clientOptions *interfaces.ClientOptions, settings *interfaces.SettingsOptions, handler DeepgramHandler) (*voiceV2, error) {
	cancel := context.GetCancel(ctx)
	callback := msginterfaces.AgentMessageChan(handler)
	dgWS, err := client.NewWSUsingChanWithCancel(ctx, cancel, dgApiKey, clientOptions, settings, callback)
	if err != nil {
		return nil, err
	}
	return &voiceV2{
		dgWS,
		handler,
		svcCtx,
	}, nil
}

func (g *voiceV2) Options() models.WebSocketOptions {
	return models.WebSocketOptions{}
}

func (v *voiceV2) HandleRequestWithStreaming(ctx gctx.Context, r models.StreamingReader, w models.StreamingWriter[models.StreamingResponseBody]) {
	lgr := v.Lgr("HandleRequestWithStreaming")
	pr, pw := io.Pipe()
	defer pw.Close()

	lgr.Info("Starting streaming: user <- agent")
	go v.callback.Run(w) // user <- agent
	if !v.dgWS.Connect() {
		lgr.Error("Failed to connect to Deepgram WebSocket")
		return
	}
	defer v.dgWS.Stop()
	lgr.Info("Starting streaming: user -> agent")
	go v.dgWS.Stream(pr) // user => agent

	// handle streaming data from our websocket to the deepgram websocket
	for {
		select {
		case <-ctx.Done():
			return
		case res := <-r:
			if _, err := pw.Write(res); err != nil {
				lgr.Error("Failed to write data to pipe writer")
				return
			}
		}
	}
}

func (dch DeepgramHandler) Run(w models.StreamingWriter[models.StreamingResponseBody]) error {
	wgReceivers := sync.WaitGroup{}

	// Handle binary data
	wgReceivers.Add(1)
	go func() {
		defer wgReceivers.Done()

		for br := range dch.binaryChan {
			if br == nil {
				continue
			}
			fmt.Printf("[Binary Data Received]\n")
			models.WriteBinary(w, models.StreamingResponseBody{Type: models.SR_AGENT_SPEAK, Data: *br})
		}
	}()

	// Handle conversation text
	wgReceivers.Add(1)
	go func() {
		defer wgReceivers.Done()

		var currentSpeaker string
		var currentMessage strings.Builder
		lastUpdate := time.Now()

		for ctr := range dch.conversationTextResponse {
			// If speaker changed or it's been more than 2 seconds, print accumulated message
			if currentSpeaker != ctr.Role || time.Since(lastUpdate) > 2*time.Second {
				if currentMessage.Len() > 0 {
					fmt.Printf("[ConversationTextResponse]\n")
					fmt.Printf("%s: %s", currentSpeaker, currentMessage.String())

					// Write to chat log
					if err := dch.writeToChatLog(currentSpeaker, currentMessage.String()); err != nil {
						fmt.Printf("Failed to write to chat log: %v\n", err)
					}
				}
				currentSpeaker = ctr.Role
				currentMessage.Reset()
			}

			// Add new content to current message
			if currentMessage.Len() > 0 {
				currentMessage.WriteString(" ")
			}
			currentMessage.WriteString(ctr.Content)
			lastUpdate = time.Now()

			// Track conversation flow
			switch ctr.Role {
			case "user":
				fmt.Printf("Received user message: %s\n", ctr.Content)
				fmt.Printf("Waiting for agent to process...\n")
			case "assistant":
				fmt.Printf("Agent response: %s\n", ctr.Content)
				fmt.Printf("Waiting for next user input...\n")
				models.WriteBinary(w, models.StreamingResponseBody{Type: models.SR_AGENT_TRANSCRIBE, Data: []byte(ctr.Content)})
			default:
				fmt.Printf("Received message from %s: %s\n", ctr.Role, ctr.Content)
			}
		}

		// Print any remaining message
		if currentMessage.Len() > 0 {
			fmt.Printf("[ConversationTextResponse]\n")
			fmt.Printf("%s: %s", currentSpeaker, currentMessage.String())

			if err := dch.writeToChatLog(currentSpeaker, currentMessage.String()); err != nil {
				fmt.Printf("Failed to write to chat log: %v\n", err)
			}
		}
	}()

	// Handle user started speaking
	wgReceivers.Add(1)
	go func() {
		defer wgReceivers.Done()

		for typ := range dch.userStartedSpeakingResponse {
			fmt.Printf("[UserStartedSpeakingResponse]: %s\n", typ)
			fmt.Printf("User has started speaking, waiting for completion...")

			// Write to chat log
			if err := dch.writeToChatLog("system", fmt.Sprintf("User has started speaking: %s", typ)); err != nil {
				fmt.Printf("Failed to write to chat log: %v\n", err)
			}
		}
	}()

	// Handle agent thinking
	wgReceivers.Add(1)
	go func() {
		defer wgReceivers.Done()

		for atr := range dch.agentThinkingResponse {
			fmt.Printf("[AgentThinkingResponse]\n")
			fmt.Printf("Agent is processing input: %s\n", atr.Content)
			fmt.Printf("Waiting for agent's response...")

			// Write to chat log
			if err := dch.writeToChatLog("system", fmt.Sprintf("Agent is processing: %s - %s", atr.Type, atr.Content)); err != nil {
				fmt.Printf("Failed to write to chat log: %v\n", err)
			}
		}
	}()

	// Handle agent started speaking
	wgReceivers.Add(1)
	go func() {
		defer wgReceivers.Done()

		for asr := range dch.agentStartedSpeakingResponse {
			fmt.Printf("Agent is starting to respond (latency: %.2fms)\n", asr.TotalLatency)
			var latencyBytes [8]byte
			binary.BigEndian.PutUint64(latencyBytes[:], math.Float64bits(asr.TotalLatency))
			models.WriteBinary(w, models.StreamingResponseBody{Type: models.SR_AGENT_SPEAK, Data: latencyBytes[:]})
			// Write to chat log
			if err := dch.writeToChatLog("system", "Agent is starting to respond"); err != nil {
				fmt.Printf("Failed to write to chat log: %v\n", err)
			}
		}
	}()

	// Handle agent audio done
	wgReceivers.Add(1)
	go func() {
		defer wgReceivers.Done()

		for range dch.agentAudioDoneResponse {
			fmt.Printf("[AgentAudioDoneResponse]\n")
			fmt.Printf("Agent finished speaking, waiting for next user input...")

			// Write to chat log
			if err := dch.writeToChatLog("system", "Agent finished speaking"); err != nil {
				fmt.Printf("Failed to write to chat log: %v\n", err)
			}
		}
	}()

	// Handle keep alive responses
	wgReceivers.Add(1)
	go func() {
		defer wgReceivers.Done()

		for range dch.keepAliveResponse {
			fmt.Printf("[KeepAliveResponse]\n")
			fmt.Printf("Connection is alive, waiting for next event...")

			// Write to chat log
			if err := dch.writeToChatLog("system", "Keep alive received"); err != nil {
				fmt.Printf("Failed to write to chat log: %v\n", err)
			}
		}
	}()

	// Handle other events
	wgReceivers.Add(1)
	go func() {
		defer wgReceivers.Done()
		for range dch.openChan {
			fmt.Printf("[OpenResponse]")
		}
	}()

	// welcome channel
	wgReceivers.Add(1)
	go func() {
		defer wgReceivers.Done()
		for range dch.welcomeResponse {
			fmt.Printf("[WelcomeResponse]")
		}
	}()

	// settings applied channel
	wgReceivers.Add(1)
	go func() {
		defer wgReceivers.Done()
		for range dch.settingsAppliedResponse {
			fmt.Printf("[SettingsAppliedResponse]")
		}
	}()

	// close channel
	wgReceivers.Add(1)
	go func() {
		defer wgReceivers.Done()
		for closeResp := range dch.closeChan {
			fmt.Printf("[CloseResponse]\n")
			fmt.Printf(" Close response received\n")
			fmt.Printf(" Close response type: %+v\n", closeResp)
			fmt.Printf("\n")
		}
	}()

	// error channel
	wgReceivers.Add(1)
	go func() {
		defer wgReceivers.Done()
		for er := range dch.errorChan {
			fmt.Printf("\n[ErrorResponse]\n")
			fmt.Printf("\nError.Type: %s\n", er.ErrCode)
			fmt.Printf("Error.Message: %s\n", er.ErrMsg)
			fmt.Printf("Error.Description: %s", er.Description)
			fmt.Printf("Error.Variant: %s", er.Variant)
		}
	}()

	// unhandled event channel
	wgReceivers.Add(1)
	go func() {
		defer wgReceivers.Done()
		for _ = range dch.unhandledChan {
			fmt.Printf("\n[UnhandledEvent]\n")
			// fmt.Printf("Raw message: %s\n", string(*byData))
		}
	}()

	// Handle function call request
	wgReceivers.Add(1)
	go func() {
		defer wgReceivers.Done()
		for range dch.functionCallRequestResponse {
			fmt.Printf("[FunctionCallRequestResponse]")
		}
	}()

	// Wait for all receivers to finish
	wgReceivers.Wait()
	return nil
}

// Helper function to write to chat log
func (dch *DeepgramHandler) writeToChatLog(role, content string) error {
	if dch.log == nil {
		return fmt.Errorf("chat log file not initialized")
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s: %s\n", timestamp, role, content)

	_, err := dch.log.Write([]byte(logEntry))
	if err != nil {
		return fmt.Errorf("failed to write to chat log: %v", err)
	}

	return nil
}
