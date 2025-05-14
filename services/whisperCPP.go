package services

import (
	"strings"

	"github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
)

type WhisperCPP interface {
	SpeechToTextConverter
}

type whisperCPP struct {
	ServiceContext
	modelPath string
}

func NewWhisperCPPService(ctx ServiceContext, modelPath string) WhisperCPP {
	return &whisperCPP{
		ServiceContext: ctx,
		modelPath:      modelPath,
	}
}

func (w *whisperCPP) SpeechToText(audio []byte) (string, error) {
	m, err := whisper.New(w.modelPath)
	if err != nil {
		return "", err
	}
	defer m.Close()

	voiceCtx, err := m.NewContext()
	audioFloat32 := make([]float32, len(audio))
	for i := range audio {
		audioFloat32[i] = float32(audio[i])
	}

	if err := voiceCtx.Process(audioFloat32, nil, nil, nil); err != nil {
		return "", err
	}

	var str strings.Builder

	for {
		segment, err := voiceCtx.NextSegment()
		if err != nil {
			break
		}
		if _, err := str.WriteString(segment.Text); err != nil {
			return "", err
		}
	}

	return str.String(), nil
}
