package voice

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/carsonkrueger/main/services"
	"github.com/carsonkrueger/main/tools"
	"github.com/gorilla/websocket"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
	"go.uber.org/zap"
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
	lgr := m.Lgr("HandleRequestWithStreaming")
	m.handling.Store(true)
	defer m.handling.Store(false)

	fmt.Println("HANDLING")
	defer func() {
		out <- services.StreamResponse{Done: true}
	}()

	// check every m.waitCheckInterval to see when the user stops speaking
	ticker := time.NewTicker(m.waitCheckInterval)
	defer ticker.Stop()

outer:
	for {
		select {
		case <-ticker.C:
			if time.Since(m.lastUserSpeak) > m.waitDuration {
				lgr.Info("wait duration reached")
				break outer
			}
		case <-ctx.Done():
			return
		}
	}

	m.bufMutex.Lock()
	wav := tools.Int16ToWAV(tools.BytesToInt16Slice(m.audioBuffer), 16000)
	m.audioBuffer = []byte{}
	m.bufMutex.Unlock()

	if err := m.SaveUserStreamResponse(ctx, wav); err != nil {
		lgr.Error("SaveUserStreamResponse", zap.Error(err))
		return
	}

	// may need this to stream response from gpt to eleven labs
	stream := m.openaiClient.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Messages:   *m.chatHistory,
		Modalities: []string{"text"},
		Model:      shared.ChatModelGPT4oMiniAudioPreview,
	})
	defer stream.Close()

	var fullResponse strings.Builder
	pr1, pw1 := io.Pipe()
	defer pr1.Close()

	go func() {
		defer pw1.Close()
		defer fmt.Println("gpt streaming done")
		defer m.SaveAssistantTextResponse(fullResponse.String())
		for stream.Next() {
			select {
			default:
				if err := stream.Err(); err != nil {
					fmt.Println("error streaming from gpt:", err)
					return
				}
				chunk := stream.Current()
				data := []byte(chunk.Choices[0].Delta.Content)
				_, err := pw1.Write(data)
				if err != nil {
					lgr.Error("gpt streaming", zap.Error(err))
					pw1.CloseWithError(err)
					return
				}
				fullResponse.Write(data)
			case <-ctx.Done():
				return
			}
		}
	}()

	var currentResponse strings.Builder
	pr2, pw2 := io.Pipe()
	defer pr2.Close()

	buf2 := make([]byte, 1024)
	go func() {
		defer pw2.Close()
		defer fmt.Println("gpt-to-elevenlabs done")
	outer:
		for {
			select {
			default:
				n, errRead := pr1.Read(buf2)
				if errRead != nil {
					if errRead == io.EOF {
						break outer
					}
					return
				}
				_, err := currentResponse.Write(buf2[:n])
				if err != nil {
					lgr.Error("gpt-to-elevenlabs write", zap.Error(err))
					return
				}
				res := currentResponse.String()
				if tools.IsBoundary(res, true) {
					fmt.Println("found bound:", res)
					_, err = pw2.Write([]byte(res))
					if err != nil {
						lgr.Error("gpt-to-elevenlabs bounds write", zap.Error(err))
						pw2.CloseWithError(err)
						return
					}
					currentResponse.Reset()
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Start ElevenLabs streaming into the pipe writer in a goroutine
	pr3, pw3 := io.Pipe()
	defer pr3.Close()
	buf3 := make([]byte, 1024)

	go func() {
		defer pw3.Close() // Close when done to signal EOF
		defer fmt.Println("ElevenLabs streaming done")
	outer:
		for {
			select {
			default:
				n, errRead := pr2.Read(buf3)
				if errRead != nil {
					if errRead == io.EOF {
						break outer
					}
					return
				}
				err := m.SM().ElevenLabsService().TextToSpeechStream(string(buf3[:n]), pw3)
				if err != nil {
					lgr.Error("elevenlabs stream", zap.Error(err))
					pw3.CloseWithError(err)
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// reading eleven labs response
	buf4 := make([]byte, 4096)
outer2:
	for {
		select {
		default:
			n, errRead := pr3.Read(buf4)
			if errRead != nil {
				if errRead == io.EOF {
					break outer2
				}
				lgr.Error("elevenlabs read", zap.Error(errRead))
				return
			}
			currentResponse := services.StreamResponse{
				Data:    slices.Clone(buf4[:n]), // copy to avoid reuse
				MsgType: tools.Ptr(websocket.BinaryMessage),
			}
			out <- currentResponse
		case <-ctx.Done():
			return
		}
	}
}

func (w *gpt4oV1) HandleClose() {}

func (w *gpt4oV1) IsHandling() bool {
	return w.handling.Load()
}
