package cmd

import (
	gctx "context"
	"database/sql"
	"time"

	"github.com/carsonkrueger/main/cfg"
	"github.com/carsonkrueger/main/context"
	"github.com/carsonkrueger/main/database/DAO"
	"github.com/carsonkrueger/main/logger"
	"github.com/carsonkrueger/main/router"
	"github.com/carsonkrueger/main/services"
	"github.com/haguro/elevenlabs-go"
	"github.com/tmc/langchaingo/llms/openai"

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

	openAILLM, err := openai.New(
		openai.WithToken(cfg.OpenAIAPIKey),
		openai.WithModel("gpt-4o-mini"),
	)
	if err != nil || openAILLM == nil {
		panic(err)
	}

	elevenLabsClient := elevenlabs.NewClient(ctx, cfg.ElevenLabsAPIKey, 10*time.Second)

	svcManagerCtx := context.NewServiceManagerContext(openAILLM, elevenLabsClient)

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

	appRouter := router.NewAppRouter(appCtx)
	appRouter.BuildRouter()
	if err := appRouter.Start(cfg); err != nil {
		panic(err)
	}
}
