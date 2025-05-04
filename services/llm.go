package services

import (
	"context"

	"github.com/tmc/langchaingo/llms"
)

type LLMService interface {
	Generate(ctx context.Context) error
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

func (l *llmService) Generate(ctx context.Context) error {
	lgr := l.Lgr("llmService.Generate")
	lgr.Info("Called")

	res, err := llms.GenerateFromSinglePrompt(ctx, l.llm, "Hello this is a test")
	if err != nil {
		return err
	}

	lgr.Info(res)
	return nil
}
