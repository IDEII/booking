package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"booking-service/internal/domain"
	"booking-service/internal/mocks"
	"booking-service/internal/service"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupBookingHandler() (*BookingHandler, *mocks.MockBookingRepository, *mocks.MockSlotRepository, *mocks.MockConferenceService) {
	mockBookingRepo := new(mocks.MockBookingRepository)
	mockSlotRepo := new(mocks.MockSlotRepository)
	mockConfSvc := new(mocks.MockConferenceService)

	bookingService := service.NewBookingService(mockBookingRepo, mockSlotRepo, mockConfSvc, 100, 20)
	bookingHandler := NewBookingHandler(bookingService)

	return bookingHandler, mockBookingRepo, mockSlotRepo, mockConfSvc
}

func TestBookingHandler_Create(t *testing.T) {
	userID := uuid.New()
	slotID := uuid.New()
	now := time.Now()
	futureSlot := &domain.Slot{
		ID:     slotID,
		RoomID: uuid.New(),
		Start:  now.Add(24 * time.Hour),
		End:    now.Add(24*time.Hour + 30*time.Minute),
	}

	tests := []struct {
		name           string
		requestBody    CreateBookingRequest
		setupMock      func(*mocks.MockSlotRepository, *mocks.MockBookingRepository, *mocks.MockConferenceService)
		expectedStatus int
	}{
		{
			name: "successful booking creation",
			requestBody: CreateBookingRequest{
				SlotID:               slotID,
				CreateConferenceLink: false,
			},
			setupMock: func(slotRepo *mocks.MockSlotRepository, bookingRepo *mocks.MockBookingRepository, confSvc *mocks.MockConferenceService) {
				slotRepo.On("GetByID", mock.Anything, slotID).Return(futureSlot, nil)
				bookingRepo.On("GetBySlotID", mock.Anything, slotID).Return(nil, nil)
				bookingRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Booking")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "slot not found",
			requestBody: CreateBookingRequest{
				SlotID:               slotID,
				CreateConferenceLink: false,
			},
			setupMock: func(slotRepo *mocks.MockSlotRepository, bookingRepo *mocks.MockBookingRepository, confSvc *mocks.MockConferenceService) {
				slotRepo.On("GetByID", mock.Anything, slotID).Return(nil, nil)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "slot already booked",
			requestBody: CreateBookingRequest{
				SlotID:               slotID,
				CreateConferenceLink: false,
			},
			setupMock: func(slotRepo *mocks.MockSlotRepository, bookingRepo *mocks.MockBookingRepository, confSvc *mocks.MockConferenceService) {
				slotRepo.On("GetByID", mock.Anything, slotID).Return(futureSlot, nil)
				existingBooking := &domain.Booking{
					ID:     uuid.New(),
					SlotID: slotID,
					Status: domain.BookingStatusActive,
				}
				bookingRepo.On("GetBySlotID", mock.Anything, slotID).Return(existingBooking, nil)
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockBookingRepo, mockSlotRepo, mockConfSvc := setupBookingHandler()
			tt.setupMock(mockSlotRepo, mockBookingRepo, mockConfSvc)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/bookings/create", bytes.NewBuffer(body))
			ctx := addClaimsToContext(req.Context(), userID, domain.RoleUser)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.Create(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			mockSlotRepo.AssertExpectations(t)
			mockBookingRepo.AssertExpectations(t)
		})
	}
}

func TestBookingHandler_Cancel(t *testing.T) {
	userID := uuid.New()
	bookingID := uuid.New()

	tests := []struct {
		name           string
		bookingID      string
		setupMock      func(*mocks.MockBookingRepository)
		expectedStatus int
	}{
		{
			name:      "successful cancellation",
			bookingID: bookingID.String(),
			setupMock: func(bookingRepo *mocks.MockBookingRepository) {
				booking := &domain.Booking{
					ID:     bookingID,
					UserID: userID,
					Status: domain.BookingStatusActive,
				}
				bookingRepo.On("GetByID", mock.Anything, bookingID).Return(booking, nil)
				bookingRepo.On("UpdateStatus", mock.Anything, bookingID, domain.BookingStatusCancelled).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "booking not found",
			bookingID: bookingID.String(),
			setupMock: func(bookingRepo *mocks.MockBookingRepository) {
				bookingRepo.On("GetByID", mock.Anything, bookingID).Return(nil, nil)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid booking id",
			bookingID:      "invalid-uuid",
			setupMock:      func(bookingRepo *mocks.MockBookingRepository) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockBookingRepo, _, _ := setupBookingHandler()
			tt.setupMock(mockBookingRepo)

			req := httptest.NewRequest(http.MethodPost, "/bookings/"+tt.bookingID+"/cancel", nil)
			req = mux.SetURLVars(req, map[string]string{"bookingId": tt.bookingID})
			ctx := addClaimsToContext(req.Context(), userID, domain.RoleUser)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.Cancel(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Contains(t, response, "booking")
			}

			mockBookingRepo.AssertExpectations(t)
		})
	}
}

func TestBookingHandler_ListMy(t *testing.T) {
	userID := uuid.New()
	now := time.Now()
	futureSlot := &domain.Slot{
		ID:    uuid.New(),
		Start: now.Add(24 * time.Hour),
		End:   now.Add(24*time.Hour + 30*time.Minute),
	}

	tests := []struct {
		name           string
		setupMock      func(*mocks.MockBookingRepository, *mocks.MockSlotRepository)
		expectedStatus int
	}{
		{
			name: "successful list my bookings",
			setupMock: func(bookingRepo *mocks.MockBookingRepository, slotRepo *mocks.MockSlotRepository) {
				bookings := []domain.Booking{
					{ID: uuid.New(), SlotID: uuid.New(), UserID: userID, Status: domain.BookingStatusActive},
				}
				bookingRepo.On("GetByUserID", mock.Anything, userID).Return(bookings, nil)
				slotRepo.On("GetByID", mock.Anything, bookings[0].SlotID).Return(futureSlot, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "no bookings",
			setupMock: func(bookingRepo *mocks.MockBookingRepository, slotRepo *mocks.MockSlotRepository) {
				bookingRepo.On("GetByUserID", mock.Anything, userID).Return([]domain.Booking{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockBookingRepo, mockSlotRepo, _ := setupBookingHandler()
			tt.setupMock(mockBookingRepo, mockSlotRepo)

			req := httptest.NewRequest(http.MethodGet, "/bookings/my", nil)
			ctx := addClaimsToContext(req.Context(), userID, domain.RoleUser)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.ListMy(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Contains(t, response, "bookings")
			}

			mockBookingRepo.AssertExpectations(t)
		})
	}
}

func TestBookingHandler_Create_WithConferenceLink(t *testing.T) {
	userID := uuid.New()
	slotID := uuid.New()
	now := time.Now()
	futureSlot := &domain.Slot{
		ID:     slotID,
		RoomID: uuid.New(),
		Start:  now.Add(24 * time.Hour),
		End:    now.Add(24*time.Hour + 30*time.Minute),
	}
	conferenceLink := "https://meet.example.com/test-conference"

	tests := []struct {
		name           string
		requestBody    CreateBookingRequest
		setupMock      func(*mocks.MockSlotRepository, *mocks.MockBookingRepository, *mocks.MockConferenceService)
		expectedStatus int
	}{
		{
			name: "successful booking with conference link",
			requestBody: CreateBookingRequest{
				SlotID:               slotID,
				CreateConferenceLink: true,
			},
			setupMock: func(slotRepo *mocks.MockSlotRepository, bookingRepo *mocks.MockBookingRepository, confSvc *mocks.MockConferenceService) {
				slotRepo.On("GetByID", mock.Anything, slotID).Return(futureSlot, nil)
				bookingRepo.On("GetBySlotID", mock.Anything, slotID).Return(nil, nil)
				confSvc.On("CreateConference", mock.Anything, futureSlot).Return(conferenceLink, nil)
				bookingRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Booking")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "conference service failure",
			requestBody: CreateBookingRequest{
				SlotID:               slotID,
				CreateConferenceLink: true,
			},
			setupMock: func(slotRepo *mocks.MockSlotRepository, bookingRepo *mocks.MockBookingRepository, confSvc *mocks.MockConferenceService) {
				slotRepo.On("GetByID", mock.Anything, slotID).Return(futureSlot, nil)
				bookingRepo.On("GetBySlotID", mock.Anything, slotID).Return(nil, nil)
				confSvc.On("CreateConference", mock.Anything, futureSlot).Return("", assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBookingRepo := new(mocks.MockBookingRepository)
			mockSlotRepo := new(mocks.MockSlotRepository)
			mockConfSvc := new(mocks.MockConferenceService)

			tt.setupMock(mockSlotRepo, mockBookingRepo, mockConfSvc)

			bookingService := service.NewBookingService(mockBookingRepo, mockSlotRepo, mockConfSvc, 100, 20)
			bookingHandler := NewBookingHandler(bookingService)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/bookings/create", bytes.NewBuffer(body))
			ctx := addClaimsToContext(req.Context(), userID, domain.RoleUser)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			bookingHandler.Create(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated {
				var response map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				booking := response["booking"].(map[string]interface{})
				assert.NotNil(t, booking["conferenceLink"])
			}

			mockSlotRepo.AssertExpectations(t)
			mockBookingRepo.AssertExpectations(t)
			mockConfSvc.AssertExpectations(t)
		})
	}
}

func TestBookingHandler_ListAll_Pagination(t *testing.T) {
	adminID := domain.TestAdminID
	now := time.Now()

	tests := []struct {
		name           string
		page           string
		pageSize       string
		setupMock      func(*mocks.MockBookingRepository)
		expectedStatus int
		expectedPage   int
		expectedSize   int
	}{
		{
			name:     "first page with default size",
			page:     "",
			pageSize: "",
			setupMock: func(bookingRepo *mocks.MockBookingRepository) {
				bookings := []domain.Booking{
					{ID: uuid.New(), CreatedAt: &now},
					{ID: uuid.New(), CreatedAt: &now},
				}
				bookingRepo.On("List", mock.Anything, 1, 20).Return(bookings, 25, nil)
			},
			expectedStatus: http.StatusOK,
			expectedPage:   1,
			expectedSize:   20,
		},
		{
			name:     "second page with custom size",
			page:     "2",
			pageSize: "10",
			setupMock: func(bookingRepo *mocks.MockBookingRepository) {
				bookings := []domain.Booking{
					{ID: uuid.New(), CreatedAt: &now},
				}
				bookingRepo.On("List", mock.Anything, 2, 10).Return(bookings, 15, nil)
			},
			expectedStatus: http.StatusOK,
			expectedPage:   2,
			expectedSize:   10,
		},
		{
			name:     "page size exceeds max",
			page:     "1",
			pageSize: "200",
			setupMock: func(bookingRepo *mocks.MockBookingRepository) {
				bookings := []domain.Booking{
					{ID: uuid.New(), CreatedAt: &now},
				}
				bookingRepo.On("List", mock.Anything, 1, 100).Return(bookings, 50, nil)
			},
			expectedStatus: http.StatusOK,
			expectedPage:   1,
			expectedSize:   100,
		},
		{
			name:     "negative page - should default to 1",
			page:     "-1",
			pageSize: "20",
			setupMock: func(bookingRepo *mocks.MockBookingRepository) {
				bookings := []domain.Booking{
					{ID: uuid.New(), CreatedAt: &now},
				}
				bookingRepo.On("List", mock.Anything, 1, 20).Return(bookings, 10, nil)
			},
			expectedStatus: http.StatusOK,
			expectedPage:   1,
			expectedSize:   20,
		},
		{
			name:     "invalid page - should default to 1",
			page:     "invalid",
			pageSize: "20",
			setupMock: func(bookingRepo *mocks.MockBookingRepository) {
				bookings := []domain.Booking{
					{ID: uuid.New(), CreatedAt: &now},
				}
				bookingRepo.On("List", mock.Anything, 1, 20).Return(bookings, 10, nil)
			},
			expectedStatus: http.StatusOK,
			expectedPage:   1,
			expectedSize:   20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBookingRepo := new(mocks.MockBookingRepository)
			mockSlotRepo := new(mocks.MockSlotRepository)
			mockConfSvc := new(mocks.MockConferenceService)

			tt.setupMock(mockBookingRepo)

			bookingService := service.NewBookingService(mockBookingRepo, mockSlotRepo, mockConfSvc, 100, 20)
			bookingHandler := NewBookingHandler(bookingService)

			url := "/bookings/list"
			if tt.page != "" || tt.pageSize != "" {
				url += "?"
				if tt.page != "" {
					url += "page=" + tt.page
				}
				if tt.pageSize != "" {
					if tt.page != "" {
						url += "&"
					}
					url += "pageSize=" + tt.pageSize
				}
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			ctx := addClaimsToContext(req.Context(), adminID, domain.RoleAdmin)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			bookingHandler.ListAll(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Contains(t, response, "pagination")
				pagination := response["pagination"].(map[string]interface{})
				assert.Equal(t, float64(tt.expectedPage), pagination["page"])
				assert.Equal(t, float64(tt.expectedSize), pagination["pageSize"])
			}

			mockBookingRepo.AssertExpectations(t)
		})
	}
}

func TestBookingHandler_ListMy_Filtering(t *testing.T) {
	userID := uuid.New()
	now := time.Now()
	futureSlot := &domain.Slot{
		ID:    uuid.New(),
		Start: now.Add(24 * time.Hour),
		End:   now.Add(24*time.Hour + 30*time.Minute),
	}
	pastSlot := &domain.Slot{
		ID:    uuid.New(),
		Start: now.Add(-48 * time.Hour),
		End:   now.Add(-48*time.Hour + 30*time.Minute),
	}

	tests := []struct {
		name           string
		setupMock      func(*mocks.MockBookingRepository, *mocks.MockSlotRepository)
		expectedStatus int
		expectedCount  int
	}{
		{
			name: "only future bookings returned",
			setupMock: func(bookingRepo *mocks.MockBookingRepository, slotRepo *mocks.MockSlotRepository) {
				bookings := []domain.Booking{
					{ID: uuid.New(), SlotID: futureSlot.ID, UserID: userID, Status: domain.BookingStatusActive},
					{ID: uuid.New(), SlotID: pastSlot.ID, UserID: userID, Status: domain.BookingStatusActive},
				}
				bookingRepo.On("GetByUserID", mock.Anything, userID).Return(bookings, nil)
				slotRepo.On("GetByID", mock.Anything, futureSlot.ID).Return(futureSlot, nil)
				slotRepo.On("GetByID", mock.Anything, pastSlot.ID).Return(pastSlot, nil)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name: "all future bookings",
			setupMock: func(bookingRepo *mocks.MockBookingRepository, slotRepo *mocks.MockSlotRepository) {
				bookings := []domain.Booking{
					{ID: uuid.New(), SlotID: futureSlot.ID, UserID: userID, Status: domain.BookingStatusActive},
					{ID: uuid.New(), SlotID: uuid.New(), UserID: userID, Status: domain.BookingStatusActive},
				}
				bookingRepo.On("GetByUserID", mock.Anything, userID).Return(bookings, nil)
				slotRepo.On("GetByID", mock.Anything, futureSlot.ID).Return(futureSlot, nil)
				slotRepo.On("GetByID", mock.Anything, mock.Anything).Return(futureSlot, nil)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name: "no future bookings",
			setupMock: func(bookingRepo *mocks.MockBookingRepository, slotRepo *mocks.MockSlotRepository) {
				bookings := []domain.Booking{
					{ID: uuid.New(), SlotID: pastSlot.ID, UserID: userID, Status: domain.BookingStatusActive},
				}
				bookingRepo.On("GetByUserID", mock.Anything, userID).Return(bookings, nil)
				slotRepo.On("GetByID", mock.Anything, pastSlot.ID).Return(pastSlot, nil)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBookingRepo := new(mocks.MockBookingRepository)
			mockSlotRepo := new(mocks.MockSlotRepository)
			mockConfSvc := new(mocks.MockConferenceService)

			tt.setupMock(mockBookingRepo, mockSlotRepo)

			bookingService := service.NewBookingService(mockBookingRepo, mockSlotRepo, mockConfSvc, 100, 20)
			bookingHandler := NewBookingHandler(bookingService)

			req := httptest.NewRequest(http.MethodGet, "/bookings/my", nil)
			ctx := addClaimsToContext(req.Context(), userID, domain.RoleUser)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			bookingHandler.ListMy(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				bookings := response["bookings"].([]interface{})
				assert.Len(t, bookings, tt.expectedCount)
			}

			mockBookingRepo.AssertExpectations(t)
			mockSlotRepo.AssertExpectations(t)
		})
	}
}

func TestBookingHandler_Cancel_Idempotent(t *testing.T) {
	userID := uuid.New()
	bookingID := uuid.New()

	tests := []struct {
		name           string
		setupMock      func(*mocks.MockBookingRepository)
		expectedStatus int
	}{
		{
			name: "cancel already cancelled booking",
			setupMock: func(bookingRepo *mocks.MockBookingRepository) {
				booking := &domain.Booking{
					ID:     bookingID,
					UserID: userID,
					Status: domain.BookingStatusCancelled,
				}
				bookingRepo.On("GetByID", mock.Anything, bookingID).Return(booking, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "cancel active booking first time",
			setupMock: func(bookingRepo *mocks.MockBookingRepository) {
				booking := &domain.Booking{
					ID:     bookingID,
					UserID: userID,
					Status: domain.BookingStatusActive,
				}
				bookingRepo.On("GetByID", mock.Anything, bookingID).Return(booking, nil)
				bookingRepo.On("UpdateStatus", mock.Anything, bookingID, domain.BookingStatusCancelled).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBookingRepo := new(mocks.MockBookingRepository)
			mockSlotRepo := new(mocks.MockSlotRepository)
			mockConfSvc := new(mocks.MockConferenceService)

			tt.setupMock(mockBookingRepo)

			bookingService := service.NewBookingService(mockBookingRepo, mockSlotRepo, mockConfSvc, 100, 20)
			bookingHandler := NewBookingHandler(bookingService)

			req := httptest.NewRequest(http.MethodPost, "/bookings/"+bookingID.String()+"/cancel", nil)
			req = mux.SetURLVars(req, map[string]string{"bookingId": bookingID.String()})
			ctx := addClaimsToContext(req.Context(), userID, domain.RoleUser)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			bookingHandler.Cancel(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				booking := response["booking"].(map[string]interface{})
				assert.Equal(t, "cancelled", booking["status"])
			}

			mockBookingRepo.AssertExpectations(t)
		})
	}
}
