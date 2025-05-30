package context

import (
	gctx "context"
	"database/sql"
	"io"
	"net/http"

	"github.com/carsonkrueger/main/database/DAO"
	"github.com/carsonkrueger/main/gen/go_db/auth/model"
	"github.com/carsonkrueger/main/models"
	"github.com/carsonkrueger/main/models/authModels"
	"github.com/carsonkrueger/main/templates/datadisplay"
	"github.com/gorilla/websocket"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/server"
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

type ServiceManager interface {
	UsersService() UsersService
	PrivilegesService() PrivilegesService
	LLMService() LLMService
	PhoneService() PhoneService
	WebSocketService() WebSocketService
	MCPService() AppMCPService
	ElevenLabsService() ElevenLabsService
}

type ElevenLabsService interface {
	TextToSpeechStream(msg string, w io.Writer) error
}

type UsersService interface {
	Login(email string, password string, req *http.Request) (*string, error)
	Logout(id int64, token string) error
	LogoutRequest(req *http.Request) error
	GetAuthParts(req *http.Request) (string, int64, error)
}

type PrivilegesService interface {
	CreatePrivilegeAssociation(levelID int64, privID int64) error
	DeletePrivilegeAssociation(levelID int64, privID int64) error
	CreateLevel(name string) error
	HasPermissionByID(levelID int64, permissionID int64) bool
	SetUserPrivilegeLevel(levelID int64, userID int64) error
	UserPrivilegeLevelJoinAsRowData(upl []authModels.UserPrivilegeLevelJoin, allLevels []*model.PrivilegeLevels) []datadisplay.RowData
	JoinedPrivilegeLevelAsRowData(jpl []authModels.JoinedPrivilegeLevel) []datadisplay.RowData
	JoinedPrivilegesAsRowData(jpl []authModels.JoinedPrivilegesRaw) []datadisplay.RowData
}

type LLMService interface {
	LLM() llms.Model
	OpenaiClient() *openai.Client
}

type AppMCPService interface {
	Server() *server.MCPServer
	Client() *client.Client
}

type PhoneService interface {
	StartCall(ctx gctx.Context) error
	EndCall(ctx gctx.Context) error
}

type WebSocketService interface {
	StartSocket(conn *websocket.Conn, handler WebSocketHandler, opts *models.WebSocketOptions)
	StartStreamingResponseSocket(conn *websocket.Conn, handler WebSocketHandler, opts *models.WebSocketOptions)
}

type WebSocketHandler interface {
	HandleRequest(ctx gctx.Context, msgType int, req []byte) (*int, []byte, error)
	HandleRequestWithStreaming(ctx gctx.Context, req []byte, out chan<- models.StreamResponse)
	HandleClose()
}
