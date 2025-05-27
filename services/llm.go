package services

import (
	"context"

	"github.com/openai/openai-go"
	"github.com/tmc/langchaingo/llms"
)

type Generator interface {
	Generate(ctx context.Context, res []byte) ([]byte, error)
}

type StreamGenerator interface {
	GenerateStream(ctx context.Context, res []byte, out chan<- StreamResponse) ([]byte, error)
}

type AssistantStreamMemoryHandler interface {
	SaveAssistantStreamResponse(ctx context.Context, res []byte) error
}

type AssistantMemoryHandler interface {
	SaveAssistantResponse(ctx context.Context, res []byte) error
}

type UserMemoryHandler interface {
	SaveUserResponse(ctx context.Context, res []byte) error
}

type NeuralDialModel interface {
	UserMemoryHandler
	AssistantMemoryHandler
	Generator
}

type NeuralDialStreamModel interface {
	UserMemoryHandler
	AssistantStreamMemoryHandler
	StreamGenerator
}

type LLMService interface {
	GenerateResponse(ctx context.Context, model NeuralDialModel, req []byte) ([]byte, error)
	GenerateResponseStream(ctx context.Context, model NeuralDialStreamModel, req []byte, out chan<- StreamResponse) error
	LLM() llms.Model
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

func (l *llmService) GenerateResponse(ctx context.Context, model NeuralDialModel, req []byte) ([]byte, error) {
	if err := model.SaveUserResponse(ctx, req); err != nil {
		return nil, err
	}
	res, err := model.Generate(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := model.SaveAssistantResponse(ctx, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (l *llmService) GenerateResponseStream(ctx context.Context, model NeuralDialStreamModel, req []byte, out chan<- StreamResponse) error {
	if err := model.SaveUserResponse(ctx, req); err != nil {
		return err
	}
	res, err := model.GenerateStream(ctx, req, out)
	if err != nil {
		return err
	}
	if err := model.SaveAssistantStreamResponse(ctx, res); err != nil {
		return err
	}
	return nil
}

func (w *llmService) LLM() llms.Model {
	return w.llm
}
