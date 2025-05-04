package services

import (
	"context"
)

type PhoneService interface {
	StartCall(ctx context.Context) error
	EndCall(ctx context.Context) error
}
