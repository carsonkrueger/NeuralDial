package services

import (
	"context"
	"database/sql"

	"github.com/carsonkrueger/main/database/DAO"
	"github.com/openai/openai-go"

	// "github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
	"github.com/carsonkrueger/elevenlabs-go"
	"github.com/tmc/langchaingo/llms"
	"go.uber.org/zap"
)

type ServiceManagerContext interface {
	PrimaryModel() llms.Model
	ElevenLabsClient() *elevenlabs.Client
	OpenaiClient() *openai.Client

	// WhisperCPPModel() whisper.Model
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
	PhoneService() PhoneService
	WebSocketService() WebSocketService
	MCPService() AppMCPService
	ElevenLabsService() ElevenLabsService
}

type serviceManager struct {
	usersService      UsersService
	privilegesService PrivilegesService
	llmService        LLMService
	phoneService      PhoneService
	webSocketService  WebSocketService
	mcpService        AppMCPService
	elevenLabsService ElevenLabsService
	svcCtx            ServiceContext
	ctx               ServiceManagerContext
}

func NewServiceManager(svcCtx ServiceContext, svcManagerCtx ServiceManagerContext) *serviceManager {
	return &serviceManager{
		svcCtx: svcCtx,
		ctx:    svcManagerCtx,
	}
}

type PhoneService interface {
	StartCall(ctx context.Context) error
	EndCall(ctx context.Context) error
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
		sm.llmService = NewLLMService(sm.svcCtx, sm.ctx.PrimaryModel(), sm.ctx.OpenaiClient())
	}
	return sm.llmService
}

func (sm *serviceManager) PhoneService() PhoneService {
	if sm.phoneService == nil {
		// implement concrete phone service here
		// sm.phoneService = NewTwilioService(sm.svcCtx)
	}
	return sm.phoneService
}

func (sm *serviceManager) WebSocketService() WebSocketService {
	if sm.webSocketService == nil {
		sm.webSocketService = NewWebSocketService(sm.svcCtx)
	}
	return sm.webSocketService
}

func (sm *serviceManager) MCPService() AppMCPService {
	if sm.mcpService == nil {
		sm.mcpService = NewMcpService(sm.svcCtx)
	}
	return sm.mcpService
}

func (sm *serviceManager) ElevenLabsService() ElevenLabsService {
	if sm.elevenLabsService == nil {
		sm.elevenLabsService = NewElevenLabsService(sm.svcCtx, sm.ctx.ElevenLabsClient())
	}
	return sm.elevenLabsService
}
