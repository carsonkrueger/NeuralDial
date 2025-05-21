package services

// "github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"

type WhisperCPP interface {
	SpeechToTextConverter
}

type whisperCPP struct {
	ServiceContext
	// model whisper.Model
}

func NewWhisperCPPService(
	ctx ServiceContext,
	// model whisper.Model
) WhisperCPP {
	return &whisperCPP{
		ServiceContext: ctx,
		// model:          model,
	}
}

func (w *whisperCPP) SpeechToText(audio []byte) (string, error) {
	// voiceCtx, err := w.model.NewContext()
	// if err != nil {
	// 	return "", err
	// }

	// audioFloat32 := make([]float32, len(audio))
	// for i := range audio {
	// 	audioFloat32[i] = float32(audio[i])
	// }

	// if err := voiceCtx.Process(audioFloat32, nil, nil, nil); err != nil {
	// 	return "", err
	// }

	// var str strings.Builder

	// for {
	// 	segment, err := voiceCtx.NextSegment()
	// 	if err != nil {
	// 		break
	// 	}
	// 	if _, err := str.WriteString(segment.Text); err != nil {
	// 		return "", err
	// 	}
	// }

	// return str.String(), nil
	return "", nil
}
