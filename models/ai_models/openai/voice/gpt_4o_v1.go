package voice

import (
	gctx "context"
	"encoding/base64"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/carsonkrueger/main/context"
	"github.com/carsonkrueger/main/models"
	"github.com/carsonkrueger/main/tools"
	"github.com/gorilla/websocket"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
	"go.uber.org/zap"
)

type gpt4oV1 struct {
	interrupt         *models.Interrupt                         // used to interrupted ai response pipeline
	handling          bool                                      // whether ai has begun to handle the request
	handleMutex       sync.Mutex                                // mutex for handling
	aiResponding      bool                                      // mutex whether ai is responding to user
	waitDuration      time.Duration                             // how long to wait for user to stop speaking
	waitCheckInterval time.Duration                             // how often to check if user has stopped speaking
	lastUserSpeak     time.Time                                 // time of last user audio req
	audioBuffer       []byte                                    // raw audio buffer
	bufMutex          sync.Mutex                                // mutex for audio buffer
	chatHistory       *[]openai.ChatCompletionMessageParamUnion // entire chat history of conversation
	chatMutex         sync.Mutex                                // mutex for chat history
	openaiClient      *openai.Client
	context.ServiceContext
}

func NewGPT4oV1(svcCtx context.ServiceContext) *gpt4oV1 {
	return &gpt4oV1{
		interrupt:         models.NewInterrupt(),
		waitDuration:      time.Millisecond * 500,
		waitCheckInterval: time.Millisecond * 50,
		ServiceContext:    svcCtx,
		chatHistory:       &[]openai.ChatCompletionMessageParamUnion{},
		openaiClient:      svcCtx.SM().LLMService().OpenaiClient(),
	}
}

func (m *gpt4oV1) SaveUserStreamResponse(ctx gctx.Context, wav []byte) error {
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

func (m *gpt4oV1) Generate(ctx gctx.Context, req []byte) (*string, []byte, error) {
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

func (m *gpt4oV1) GenerateStream(ctx gctx.Context, req []byte) (*string, []byte, error) {
	return nil, nil, nil
}

func (m *gpt4oV1) HandleRequest(ctx gctx.Context, msgType int, req []byte) (*int, []byte, error) {
	return nil, nil, nil
}

func (m *gpt4oV1) HandleRequestWithStreaming(ctx gctx.Context, req []byte, out chan<- models.StreamResponse) {
	lgr := m.Lgr("HandleRequestWithStreaming")
	var err error

	defer func() {
		out <- models.StreamResponse{Done: true, Err: err}
	}()

	// save audio buffer
	m.bufMutex.Lock()
	m.audioBuffer = append(m.audioBuffer, req...)
	m.lastUserSpeak = time.Now()
	m.bufMutex.Unlock()

	// PRE AI RESPONSE
	// if AI is responding, interrupt - maybe wait n time for
	if m.aiResponding {
		lgr.Debug("Canceling")
		m.interrupt.Signal()
		m.handleMutex.Lock()
		m.handling = false
		m.aiResponding = false
		m.handleMutex.Unlock()
	}

	m.handleMutex.Lock()
	if m.handling {
		m.handleMutex.Unlock()
		return
	}
	m.handling = true
	m.handleMutex.Unlock()

	// POST AI RESPONSE
	lgr.Info("HANDLING")
	ticker := time.NewTicker(m.waitCheckInterval)
	defer func() {
		ticker.Stop()
		m.interrupt.Reset()
		m.handleMutex.Lock()
		m.handling = false
		m.handleMutex.Unlock()
	}()

userSpeak:
	for {
		select {
		case <-ticker.C:
			if time.Since(m.lastUserSpeak) > m.waitDuration {
				lgr.Info("wait duration reached")
				break userSpeak
			}
		case <-ctx.Done():
			return
		case <-m.interrupt.Done():
			lgr.Warn("Interrupted")
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
		defer func() {
			fmt.Println("gpt streaming done:")
			pw1.Close()
			m.SaveAssistantTextResponse(fullResponse.String())
		}()
		for stream.Next() {
			select {
			default:
				if err := stream.Err(); err != nil {
					lgr.Error("GPT Err", zap.Error(err))
					return
				}
				m.aiResponding = true
				chunk := stream.Current()
				data := []byte(chunk.Choices[0].Delta.Content)
				if len(data) == 0 {
					lgr.Warn("Empty response from GPT")
					continue
				}
				_, err := pw1.Write(data)
				if err != nil {
					if err != io.ErrClosedPipe {
						lgr.Error("gpt streaming", zap.Error(err))
					}
					return
				}
				fmt.Println(string(data))
				fullResponse.Write(data)
			case <-ctx.Done():
				lgr.Warn("Context done")
				return
			case <-m.interrupt.Done():
				lgr.Warn("Interrupted")
				return
			}
		}
	}()

	var currentResponse strings.Builder
	pr2, pw2 := io.Pipe()
	defer pr2.Close()

	buf2 := make([]byte, 1024)
	go func() {
		defer fmt.Println("gpt-to-elevenlabs done")
		defer pw2.Close()
		for {
			select {
			default:
				n, errRead := pr1.Read(buf2)
				if errRead != nil {
					return
				}
				currentResponse.Write(buf2[:n])
				res := currentResponse.String()
				if !tools.IsBoundary(res, true) {
					continue
				}
				fmt.Println("found bound:", res)
				_, err = pw2.Write([]byte(res))
				if err != nil {
					if err != io.ErrClosedPipe {
						lgr.Error("gpt-to-elevenlabs bounds write", zap.Error(err))
					}
					return
				}
				currentResponse.Reset()
			case <-ctx.Done():
				lgr.Warn("Context done")
				return
			case <-m.interrupt.Done():
				lgr.Warn("Interrupted")
				return
			}
		}
	}()

	// Start ElevenLabs streaming into the pipe writer in a goroutine
	pr3, pw3 := io.Pipe()
	defer pr3.Close()
	buf3 := make([]byte, 2048)

	go func() {
		defer fmt.Println("ElevenLabs streaming done")
		defer pw3.Close()
		for {
			select {
			default:
				n, errRead := pr2.Read(buf3)
				if errRead != nil {
					if errRead == io.EOF {
						return
					}
					return
				}
				err := m.SM().ElevenLabsService().TextToSpeechStream(string(buf3[:n]), pw3)
				if err != nil {
					if err != io.ErrClosedPipe {
						lgr.Error("elevenlabs stream", zap.Error(err))
					}
					return
				}
			case <-ctx.Done():
				lgr.Warn("Context done")
				return
			case <-m.interrupt.Done():
				lgr.Warn("Interrupted")
				return
			}
		}
	}()

	// reading eleven labs response
	buf4 := make([]byte, 4096)
outer:
	for {
		select {
		default:
			n, errRead := pr3.Read(buf4)
			if errRead != nil {
				if errRead == io.EOF {
					break outer
				}
				if errRead != io.ErrClosedPipe {
					lgr.Error("elevenlabs read", zap.Error(errRead))
				}
				return
			}
			out <- models.StreamResponse{
				Data:    slices.Clone(buf4[:n]), // copy to avoid reuse
				MsgType: tools.Ptr(websocket.BinaryMessage),
			}
		case <-ctx.Done():
			lgr.Warn("Context done")
			return
		case <-m.interrupt.Done():
			lgr.Warn("Interrupted")
			return
		}
	}

	fmt.Println("RETURNING")
}

func (w *gpt4oV1) HandleClose() {}
