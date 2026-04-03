package service

import (
	"context"
	"testing"
	"time"

	"booking-service/internal/domain"
	"booking-service/internal/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSlotService_GetAvailableSlots(t *testing.T) {
	roomID := uuid.New()

	now := time.Now().UTC()
	futureDate := now.AddDate(0, 0, 1)

	for futureDate.Weekday() != time.Monday {
		futureDate = futureDate.AddDate(0, 0, 1)
	}

	normalizedDate := time.Date(futureDate.Year(), futureDate.Month(), futureDate.Day(), 0, 0, 0, 0, time.UTC)

	weekday := int(normalizedDate.Weekday())
	mappedDay := weekday
	if mappedDay == 0 {
		mappedDay = 7
	}

	notMondayDate := normalizedDate.AddDate(0, 0, 1)
	for notMondayDate.Weekday() == time.Monday {
		notMondayDate = notMondayDate.AddDate(0, 0, 1)
	}
	notMondayNormalized := time.Date(notMondayDate.Year(), notMondayDate.Month(), notMondayDate.Day(), 0, 0, 0, 0, time.UTC)
	notMondayMappedDay := int(notMondayNormalized.Weekday())
	if notMondayMappedDay == 0 {
		notMondayMappedDay = 7
	}

	availableSlots := []domain.Slot{
		{
			ID:     uuid.New(),
			RoomID: roomID,
			Start:  normalizedDate.Add(9 * time.Hour),
			End:    normalizedDate.Add(9*time.Hour + 30*time.Minute),
		},
		{
			ID:     uuid.New(),
			RoomID: roomID,
			Start:  normalizedDate.Add(10 * time.Hour),
			End:    normalizedDate.Add(10*time.Hour + 30*time.Minute),
		},
	}

	tests := []struct {
		name        string
		roomID      uuid.UUID
		date        time.Time
		setupMock   func(*mocks.MockRoomRepository, *mocks.MockScheduleRepository, *mocks.MockSlotRepository)
		expected    []domain.Slot
		expectedErr error
	}{
		{
			name:   "successful get available slots",
			roomID: roomID,
			date:   normalizedDate,
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository, slotRepo *mocks.MockSlotRepository) {
				roomRepo.On("Exists", mock.Anything, roomID).Return(true, nil)
				schedule := &domain.Schedule{
					ID:         uuid.New(),
					RoomID:     roomID,
					DaysOfWeek: []int{mappedDay},
					StartTime:  "09:00",
					EndTime:    "18:00",
				}
				scheduleRepo.On("GetByRoomID", mock.Anything, roomID).Return(schedule, nil)
				slotRepo.On("GetByRoomAndDate", mock.Anything, roomID, normalizedDate).Return(availableSlots, nil)
			},
			expected:    availableSlots,
			expectedErr: nil,
		},
		{
			name:   "room not found",
			roomID: roomID,
			date:   normalizedDate,
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository, slotRepo *mocks.MockSlotRepository) {
				roomRepo.On("Exists", mock.Anything, roomID).Return(false, nil)
			},
			expected:    nil,
			expectedErr: domain.ErrRoomNotFound,
		},
		{
			name:   "no schedule for room - returns empty slots",
			roomID: roomID,
			date:   normalizedDate,
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository, slotRepo *mocks.MockSlotRepository) {
				roomRepo.On("Exists", mock.Anything, roomID).Return(true, nil)
				scheduleRepo.On("GetByRoomID", mock.Anything, roomID).Return(nil, nil)
			},
			expected:    []domain.Slot{},
			expectedErr: nil,
		},
		{
			name:   "day not in schedule - returns empty slots",
			roomID: roomID,
			date:   notMondayNormalized,
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository, slotRepo *mocks.MockSlotRepository) {
				roomRepo.On("Exists", mock.Anything, roomID).Return(true, nil)
				schedule := &domain.Schedule{
					ID:         uuid.New(),
					RoomID:     roomID,
					DaysOfWeek: []int{mappedDay},
					StartTime:  "09:00",
					EndTime:    "18:00",
				}
				scheduleRepo.On("GetByRoomID", mock.Anything, roomID).Return(schedule, nil)

			},
			expected:    []domain.Slot{},
			expectedErr: nil,
		},
		{
			name:   "repository error on room exists check",
			roomID: roomID,
			date:   normalizedDate,
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository, slotRepo *mocks.MockSlotRepository) {
				roomRepo.On("Exists", mock.Anything, roomID).Return(false, assert.AnError)
			},
			expected:    nil,
			expectedErr: assert.AnError,
		},
		{
			name:   "no slots available - queue background generation",
			roomID: roomID,
			date:   normalizedDate,
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository, slotRepo *mocks.MockSlotRepository) {
				roomRepo.On("Exists", mock.Anything, roomID).Return(true, nil)
				schedule := &domain.Schedule{
					ID:         uuid.New(),
					RoomID:     roomID,
					DaysOfWeek: []int{mappedDay},
					StartTime:  "09:00",
					EndTime:    "18:00",
				}
				scheduleRepo.On("GetByRoomID", mock.Anything, roomID).Return(schedule, nil)
				slotRepo.On("GetByRoomAndDate", mock.Anything, roomID, normalizedDate).Return([]domain.Slot{}, nil)

				slotRepo.On("CreateBatch", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			expected:    []domain.Slot{},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRoomRepo := new(mocks.MockRoomRepository)
			mockScheduleRepo := new(mocks.MockScheduleRepository)
			mockSlotRepo := new(mocks.MockSlotRepository)

			tt.setupMock(mockRoomRepo, mockScheduleRepo, mockSlotRepo)

			slotService := NewSlotService(mockSlotRepo, mockRoomRepo, mockScheduleRepo)

			time.Sleep(10 * time.Millisecond)

			slots, err := slotService.GetAvailableSlots(context.Background(), tt.roomID, tt.date)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				if tt.expectedErr != assert.AnError {
					assert.ErrorContains(t, err, tt.expectedErr.Error())
				}
				assert.Nil(t, slots)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, slots)
			}

			mockRoomRepo.AssertExpectations(t)
			mockScheduleRepo.AssertExpectations(t)
			mockSlotRepo.AssertExpectations(t)
		})
	}
}

