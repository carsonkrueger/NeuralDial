package voice

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/carsonkrueger/main/services"
	"github.com/carsonkrueger/main/tools"
	"github.com/gorilla/websocket"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
)

type gpt4oV1 struct {
	bufMutex          sync.Mutex
	chatMutex         sync.Mutex
	handling          atomic.Bool
	waitDuration      time.Duration
	waitCheckInterval time.Duration
	lastUserSpeak     time.Time
	audioBuffer       []byte
	chatHistory       *[]openai.ChatCompletionMessageParamUnion
	services.ServiceContext
	openaiClient *openai.Client
}

func NewGPT4oV1(svcCtx services.ServiceContext, waitDuration time.Duration, waitCheckInterval time.Duration, client *openai.Client) *gpt4oV1 {
	return &gpt4oV1{
		waitDuration:      waitDuration,
		ServiceContext:    svcCtx,
		chatHistory:       &[]openai.ChatCompletionMessageParamUnion{},
		openaiClient:      client,
		waitCheckInterval: waitCheckInterval,
	}
}

func (m *gpt4oV1) SaveUserStreamResponse(ctx context.Context, wav []byte) error {
	data := base64.StdEncoding.EncodeToString(wav)
	m.chatMutex.Lock()
	*m.chatHistory = append(*m.chatHistory, openai.ChatCompletionMessageParamUnion{
		OfUser: &openai.ChatCompletionUserMessageParam{
			Content: openai.ChatCompletionUserMessageParamContentUnion{
				OfArrayOfContentParts: []openai.ChatCompletionContentPartUnionParam{
					openai.InputAudioContentPart(openai.ChatCompletionContentPartInputAudioInputAudioParam{
						Data:   data,
						Format: "wav",
					}),
				},
			},
		},
	})
	m.chatMutex.Unlock()
	return nil
}

func (m *gpt4oV1) SaveAssistantTextResponse(text string) error {
	*m.chatHistory = append(*m.chatHistory, openai.ChatCompletionMessageParamUnion{
		OfAssistant: &openai.ChatCompletionAssistantMessageParam{
			Content: openai.ChatCompletionAssistantMessageParamContentUnion{
				OfString: openai.String(text),
			},
		},
	})
	return nil
}

func (m *gpt4oV1) Generate(ctx context.Context, req []byte) (*string, []byte, error) {
	completion, err := m.openaiClient.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: *m.chatHistory,
		// Audio: openai.ChatCompletionAudioParam{
		// 	Format: "wav",
		// 	Voice:  "alloy",
		// },
		Modalities: []string{"text"},
		Model:      shared.ChatModelGPT4oMiniAudioPreview,
	})
	if err != nil {
		return nil, nil, err
	}
	return nil, []byte(completion.Choices[0].Message.Content), nil
}

func (m *gpt4oV1) GenerateStream(ctx context.Context, req []byte) (*string, []byte, error) {
	return nil, nil, nil
}

func (m *gpt4oV1) HandleRequest(ctx context.Context, msgType int, req []byte) (*int, []byte, error) {
	return nil, nil, nil
}

func (m *gpt4oV1) PreprocessRequest(ctx context.Context, req []byte) {
	m.bufMutex.Lock()
	m.audioBuffer = append(m.audioBuffer, req...)
	m.lastUserSpeak = time.Now()
	m.bufMutex.Unlock()
}

func (m *gpt4oV1) HandleRequestWithStreaming(ctx context.Context, req []byte, out chan<- services.StreamResponse) {
	m.handling.Store(true)
	defer m.handling.Store(false)

	fmt.Println("HANDLING")
	var err error
	defer func() {
		out <- services.StreamResponse{Done: true, Err: err}
	}()

	// check every 100 ms to see when the user stops speaking
	ticker := time.NewTicker(m.waitCheckInterval)
	defer ticker.Stop()

outer:
	for {
		select {
		case <-ticker.C:
			if time.Since(m.lastUserSpeak) > m.waitDuration {
				fmt.Println("wait duration reached")
				break outer
			}
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}

	m.bufMutex.Lock()
	wav := tools.Int16ToWAV(tools.BytesToInt16Slice(m.audioBuffer), 16000)
	m.audioBuffer = []byte{}
	m.bufMutex.Unlock()

	if err = m.SaveUserStreamResponse(ctx, wav); err != nil {
		return
	}

	// may need this to stream response from gpt to eleven labs
	// stream := m.openaiClient.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
	// 	Messages:   *m.chatHistory,
	// 	Modalities: []string{"text"},
	// 	Model:      shared.ChatModelGPT4oMiniAudioPreview,
	// })
	// defer stream.Close()

	_, res, err := m.Generate(ctx, nil)
	fmt.Println(string(res))

	// create buffer for io.Writer to write to
	pr, pw := io.Pipe()

	// Start ElevenLabs streaming into the pipe writer in a goroutine
	go func() {
		defer pw.Close() // Close when done to signal EOF
		err := m.SM().ElevenLabsService().TextToSpeechStream(string(res), pw)
		if err != nil {
			pw.CloseWithError(err)
		}
	}()

	fmt.Println("Begin TTS")
	buf := make([]byte, 4096)
	for {
		n, errRead := pr.Read(buf)
		if errRead != nil {
			if errRead == io.EOF {
				break // done
			}
			err = errors.New("Error reading from pipe")
			return
		}

		currentResponse := services.StreamResponse{
			Data:    slices.Clone(buf[:n]), // copy to avoid reuse
			MsgType: tools.Ptr(websocket.BinaryMessage),
		}

		out <- currentResponse
	}

	// 	for stream.Next() {
	// 		if err = stream.Err(); err != nil {
	// 			return
	// 		}
	// 		chunk := stream.Current()
	// 		data := []byte(chunk.Choices[0].Delta.Content)
	// 		stringBuilder.Write(data)
	// 		currentResponse.Data = data
	// 		currentResponse.MsgType = tools.Ptr(websocket.BinaryMessage)
	// 		out <- currentResponse
	// 	}

	// fmt.Println(stringBuilder.String())
	// m.SaveAssistantTextResponse(stringBuilder.String())
}

func (w *gpt4oV1) HandleClose() {}

func (w *gpt4oV1) IsHandling() bool {
	return w.handling.Load()
}
