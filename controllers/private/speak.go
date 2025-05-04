package private

import (
	"net/http"

	"github.com/carsonkrueger/main/builders"
	"github.com/carsonkrueger/main/context"
	"github.com/carsonkrueger/main/templates/pageLayouts"
	"github.com/carsonkrueger/main/templates/pages"
	"github.com/carsonkrueger/main/tools"
)

const (
	SpeakGet    = "SpeakGet"
	SpeakPost   = "SpeakPost"
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

func (um speak) Path() string {
	return "/speak"
}

func (um *speak) PrivateRoute(b *builders.PrivateRouteBuilder) {
	b.NewHandle().Register(builders.GET, "/", um.speakGet).SetPermissionName(SpeakGet).Build()
	b.NewHandle().Register(builders.POST, "/", um.speakPost).SetPermissionName(SpeakPost).Build()
	b.NewHandle().Register(builders.PUT, "/", um.speakPut).SetPermissionName(SpeakPut).Build()
	b.NewHandle().Register(builders.PATCH, "/", um.speakPatch).SetPermissionName(SpeakPatch).Build()
	b.NewHandle().Register(builders.DELETE, "/", um.speakDelete).SetPermissionName(SpeakDelete).Build()
}

func (r *speak) speakGet(res http.ResponseWriter, req *http.Request) {
	lgr := r.Lgr("speakGet")
	lgr.Info("Called")
	ctx := req.Context()
	page := pageLayouts.Index(pages.Speak())
	page.Render(ctx, res)
}

func (r *speak) speakPost(res http.ResponseWriter, req *http.Request) {
	lgr := r.Lgr("speakPost")
	lgr.Info("Called")
	ctx := req.Context()

	err := r.SM().LLMService().Generate(ctx)
	if err != nil {
		tools.HandleError(req, res, lgr, err, 500, "Could not generate response")
		return
	}
}

func (r *speak) speakPut(res http.ResponseWriter, req *http.Request) {
	lgr := r.Lgr("speakPut")
	lgr.Info("Called")
}

func (r *speak) speakPatch(res http.ResponseWriter, req *http.Request) {
	lgr := r.Lgr("speakPatch")
	lgr.Info("Called")
}

func (r *speak) speakDelete(res http.ResponseWriter, req *http.Request) {
	lgr := r.Lgr("speakDelete")
	lgr.Info("Called")
}
