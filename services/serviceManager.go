package services

import (
	"context"
	"database/sql"

	"github.com/carsonkrueger/main/database/DAO"
	// "github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
	"github.com/haguro/elevenlabs-go"
	"github.com/tmc/langchaingo/llms"
	"go.uber.org/zap"
)

type ServiceManagerContext interface {
	PrimaryModel() llms.Model
	ElevenLabsClient() *elevenlabs.Client
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
	TextToVoice() TextToVoiceConverter
	SpeechToText() SpeechToTextConverter
	WebSocketService() WebSocketService
	MCPService() AppMCPService
}

type serviceManager struct {
	usersService        UsersService
	privilegesService   PrivilegesService
	llmService          LLMService
	phoneService        PhoneService
	textToVoiceService  TextToVoiceConverter
	speechToTextService SpeechToTextConverter
	webSocketService    WebSocketService
	mcpService          AppMCPService
	svcCtx              ServiceContext
	ctx                 ServiceManagerContext
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

type TextToVoiceConverter interface {
	TextToSpeech(msg string) ([]byte, error)
}

type SpeechToTextConverter interface {
	SpeechToText(audio []byte) (string, error)
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

func (sm *serviceManager) PhoneService() PhoneService {
	if sm.phoneService == nil {
		// implement concrete phone service here
		// sm.phoneService = NewTwilioService(sm.svcCtx)
	}
	return sm.phoneService
}

func (sm *serviceManager) TextToVoice() TextToVoiceConverter {
	if sm.textToVoiceService == nil {
		sm.textToVoiceService = NewElevenLabsService(sm.svcCtx, sm.ctx.ElevenLabsClient())
	}
	return sm.textToVoiceService
}

func (sm *serviceManager) SpeechToText() SpeechToTextConverter {
	if sm.speechToTextService == nil {
		// sm.speechToTextService = NewWhisperCPPService(sm.svcCtx, sm.ctx.WhisperCPPModel())
	}
	return sm.speechToTextService
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
