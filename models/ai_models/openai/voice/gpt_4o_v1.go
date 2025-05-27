package voice

import (
	"context"
	"encoding/base64"
	"errors"
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
	sync.Mutex
	waitDuration  time.Duration
	lastUserSpeak time.Time
	audioBuffer   []byte
	chatHistory   *[]openai.ChatCompletionMessageParamUnion
	services.ServiceContext
	openaiClient *openai.Client
}

func NewGPT4oV1(svcCtx services.ServiceContext, waitDuration time.Duration, client *openai.Client) *gpt4oV1 {
	return &gpt4oV1{
		waitDuration:   waitDuration,
		ServiceContext: svcCtx,
		chatHistory:    &[]openai.ChatCompletionMessageParamUnion{},
		openaiClient:   client,
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

func (m *gpt4oV1) SaveAssistantResponse(ctx context.Context, id *string, res []byte) error {
	if id == nil {
		return errors.New("id is nil")
	}
	*m.chatHistory = append(*m.chatHistory, openai.ChatCompletionMessageParamUnion{
		OfAssistant: &openai.ChatCompletionAssistantMessageParam{
			Audio: openai.ChatCompletionAssistantMessageParamAudio{
				ID: *id,
			},
		},
	})
	return nil
}

func (m *gpt4oV1) SaveAssistantStreamResponse(ctx context.Context, res []byte) error {
	// *m.chatHistory = append(*m.chatHistory, openai.ChatCompletionMessageParamUnion{
	// 	OfAssistant: &openai.ChatCompletionAssistantMessageParam{
	// 		Audio: openai.ChatCompletionAssistantMessageParamAudio{
	// 			// ID: string(res),
	// 		},
	// 	},
	// })
	return nil
}

func (m *gpt4oV1) Generate(ctx context.Context, req []byte) (*string, []byte, error) {
	completion, err := m.openaiClient.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: *m.chatHistory,
		Audio: openai.ChatCompletionAudioParam{
			Format: "wav",
			Voice:  "alloy",
		},
		Modalities: []string{"audio", "text"},
		Model:      shared.ChatModelGPT4oMiniAudioPreview,
	})
	if err != nil {
		return nil, nil, err
	}
	return &completion.Choices[0].Message.Audio.ID, []byte(completion.Choices[0].Message.Audio.Data), nil
}

func (m *gpt4oV1) GenerateStream(ctx context.Context, res []byte, out chan<- services.StreamResponse) ([]byte, error) {
	fmt.Println("acquiring stream...")
	stream := m.openaiClient.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Messages: *m.chatHistory,
		Audio: openai.ChatCompletionAudioParam{
			Format: "pcm16",
			Voice:  "alloy",
		},
		Modalities: []string{"text", "audio"},
		Model:      shared.ChatModelGPT4oMiniAudioPreview,
	})
	fmt.Println("stream acquired")
	defer stream.Close()

	fullResponse := []byte{}
	currentResponse := services.StreamResponse{}

	defer func() {
		currentResponse.Data = nil
		currentResponse.MsgType = nil
		currentResponse.Done = true
		out <- currentResponse
	}()

	fmt.Println("listening...")

	for stream.Next() {
		fmt.Println("next...")
		if err := stream.Err(); err != nil {
			return nil, err
		}
		chunk := stream.Current()
		fmt.Println(len(chunk.Choices[0].Delta.Content))
		fmt.Println(chunk.Choices[0].Delta.Content)
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
