package private

import (
	"net/http"
	"time"

	"github.com/carsonkrueger/main/builders"
	"github.com/carsonkrueger/main/context"
	"github.com/carsonkrueger/main/models"
	"github.com/carsonkrueger/main/templates/pageLayouts"
	"github.com/carsonkrueger/main/templates/pages"
	"github.com/carsonkrueger/main/tools"
	"github.com/gorilla/websocket"
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
	// lgr := r.Lgr("textWebSocket")
	// lgr.Info("Called")

	// conn, err := upgrader.Upgrade(res, req, nil)
	// if err != nil {
	// 	tools.HandleError(req, res, lgr, err, 500, "Error setting up websocket")
	// 	return
	// }
	// defer conn.Close()

	// mcpClient := r.SM().MCPService().Client()
	// llm := r.SM().LLMService().LLM()
	// agent, memoryBuffer, err := models.NewLangChainConversationalAgent(nil, mcpClient, llm)
	// if err != nil {
	// 	tools.HandleError(req, res, lgr, err, 500, "Error creating agent")
	// 	return
	// }

	// textHandler := services.NewGPT4oTextV1(&agent, memoryBuffer, r.AppContext)
	// r.SM().WebSocketService().StartSocket(conn, textHandler)
}

func (r *webText) Options() models.WebSocketOptions {
	return models.WebSocketOptions{
		KeepAliveDuration:   10 * time.Minute,
		PongDeadline:        tools.Ptr(10 * time.Minute),
		PongInterval:        tools.Ptr(10 * time.Second),
		AllowedMessageTypes: []int{websocket.TextMessage},
	}
}
