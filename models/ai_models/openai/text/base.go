package text

import (
	"context"

	"github.com/carsonkrueger/main/services"
	"github.com/carsonkrueger/main/tools"
	"github.com/gorilla/websocket"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/memory"
)

// base langchain memory

type BaseLangChainMemory struct {
	Agent *agents.Agent
	Mem   *memory.ConversationBuffer
}

func (b *BaseLangChainMemory) SaveUserResponse(ctx context.Context, res []byte) error {
	return b.Mem.ChatHistory.AddUserMessage(ctx, string(res))
}

func (b *BaseLangChainMemory) SaveAssistantResponse(ctx context.Context, res []byte) error {
	return b.Mem.ChatHistory.AddAIMessage(ctx, string(res))
}

// base generate methods

func BaseLangChainGenerate(ctx context.Context, req []byte, svcCtx services.ServiceContext, m *BaseLangChainMemory) ([]byte, error) {
	executor := agents.NewExecutor(*m.Agent, agents.WithMemory(m.Mem))
	res, err := chains.Run(ctx, executor, string(req))
	if err != nil {
		return nil, err
	}
	return []byte(res), nil
}

func BaseLangChainHandleRequest(ctx context.Context, msgType int, req []byte, svcCtx services.ServiceContext, m *BaseLangChainMemory) (*int, []byte, error) {
	res, err := BaseLangChainGenerate(ctx, req, svcCtx, m)
	if err != nil {
		return nil, nil, err
	}
	resBytes := []byte(res)
	return tools.Ptr(websocket.BinaryMessage), resBytes, nil
}

func BaseHandleClose() {}
