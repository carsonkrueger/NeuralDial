package private

import (
	"net/http"
	"time"

	"github.com/carsonkrueger/main/builders"
	"github.com/carsonkrueger/main/context"
	"github.com/carsonkrueger/main/models/ai_models/openai"
	"github.com/carsonkrueger/main/models/ai_models/openai/voice"
	"github.com/carsonkrueger/main/templates/pageLayouts"
	"github.com/carsonkrueger/main/templates/pages"
	"github.com/carsonkrueger/main/tools"
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

	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		tools.HandleError(req, res, lgr, err, 500, "Error setting up websocket")
		return
	}
	defer conn.Close()

	handler := voice.NewGPT4oV1(r.AppContext, time.Second*1, time.Millisecond*100, r.SM().LLMService().OpenaiClient())
	opts := openai.NewVoiceOptions()
	r.SM().WebSocketService().StartStreamingResponseSocket(conn, handler, &opts)

	lgr.Info("Leaving...")
}
