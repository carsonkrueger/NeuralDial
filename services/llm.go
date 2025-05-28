package services

import (
	"github.com/openai/openai-go"
	"github.com/tmc/langchaingo/llms"
)

type LLMService interface {
	LLM() llms.Model
	OpenaiClient() *openai.Client
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

func (w *llmService) LLM() llms.Model {
	return w.llm
}

func (w *llmService) OpenaiClient() *openai.Client {
	return w.openaiClient
}
