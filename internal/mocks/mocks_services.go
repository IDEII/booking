package mocks

import (
	"context"

	"booking-service/internal/domain"

	"github.com/stretchr/testify/mock"
)

type MockConferenceService struct {
	mock.Mock
}

func (m *MockConferenceService) CreateConference(ctx context.Context, slot *domain.Slot) (string, error) {
	args := m.Called(ctx, slot)
	return args.String(0), args.Error(1)
}
