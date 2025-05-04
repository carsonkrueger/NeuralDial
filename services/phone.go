package services

import (
	"context"
)

type PhoneService interface {
	StartCall(ctx context.Context) error
}

type phoneService struct {
	ServiceContext
}

func NewPhoneService(ctx ServiceContext) *phoneService {
	return &phoneService{
		ctx,
	}
}

func (p *phoneService) StartCall(ctx context.Context) error {
	return nil
}
