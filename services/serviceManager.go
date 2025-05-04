package services

import (
	"database/sql"

	"github.com/carsonkrueger/main/database/DAO"
	"go.uber.org/zap"
)

type ServiceContext interface {
	Lgr(name string) *zap.Logger
	SM() ServiceManager
	DM() DAO.DAOManager
	DB() *sql.DB
}

type appContext struct {
	Lgr *zap.Logger
	SM  ServiceManager
	DM DAO.DAOManager
	DB *sql.DB
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
}

type serviceManager struct {
	usersService      UsersService
	privilegesService PrivilegesService
	svcCtx ServiceContext
}

func NewServiceManager(svcCtx ServiceContext) *serviceManager {
	return &serviceManager{
		svcCtx: svcCtx,
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

