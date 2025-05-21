package services

import (
	"context"
	"errors"

	"github.com/carsonkrueger/main/models"
	langchaingo_mcp_adapter "github.com/i2y/langchaingo-mcp-adapter"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/llms"
)

type LLMService interface {
	StreamFromLLM(ctx context.Context, previousChats *models.LLMStreamingModel, msg string, streamingFunc func(ctx context.Context, chunk []byte) error) (string, error)
	Generate(ctx context.Context, previousChats *models.LLMStreamingModel, msg string) (string, error)
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

func (l *llmService) Generate(
	ctx context.Context,
	previousChats *models.LLMStreamingModel,
	msg string) (string, error) {

	lgr := l.Lgr("llmService.Generate")
	lgr.Info("Called")

	newMsg := l.BuildTextMessage(llms.ChatMessageTypeHuman, msg)
	previousChats.AddText(newMsg)

	adapter, err := langchaingo_mcp_adapter.New(l.SM().MCPService().Client())
	if err != nil {
		return "", err
	}

	tools, err := adapter.Tools()
	if err != nil {
		return "", err
	}

	agents.NewConversationalAgent(l.llm, tools)

	res, err := l.llm.GenerateContent(ctx, previousChats.Messages())
	if err != nil {
		return "", err
	} else if res == nil || len(res.Choices) == 0 {
		return "", errors.New("no response returned")
	}

	aiMsg := l.BuildTextMessage(llms.ChatMessageTypeAI, res.Choices[0].Content)
	previousChats.AddText(aiMsg)

	return res.Choices[0].Content, nil
}
