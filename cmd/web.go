package cmd

import (
	gctx "context"
	"database/sql"
	"net/http"
	"time"

	"github.com/carsonkrueger/elevenlabs-go"
	"github.com/carsonkrueger/main/cfg"
	"github.com/carsonkrueger/main/context"
	"github.com/carsonkrueger/main/database/DAO"
	"github.com/carsonkrueger/main/logger"
	"github.com/carsonkrueger/main/router"
	"github.com/carsonkrueger/main/services"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	// "github.com/mark3labs/mcp-go/server"
	lang_openai "github.com/tmc/langchaingo/llms/openai"

	_ "github.com/lib/pq"
)

func web() {
	cfg := cfg.LoadConfig()
	lgr := logger.NewLogger(&cfg)
	ctx := gctx.Background()

	db, err := sql.Open("postgres", cfg.DbUrl())
	defer db.Close()
	if err != nil {
		panic(err)
	}
	if db == nil {
		panic("Database connection is nil")
	}

	open4oMini, err := lang_openai.New(
		lang_openai.WithToken(cfg.OpenAIAPIKey),
		lang_openai.WithModel("gpt-4o-mini"),
	)
	openClient := openai.NewClient(option.WithAPIKey(cfg.OpenAIAPIKey))
	if err != nil || open4oMini == nil {
		panic(err)
	}

	httpClient := http.Client{}
	elevenLabsClient := elevenlabs.NewClient(&httpClient, ctx, cfg.ElevenLabsAPIKey, 10*time.Second)
	svcManagerCtx := context.NewServiceManagerContext(open4oMini, openClient, elevenLabsClient, cfg.WhisperModelPath)

	dm := DAO.NewDAOManager(db)
	sm := services.NewServiceManager(nil, svcManagerCtx)
	appCtx := context.NewAppContext(
		lgr,
		sm,
		dm,
		db,
	)
	sm.SetAppContext(appCtx)
	defer appCtx.CleanUp()

	appRouter := router.NewAppRouter(appCtx, cfg)
	appRouter.BuildRouter()
	if err := appRouter.Start(cfg); err != nil {
		panic(err)
	}
}
