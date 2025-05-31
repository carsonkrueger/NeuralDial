package services

import (
	gctx "context"

	"github.com/carsonkrueger/elevenlabs-go"
	"github.com/carsonkrueger/main/context"
	"github.com/carsonkrueger/main/models"
	"github.com/carsonkrueger/main/tools"
	"github.com/gorilla/websocket"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
)

type baseLangChainService struct {
	context.ServiceContext
	client               *elevenlabs.Client
	voices               []elevenlabs.Voice
	models               []elevenlabs.Model
	defaultVoiceSettings *elevenlabs.VoiceSettings
}

func NewBaseLangChainService(ctx context.ServiceContext, client *elevenlabs.Client) *elevenLabsService {
	return &elevenLabsService{
		ServiceContext:       ctx,
		client:               client,
		voices:               []elevenlabs.Voice{},
		defaultVoiceSettings: nil,
	}
}

func SaveUserResponse(ctx gctx.Context, b *models.BaseLangChainMemory, res []byte) error {
	return b.Mem.ChatHistory.AddUserMessage(ctx, string(res))
}

func SaveAssistantResponse(ctx gctx.Context, b *models.BaseLangChainMemory, res []byte) error {
	return b.Mem.ChatHistory.AddAIMessage(ctx, string(res))
}

func BaseLangChainGenerate(ctx gctx.Context, req []byte, svcCtx context.ServiceContext, m *models.BaseLangChainMemory) ([]byte, error) {
	executor := agents.NewExecutor(*m.Agent, agents.WithMemory(m.Mem))
	res, err := chains.Run(ctx, executor, string(req))
	if err != nil {
		return nil, err
	}
	return []byte(res), nil
}

func BaseLangChainHandleRequest(ctx gctx.Context, msgType int, req []byte, svcCtx context.ServiceContext, m *models.BaseLangChainMemory) (*int, []byte, error) {
	res, err := BaseLangChainGenerate(ctx, req, svcCtx, m)
	if err != nil {
		return nil, nil, err
	}
	resBytes := []byte(res)
	return tools.Ptr(websocket.BinaryMessage), resBytes, nil
}

func BaseHandleClose() {}
