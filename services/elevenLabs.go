package services

import (
	"errors"

	"github.com/haguro/elevenlabs-go"
)

type ElevenLabsService interface {
}

type elevenLabsService struct {
	ServiceContext
	client               *elevenlabs.Client
	voices               []elevenlabs.Voice
	models               []elevenlabs.Model
	defaultVoiceSettings *elevenlabs.VoiceSettings
}

func NewElevenLabsService(ctx ServiceContext, client *elevenlabs.Client) *elevenLabsService {
	return &elevenLabsService{
		ServiceContext:       ctx,
		client:               client,
		voices:               []elevenlabs.Voice{},
		defaultVoiceSettings: nil,
	}
}

func (el *elevenLabsService) GetVoice(name string) (elevenlabs.Voice, error) {
	if len(el.voices) == 0 {
		voices, err := el.client.GetVoices()
		if err != nil {
			return elevenlabs.Voice{}, err
		}
		el.voices = voices
	}
	for i := range el.voices {
		v := el.voices[i]
		if v.Name == name {
			return v, nil
		}
	}
	return elevenlabs.Voice{}, errors.New("Voice Not Found")
}

func (el *elevenLabsService) GetModel(name string) (elevenlabs.Model, error) {
	if len(el.models) == 0 {
		models, err := el.client.GetModels()
		if err != nil {
			return elevenlabs.Model{}, err
		}
		el.models = models
	}
	for i := range el.models {
		m := el.models[i]
		if m.Name == name {
			return m, nil
		}
	}
	return elevenlabs.Model{}, errors.New("Model Not Found")
}

func (el *elevenLabsService) GetDefaultVoiceSettings() (*elevenlabs.VoiceSettings, error) {
	if el.defaultVoiceSettings == nil {
		settings, err := el.client.GetDefaultVoiceSettings()
		if err != nil {
			return nil, err
		}
		el.defaultVoiceSettings = &settings
	}
	return el.defaultVoiceSettings, nil
}

func (el *elevenLabsService) TextToSpeech(msg string) ([]byte, error) {
	var bytes []byte
	voice, err := el.GetVoice("Bill")
	if err != nil {
		return bytes, err
	}

	model, err := el.GetModel("Eleven Flash v2.5")
	if err != nil {
		return bytes, err
	}

	settings, err := el.GetDefaultVoiceSettings()
	if err != nil {
		return bytes, err
	}

	req := elevenlabs.TextToSpeechRequest{
		Text:          msg,
		ModelID:       model.ModelId,
		VoiceSettings: settings,
	}
	bytes, err = el.client.TextToSpeech(voice.VoiceId, req)
	if err != nil {
		return bytes, err
	}
	return bytes, nil
}
