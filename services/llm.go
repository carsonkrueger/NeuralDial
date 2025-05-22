package services

import (
	"context"
	"errors"

	"github.com/carsonkrueger/main/models"
	langchaingo_mcp_adapter "github.com/i2y/langchaingo-mcp-adapter"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory"
)

type LLMService interface {
	StreamFromLLM(ctx context.Context, previousChats *models.LLMStreamingModel, msg string, streamingFunc func(ctx context.Context, chunk []byte) error) (string, error)
	Generate(ctx context.Context, agent agents.Agent, memory *memory.ConversationBuffer, msg string) (string, error)
	NewConversationalAgent(initialMessages []llms.ChatMessage) (agents.Agent, *memory.ConversationBuffer, error)
}

type llmService struct {
	ServiceContext
	llm llms.Model
}

func NewLLMService(ctx ServiceContext, llm llms.Model) *llmService {
	return &llmService{
		ctx,
		llm,
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
