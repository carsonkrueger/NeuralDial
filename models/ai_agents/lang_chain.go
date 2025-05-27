package ai_agents

import (
	langchaingo_mcp_adapter "github.com/i2y/langchaingo-mcp-adapter"
	"github.com/mark3labs/mcp-go/client"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory"
)

func NewLangChainConversationalAgent(initialMessages []llms.ChatMessage, client *client.Client, llm llms.Model) (agents.Agent, *memory.ConversationBuffer, error) {
	adapter, err := langchaingo_mcp_adapter.New(client)
	if err != nil {
		return nil, nil, err
	}
	tools, err := adapter.Tools()
	if err != nil {
		return nil, nil, err
	}
	mem := memory.NewChatMessageHistory(memory.WithPreviousMessages(initialMessages))
	memoryBuffer := memory.NewConversationBuffer(memory.WithChatHistory(mem))
	return agents.NewConversationalAgent(llm, tools), memoryBuffer, nil
}
