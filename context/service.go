package context

import (
	"github.com/haguro/elevenlabs-go"
	"github.com/tmc/langchaingo/llms"
)

type serviceManagerContext struct {
	primaryModel     llms.Model
	elevenLabsClient *elevenlabs.Client
}

func NewServiceManagerContext(primaryModel llms.Model, elevenLabsClient *elevenlabs.Client) *serviceManagerContext {
	return &serviceManagerContext{
		primaryModel,
		elevenLabsClient,
	}
}

func (c *serviceManagerContext) PrimaryModel() llms.Model {
	return c.primaryModel
}

func (c *serviceManagerContext) ElevenLabsClient() *elevenlabs.Client {
	return c.elevenLabsClient
}
