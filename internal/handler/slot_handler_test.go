package handler

import (
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

func setupSlotHandler() (*SlotHandler, *mocks.MockSlotRepository, *mocks.MockRoomRepository, *mocks.MockScheduleRepository) {
	mockSlotRepo := new(mocks.MockSlotRepository)
	mockRoomRepo := new(mocks.MockRoomRepository)
	mockScheduleRepo := new(mocks.MockScheduleRepository)

	slotService := service.NewSlotService(mockSlotRepo, mockRoomRepo, mockScheduleRepo)
	slotHandler := NewSlotHandler(slotService)

	time.Sleep(10 * time.Millisecond)

	return slotHandler, mockSlotRepo, mockRoomRepo, mockScheduleRepo
}

func TestSlotHandler_ListAvailable(t *testing.T) {
	roomID := uuid.New()

	now := time.Now().UTC()
	futureDate := now.AddDate(0, 0, 1)

	for futureDate.Weekday() != time.Monday {
		futureDate = futureDate.AddDate(0, 0, 1)
	}

	dateForHandler := time.Date(futureDate.Year(), futureDate.Month(), futureDate.Day(), 0, 0, 0, 0, time.UTC)
	dateStr := dateForHandler.Format("2006-01-02")

	weekday := int(dateForHandler.Weekday())
	mappedDay := weekday
	if mappedDay == 0 {
		mappedDay = 7
	}

	availableSlots := []domain.Slot{
		{
			ID:     uuid.New(),
			RoomID: roomID,
			Start:  dateForHandler.Add(9 * time.Hour),
			End:    dateForHandler.Add(9*time.Hour + 30*time.Minute),
		},
		{
			ID:     uuid.New(),
			RoomID: roomID,
			Start:  dateForHandler.Add(10 * time.Hour),
			End:    dateForHandler.Add(10*time.Hour + 30*time.Minute),
		},
	}

	tests := []struct {
		name           string
		roomID         string
		dateParam      string
		setupMock      func(*mocks.MockRoomRepository, *mocks.MockSlotRepository, *mocks.MockScheduleRepository)
		expectedStatus int
		expectedCount  int
	}{
		{
			name:      "successful list available slots",
			roomID:    roomID.String(),
			dateParam: dateStr,
			setupMock: func(roomRepo *mocks.MockRoomRepository, slotRepo *mocks.MockSlotRepository, scheduleRepo *mocks.MockScheduleRepository) {
				roomRepo.On("Exists", mock.Anything, roomID).Return(true, nil)
				schedule := &domain.Schedule{
					ID:         uuid.New(),
					RoomID:     roomID,
					DaysOfWeek: []int{mappedDay},
					StartTime:  "09:00",
					EndTime:    "18:00",
				}
				scheduleRepo.On("GetByRoomID", mock.Anything, roomID).Return(schedule, nil)

				slotRepo.On("GetByRoomAndDate", mock.Anything, roomID, dateForHandler).Return(availableSlots, nil)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:      "no slots available - returns empty list",
			roomID:    roomID.String(),
			dateParam: dateStr,
			setupMock: func(roomRepo *mocks.MockRoomRepository, slotRepo *mocks.MockSlotRepository, scheduleRepo *mocks.MockScheduleRepository) {
				roomRepo.On("Exists", mock.Anything, roomID).Return(true, nil)
				schedule := &domain.Schedule{
					ID:         uuid.New(),
					RoomID:     roomID,
					DaysOfWeek: []int{mappedDay},
					StartTime:  "09:00",
					EndTime:    "18:00",
				}
				scheduleRepo.On("GetByRoomID", mock.Anything, roomID).Return(schedule, nil)
				slotRepo.On("GetByRoomAndDate", mock.Anything, roomID, dateForHandler).Return([]domain.Slot{}, nil)

				slotRepo.On("CreateBatch", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name:      "invalid room id",
			roomID:    "invalid-uuid",
			dateParam: dateStr,
			setupMock: func(roomRepo *mocks.MockRoomRepository, slotRepo *mocks.MockSlotRepository, scheduleRepo *mocks.MockScheduleRepository) {
			},
			expectedStatus: http.StatusBadRequest,
			expectedCount:  0,
		},
		{
			name:      "missing date parameter",
			roomID:    roomID.String(),
			dateParam: "",
			setupMock: func(roomRepo *mocks.MockRoomRepository, slotRepo *mocks.MockSlotRepository, scheduleRepo *mocks.MockScheduleRepository) {
			},
			expectedStatus: http.StatusBadRequest,
			expectedCount:  0,
		},
		{
			name:      "invalid date format",
			roomID:    roomID.String(),
			dateParam: "2024-13-45",
			setupMock: func(roomRepo *mocks.MockRoomRepository, slotRepo *mocks.MockSlotRepository, scheduleRepo *mocks.MockScheduleRepository) {
			},
			expectedStatus: http.StatusBadRequest,
			expectedCount:  0,
		},
		{
			name:      "room not found",
			roomID:    roomID.String(),
			dateParam: dateStr,
			setupMock: func(roomRepo *mocks.MockRoomRepository, slotRepo *mocks.MockSlotRepository, scheduleRepo *mocks.MockScheduleRepository) {
				roomRepo.On("Exists", mock.Anything, roomID).Return(false, nil)
			},
			expectedStatus: http.StatusNotFound,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockSlotRepo, mockRoomRepo, mockScheduleRepo := setupSlotHandler()
			tt.setupMock(mockRoomRepo, mockSlotRepo, mockScheduleRepo)

			url := "/rooms/" + tt.roomID + "/slots/list"
			if tt.dateParam != "" {
				url += "?date=" + tt.dateParam
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			req = mux.SetURLVars(req, map[string]string{"roomId": tt.roomID})

			w := httptest.NewRecorder()
			handler.ListAvailable(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Contains(t, response, "slots")
				slots, ok := response["slots"].([]interface{})
				if ok {
					assert.Len(t, slots, tt.expectedCount)
				}
			}

			time.Sleep(50 * time.Millisecond)

			mockRoomRepo.AssertExpectations(t)
			mockSlotRepo.AssertExpectations(t)
			mockScheduleRepo.AssertExpectations(t)
		})
	}
}

func TestSlotHandler_ListAvailable_WithDifferentDates(t *testing.T) {
	roomID := uuid.New()

	allDays := []int{1, 2, 3, 4, 5, 6, 7}

	tests := []struct {
		name           string
		dateStr        string
		expectedStatus int
	}{
		{
			name:           "today's date",
			dateStr:        time.Now().UTC().Format("2006-01-02"),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "tomorrow's date",
			dateStr:        time.Now().UTC().Add(24 * time.Hour).Format("2006-01-02"),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "next week date",
			dateStr:        time.Now().UTC().Add(7 * 24 * time.Hour).Format("2006-01-02"),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "past date",
			dateStr:        time.Now().UTC().Add(-24 * time.Hour).Format("2006-01-02"),
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockSlotRepo, mockRoomRepo, mockScheduleRepo := setupSlotHandler()

			parsedDate, err := time.Parse("2006-01-02", tt.dateStr)
			if err != nil {
				t.Fatalf("Failed to parse date: %v", err)
			}

			parsedDate = time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 0, 0, 0, 0, time.UTC)

			mockRoomRepo.On("Exists", mock.Anything, roomID).Return(true, nil)

			schedule := &domain.Schedule{
				ID:         uuid.New(),
				RoomID:     roomID,
				DaysOfWeek: allDays,
				StartTime:  "09:00",
				EndTime:    "18:00",
			}
			mockScheduleRepo.On("GetByRoomID", mock.Anything, roomID).Return(schedule, nil)

			mockSlotRepo.On("GetByRoomAndDate", mock.Anything, roomID, parsedDate).Return([]domain.Slot{}, nil)

			mockSlotRepo.On("CreateBatch", mock.Anything, mock.Anything).Return(nil).Maybe()

			url := "/rooms/" + roomID.String() + "/slots/list?date=" + tt.dateStr
			req := httptest.NewRequest(http.MethodGet, url, nil)
			req = mux.SetURLVars(req, map[string]string{"roomId": roomID.String()})

			w := httptest.NewRecorder()
			handler.ListAvailable(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			time.Sleep(50 * time.Millisecond)

			mockRoomRepo.AssertExpectations(t)
			mockSlotRepo.AssertExpectations(t)
			mockScheduleRepo.AssertExpectations(t)
		})
	}
}
