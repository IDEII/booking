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

func TestCompleteBookingFlow(t *testing.T) {
	mockRoomRepo := new(mocks.MockRoomRepository)
	mockScheduleRepo := new(mocks.MockScheduleRepository)
	mockSlotRepo := new(mocks.MockSlotRepository)
	mockBookingRepo := new(mocks.MockBookingRepository)
	mockConfSvc := new(mocks.MockConferenceService)

	roomService := NewRoomService(mockRoomRepo)

	slotService := NewSlotService(mockSlotRepo, mockRoomRepo, mockScheduleRepo)
	scheduleService := NewScheduleService(mockScheduleRepo, mockRoomRepo, mockSlotRepo, 30*time.Minute, slotService)
	bookingService := NewBookingService(mockBookingRepo, mockSlotRepo, mockConfSvc, 100, 20)

	ctx := context.Background()

	roomID := uuid.New()
	mockRoomRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Room")).Return(nil)

	room, err := roomService.Create(ctx, "Test Room", nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, room)

	mockRoomRepo.On("Exists", mock.Anything, roomID).Return(true, nil)
	mockScheduleRepo.On("Exists", mock.Anything, roomID).Return(false, nil)
	mockScheduleRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Schedule")).Return(nil)

	mockScheduleRepo.On("GetByRoomID", mock.Anything, roomID).Return(&domain.Schedule{
		ID:         uuid.New(),
		RoomID:     roomID,
		DaysOfWeek: []int{1, 2, 3, 4, 5},
		StartTime:  "09:00",
		EndTime:    "17:00",
	}, nil).Maybe()
	mockSlotRepo.On("CreateBatch", mock.Anything, mock.Anything).Return(nil).Maybe()

	schedule, err := scheduleService.Create(ctx, roomID, []int{1, 2, 3, 4, 5}, "09:00", "17:00")
	assert.NoError(t, err)
	assert.NotNil(t, schedule)

	time.Sleep(100 * time.Millisecond)

	now := time.Now().UTC()
	startDate := now.Truncate(24 * time.Hour)
	endDate := startDate.AddDate(0, 0, 7)

	mockScheduleRepo.On("GetByRoomID", mock.Anything, roomID).Return(schedule, nil).Maybe()
	mockSlotRepo.On("CreateBatch", mock.Anything, mock.AnythingOfType("[]domain.Slot")).Return(nil).Maybe()

	err = scheduleService.GenerateSlots(ctx, roomID, startDate, endDate)
	assert.NoError(t, err)

	futureDate := now.AddDate(0, 0, 1).Truncate(24 * time.Hour)
	availableSlots := []domain.Slot{
		{
			ID:     uuid.New(),
			RoomID: roomID,
			Start:  futureDate.Add(9 * time.Hour),
			End:    futureDate.Add(9*time.Hour + 30*time.Minute),
		},
	}

	mockRoomRepo.On("Exists", mock.Anything, roomID).Return(true, nil)
	mockScheduleRepo.On("GetByRoomID", mock.Anything, roomID).Return(schedule, nil)
	mockSlotRepo.On("GetByRoomAndDate", mock.Anything, roomID, futureDate).Return(availableSlots, nil)

	slots, err := slotService.GetAvailableSlots(ctx, roomID, futureDate)
	assert.NoError(t, err)
	assert.Len(t, slots, 1)

	userID := uuid.New()
	slot := slots[0]

	mockSlotRepo.On("GetByID", mock.Anything, slot.ID).Return(&slot, nil)
	mockBookingRepo.On("GetBySlotID", mock.Anything, slot.ID).Return(nil, nil)
	mockBookingRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Booking")).Return(nil)

	booking, err := bookingService.Create(ctx, userID, slot.ID, false)
	assert.NoError(t, err)
	assert.NotNil(t, booking)
	assert.Equal(t, userID, booking.UserID)
	assert.Equal(t, slot.ID, booking.SlotID)
	assert.Equal(t, domain.BookingStatusActive, booking.Status)

	userBookings := []domain.Booking{*booking}
	mockBookingRepo.On("GetByUserID", mock.Anything, userID).Return(userBookings, nil)
	mockSlotRepo.On("GetByID", mock.Anything, slot.ID).Return(&slot, nil)

	bookings, err := bookingService.ListUserBookings(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, bookings, 1)

	mockBookingRepo.On("GetByID", mock.Anything, booking.ID).Return(booking, nil)
	mockBookingRepo.On("UpdateStatus", mock.Anything, booking.ID, domain.BookingStatusCancelled).Return(nil)

	cancelledBooking, err := bookingService.Cancel(ctx, booking.ID, userID)
	assert.NoError(t, err)
	assert.Equal(t, domain.BookingStatusCancelled, cancelledBooking.Status)

	mockRoomRepo.AssertExpectations(t)
	mockScheduleRepo.AssertExpectations(t)
	mockSlotRepo.AssertExpectations(t)
	mockBookingRepo.AssertExpectations(t)
}