func TestSlotService_GetAvailableSlots_WithPastDate(t *testing.T) {
	roomID := uuid.New()
	pastDate := time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour)

	mockRoomRepo := new(mocks.MockRoomRepository)
	mockScheduleRepo := new(mocks.MockScheduleRepository)
	mockSlotRepo := new(mocks.MockSlotRepository)

	mockRoomRepo.On("Exists", mock.Anything, roomID).Return(true, nil)
	schedule := &domain.Schedule{
		ID:         uuid.New(),
		RoomID:     roomID,
		DaysOfWeek: []int{1, 2, 3, 4, 5},
		StartTime:  "09:00",
		EndTime:    "18:00",
	}
	mockScheduleRepo.On("GetByRoomID", mock.Anything, roomID).Return(schedule, nil)
	mockSlotRepo.On("GetByRoomAndDate", mock.Anything, roomID, pastDate).Return([]domain.Slot{}, nil)

	slotService := NewSlotService(mockSlotRepo, mockRoomRepo, mockScheduleRepo)
	time.Sleep(10 * time.Millisecond)

	slots, err := slotService.GetAvailableSlots(context.Background(), roomID, pastDate)

	assert.NoError(t, err)
	assert.Empty(t, slots)

	mockRoomRepo.AssertExpectations(t)
	mockScheduleRepo.AssertExpectations(t)
	mockSlotRepo.AssertExpectations(t)
}

