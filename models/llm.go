package models

import (
	"sync"

	"github.com/tmc/langchaingo/llms"
)

type LLMStreamingModel struct {
	messages []llms.MessageContent
	mutex    sync.RWMutex
}

func (m *LLMStreamingModel) AddText(msg llms.MessageContent, msgs ...llms.MessageContent) {
	m.mutex.Lock()
	m.messages = append(m.messages, msg)
	m.messages = append(m.messages, msgs...)
	m.mutex.Unlock()
}

func (m *LLMStreamingModel) Messages() []llms.MessageContent {
	return m.messages
}
