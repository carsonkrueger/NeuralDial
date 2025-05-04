package services

import (
	"context"

	"github.com/tmc/langchaingo/llms"
)

type LLMService interface {
	Generate(ctx context.Context, msg string) (string, error)
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

func (l *llmService) Generate(ctx context.Context, msg string) (string, error) {
	lgr := l.Lgr("llmService.Generate")
	lgr.Info("Called")

	res, err := llms.GenerateFromSinglePrompt(ctx, l.llm, msg)
	if err != nil {
		return res, err
	}

	return res, nil
}