func TestSlotService_GetByID(t *testing.T) {
	slotID := uuid.New()
	expectedSlot := &domain.Slot{
		ID:     slotID,
		RoomID: uuid.New(),
		Start:  time.Now().Add(24 * time.Hour),
		End:    time.Now().Add(24*time.Hour + 30*time.Minute),
	}

	tests := []struct {
		name        string
		slotID      uuid.UUID
		setupMock   func(*mocks.MockSlotRepository)
		expected    *domain.Slot
		expectedErr error
	}{
		{
			name:   "successful get by id",
			slotID: slotID,
			setupMock: func(slotRepo *mocks.MockSlotRepository) {
				slotRepo.On("GetByID", mock.Anything, slotID).Return(expectedSlot, nil)
			},
			expected:    expectedSlot,
			expectedErr: nil,
		},
		{
			name:   "slot not found",
			slotID: slotID,
			setupMock: func(slotRepo *mocks.MockSlotRepository) {
				slotRepo.On("GetByID", mock.Anything, slotID).Return(nil, nil)
			},
			expected:    nil,
			expectedErr: nil,
		},
		{
			name:   "repository error",
			slotID: slotID,
			setupMock: func(slotRepo *mocks.MockSlotRepository) {
				slotRepo.On("GetByID", mock.Anything, slotID).Return(nil, assert.AnError)
			},
			expected:    nil,
			expectedErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSlotRepo := new(mocks.MockSlotRepository)
			tt.setupMock(mockSlotRepo)

			slotService := NewSlotService(mockSlotRepo, nil, nil)

			slot, err := slotService.GetByID(context.Background(), tt.slotID)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr, err)
				assert.Nil(t, slot)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, slot)
			}

			mockSlotRepo.AssertExpectations(t)
		})
	}
}

func TestSlotService_IsSlotInPast(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	pastSlotID := uuid.New()
	futureSlotID := uuid.New()
	nonExistentSlotID := uuid.New()

	tests := []struct {
		name        string
		slotID      uuid.UUID
		setupMock   func(*mocks.MockSlotRepository)
		expected    bool
		expectedErr error
	}{
		{
			name:   "slot is in past",
			slotID: pastSlotID,
			setupMock: func(slotRepo *mocks.MockSlotRepository) {
				slotRepo.On("GetByID", mock.Anything, pastSlotID).Return(&domain.Slot{
					ID:    pastSlotID,
					Start: now.Add(-24 * time.Hour),
					End:   now.Add(-24*time.Hour + 30*time.Minute),
				}, nil)
			},
			expected:    true,
			expectedErr: nil,
		},
		{
			name:   "slot is in future",
			slotID: futureSlotID,
			setupMock: func(slotRepo *mocks.MockSlotRepository) {
				slotRepo.On("GetByID", mock.Anything, futureSlotID).Return(&domain.Slot{
					ID:    futureSlotID,
					Start: now.Add(24 * time.Hour),
					End:   now.Add(24*time.Hour + 30*time.Minute),
				}, nil)
			},
			expected:    false,
			expectedErr: nil,
		},
		{
			name:   "slot is exactly now",
			slotID: uuid.New(),
			setupMock: func(slotRepo *mocks.MockSlotRepository) {
				slotRepo.On("GetByID", mock.Anything, mock.Anything).Return(&domain.Slot{
					ID:    uuid.New(),
					Start: now,
					End:   now.Add(30 * time.Minute),
				}, nil)
			},
			expected:    false,
			expectedErr: nil,
		},
		{
			name:   "slot not found - returns false",
			slotID: nonExistentSlotID,
			setupMock: func(slotRepo *mocks.MockSlotRepository) {
				slotRepo.On("GetByID", mock.Anything, nonExistentSlotID).Return(nil, nil)
			},
			expected:    false,
			expectedErr: nil,
		},
		{
			name:   "repository error",
			slotID: uuid.New(),
			setupMock: func(slotRepo *mocks.MockSlotRepository) {
				slotRepo.On("GetByID", mock.Anything, mock.Anything).Return(nil, assert.AnError)
			},
			expected:    false,
			expectedErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSlotRepo := new(mocks.MockSlotRepository)
			tt.setupMock(mockSlotRepo)

			slotService := NewSlotService(mockSlotRepo, nil, nil)

			isPast, err := slotService.IsSlotInPast(context.Background(), tt.slotID)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, isPast)
			}

			mockSlotRepo.AssertExpectations(t)
		})
	}
}

