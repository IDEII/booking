package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"booking-service/internal/domain"
	"booking-service/internal/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestScheduleService_Create(t *testing.T) {
	roomID := uuid.New()
	validDays := []int{1, 2, 3, 4, 5}

	tests := []struct {
		name        string
		roomID      uuid.UUID
		daysOfWeek  []int
		startTime   string
		endTime     string
		setupMock   func(*mocks.MockRoomRepository, *mocks.MockScheduleRepository)
		expectedErr bool
		errMessage  string
	}{
		{
			name:       "successful schedule creation",
			roomID:     roomID,
			daysOfWeek: validDays,
			startTime:  "09:00",
			endTime:    "18:00",
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository) {
				roomRepo.On("Exists", mock.Anything, roomID).Return(true, nil)
				scheduleRepo.On("Exists", mock.Anything, roomID).Return(false, nil)
				scheduleRepo.On("Create", mock.Anything, mock.MatchedBy(func(schedule *domain.Schedule) bool {
					return schedule.RoomID == roomID &&
						len(schedule.DaysOfWeek) == len(validDays) &&
						schedule.StartTime == "09:00" &&
						schedule.EndTime == "18:00"
				})).Return(nil)
			},
			expectedErr: false,
		},
		{
			name:       "room not found",
			roomID:     roomID,
			daysOfWeek: validDays,
			startTime:  "09:00",
			endTime:    "18:00",
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository) {
				roomRepo.On("Exists", mock.Anything, roomID).Return(false, nil)
			},
			expectedErr: true,
			errMessage:  "room not found",
		},
		{
			name:       "schedule already exists",
			roomID:     roomID,
			daysOfWeek: validDays,
			startTime:  "09:00",
			endTime:    "18:00",
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository) {
				roomRepo.On("Exists", mock.Anything, roomID).Return(true, nil)
				scheduleRepo.On("Exists", mock.Anything, roomID).Return(true, nil)
			},
			expectedErr: true,
			errMessage:  "schedule already exists",
		},
		{
			name:       "invalid day of week - too low",
			roomID:     roomID,
			daysOfWeek: []int{0},
			startTime:  "09:00",
			endTime:    "18:00",
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository) {
			},
			expectedErr: true,
			errMessage:  "invalid day of week: 0",
		},
		{
			name:       "invalid day of week - too high",
			roomID:     roomID,
			daysOfWeek: []int{8},
			startTime:  "09:00",
			endTime:    "18:00",
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository) {
			},
			expectedErr: true,
			errMessage:  "invalid day of week: 8",
		},
		{
			name:       "multiple invalid days",
			roomID:     roomID,
			daysOfWeek: []int{1, 2, 8, 9},
			startTime:  "09:00",
			endTime:    "18:00",
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository) {
			},
			expectedErr: true,
			errMessage:  "invalid day of week: 8",
		},
		{
			name:       "start time after end time",
			roomID:     roomID,
			daysOfWeek: validDays,
			startTime:  "18:00",
			endTime:    "09:00",
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository) {
			},
			expectedErr: true,
			errMessage:  "start time must be before end time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRoomRepo := new(mocks.MockRoomRepository)
			mockScheduleRepo := new(mocks.MockScheduleRepository)
			mockSlotRepo := new(mocks.MockSlotRepository)

			tt.setupMock(mockRoomRepo, mockScheduleRepo)

			scheduleService := NewScheduleService(mockScheduleRepo, mockRoomRepo, mockSlotRepo, 30*time.Minute, nil)

			schedule, err := scheduleService.Create(context.Background(), tt.roomID, tt.daysOfWeek, tt.startTime, tt.endTime)

			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, schedule)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
				mockScheduleRepo.AssertNotCalled(t, "Create")
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, schedule)
				assert.Equal(t, tt.roomID, schedule.RoomID)
				assert.Equal(t, tt.daysOfWeek, schedule.DaysOfWeek)
				assert.Equal(t, tt.startTime, schedule.StartTime)
				assert.Equal(t, tt.endTime, schedule.EndTime)
			}

			if tt.name == "successful schedule creation" || tt.name == "room not found" || tt.name == "schedule already exists" {
				mockRoomRepo.AssertExpectations(t)
				mockScheduleRepo.AssertExpectations(t)
			} else {
				mockRoomRepo.AssertNotCalled(t, "Exists")
				mockScheduleRepo.AssertNotCalled(t, "Exists")
				mockScheduleRepo.AssertNotCalled(t, "Create")
			}
		})
	}
}

