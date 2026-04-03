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

func TestBookingService_Create(t *testing.T) {
	userID := uuid.New()
	slotID := uuid.New()
	now := time.Now()
	futureSlot := &domain.Slot{
		ID:     slotID,
		RoomID: uuid.New(),
		Start:  now.Add(24 * time.Hour),
		End:    now.Add(24*time.Hour + 30*time.Minute),
	}
	pastSlot := &domain.Slot{
		ID:     slotID,
		RoomID: uuid.New(),
		Start:  now.Add(-24 * time.Hour),
		End:    now.Add(-24*time.Hour + 30*time.Minute),
	}

	tests := []struct {
		name                  string
		userID                uuid.UUID
		slotID                uuid.UUID
		createConferenceLink  bool
		setupMock             func(*mocks.MockSlotRepository, *mocks.MockBookingRepository, *mocks.MockConferenceService)
		expectedErr           error
		expectedConferenceSet bool
	}{
		{
			name:                 "successful booking without conference link",
			userID:               userID,
			slotID:               slotID,
			createConferenceLink: false,
			setupMock: func(slotRepo *mocks.MockSlotRepository, bookingRepo *mocks.MockBookingRepository, confSvc *mocks.MockConferenceService) {
				slotRepo.On("GetByID", mock.Anything, slotID).Return(futureSlot, nil)
				bookingRepo.On("GetBySlotID", mock.Anything, slotID).Return(nil, nil)
				bookingRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Booking")).Return(nil)
			},
			expectedErr:           nil,
			expectedConferenceSet: false,
		},
		{
			name:                 "successful booking with conference link",
			userID:               userID,
			slotID:               slotID,
			createConferenceLink: true,
			setupMock: func(slotRepo *mocks.MockSlotRepository, bookingRepo *mocks.MockBookingRepository, confSvc *mocks.MockConferenceService) {
				slotRepo.On("GetByID", mock.Anything, slotID).Return(futureSlot, nil)
				bookingRepo.On("GetBySlotID", mock.Anything, slotID).Return(nil, nil)
				confSvc.On("CreateConference", mock.Anything, futureSlot).Return("https://meet.example.com/test", nil)
				bookingRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Booking")).Return(nil)
			},
			expectedErr:           nil,
			expectedConferenceSet: true,
		},
		{
			name:                 "slot not found",
			userID:               userID,
			slotID:               slotID,
			createConferenceLink: false,
			setupMock: func(slotRepo *mocks.MockSlotRepository, bookingRepo *mocks.MockBookingRepository, confSvc *mocks.MockConferenceService) {
				slotRepo.On("GetByID", mock.Anything, slotID).Return(nil, nil)
			},
			expectedErr: domain.ErrSlotNotFound,
		},
		{
			name:                 "slot in past",
			userID:               userID,
			slotID:               slotID,
			createConferenceLink: false,
			setupMock: func(slotRepo *mocks.MockSlotRepository, bookingRepo *mocks.MockBookingRepository, confSvc *mocks.MockConferenceService) {
				slotRepo.On("GetByID", mock.Anything, slotID).Return(pastSlot, nil)
			},
			expectedErr: domain.ErrSlotInPast,
		},
		{
			name:                 "slot already booked",
			userID:               userID,
			slotID:               slotID,
			createConferenceLink: false,
			setupMock: func(slotRepo *mocks.MockSlotRepository, bookingRepo *mocks.MockBookingRepository, confSvc *mocks.MockConferenceService) {
				slotRepo.On("GetByID", mock.Anything, slotID).Return(futureSlot, nil)
				existingBooking := &domain.Booking{
					ID:     uuid.New(),
					SlotID: slotID,
					Status: domain.BookingStatusActive,
				}
				bookingRepo.On("GetBySlotID", mock.Anything, slotID).Return(existingBooking, nil)
			},
			expectedErr: domain.ErrSlotAlreadyBooked,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSlotRepo := new(mocks.MockSlotRepository)
			mockBookingRepo := new(mocks.MockBookingRepository)
			mockConfSvc := new(mocks.MockConferenceService)

			tt.setupMock(mockSlotRepo, mockBookingRepo, mockConfSvc)

			bookingService := NewBookingService(mockBookingRepo, mockSlotRepo, mockConfSvc, 100, 20)

			booking, err := bookingService.Create(context.Background(), tt.userID, tt.slotID, tt.createConferenceLink)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr, err)
				assert.Nil(t, booking)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, booking)
				assert.Equal(t, tt.userID, booking.UserID)
				assert.Equal(t, slotID, booking.SlotID)
				assert.Equal(t, domain.BookingStatusActive, booking.Status)

				if tt.expectedConferenceSet {
					assert.NotNil(t, booking.ConferenceLink)
				} else {
					assert.Nil(t, booking.ConferenceLink)
				}
			}

			mockSlotRepo.AssertExpectations(t)
			mockBookingRepo.AssertExpectations(t)
			if tt.createConferenceLink {
				mockConfSvc.AssertExpectations(t)
			}
		})
	}
}

