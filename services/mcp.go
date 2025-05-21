package services

import (
	gctx "context"
	"fmt"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type AppMCPService interface {
	Server() *server.MCPServer
	Client() *client.Client
}

type appMCP struct {
	ServiceContext
	server *server.MCPServer
	client *client.Client
}

func NewMcpService(ctx ServiceContext) *appMCP {
	s := server.NewMCPServer("llm-agent", "1.0")
	client, err := client.NewInProcessClient(s)
	if err != nil {
		panic(err)
	}
	m := &appMCP{
		ServiceContext: ctx,
		server:         s,
		client:         client,
	}
	tool := mcp.NewTool("console_log", mcp.WithNumber("number", mcp.Min(0), mcp.Max(64), mcp.Required()))
	s.AddTool(tool, m.loggingTool)
	return m
}

func (s *appMCP) loggingTool(ctx gctx.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	number := request.GetInt("number", 0)
	fmt.Println(number)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent("logged"),
		},
	}, nil
}

func (s *appMCP) Server() *server.MCPServer {
	return s.server
}

func (s *appMCP) Client() *client.Client {
	return s.client
}
