package services

import (
	"context"
	"errors"
	"time"

	"github.com/carsonkrueger/main/models"
	"github.com/carsonkrueger/main/tools"
	"github.com/gorilla/websocket"
	langchaingo_mcp_adapter "github.com/i2y/langchaingo-mcp-adapter"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory"
)

type LLMService interface {
	StreamFromLLM(ctx context.Context, previousChats *models.LLMStreamingModel, msg string, streamingFunc func(ctx context.Context, chunk []byte) error) (string, error)
	Generate(ctx context.Context, agent agents.Agent, memory *memory.ConversationBuffer, msg string) (string, error)
	NewConversationalAgent(initialMessages []llms.ChatMessage) (agents.Agent, *memory.ConversationBuffer, error)
	WebTextHandler(agent *agents.Agent, memory *memory.ConversationBuffer) (WebSocketHandler, models.WebSocketOptions)
}

type llmService struct {
	ServiceContext
	llm          llms.Model
	openaiClient *openai.Client
}

func NewLLMService(ctx ServiceContext, llm llms.Model, openaiClient *openai.Client) *llmService {
	return &llmService{
		ctx,
		llm,
		openaiClient,
	}
}

func (l *llmService) BuildTextMessage(role llms.ChatMessageType, msg string, msgs ...string) llms.MessageContent {
	msgContent := llms.MessageContent{
		Role:  role,
		Parts: []llms.ContentPart{llms.TextPart(msg)},
	}
	for i := range msgs {
		msgContent.Parts = append(msgContent.Parts, llms.TextPart(msgs[i]))
	}
	return msgContent
}

func (l *llmService) StreamFromLLM(
	ctx context.Context,
	previousChats *models.LLMStreamingModel,
	msg string,
	streamingFunc func(ctx context.Context, chunk []byte) error) (string, error) {

	lgr := l.Lgr("llmService.StreamFromLLM")
	lgr.Info("Called")

	newMsg := l.BuildTextMessage(llms.ChatMessageTypeHuman, msg)
	previousChats.AddText(newMsg)

	streamOption := llms.WithStreamingFunc(streamingFunc)
	res, err := l.llm.GenerateContent(ctx, previousChats.Messages(), streamOption)
	if err != nil {
		return "", err
	} else if res == nil || len(res.Choices) == 0 {
		return "", errors.New("no response returned")
	}

	aiMsg := l.BuildTextMessage(llms.ChatMessageTypeAI, res.Choices[0].Content)
	previousChats.AddText(aiMsg)

	return res.Choices[0].Content, nil
}

func (l *llmService) NewConversationalAgent(initialMessages []llms.ChatMessage) (agents.Agent, *memory.ConversationBuffer, error) {
	adapter, err := langchaingo_mcp_adapter.New(l.SM().MCPService().Client())
	if err != nil {
		return nil, nil, err
	}

	tools, err := adapter.Tools()
	if err != nil {
		return nil, nil, err
	}

	mem := memory.NewChatMessageHistory(memory.WithPreviousMessages(initialMessages))
	memoryBuffer := memory.NewConversationBuffer(memory.WithChatHistory(mem))
	return agents.NewConversationalAgent(l.llm, tools), memoryBuffer, nil
}

func (l *llmService) Generate(ctx context.Context, agent agents.Agent, memoryBuffer *memory.ConversationBuffer, msg string) (string, error) {
	memoryBuffer.ChatHistory.AddUserMessage(ctx, msg)
	executor := agents.NewExecutor(agent, agents.WithMemory(memoryBuffer))
	res, err := chains.Run(ctx, executor, msg)
	if err != nil {
		return "", err
	}
	memoryBuffer.ChatHistory.AddAIMessage(ctx, res)
	return res, nil
}

func (llms *llmService) WebTextHandler(agent *agents.Agent, memory *memory.ConversationBuffer) (WebSocketHandler, models.WebSocketOptions) {
	opts := models.WebSocketOptions{
		PongDeadline:        tools.Ptr(10 * time.Minute),
		PongInterval:        tools.Ptr(10 * time.Second),
		AllowedMessageTypes: []int{websocket.TextMessage},
	}
	handler := webTextHandler{
		agent:      agent,
		mem:        memory,
		llmService: llms,
	}
	return &handler, opts
}

func (l *llmService) Open4oAudioResponse(ctx context.Context, chatHistory *[]openai.ChatCompletionMessageParamUnion, audio []byte) ([]byte, error) {
	data := string(audio)
	*chatHistory = append(*chatHistory, openai.ChatCompletionMessageParamUnion{
		OfUser: &openai.ChatCompletionUserMessageParam{
			Content: openai.ChatCompletionUserMessageParamContentUnion{
				OfArrayOfContentParts: []openai.ChatCompletionContentPartUnionParam{
					openai.InputAudioContentPart(openai.ChatCompletionContentPartInputAudioInputAudioParam{
						Data:   data,
						Format: "wav",
					}),
				},
			},
		},
	})
	completion, err := l.openaiClient.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: *chatHistory,
		Audio: openai.ChatCompletionAudioParam{
			Format: "wav",
			Voice:  "alloy",
		},
		Modalities: []string{"audio", "text"},
		Model:      shared.ChatModelGPT4oAudioPreview,
	})
	if err != nil {
		return nil, err
	}
	*chatHistory = append(*chatHistory, openai.ChatCompletionMessageParamUnion{
		OfAssistant: &openai.ChatCompletionAssistantMessageParam{
			Content: openai.ChatCompletionAssistantMessageParamContentUnion{
				OfArrayOfContentParts: []openai.ChatCompletionAssistantMessageParamContentArrayOfContentPartUnion{
					openai.ChatCompletionAssistantMessageParamContentArrayOfContentPartUnion{
						OfText: &openai.ChatCompletionContentPartTextParam{
							Text: completion.Choices[0].Message.Audio.Transcript,
							Type: "text",
						},
					},
				},
			},
		},
	})
	return []byte(completion.Choices[0].Message.Audio.Data), nil
}

type webTextHandler struct {
	agent      *agents.Agent
	mem        *memory.ConversationBuffer
	llmService *llmService
}

func (w *webTextHandler) HandleRequest(ctx context.Context, msgType int, req []byte) (int, []byte, error) {
	msg := string(req)
	res, err := w.llmService.Generate(ctx, *w.agent, w.mem, msg)
	if err != nil {
		return 0, nil, err
	}
	resBytes := []byte(res)
	return websocket.BinaryMessage, resBytes, nil
}

type webVoiceHandler struct {
	agent      *agents.Agent
	mem        *memory.ConversationBuffer
	llmService *llmService
}

func (w *webVoiceHandler) HandleRequest(ctx context.Context, msgType int, req []byte) (int, []byte, error) {
	msg := string(req)
	res, err := w.llmService.Generate(ctx, *w.agent, w.mem, msg)
	if err != nil {
		return 0, nil, err
	}
	resBytes := []byte(res)
	return websocket.BinaryMessage, resBytes, nil
}
