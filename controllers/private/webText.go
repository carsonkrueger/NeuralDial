package private

import (
	"net/http"

	"github.com/carsonkrueger/main/builders"
	"github.com/carsonkrueger/main/context"
	"github.com/carsonkrueger/main/models/ai_agents"
	"github.com/carsonkrueger/main/models/ai_models/openai"
	"github.com/carsonkrueger/main/models/ai_models/openai/text"
	"github.com/carsonkrueger/main/templates/pageLayouts"
	"github.com/carsonkrueger/main/templates/pages"
	"github.com/carsonkrueger/main/tools"
)

const (
	WebTextGet = "WebTextGet"
	WebTextWS  = "WebTextWS"
)

type webText struct {
	context.AppContext
}

func NewWebText(ctx context.AppContext) *webText {
	return &webText{
		AppContext: ctx,
	}
}

func (um webText) Path() string {
	return "/web_text"
}

func (um *webText) PrivateRoute(b *builders.PrivateRouteBuilder) {
	b.NewHandle().Register(builders.GET, "/", um.webTextGet).SetPermissionName(WebTextGet).Build()
	b.NewHandle().Register(builders.GET, "/ws", um.textWebSocket).SetPermissionName(WebTextWS).Build()
}

func (r *webText) webTextGet(res http.ResponseWriter, req *http.Request) {
	lgr := r.Lgr("webTextGet")
	lgr.Info("Called")
	ctx := req.Context()
	page := pageLayouts.Index(pages.TextChat())
	page.Render(ctx, res)
}

func (r *webText) textWebSocket(res http.ResponseWriter, req *http.Request) {
	lgr := r.Lgr("textWebSocket")
	lgr.Info("Called")

	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		tools.HandleError(req, res, lgr, err, 500, "Error setting up websocket")
		return
	}
	defer conn.Close()

	mcpClient := r.SM().MCPService().Client()
	llm := r.SM().LLMService().LLM()
	agent, memoryBuffer, err := ai_agents.NewLangChainConversationalAgent(nil, mcpClient, llm)
	if err != nil {
		tools.HandleError(req, res, lgr, err, 500, "Error creating agent")
		return
	}

	opts := openai.NewTextOptions()
	textHandler := text.NewGPT4oV1(&agent, memoryBuffer, r.AppContext)
	r.SM().WebSocketService().StartSocket(conn, textHandler, &opts)
}
