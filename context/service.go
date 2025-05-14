package context

import (
	"github.com/haguro/elevenlabs-go"
	"github.com/tmc/langchaingo/llms"
)

type serviceManagerContext struct {
	primaryModel        llms.Model
	elevenLabsClient    *elevenlabs.Client
	whisperCPPModelPath string
}

func NewServiceManagerContext(primaryModel llms.Model, elevenLabsClient *elevenlabs.Client, whisperCPPModelPath string) *serviceManagerContext {
	return &serviceManagerContext{
		primaryModel,
		elevenLabsClient,
		whisperCPPModelPath,
	}
}

func (c *serviceManagerContext) PrimaryModel() llms.Model {
	return c.primaryModel
}

func (c *serviceManagerContext) ElevenLabsClient() *elevenlabs.Client {
	return c.elevenLabsClient
}

func (c *serviceManagerContext) WhisperCPPModelPath() string {
	return c.whisperCPPModelPath
}
