package services

import (
	gctx "context"
	"encoding/base64"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/carsonkrueger/main/context"
	"github.com/carsonkrueger/main/models"
	"github.com/carsonkrueger/main/tools"
	"github.com/gorilla/websocket"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
	"go.uber.org/zap"
)

type gpt4oVoiceV1 struct {
	speakCadence          time.Duration                             // how often the AI speaks per word
	interrupt             *models.Interrupt                         // used to interrupted ai response pipeline
	interruptMutex        sync.Mutex                                // mutex for interrupt handling
	interruptWaitDuration time.Duration                             // how long the user has to speak to interrupt the ai response pipeline
	waitDuration          time.Duration                             // how long to wait for user to stop speaking
	waitCheckInterval     time.Duration                             // how often to check if user has stopped speaking
	lastUserSpeak         time.Time                                 // time of last user audio req
	lastUserSpeakMutex    sync.Mutex                                // mutex for last user speak time
	audioBuffer           []byte                                    // raw audio buffer
	audioMutex            sync.Mutex                                // mutex for audio buffer
	chatHistory           *[]openai.ChatCompletionMessageParamUnion // entire chat history of conversation
	chatMutex             sync.Mutex                                // mutex for chat history
	openaiClient          *openai.Client
	context.ServiceContext
}

func NewGPT4oVoiceV1(svcCtx context.ServiceContext) *gpt4oVoiceV1 {
	return &gpt4oVoiceV1{
		interrupt:             models.NewInterrupt(),
		interruptWaitDuration: time.Second * 1,
		waitDuration:          time.Millisecond * 500,
		waitCheckInterval:     time.Millisecond * 50,
		ServiceContext:        svcCtx,
		chatHistory:           &[]openai.ChatCompletionMessageParamUnion{},
		speakCadence:          time.Millisecond * 400,
		openaiClient:          svcCtx.SM().LLMService().OpenaiClient(),
	}
}

func (g *gpt4oVoiceV1) Options() models.WebSocketOptions {
	return models.WebSocketOptions{
		PongDeadline:        tools.Ptr(1 * time.Second),
		PongInterval:        tools.Ptr(8 * time.Second),
		AllowedMessageTypes: []int{websocket.BinaryMessage},
	}
}

