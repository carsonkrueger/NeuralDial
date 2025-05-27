package voice

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/carsonkrueger/main/services"
	"github.com/carsonkrueger/main/tools"
	"github.com/gorilla/websocket"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
)

type gpt4oV1 struct {
	waitDuration  time.Duration
	lastUserSpeak time.Time
	audioBuffer   []byte
	chatHistory   *[]openai.ChatCompletionMessageParamUnion
	take          chan bool
	sync.Mutex
	services.ServiceContext
	openaiClient openai.Client
}

func NewGPT4oV1(svcCtx services.ServiceContext, waitDuration time.Duration) *gpt4oV1 {
	return &gpt4oV1{
		waitDuration:   waitDuration,
		ServiceContext: svcCtx,
	}
}

func (m *gpt4oV1) SaveUserResponse(ctx context.Context, res []byte) error {
	data := base64.StdEncoding.EncodeToString(res)
	*m.chatHistory = append(*m.chatHistory, openai.ChatCompletionMessageParamUnion{
		OfUser: &openai.ChatCompletionUserMessageParam{
			Content: openai.ChatCompletionUserMessageParamContentUnion{
				// OfString: openai.String("Hello there!"),
				OfArrayOfContentParts: []openai.ChatCompletionContentPartUnionParam{
					openai.InputAudioContentPart(openai.ChatCompletionContentPartInputAudioInputAudioParam{
						Data:   data,
						Format: "wav",
					}),
				},
			},
		},
	})
	return nil
}

func (m *gpt4oV1) SaveAssistantResponse(ctx context.Context, res []byte) error {
	*m.chatHistory = append(*m.chatHistory, openai.ChatCompletionMessageParamUnion{
		OfAssistant: &openai.ChatCompletionAssistantMessageParam{
			Audio: openai.ChatCompletionAssistantMessageParamAudio{
				ID: string(res),
			},
		},
	})
	return nil
}

func (m *gpt4oV1) SaveAssistantStreamResponse(ctx context.Context, res []byte) error {
	*m.chatHistory = append(*m.chatHistory, openai.ChatCompletionMessageParamUnion{
		OfAssistant: &openai.ChatCompletionAssistantMessageParam{
			Audio: openai.ChatCompletionAssistantMessageParamAudio{
				ID: string(res),
			},
		},
	})
	return nil
}

func (m *gpt4oV1) Generate(ctx context.Context, req []byte) ([]byte, error) {
	completion, err := m.openaiClient.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: *m.chatHistory,
		Audio: openai.ChatCompletionAudioParam{
			Format: "wav",
			Voice:  "alloy",
		},
		Modalities: []string{"audio", "text"},
		Model:      shared.ChatModelGPT4oAudioPreview2024_12_17,
	})
	if err != nil {
		return nil, err
	}

	return []byte(completion.Choices[0].Message.Audio.Data), nil
}

func (m *gpt4oV1) GenerateStream(ctx context.Context, res []byte, out chan<- services.StreamResponse) ([]byte, error) {
	data := base64.StdEncoding.EncodeToString(res)
	*m.chatHistory = append(*m.chatHistory, openai.ChatCompletionMessageParamUnion{
		OfUser: &openai.ChatCompletionUserMessageParam{
			Content: openai.ChatCompletionUserMessageParamContentUnion{
				// OfString: openai.String("Hello there!"),
				OfArrayOfContentParts: []openai.ChatCompletionContentPartUnionParam{
					openai.InputAudioContentPart(openai.ChatCompletionContentPartInputAudioInputAudioParam{
						Data:   data,
						Format: "wav",
					}),
				},
			},
		},
	})
	stream := m.openaiClient.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Messages: *m.chatHistory,
		Audio: openai.ChatCompletionAudioParam{
			Format: "wav",
			Voice:  "alloy",
		},
		Modalities: []string{"audio", "text"},
		Model:      shared.ChatModelGPT4oAudioPreview2024_12_17,
	})
	defer stream.Close()

	fullResponse := []byte{}
	currentResponse := services.StreamResponse{}

	go func() {
		currentResponse.Data = nil
		currentResponse.MsgType = nil
		currentResponse.Done = true
		out <- currentResponse
	}()

	for stream.Next() {
		if err := stream.Err(); err != nil {
			return nil, err
		}
		chunk := stream.Current()
		data := []byte(chunk.Choices[0].Delta.Content)
		fullResponse = append(fullResponse, data...)
		currentResponse.Data = data
		currentResponse.MsgType = tools.Ptr(websocket.BinaryMessage)
		out <- currentResponse
	}

	return fullResponse, nil
}

func (m *gpt4oV1) HandleRequest(ctx context.Context, msgType int, req []byte) (*int, []byte, error) {
	fmt.Println("sending data to open ai", len(req))
	res, err := m.SM().LLMService().GenerateResponse(ctx, m, req)
	if err != nil {
		return nil, nil, err
	}
	return tools.Ptr(websocket.TextMessage), res, nil
	// str := base64.StdEncoding.EncodeToString(req)
	// return tools.Ptr(websocket.TextMessage), []byte(str), nil
}

func (m *gpt4oV1) HandleRequestWithStreaming(ctx context.Context, msgType int, req []byte, out chan<- services.StreamResponse) error {
	fmt.Println("sending data to open ai", len(req))
	err := m.SM().LLMService().GenerateResponseStream(ctx, m, req, out)
	if err != nil {
		return err
	}
	return nil
}

func (w *gpt4oV1) HandleClose() {}
