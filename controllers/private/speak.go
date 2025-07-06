package private

import (
	gctx "context"
	"fmt"
	"net/http"
	"os"

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
	deepgramKey string
	// in memory cache for agent settings
	settings map[string]*interfaces.SettingsOptions
}

func NewSpeak(ctx context.AppContext, deepgramKey string) *speak {
	return &speak{
		AppContext:  ctx,
		deepgramKey: deepgramKey,
	}
}

func (r speak) Path() string {
	return "/speak"
}

func (r *speak) PrivateRoute(b *builders.PrivateRouteBuilder) {
	b.NewHandle().Register(builders.GET, "/", r.speakGet).SetPermissionName(SpeakGet).Build()
	b.NewHandle().Register(builders.GET, "/ws", r.speakWebSocket).SetPermissionName(SpeakWS).Build()
	b.NewHandle().Register(builders.POST, "/options", r.speakOptionsPost).SetPermissionName(SpeakWS).Build()
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
	ctx, cancel := gctx.WithCancel(req.Context())
	ctx = context.WithCancel(ctx, cancel)
	userID := context.GetUserId(ctx)

	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		tools.HandleError(req, res, lgr, err, 500, "Error setting up websocket")
		return
	}
	defer conn.Close()

	clientOptions := interfaces.ClientOptions{
		EnableKeepAlive: true,
	}

	logFile, err := os.OpenFile("out.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		tools.HandleError(req, res, lgr, err, 500, "Error opening file")
		return
	}
	defer logFile.Close()
	handler := services.NewDeepgramHandler(logFile)

	tOptions := r.GetOptions(userID)
	voiceHandler, err := services.NewVoiceV2(ctx, r.AppContext, r.deepgramKey, &clientOptions, tOptions, handler)
	r.SM().WebSocketService().StartStreamingResponseSocket(conn, voiceHandler)

	lgr.Info("Leaving...")
}

func (r *speak) speakOptionsPost(res http.ResponseWriter, req *http.Request) {
	lgr := r.Lgr("speakOptionsPost")
	lgr.Info("Called")
	ctx := req.Context()

	if err := req.ParseForm(); err != nil {
		tools.HandleError(req, res, lgr, err, 400, "Error parsing form")
		return
	}

	userID := context.GetUserId(ctx)
	opts := r.GetOptions(userID)
	opts.Agent.Think.Prompt = req.FormValue("think-prompt")
	opts.Agent.Think.Provider["model"] = req.FormValue("think-model")
	opts.Agent.Think.Provider["temperature"] = req.FormValue("think-temperature")
	opts.Agent.Think.Provider["model"] = req.FormValue("think-model")
	opts.Agent.Listen.Provider["model"] = req.FormValue("listen-model")
	opts.Agent.Listen.Provider["keyterms"] = req.FormValue("listen-keyterms")
	opts.Agent.Greeting = req.FormValue("greeting")

	fmt.Println("%v", opts.Agent)
}

func (r *speak) DefaultOptions() *interfaces.SettingsOptions {
	tOptions := agent.NewSettingsConfigurationOptions()
	tOptions.Agent.Think.Provider["type"] = "open_ai"
	tOptions.Agent.Think.Provider["model"] = "gpt-4o-mini"
	tOptions.Agent.Think.Provider["temperature"] = "0.7"
	tOptions.Agent.Think.Prompt = "You are a helpful AI assistant."
	tOptions.Agent.Listen.Provider["type"] = "deepgram"
	tOptions.Agent.Listen.Provider["model"] = "nova-3"
	tOptions.Agent.Language = "en"
	tOptions.Agent.Greeting = "Hello! How can I help you today?"
	return tOptions
}

func (r *speak) GetOptions(userID int64) *interfaces.SettingsOptions {
	key := fmt.Sprintf("as:%d", userID)
	if opts, ok := r.settings[key]; ok {
		return opts
	}
	return r.DefaultOptions()
}

func (r *speak) SetOptions(userID int64, opts *interfaces.SettingsOptions) {
	key := fmt.Sprintf("as:%d", userID)
	r.settings[key] = opts
}
