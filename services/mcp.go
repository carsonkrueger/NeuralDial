package services

import (
	gctx "context"
	"fmt"
	"io"
	"net/http"

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
	logTool := mcp.NewTool("console_log", mcp.WithNumber("number", mcp.Min(0), mcp.Max(64), mcp.Required()), mcp.WithDescription("Use if the user gives you number to log"))
	webSearchTool := mcp.NewTool("web_search", mcp.WithString("url", mcp.Required()), mcp.WithDescription("Use if the user gives you a url to answer questions about"))
	s.AddTool(logTool, m.loggingTool)
	s.AddTool(webSearchTool, m.webSearchTool)
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

func (s *appMCP) webSearchTool(ctx gctx.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url := request.GetString("url", "")
	if url == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent("Please provide a valid URL"),
			},
			IsError: true,
		}, nil
	}
	fmt.Println(url)
	// make request to url
	res, err := http.Get(url)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent("Could not fetch URL"),
			},
			IsError: true,
		}, nil
	}
	defer res.Body.Close()
	bodyBytes, err := io.ReadAll(io.LimitReader(res.Body, 1_000)) // 1MB limit for safety
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent("Error reading response body"),
			},
			IsError: true,
		}, nil
	}
	bodyText := string(bodyBytes)
	fmt.Println(bodyText)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(bodyText),
		},
	}, nil
}

func (s *appMCP) Server() *server.MCPServer {
	return s.server
}

func (s *appMCP) Client() *client.Client {
	return s.client
}