func TestBookingService_Cancel(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()
	bookingID := uuid.New()

	tests := []struct {
		name        string
		bookingID   uuid.UUID
		userID      uuid.UUID
		setupMock   func(*mocks.MockBookingRepository)
		expectedErr error
	}{
		{
			name:      "successful cancellation",
			bookingID: bookingID,
			userID:    userID,
			setupMock: func(bookingRepo *mocks.MockBookingRepository) {
				booking := &domain.Booking{
					ID:     bookingID,
					UserID: userID,
					Status: domain.BookingStatusActive,
				}
				bookingRepo.On("GetByID", mock.Anything, bookingID).Return(booking, nil)
				bookingRepo.On("UpdateStatus", mock.Anything, bookingID, domain.BookingStatusCancelled).Return(nil)
			},
			expectedErr: nil,
		},
		{
			name:      "cancel already cancelled booking (idempotent)",
			bookingID: bookingID,
			userID:    userID,
			setupMock: func(bookingRepo *mocks.MockBookingRepository) {
				booking := &domain.Booking{
					ID:     bookingID,
					UserID: userID,
					Status: domain.BookingStatusCancelled,
				}
				bookingRepo.On("GetByID", mock.Anything, bookingID).Return(booking, nil)
			},
			expectedErr: nil,
		},
		{
			name:      "booking not found",
			bookingID: bookingID,
			userID:    userID,
			setupMock: func(bookingRepo *mocks.MockBookingRepository) {
				bookingRepo.On("GetByID", mock.Anything, bookingID).Return(nil, nil)
			},
			expectedErr: domain.ErrBookingNotFound,
		},
		{
			name:      "forbidden - wrong user",
			bookingID: bookingID,
			userID:    otherUserID,
			setupMock: func(bookingRepo *mocks.MockBookingRepository) {
				booking := &domain.Booking{
					ID:     bookingID,
					UserID: userID,
					Status: domain.BookingStatusActive,
				}
				bookingRepo.On("GetByID", mock.Anything, bookingID).Return(booking, nil)
			},
			expectedErr: domain.ErrForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBookingRepo := new(mocks.MockBookingRepository)
			tt.setupMock(mockBookingRepo)

			mockSlotRepo := new(mocks.MockSlotRepository)
			mockConfSvc := new(mocks.MockConferenceService)

			bookingService := NewBookingService(mockBookingRepo, mockSlotRepo, mockConfSvc, 100, 20)

			booking, err := bookingService.Cancel(context.Background(), tt.bookingID, tt.userID)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr, err)
				assert.Nil(t, booking)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, booking)
				assert.Equal(t, domain.BookingStatusCancelled, booking.Status)
			}

			mockBookingRepo.AssertExpectations(t)
		})
	}
}

func TestBookingService_ListAll(t *testing.T) {
	tests := []struct {
		name      string
		page      int
		pageSize  int
		setupMock func(*mocks.MockBookingRepository)
		expected  int
	}{
		{
			name:     "list with default pagination",
			page:     0,
			pageSize: 0,
			setupMock: func(bookingRepo *mocks.MockBookingRepository) {
				bookings := []domain.Booking{
					{ID: uuid.New()},
					{ID: uuid.New()},
				}
				bookingRepo.On("List", mock.Anything, 1, 20).Return(bookings, 2, nil)
			},
			expected: 2,
		},
		{
			name:     "list with custom pagination",
			page:     2,
			pageSize: 10,
			setupMock: func(bookingRepo *mocks.MockBookingRepository) {
				bookings := []domain.Booking{
					{ID: uuid.New()},
				}
				bookingRepo.On("List", mock.Anything, 2, 10).Return(bookings, 11, nil)
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBookingRepo := new(mocks.MockBookingRepository)
			tt.setupMock(mockBookingRepo)

			mockSlotRepo := new(mocks.MockSlotRepository)
			mockConfSvc := new(mocks.MockConferenceService)

			bookingService := NewBookingService(mockBookingRepo, mockSlotRepo, mockConfSvc, 100, 20)

			bookings, pagination, err := bookingService.ListAll(context.Background(), tt.page, tt.pageSize)

			assert.NoError(t, err)
			assert.Len(t, bookings, tt.expected)
			assert.NotNil(t, pagination)
			mockBookingRepo.AssertExpectations(t)
		})
	}
}
