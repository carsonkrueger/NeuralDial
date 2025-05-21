package context

import (
	// "github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
	"github.com/haguro/elevenlabs-go"
	"github.com/tmc/langchaingo/llms"
)

type serviceManagerContext struct {
	primaryModel     llms.Model
	elevenLabsClient *elevenlabs.Client
	// whisperCPPModel  whisper.Model
}

func NewServiceManagerContext(primaryModel llms.Model, elevenLabsClient *elevenlabs.Client, whisperCPPModelPath string) *serviceManagerContext {
	// whisperCPPModel, err := whisper.New(whisperCPPModelPath)
	// if err != nil {
	// 	panic(err)
	// }
	// var whisperCPPModel whisper.Model

	return &serviceManagerContext{
		primaryModel,
		elevenLabsClient,
		// whisperCPPModel,
	}
}

func (c *serviceManagerContext) PrimaryModel() llms.Model {
	return c.primaryModel
}

func (c *serviceManagerContext) ElevenLabsClient() *elevenlabs.Client {
	return c.elevenLabsClient
}

// func (c *serviceManagerContext) WhisperCPPModel() whisper.Model {
// 	return c.whisperCPPModel
// }
