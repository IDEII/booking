package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"booking-service/internal/domain"
)

type ConferenceService interface {
	CreateConference(ctx context.Context, slot *domain.Slot) (string, error)
}

type MockConferenceService struct {
	shouldFail bool
	failRate   float64
}

func NewMockConferenceService(shouldFail bool) *MockConferenceService {
	return &MockConferenceService{
		shouldFail: shouldFail,
		failRate:   1.0,
	}
}

func NewMockConferenceServiceWithFailRate(failRate float64) *MockConferenceService {
	return &MockConferenceService{
		shouldFail: true,
		failRate:   failRate,
	}
}

func (s *MockConferenceService) CreateConference(ctx context.Context, slot *domain.Slot) (string, error) {
	select {
	case <-time.After(100 * time.Millisecond):
	case <-ctx.Done():
		return "", ctx.Err()
	}

	if s.shouldFail {
		if s.failRate >= 1.0 {
			return "", fmt.Errorf("conference service unavailable")
		}
		if rand.Float64() < s.failRate {
			return "", fmt.Errorf("conference service unavailable")
		}
	}

	link := fmt.Sprintf("https://meet.example.com/%s-%d", slot.ID.String(), time.Now().Unix())
	return link, nil
}
