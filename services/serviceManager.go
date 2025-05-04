package services

import (
	"database/sql"

	"github.com/carsonkrueger/main/database/DAO"
	"github.com/tmc/langchaingo/llms"
	"go.uber.org/zap"
)

type ServiceManagerContext interface {
	PrimaryModel() llms.Model
}

type ServiceContext interface {
	Lgr(name string) *zap.Logger
	SM() ServiceManager
	DM() DAO.DAOManager
	DB() *sql.DB
}

type appContext struct {
	Lgr *zap.Logger
	SM  ServiceManager
	DM  DAO.DAOManager
	DB  *sql.DB
}

func NewAppContext(
	lgr *zap.Logger,
	sm ServiceManager,
	dm DAO.DAOManager,
	db *sql.DB,
) *appContext {
	return &appContext{
		lgr,
		sm,
		dm,
		db,
	}
}

type ServiceManager interface {
	UsersService() UsersService
	PrivilegesService() PrivilegesService
	LLMService() LLMService
}

type serviceManager struct {
	usersService      UsersService
	privilegesService PrivilegesService
	llmService        LLMService
	svcCtx            ServiceContext
	ctx               ServiceManagerContext
}

func NewServiceManager(svcCtx ServiceContext, svcManagerCtx ServiceManagerContext) *serviceManager {
	return &serviceManager{
		svcCtx: svcCtx,
		ctx:    svcManagerCtx,
	}
}

func (sm *serviceManager) SetAppContext(svcCtx ServiceContext) {
	sm.svcCtx = svcCtx
}

func (sm *serviceManager) UsersService() UsersService {
	if sm.usersService == nil {
		sm.usersService = NewUsersService(sm.svcCtx)
	}
	return sm.usersService
}

func (sm *serviceManager) PrivilegesService() PrivilegesService {
	if sm.privilegesService == nil {
		sm.privilegesService = NewPrivilegesService(sm.svcCtx)
	}
	return sm.privilegesService
}

func (sm *serviceManager) LLMService() LLMService {
	if sm.llmService == nil {
		sm.llmService = NewLLMService(sm.svcCtx, sm.ctx.PrimaryModel())
	}
	return sm.llmService
}