func TestScheduleService_Create_WithInvalidTimeFormat(t *testing.T) {
	roomID := uuid.New()
	validDays := []int{1, 2, 3, 4, 5}

	mockRoomRepo := new(mocks.MockRoomRepository)
	mockScheduleRepo := new(mocks.MockScheduleRepository)
	mockSlotRepo := new(mocks.MockSlotRepository)

	scheduleService := NewScheduleService(mockScheduleRepo, mockRoomRepo, mockSlotRepo, 30*time.Minute, nil)

	schedule, err := scheduleService.Create(context.Background(), roomID, validDays, "25:00", "18:00")
	assert.Error(t, err)
	assert.Nil(t, schedule)
	assert.Contains(t, err.Error(), "invalid start time format")

	schedule, err = scheduleService.Create(context.Background(), roomID, validDays, "09:00", "25:00")
	assert.Error(t, err)
	assert.Nil(t, schedule)
	assert.Contains(t, err.Error(), "invalid end time format")

	mockRoomRepo.AssertNotCalled(t, "Exists")
	mockScheduleRepo.AssertNotCalled(t, "Exists")
	mockScheduleRepo.AssertNotCalled(t, "Create")
}

func TestScheduleService_Create_WithRepositoryError(t *testing.T) {
	roomID := uuid.New()
	validDays := []int{1, 2, 3, 4, 5}

	mockRoomRepo := new(mocks.MockRoomRepository)
	mockScheduleRepo := new(mocks.MockScheduleRepository)
	mockSlotRepo := new(mocks.MockSlotRepository)

	mockRoomRepo.On("Exists", mock.Anything, roomID).Return(true, nil)
	mockScheduleRepo.On("Exists", mock.Anything, roomID).Return(false, nil)
	mockScheduleRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Schedule")).Return(errors.New("database error"))

	scheduleService := NewScheduleService(mockScheduleRepo, mockRoomRepo, mockSlotRepo, 30*time.Minute, nil)

	schedule, err := scheduleService.Create(context.Background(), roomID, validDays, "09:00", "18:00")
	assert.Error(t, err)
	assert.Nil(t, schedule)
	assert.Contains(t, err.Error(), "database error")
}

func TestScheduleService_GenerateSlots(t *testing.T) {
	roomID := uuid.New()
	now := time.Now().UTC()
	startDate := now.Truncate(24 * time.Hour)
	endDate := startDate.AddDate(0, 0, 7)

	tests := []struct {
		name        string
		roomID      uuid.UUID
		startDate   time.Time
		endDate     time.Time
		setupMock   func(*mocks.MockScheduleRepository, *mocks.MockSlotRepository)
		expectedErr bool
	}{
		{
			name:      "successful slot generation for weekdays",
			roomID:    roomID,
			startDate: startDate,
			endDate:   endDate,
			setupMock: func(scheduleRepo *mocks.MockScheduleRepository, slotRepo *mocks.MockSlotRepository) {
				schedule := &domain.Schedule{
					ID:         uuid.New(),
					RoomID:     roomID,
					DaysOfWeek: []int{1, 2, 3, 4, 5},
					StartTime:  "09:00",
					EndTime:    "17:00",
				}
				scheduleRepo.On("GetByRoomID", mock.Anything, roomID).Return(schedule, nil)
				slotRepo.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]domain.Slot")).Return(nil)
			},
			expectedErr: false,
		},
		{
			name:      "no schedule for room",
			roomID:    roomID,
			startDate: startDate,
			endDate:   endDate,
			setupMock: func(scheduleRepo *mocks.MockScheduleRepository, slotRepo *mocks.MockSlotRepository) {
				scheduleRepo.On("GetByRoomID", mock.Anything, roomID).Return(nil, nil)
			},
			expectedErr: false,
		},
		{
			name:      "schedule with weekend days only",
			roomID:    roomID,
			startDate: startDate,
			endDate:   endDate,
			setupMock: func(scheduleRepo *mocks.MockScheduleRepository, slotRepo *mocks.MockSlotRepository) {
				schedule := &domain.Schedule{
					ID:         uuid.New(),
					RoomID:     roomID,
					DaysOfWeek: []int{6, 7},
					StartTime:  "10:00",
					EndTime:    "15:00",
				}
				scheduleRepo.On("GetByRoomID", mock.Anything, roomID).Return(schedule, nil)
				slotRepo.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]domain.Slot")).Return(nil)
			},
			expectedErr: false,
		},
		{
			name:      "schedule with full week",
			roomID:    roomID,
			startDate: startDate,
			endDate:   endDate,
			setupMock: func(scheduleRepo *mocks.MockScheduleRepository, slotRepo *mocks.MockSlotRepository) {
				schedule := &domain.Schedule{
					ID:         uuid.New(),
					RoomID:     roomID,
					DaysOfWeek: []int{1, 2, 3, 4, 5, 6, 7},
					StartTime:  "00:00",
					EndTime:    "23:30",
				}
				scheduleRepo.On("GetByRoomID", mock.Anything, roomID).Return(schedule, nil)
				slotRepo.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]domain.Slot")).Return(nil)
			},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockScheduleRepo := new(mocks.MockScheduleRepository)
			mockRoomRepo := new(mocks.MockRoomRepository)
			mockSlotRepo := new(mocks.MockSlotRepository)

			tt.setupMock(mockScheduleRepo, mockSlotRepo)

			scheduleService := NewScheduleService(mockScheduleRepo, mockRoomRepo, mockSlotRepo, 30*time.Minute, nil)

			err := scheduleService.GenerateSlots(context.Background(), tt.roomID, tt.startDate, tt.endDate)

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockScheduleRepo.AssertExpectations(t)
			mockSlotRepo.AssertExpectations(t)
		})
	}
}

