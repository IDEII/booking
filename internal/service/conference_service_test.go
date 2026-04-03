package service

import (
	"context"
	"testing"
	"time"

	"booking-service/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMockConferenceService_CreateConference(t *testing.T) {
	slot := &domain.Slot{
		ID:     uuid.New(),
		RoomID: uuid.New(),
		Start:  time.Now().Add(24 * time.Hour),
		End:    time.Now().Add(24*time.Hour + 30*time.Minute),
	}

	tests := []struct {
		name        string
		shouldFail  bool
		expectedErr bool
	}{
		{
			name:        "successful conference creation",
			shouldFail:  false,
			expectedErr: false,
		},
		{
			name:        "conference service may fail",
			shouldFail:  true,
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confService := NewMockConferenceService(tt.shouldFail)

			link, err := confService.CreateConference(context.Background(), slot)

			if tt.expectedErr {
				assert.Error(t, err)
				assert.Empty(t, link)
				assert.Contains(t, err.Error(), "conference service unavailable")
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, link)
				assert.Contains(t, link, "https://meet.example.com/")
				assert.Contains(t, link, slot.ID.String())
			}
		})
	}
}

func TestMockConferenceService_CreateConference_MultipleCalls(t *testing.T) {
	slot := &domain.Slot{
		ID:     uuid.New(),
		RoomID: uuid.New(),
		Start:  time.Now().Add(24 * time.Hour),
		End:    time.Now().Add(24*time.Hour + 30*time.Minute),
	}

	confService := NewMockConferenceService(false)

	for i := 0; i < 5; i++ {
		link, err := confService.CreateConference(context.Background(), slot)
		assert.NoError(t, err)
		assert.NotEmpty(t, link)
		assert.Contains(t, link, "https://meet.example.com/")
	}
}

func TestMockConferenceService_CreateConference_WithError(t *testing.T) {
	slot := &domain.Slot{
		ID:     uuid.New(),
		RoomID: uuid.New(),
		Start:  time.Now().Add(24 * time.Hour),
		End:    time.Now().Add(24*time.Hour + 30*time.Minute),
	}

	confService := NewMockConferenceService(true)

	for i := 0; i < 3; i++ {
		link, err := confService.CreateConference(context.Background(), slot)
		assert.Error(t, err)
		assert.Empty(t, link)
		assert.Contains(t, err.Error(), "conference service unavailable")
	}
}

func TestMockConferenceService_CreateConference_ContextCancellation(t *testing.T) {
	slot := &domain.Slot{
		ID:     uuid.New(),
		RoomID: uuid.New(),
		Start:  time.Now().Add(24 * time.Hour),
		End:    time.Now().Add(24*time.Hour + 30*time.Minute),
	}

	confService := NewMockConferenceService(false)

	t.Run("context not cancelled - should succeed", func(t *testing.T) {
		ctx := context.Background()
		link, err := confService.CreateConference(ctx, slot)

		assert.NoError(t, err)
		assert.NotEmpty(t, link)
		assert.Contains(t, link, "https://meet.example.com/")
		assert.Contains(t, link, slot.ID.String())
	})

	t.Run("context cancelled before call - should return context.Canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		link, err := confService.CreateConference(ctx, slot)

		assert.Error(t, err)
		assert.Empty(t, link)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("context cancelled during call - should return context.Canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		link, err := confService.CreateConference(ctx, slot)

		assert.Error(t, err)
		assert.Empty(t, link)
		assert.True(t, err == context.Canceled || err == context.DeadlineExceeded)
	})
}

func TestMockConferenceService_CreateConferenceWithRetry(t *testing.T) {
	slot := &domain.Slot{
		ID:     uuid.New(),
		RoomID: uuid.New(),
		Start:  time.Now().Add(24 * time.Hour),
		End:    time.Now().Add(24*time.Hour + 30*time.Minute),
	}

	confService := NewMockConferenceService(false)

	for i := 0; i < 5; i++ {
		link, err := confService.CreateConference(context.Background(), slot)
		assert.NoError(t, err)
		assert.NotEmpty(t, link)
		assert.Contains(t, link, "https://meet.example.com/")
	}
}
