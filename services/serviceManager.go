package services

import (
	"github.com/carsonkrueger/main/context"
)

type serviceManager struct {
	usersService      context.UsersService
	privilegesService context.PrivilegesService
	llmService        context.LLMService
	phoneService      context.PhoneService
	webSocketService  context.WebSocketService
	mcpService        context.AppMCPService
	elevenLabsService context.ElevenLabsService
	svcCtx            context.ServiceContext
	ctx               context.ServiceManagerContext
}

func NewServiceManager(svcCtx context.ServiceContext, svcManagerCtx context.ServiceManagerContext) *serviceManager {
	return &serviceManager{
		svcCtx: svcCtx,
		ctx:    svcManagerCtx,
	}
}

func (sm *serviceManager) SetAppContext(svcCtx context.ServiceContext) {
	sm.svcCtx = svcCtx
}

func (sm *serviceManager) UsersService() context.UsersService {
	if sm.usersService == nil {
		sm.usersService = NewUsersService(sm.svcCtx)
	}
	return sm.usersService
}

func (sm *serviceManager) PrivilegesService() context.PrivilegesService {
	if sm.privilegesService == nil {
		sm.privilegesService = NewPrivilegesService(sm.svcCtx)
	}
	return sm.privilegesService
}

func (sm *serviceManager) LLMService() context.LLMService {
	if sm.llmService == nil {
		sm.llmService = NewLLMService(sm.svcCtx, sm.ctx.PrimaryModel(), sm.ctx.OpenaiClient())
	}
	return sm.llmService
}

func (sm *serviceManager) PhoneService() context.PhoneService {
	if sm.phoneService == nil {
		// implement concrete phone service here
		// sm.phoneService = NewTwilioService(sm.svcCtx)
	}
	return sm.phoneService
}

func (sm *serviceManager) WebSocketService() context.WebSocketService {
	if sm.webSocketService == nil {
		sm.webSocketService = NewWebSocketService(sm.svcCtx)
	}
	return sm.webSocketService
}

func (sm *serviceManager) MCPService() context.AppMCPService {
	if sm.mcpService == nil {
		sm.mcpService = NewMcpService(sm.svcCtx)
	}
	return sm.mcpService
}

func (sm *serviceManager) ElevenLabsService() context.ElevenLabsService {
	if sm.elevenLabsService == nil {
		sm.elevenLabsService = NewElevenLabsService(sm.svcCtx, sm.ctx.ElevenLabsClient())
	}
	return sm.elevenLabsService
}