func (g *gpt4oVoiceV1) SaveUserStreamResponse(ctx gctx.Context, wav []byte) error {
	data := base64.StdEncoding.EncodeToString(wav)
	g.chatMutex.Lock()
	*g.chatHistory = append(*g.chatHistory, openai.ChatCompletionMessageParamUnion{
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
	g.chatMutex.Unlock()
	return nil
}

var whitespaceRegex = regexp.MustCompile(`\s`)

func (g *gpt4oVoiceV1) SaveAssistantTextResponse(text string) error {
	g.chatMutex.Lock()
	*g.chatHistory = append(*g.chatHistory, openai.ChatCompletionMessageParamUnion{
		OfAssistant: &openai.ChatCompletionAssistantMessageParam{
			Content: openai.ChatCompletionAssistantMessageParamContentUnion{
				OfString: openai.String(text),
			},
		},
	})
	g.chatMutex.Unlock()
	return nil
}

func (g *gpt4oVoiceV1) HandleRequestWithStreaming(ctx gctx.Context, req []byte, out chan<- models.StreamResponse) {
	lgr := g.Lgr("HandleRequestWithStreaming")
	var err error

	defer func() {
		out <- models.StreamResponse{Done: true, Err: err}
	}()

	// save audio buffer
	g.audioMutex.Lock()
	g.audioBuffer = append(g.audioBuffer, req...)
	g.audioMutex.Unlock()
	g.lastUserSpeakMutex.Lock()
	g.lastUserSpeak = time.Now()
	g.lastUserSpeakMutex.Unlock()

	// PRE AI RESPONSE
	// if AI is responding, interrupt - maybe wait n time for
	g.interruptMutex.Lock()
	lgr.Warn("SIGNALING INTERRUPT")
	g.interrupt.Signal()
	g.interrupt.Wait()
	g.interrupt.Reset()
	g.interrupt.Add(1) // add interrupt here so that it can be immediately interrupted if needed
	defer g.interrupt.Remove()
	lgr.Warn("INTERRUPT DONE")
	g.interruptMutex.Unlock()

	ticker := time.NewTicker(g.waitCheckInterval)
	defer ticker.Stop()

userSpeak:
	for {
		select {
		case <-ticker.C:
			g.lastUserSpeakMutex.Lock()
			if time.Since(g.lastUserSpeak) > g.waitDuration {
				lgr.Info("wait duration reached")
				g.lastUserSpeakMutex.Unlock()
				break userSpeak
			}
			g.lastUserSpeakMutex.Unlock()
		case <-ctx.Done():
			return
		case <-g.interrupt.Done():
			return
		}
	}

	g.interrupt.Add(4)

	// convert to wav and save the response in the history
	g.audioMutex.Lock()
	if len(g.audioBuffer) > 0 {
		wav := tools.Int16ToWAV(tools.BytesToInt16Slice(g.audioBuffer), 16000)
		g.audioBuffer = []byte{}
		if err := g.SaveUserStreamResponse(ctx, wav); err != nil {
			lgr.Error("SaveUserStreamResponse", zap.Error(err))
			return
		}
	}
	g.audioMutex.Unlock()

	stream := g.openaiClient.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Messages:   *g.chatHistory,
		Modalities: []string{"text"},
		Model:      shared.ChatModelGPT4oMiniAudioPreview,
	})
	pr1, pw1 := io.Pipe()

	defer func() {
		stream.Close()
		pr1.Close()
	}()

	go func() {
		defer func() {
			fmt.Println("gpt streaming done")
			pw1.Close()
			g.interrupt.Remove()
		}()
		for stream.Next() {
			select {
			default:
				if err := stream.Err(); err != nil {
					lgr.Error("GPT Err", zap.Error(err))
					return
				}
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
			case <-ctx.Done():
				return
			case <-g.interrupt.Done():
				return
			}
		}
	}()

	var fullResponse strings.Builder
	var curResponse strings.Builder
	pr2, pw2 := io.Pipe()
	defer pr2.Close()

	buf2 := make([]byte, 1024)
	go func() {
		defer func() {
			fmt.Println("gpt-to-elevenlabs done")
			pw2.Close()
			g.interrupt.Remove()
		}()
		for {
			select {
			default:
				n, errRead := pr1.Read(buf2)
				if errRead != nil {
					return
				}
				res := string(buf2[:n])
				fullResponse.Write(buf2[:n])
				curResponse.Write(buf2[:n])
				if !tools.IsBoundary(res, true) {
					continue
				}
				fmt.Println("found bound:", curResponse.String())
				_, err = pw2.Write([]byte(curResponse.String()))
				if err != nil {
					if err != io.ErrClosedPipe {
						lgr.Error("gpt-to-elevenlabs bounds write", zap.Error(err))
					}
					return
				}
				curResponse.Reset()
			case <-ctx.Done():
				return
			case <-g.interrupt.Done():
				return
			}
		}
	}()

	pr3, pw3 := io.Pipe()
	defer pr3.Close()
	buf3 := make([]byte, 2048)

	// Start ElevenLabs streaming into the pipe writer in a goroutine
	go func() {
		defer func() {
			fmt.Println("ElevenLabs streaming done")
			pw3.Close()
			g.interrupt.Remove()
		}()
		for {
			select {
			default:
				n, errRead := pr2.Read(buf3)
				if errRead != nil {
					return
				}
				err := g.SM().ElevenLabsService().TextToSpeechStream(string(buf3[:n]), pw3)
				if err != nil {
					if err != io.ErrClosedPipe {
						lgr.Error("elevenlabs stream", zap.Error(err))
					}
					return
				}
			case <-ctx.Done():
				return
			case <-g.interrupt.Done():
				return
			}
		}
	}()

	// reading eleven labs response and sending response back through the channel passed into the function
	var speakResponseStarted *time.Time
	speakDuration := time.Duration(0)
	buf4 := make([]byte, 4096)
	defer g.interrupt.Remove()

outer:
	for {
		select {
		default:
			n, errRead := pr3.Read(buf4)
			if errRead != nil {
				if errRead == io.EOF {
					break outer
				}
				lgr.Error("elevenlabs read", zap.Error(errRead))
				break outer
			}
			if speakResponseStarted == nil {
				speakResponseStarted = tools.Ptr(time.Now())
			}
			speakDuration += tools.MsPcmDuration(n, 16000, 1, 16)
			out <- models.StreamResponse{
				Data:    slices.Clone(buf4[:n]), // copy to avoid reuse
				MsgType: tools.Ptr(websocket.BinaryMessage),
			}
		case <-ctx.Done():
			break outer
		case <-g.interrupt.Done():
			break outer
		}
	}

	if speakResponseStarted != nil {
		g.interrupt.Add(1)
		defer g.interrupt.Remove()
		startedCalc := time.Now()
		sinceStartedResponse := time.Since(*speakResponseStarted)
	outerSpeak:
		for {
			select {
			case <-time.After(speakDuration - sinceStartedResponse):
				// waited entire audio duration - save entire response
				fmt.Println("ENTIRE RESPONSE SAVED")
				g.SaveAssistantTextResponse(fullResponse.String())
				break outerSpeak
			case <-ctx.Done():
				// interrupted, calculate duration spoken and save partial response
				g.CalculateAndSaveAssistantResponse(sinceStartedResponse+time.Since(startedCalc), fullResponse.String())
				lgr.Warn("Context done")
				break outerSpeak
			case <-g.interrupt.Done():
				// interrupted, calculate duration spoken and save partial response
				g.CalculateAndSaveAssistantResponse(sinceStartedResponse+time.Since(startedCalc), fullResponse.String())
				break outerSpeak
			}
		}
	}
}

func (g *gpt4oVoiceV1) HandleClose() {}

func (g *gpt4oVoiceV1) CalculateAndSaveAssistantResponse(speakDuration time.Duration, prompt string) {
	if prompt == "" || speakDuration == 0 {
		return
	}
	numWordsSpoken := int64(speakDuration / g.speakCadence)
	fmt.Printf("Words spoken: %d\n", numWordsSpoken)
	var nthWord int64
	idx := 0
	wasWhitespace := false
	for i, r := range prompt {
		fmt.Print(r)
		if i == len(prompt)-1 {
			idx = i
			break
		}
		if !wasWhitespace && unicode.IsSpace(r) {
			wasWhitespace = true
			nthWord++
			idx = i
		} else {
			wasWhitespace = false
		}
		if nthWord >= numWordsSpoken {
			break
		}
	}
	if idx <= 0 {
		fmt.Print("No words spoken")
		return
	}
	fmt.Println(prompt[:idx])
	g.SaveAssistantTextResponse(prompt[:idx])
}
