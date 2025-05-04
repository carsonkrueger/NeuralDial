package services

type VoiceService interface {
	TextToSpeech(msg string) ([]byte, error)
}