func TestScheduleService_GenerateSlotsForDay(t *testing.T) {
	roomID := uuid.New()
	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		date          time.Time
		startTimeStr  string
		endTimeStr    string
		expectedSlots int
		expectedErr   bool
	}{
		{
			name:          "generate slots for full work day",
			date:          date,
			startTimeStr:  "09:00",
			endTimeStr:    "17:00",
			expectedSlots: 16,
			expectedErr:   false,
		},
		{
			name:          "generate slots for half day",
			date:          date,
			startTimeStr:  "09:00",
			endTimeStr:    "12:00",
			expectedSlots: 6,
			expectedErr:   false,
		},
		{
			name:          "generate slots for short period",
			date:          date,
			startTimeStr:  "10:00",
			endTimeStr:    "11:00",
			expectedSlots: 2,
			expectedErr:   false,
		},
		{
			name:          "invalid start time format",
			date:          date,
			startTimeStr:  "25:00",
			endTimeStr:    "17:00",
			expectedSlots: 0,
			expectedErr:   true,
		},
		{
			name:          "invalid end time format",
			date:          date,
			startTimeStr:  "09:00",
			endTimeStr:    "25:00",
			expectedSlots: 0,
			expectedErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockScheduleRepo := new(mocks.MockScheduleRepository)
			mockRoomRepo := new(mocks.MockRoomRepository)
			mockSlotRepo := new(mocks.MockSlotRepository)

			scheduleService := NewScheduleService(mockScheduleRepo, mockRoomRepo, mockSlotRepo, 30*time.Minute, nil)

			slots, err := scheduleService.generateSlotsForDay(tt.date, tt.startTimeStr, tt.endTimeStr, roomID)

			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, slots)
			} else {
				assert.NoError(t, err)
				assert.Len(t, slots, tt.expectedSlots)
				if len(slots) > 0 {
					assert.Equal(t, roomID, slots[0].RoomID)
					assert.Equal(t, 30*time.Minute, slots[0].End.Sub(slots[0].Start))
				}
			}
		})
	}
}

func TestScheduleService_Create_WithPreGeneration(t *testing.T) {
	roomID := uuid.New()
	validDays := []int{1, 2, 3, 4, 5}

	t.Run("successful schedule creation with pre-generation", func(t *testing.T) {
		mockRoomRepo := new(mocks.MockRoomRepository)
		mockScheduleRepo := new(mocks.MockScheduleRepository)
		mockSlotRepo := new(mocks.MockSlotRepository)

		slotService := NewSlotService(mockSlotRepo, mockRoomRepo, mockScheduleRepo)

		mockRoomRepo.On("Exists", mock.Anything, roomID).Return(true, nil)
		mockScheduleRepo.On("Exists", mock.Anything, roomID).Return(false, nil)
		mockScheduleRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Schedule")).Return(nil)

		mockScheduleRepo.On("GetByRoomID", mock.Anything, roomID).Return(&domain.Schedule{
			ID:         uuid.New(),
			RoomID:     roomID,
			DaysOfWeek: validDays,
			StartTime:  "09:00",
			EndTime:    "18:00",
		}, nil).Maybe()
		mockSlotRepo.On("CreateBatch", mock.Anything, mock.Anything).Return(nil).Maybe()

		scheduleService := NewScheduleService(mockScheduleRepo, mockRoomRepo, mockSlotRepo, 30*time.Minute, slotService)

		schedule, err := scheduleService.Create(context.Background(), roomID, validDays, "09:00", "18:00")

		assert.NoError(t, err)
		assert.NotNil(t, schedule)

		time.Sleep(100 * time.Millisecond)

		mockRoomRepo.AssertExpectations(t)
		mockScheduleRepo.AssertExpectations(t)
		mockSlotRepo.AssertExpectations(t)
	})
}

func TestScheduleService_Create_WithSlotServiceNil(t *testing.T) {
	roomID := uuid.New()
	validDays := []int{1, 2, 3, 4, 5}

	mockRoomRepo := new(mocks.MockRoomRepository)
	mockScheduleRepo := new(mocks.MockScheduleRepository)
	mockSlotRepo := new(mocks.MockSlotRepository)

	mockRoomRepo.On("Exists", mock.Anything, roomID).Return(true, nil)
	mockScheduleRepo.On("Exists", mock.Anything, roomID).Return(false, nil)
	mockScheduleRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Schedule")).Return(nil)

	scheduleService := NewScheduleService(mockScheduleRepo, mockRoomRepo, mockSlotRepo, 30*time.Minute, nil)

	schedule, err := scheduleService.Create(context.Background(), roomID, validDays, "09:00", "18:00")

	assert.NoError(t, err)
	assert.NotNil(t, schedule)

	mockRoomRepo.AssertExpectations(t)
	mockScheduleRepo.AssertExpectations(t)
}
