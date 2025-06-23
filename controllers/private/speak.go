package private

import (
	"net/http"

	"github.com/carsonkrueger/main/builders"
	"github.com/carsonkrueger/main/context"
	"github.com/carsonkrueger/main/services"
	"github.com/carsonkrueger/main/templates/pageLayouts"
	"github.com/carsonkrueger/main/templates/pages"
	"github.com/carsonkrueger/main/tools"
	"github.com/deepgram/deepgram-go-sdk/v3/pkg/client/agent"
	"github.com/deepgram/deepgram-go-sdk/v3/pkg/client/interfaces"
	"github.com/gorilla/websocket"
)

const (
	SpeakGet    = "SpeakGet"
	SpeakWS     = "SpeakWS"
	SpeakPut    = "SpeakPut"
	SpeakPatch  = "SpeakPatch"
	SpeakDelete = "SpeakDelete"
)

type speak struct {
	context.AppContext
}

func NewSpeak(ctx context.AppContext) *speak {
	return &speak{
		AppContext: ctx,
	}
}

func (r speak) Path() string {
	return "/speak"
}

func (r *speak) PrivateRoute(b *builders.PrivateRouteBuilder) {
	b.NewHandle().Register(builders.GET, "/", r.speakGet).SetPermissionName(SpeakGet).Build()
	b.NewHandle().Register(builders.GET, "/ws", r.speakWebSocket).SetPermissionName(SpeakWS).Build()
}

func (r *speak) speakGet(res http.ResponseWriter, req *http.Request) {
	lgr := r.Lgr("speakGet")
	lgr.Info("Called")
	ctx := req.Context()
	page := pageLayouts.Index(pages.Speak())
	page.Render(ctx, res)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  8192,
	WriteBufferSize: 8192,
}

func (r *speak) speakWebSocket(res http.ResponseWriter, req *http.Request) {
	lgr := r.Lgr("speakWebSocket")
	lgr.Info("Called")
	ctx := req.Context()

	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		tools.HandleError(req, res, lgr, err, 500, "Error setting up websocket")
		return
	}
	defer conn.Close()

	tOptions := agent.NewSettingsConfigurationOptions()
	tOptions.Agent.Think.Provider["type"] = "open_ai"
	tOptions.Agent.Think.Provider["model"] = "gpt-4o-mini"
	tOptions.Agent.Think.Prompt = "You are a helpful AI assistant."
	tOptions.Agent.Listen.Provider["type"] = "deepgram"
	tOptions.Agent.Listen.Provider["model"] = "nova-3"
	tOptions.Agent.Listen.Provider["keyterms"] = []string{"Bueller"}
	tOptions.Agent.Language = "en"
	tOptions.Agent.Greeting = "Hello! How can I help you today?"
	tOptions.Audio.Output.Encoding = "wav"
	tOptions.Audio.Output.Bitrate = 16
	tOptions.Audio.Output.SampleRate = 16000
	var clientOptions *interfaces.ClientOptions
	handler := services.DeepgramHandler{}

	voiceHandler, err := services.NewVoiceV2(ctx, r.AppContext, "api-key", clientOptions, tOptions, handler)
	r.SM().WebSocketService().StartStreamingResponseSocket(conn, voiceHandler)

	lgr.Info("Leaving...")
}
