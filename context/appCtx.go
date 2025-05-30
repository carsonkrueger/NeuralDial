package context

import (
	"database/sql"

	"github.com/carsonkrueger/main/database/DAO"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type AppContext interface {
	Lgr(name string) *zap.Logger
	SM() ServiceManager
	DM() DAO.DAOManager
	DB() *sql.DB
}

type appContext struct {
	Logger         *zap.Logger
	ServiceManager ServiceManager
	DAOManger      DAO.DAOManager
	Database       *sql.DB
}

func NewAppContext(
	Logger *zap.Logger,
	ServiceManager ServiceManager,
	DAOManger DAO.DAOManager,
	Database *sql.DB,
) *appContext {
	return &appContext{
		Logger,
		ServiceManager,
		DAOManger,
		Database,
	}
}

func (ctx *appContext) Lgr(name string) *zap.Logger {
	return ctx.Logger.Named(name)
}

func (ctx *appContext) SM() ServiceManager {
	return ctx.ServiceManager
}

func (ctx *appContext) DM() DAO.DAOManager {
	return ctx.DAOManger
}

func (ctx *appContext) DB() *sql.DB {
	return ctx.Database
}

func (ctx *appContext) CleanUp() {
	if err := ctx.Logger.Sync(); err != nil {
		ctx.Logger.Error("failed to sync logger", zap.Error(err))
	}
}
