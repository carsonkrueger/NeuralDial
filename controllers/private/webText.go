package private

import (
	"net/http"

	"github.com/carsonkrueger/main/builders"
	"github.com/carsonkrueger/main/context"
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

	r.SM().WebSocketService().StartConversation(conn)

	lgr.Info("Leaving...")
}
