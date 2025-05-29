package context

import (
	// "github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
	"github.com/carsonkrueger/elevenlabs-go"
	"github.com/openai/openai-go"
	"github.com/tmc/langchaingo/llms"
)

type serviceManagerContext struct {
	primaryModel     llms.Model
	openaiClient     openai.Client
	elevenLabsClient *elevenlabs.Client
}

func NewServiceManagerContext(primaryModel llms.Model, openaiClient openai.Client, elevenLabsClient *elevenlabs.Client, whisperCPPModelPath string) *serviceManagerContext {
	return &serviceManagerContext{
		primaryModel,
		openaiClient,
		elevenLabsClient,
	}
}

func (c *serviceManagerContext) PrimaryModel() llms.Model {
	return c.primaryModel
}

func (c *serviceManagerContext) OpenaiClient() *openai.Client {
	return &c.openaiClient
}

func (c *serviceManagerContext) ElevenLabsClient() *elevenlabs.Client {
	return c.elevenLabsClient
}