func TestSlotService_PreGenerateSlotsForRange(t *testing.T) {
	roomID := uuid.New()

	tests := []struct {
		name       string
		daysAhead  int
		setupMock  func(*mocks.MockScheduleRepository, *mocks.MockSlotRepository)
		expectCall bool
	}{
		{
			name:      "pre-generate slots for 7 days",
			daysAhead: 7,
			setupMock: func(scheduleRepo *mocks.MockScheduleRepository, slotRepo *mocks.MockSlotRepository) {
				schedule := &domain.Schedule{
					ID:         uuid.New(),
					RoomID:     roomID,
					DaysOfWeek: []int{1, 2, 3, 4, 5},
					StartTime:  "09:00",
					EndTime:    "18:00",
				}
				scheduleRepo.On("GetByRoomID", mock.Anything, roomID).Return(schedule, nil)

				slotRepo.On("CreateBatch", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			expectCall: true,
		},
		{
			name:      "no schedule for room",
			daysAhead: 7,
			setupMock: func(scheduleRepo *mocks.MockScheduleRepository, slotRepo *mocks.MockSlotRepository) {
				scheduleRepo.On("GetByRoomID", mock.Anything, roomID).Return(nil, nil)
			},
			expectCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRoomRepo := new(mocks.MockRoomRepository)
			mockScheduleRepo := new(mocks.MockScheduleRepository)
			mockSlotRepo := new(mocks.MockSlotRepository)

			tt.setupMock(mockScheduleRepo, mockSlotRepo)

			slotService := NewSlotService(mockSlotRepo, mockRoomRepo, mockScheduleRepo)
			time.Sleep(10 * time.Millisecond)

			err := slotService.PreGenerateSlotsForRange(context.Background(), roomID, tt.daysAhead)
			assert.NoError(t, err)

			time.Sleep(100 * time.Millisecond)

			mockScheduleRepo.AssertExpectations(t)
			if tt.expectCall {
				mockSlotRepo.AssertExpectations(t)
			}
		})
	}
}

func TestSlotService_ConcurrentAccess(t *testing.T) {
	roomID := uuid.New()
	futureDate := time.Now().UTC().AddDate(0, 0, 1).Truncate(24 * time.Hour)

	mockRoomRepo := new(mocks.MockRoomRepository)
	mockScheduleRepo := new(mocks.MockScheduleRepository)
	mockSlotRepo := new(mocks.MockSlotRepository)

	mockRoomRepo.On("Exists", mock.Anything, roomID).Return(true, nil).Maybe()
	schedule := &domain.Schedule{
		ID:         uuid.New(),
		RoomID:     roomID,
		DaysOfWeek: []int{int(futureDate.Weekday()) + 1},
		StartTime:  "09:00",
		EndTime:    "18:00",
	}
	mockScheduleRepo.On("GetByRoomID", mock.Anything, roomID).Return(schedule, nil).Maybe()
	mockSlotRepo.On("GetByRoomAndDate", mock.Anything, roomID, futureDate).Return([]domain.Slot{}, nil).Maybe()
	mockSlotRepo.On("CreateBatch", mock.Anything, mock.Anything).Return(nil).Maybe()

	slotService := NewSlotService(mockSlotRepo, mockRoomRepo, mockScheduleRepo)
	time.Sleep(10 * time.Millisecond)

	concurrency := 10
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			slots, err := slotService.GetAvailableSlots(context.Background(), roomID, futureDate)
			assert.NoError(t, err)

			assert.NotNil(t, slots)
			done <- true
		}()
	}

	for i := 0; i < concurrency; i++ {
		<-done
	}
}
