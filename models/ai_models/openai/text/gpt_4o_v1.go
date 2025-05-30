package text

import (
	gctx "context"

	"github.com/carsonkrueger/main/context"
	"github.com/carsonkrueger/main/models"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/memory"
)

type gpt4oV1 struct {
	BaseLangChainMemory
	context.ServiceContext
}

func NewGPT4oV1(agent *agents.Agent, mem *memory.ConversationBuffer, serviceContext context.ServiceContext) *gpt4oV1 {
	return &gpt4oV1{
		BaseLangChainMemory: BaseLangChainMemory{
			agent,
			mem,
		},
		ServiceContext: serviceContext,
	}
}

func (m *gpt4oV1) Generate(ctx gctx.Context, req []byte) ([]byte, error) {
	return BaseLangChainGenerate(ctx, req, m.ServiceContext, &m.BaseLangChainMemory)
}

func (m *gpt4oV1) HandleRequest(ctx gctx.Context, msgType int, req []byte) (*int, []byte, error) {
	return BaseLangChainHandleRequest(ctx, msgType, req, m.ServiceContext, &m.BaseLangChainMemory)
}

func (m *gpt4oV1) HandleRequestWithStreaming(ctx gctx.Context, req []byte, out chan<- models.StreamResponse) {
	out <- models.StreamResponse{Done: true}
}

func (w *gpt4oV1) HandleClose() {}

func (w *gpt4oV1) IsHandling() bool {
	return true
}

func (w *gpt4oV1) PreprocessRequest(ctx gctx.Context, req []byte) {}

// func (l *llmService) StreamFromLLM(
// 	ctx context.Context,
// 	previousChats *models.LLMStreamingModel,
// 	msg string,
// 	streamingFunc func(ctx context.Context, chunk []byte) error) (string, error) {

// 	lgr := l.Lgr("llmService.StreamFromLLM")
// 	lgr.Info("Called")

// 	newMsg := l.BuildTextMessage(llms.ChatMessageTypeHuman, msg)
// 	previousChats.AddText(newMsg)

// 	streamOption := llms.WithStreamingFunc(streamingFunc)
// 	res, err := l.llm.GenerateContent(ctx, previousChats.Messages(), streamOption)
// 	if err != nil {
// 		return "", err
// 	} else if res == nil || len(res.Choices) == 0 {
// 		return "", errors.New("no response returned")
// 	}

// 	aiMsg := l.BuildTextMessage(llms.ChatMessageTypeAI, res.Choices[0].Content)
// 	previousChats.AddText(aiMsg)

// 	return res.Choices[0].Content, nil
// }
