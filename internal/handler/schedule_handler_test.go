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

func setupScheduleHandler() (*ScheduleHandler, *mocks.MockScheduleRepository, *mocks.MockRoomRepository, *mocks.MockSlotRepository) {
	mockScheduleRepo := new(mocks.MockScheduleRepository)
	mockRoomRepo := new(mocks.MockRoomRepository)
	mockSlotRepo := new(mocks.MockSlotRepository)

	scheduleService := service.NewScheduleService(mockScheduleRepo, mockRoomRepo, mockSlotRepo, 30*time.Minute, nil)
	scheduleHandler := NewScheduleHandler(scheduleService)

	return scheduleHandler, mockScheduleRepo, mockRoomRepo, mockSlotRepo
}

func TestScheduleHandler_Create(t *testing.T) {
	roomID := uuid.New()
	validDays := []int{1, 2, 3, 4, 5}

	tests := []struct {
		name           string
		roomID         string
		requestBody    CreateScheduleRequest
		setupMock      func(*mocks.MockRoomRepository, *mocks.MockScheduleRepository)
		expectedStatus int
		expectedError  string
	}{
		{
			name:   "successful schedule creation",
			roomID: roomID.String(),
			requestBody: CreateScheduleRequest{
				DaysOfWeek: validDays,
				StartTime:  "09:00",
				EndTime:    "18:00",
			},
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository) {
				roomRepo.On("Exists", mock.Anything, roomID).Return(true, nil)
				scheduleRepo.On("Exists", mock.Anything, roomID).Return(false, nil)
				scheduleRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Schedule")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:   "invalid room id format",
			roomID: "invalid-uuid",
			requestBody: CreateScheduleRequest{
				DaysOfWeek: validDays,
				StartTime:  "09:00",
				EndTime:    "18:00",
			},
			setupMock:      func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "room not found",
			roomID: roomID.String(),
			requestBody: CreateScheduleRequest{
				DaysOfWeek: validDays,
				StartTime:  "09:00",
				EndTime:    "18:00",
			},
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository) {
				roomRepo.On("Exists", mock.Anything, roomID).Return(false, nil)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "schedule already exists",
			roomID: roomID.String(),
			requestBody: CreateScheduleRequest{
				DaysOfWeek: validDays,
				StartTime:  "09:00",
				EndTime:    "18:00",
			},
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository) {
				roomRepo.On("Exists", mock.Anything, roomID).Return(true, nil)
				scheduleRepo.On("Exists", mock.Anything, roomID).Return(true, nil)
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockScheduleRepo, mockRoomRepo, _ := setupScheduleHandler()
			tt.setupMock(mockRoomRepo, mockScheduleRepo)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/rooms/"+tt.roomID+"/schedule/create", bytes.NewBuffer(body))
			req = mux.SetURLVars(req, map[string]string{"roomId": tt.roomID})

			w := httptest.NewRecorder()
			handler.Create(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated {
				var response map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Contains(t, response, "schedule")

				schedule := response["schedule"].(map[string]interface{})

				daysOfWeekRaw := schedule["daysOfWeek"].([]interface{})
				actualDaysOfWeek := make([]int, len(daysOfWeekRaw))
				for i, v := range daysOfWeekRaw {
					switch val := v.(type) {
					case float64:
						actualDaysOfWeek[i] = int(val)
					case int:
						actualDaysOfWeek[i] = val
					default:
						t.Errorf("unexpected type for daysOfWeek element: %T", v)
					}
				}
				assert.Equal(t, tt.requestBody.DaysOfWeek, actualDaysOfWeek)
				assert.Equal(t, tt.requestBody.StartTime, schedule["startTime"])
				assert.Equal(t, tt.requestBody.EndTime, schedule["endTime"])
			}

			mockRoomRepo.AssertExpectations(t)
			mockScheduleRepo.AssertExpectations(t)
		})
	}
}

func TestScheduleHandler_Create_EdgeCases(t *testing.T) {
	roomID := uuid.New()

	tests := []struct {
		name           string
		requestBody    CreateScheduleRequest
		setupMock      func(*mocks.MockRoomRepository, *mocks.MockScheduleRepository)
		expectedStatus int
	}{
		{
			name: "schedule with weekend only",
			requestBody: CreateScheduleRequest{
				DaysOfWeek: []int{6, 7},
				StartTime:  "10:00",
				EndTime:    "15:00",
			},
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository) {
				roomRepo.On("Exists", mock.Anything, roomID).Return(true, nil)
				scheduleRepo.On("Exists", mock.Anything, roomID).Return(false, nil)
				scheduleRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Schedule")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "schedule with midnight to morning",
			requestBody: CreateScheduleRequest{
				DaysOfWeek: []int{1, 2, 3, 4, 5},
				StartTime:  "00:00",
				EndTime:    "06:00",
			},
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository) {
				roomRepo.On("Exists", mock.Anything, roomID).Return(true, nil)
				scheduleRepo.On("Exists", mock.Anything, roomID).Return(false, nil)
				scheduleRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Schedule")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "schedule with evening to late night",
			requestBody: CreateScheduleRequest{
				DaysOfWeek: []int{1, 2, 3, 4, 5},
				StartTime:  "18:00",
				EndTime:    "23:59",
			},
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository) {
				roomRepo.On("Exists", mock.Anything, roomID).Return(true, nil)
				scheduleRepo.On("Exists", mock.Anything, roomID).Return(false, nil)
				scheduleRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Schedule")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockScheduleRepo, mockRoomRepo, _ := setupScheduleHandler()
			tt.setupMock(mockRoomRepo, mockScheduleRepo)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID.String()+"/schedule/create", bytes.NewBuffer(body))
			req = mux.SetURLVars(req, map[string]string{"roomId": roomID.String()})

			w := httptest.NewRecorder()
			handler.Create(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			mockRoomRepo.AssertExpectations(t)
			mockScheduleRepo.AssertExpectations(t)
		})
	}
}

func TestScheduleHandler_Create_VerifyAllFields(t *testing.T) {
	roomID := uuid.New()

	tests := []struct {
		name       string
		daysOfWeek []int
		startTime  string
		endTime    string
	}{
		{
			name:       "weekdays only",
			daysOfWeek: []int{1, 2, 3, 4, 5},
			startTime:  "09:00",
			endTime:    "17:00",
		},
		{
			name:       "weekend only",
			daysOfWeek: []int{6, 7},
			startTime:  "10:00",
			endTime:    "15:00",
		},
		{
			name:       "single day",
			daysOfWeek: []int{1},
			startTime:  "13:00",
			endTime:    "14:00",
		},
		{
			name:       "full week",
			daysOfWeek: []int{1, 2, 3, 4, 5, 6, 7},
			startTime:  "00:00",
			endTime:    "23:30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRoomRepo := new(mocks.MockRoomRepository)
			mockScheduleRepo := new(mocks.MockScheduleRepository)
			mockSlotRepo := new(mocks.MockSlotRepository)

			mockRoomRepo.On("Exists", mock.Anything, roomID).Return(true, nil)
			mockScheduleRepo.On("Exists", mock.Anything, roomID).Return(false, nil)
			mockScheduleRepo.On("Create", mock.Anything, mock.MatchedBy(func(schedule *domain.Schedule) bool {
				return schedule.RoomID == roomID &&
					len(schedule.DaysOfWeek) == len(tt.daysOfWeek) &&
					schedule.StartTime == tt.startTime &&
					schedule.EndTime == tt.endTime
			})).Return(nil)

			scheduleService := service.NewScheduleService(mockScheduleRepo, mockRoomRepo, mockSlotRepo, 30*time.Minute, nil)
			handler := NewScheduleHandler(scheduleService)

			requestBody := CreateScheduleRequest{
				DaysOfWeek: tt.daysOfWeek,
				StartTime:  tt.startTime,
				EndTime:    tt.endTime,
			}

			body, _ := json.Marshal(requestBody)
			req := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID.String()+"/schedule/create", bytes.NewBuffer(body))
			req = mux.SetURLVars(req, map[string]string{"roomId": roomID.String()})

			w := httptest.NewRecorder()
			handler.Create(w, req)

			assert.Equal(t, http.StatusCreated, w.Code)

			var response map[string]interface{}
			err := json.NewDecoder(w.Body).Decode(&response)
			assert.NoError(t, err)

			schedule := response["schedule"].(map[string]interface{})

			daysOfWeekRaw := schedule["daysOfWeek"].([]interface{})
			actualDaysOfWeek := make([]int, len(daysOfWeekRaw))
			for i, v := range daysOfWeekRaw {
				actualDaysOfWeek[i] = int(v.(float64))
			}

			assert.Equal(t, tt.daysOfWeek, actualDaysOfWeek)
			assert.Equal(t, tt.startTime, schedule["startTime"])
			assert.Equal(t, tt.endTime, schedule["endTime"])

			mockRoomRepo.AssertExpectations(t)
			mockScheduleRepo.AssertExpectations(t)
		})
	}
}

func TestScheduleHandler_Create_ValidationErrors(t *testing.T) {
	roomID := uuid.New()

	tests := []struct {
		name           string
		requestBody    CreateScheduleRequest
		setupMock      func(*mocks.MockRoomRepository, *mocks.MockScheduleRepository)
		expectedStatus int
	}{
		{
			name: "empty days of week",
			requestBody: CreateScheduleRequest{
				DaysOfWeek: []int{},
				StartTime:  "09:00",
				EndTime:    "18:00",
			},
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository) {
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid start time",
			requestBody: CreateScheduleRequest{
				DaysOfWeek: []int{1, 2, 3},
				StartTime:  "25:00",
				EndTime:    "18:00",
			},
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository) {
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid end time",
			requestBody: CreateScheduleRequest{
				DaysOfWeek: []int{1, 2, 3},
				StartTime:  "09:00",
				EndTime:    "25:00",
			},
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository) {
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "start time after end time",
			requestBody: CreateScheduleRequest{
				DaysOfWeek: []int{1, 2, 3},
				StartTime:  "18:00",
				EndTime:    "09:00",
			},
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository) {
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid days of week - too low",
			requestBody: CreateScheduleRequest{
				DaysOfWeek: []int{0},
				StartTime:  "09:00",
				EndTime:    "18:00",
			},
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository) {
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid days of week - too high",
			requestBody: CreateScheduleRequest{
				DaysOfWeek: []int{8},
				StartTime:  "09:00",
				EndTime:    "18:00",
			},
			setupMock: func(roomRepo *mocks.MockRoomRepository, scheduleRepo *mocks.MockScheduleRepository) {
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockScheduleRepo, mockRoomRepo, _ := setupScheduleHandler()
			tt.setupMock(mockRoomRepo, mockScheduleRepo)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID.String()+"/schedule/create", bytes.NewBuffer(body))
			req = mux.SetURLVars(req, map[string]string{"roomId": roomID.String()})

			w := httptest.NewRecorder()
			handler.Create(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			mockScheduleRepo.AssertNotCalled(t, "Create")
			mockRoomRepo.AssertNotCalled(t, "Exists")
		})
	}
}

func TestScheduleHandler_Create_ValidationWithRoomNotFound(t *testing.T) {
	roomID := uuid.New()

	mockRoomRepo := new(mocks.MockRoomRepository)
	mockScheduleRepo := new(mocks.MockScheduleRepository)
	mockSlotRepo := new(mocks.MockSlotRepository)

	mockRoomRepo.On("Exists", mock.Anything, roomID).Return(false, nil)

	scheduleService := service.NewScheduleService(mockScheduleRepo, mockRoomRepo, mockSlotRepo, 30*time.Minute, nil)
	handler := NewScheduleHandler(scheduleService)

	requestBody := CreateScheduleRequest{
		DaysOfWeek: []int{1, 2, 3, 4, 5},
		StartTime:  "09:00",
		EndTime:    "18:00",
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID.String()+"/schedule/create", bytes.NewBuffer(body))
	req = mux.SetURLVars(req, map[string]string{"roomId": roomID.String()})

	w := httptest.NewRecorder()
	handler.Create(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)

	errorResp := response["error"].(map[string]interface{})
	assert.Equal(t, "ROOM_NOT_FOUND", errorResp["code"])

	mockRoomRepo.AssertExpectations(t)
	mockScheduleRepo.AssertNotCalled(t, "Create")
}

func TestScheduleHandler_Create_SuccessWithVariousDays(t *testing.T) {
	roomID := uuid.New()

	tests := []struct {
		name       string
		daysOfWeek []int
		startTime  string
		endTime    string
	}{
		{
			name:       "single day",
			daysOfWeek: []int{1},
			startTime:  "10:00",
			endTime:    "12:00",
		},
		{
			name:       "multiple days",
			daysOfWeek: []int{1, 3, 5},
			startTime:  "14:00",
			endTime:    "16:00",
		},
		{
			name:       "full week",
			daysOfWeek: []int{1, 2, 3, 4, 5, 6, 7},
			startTime:  "00:00",
			endTime:    "23:59",
		},
		{
			name:       "weekend only",
			daysOfWeek: []int{6, 7},
			startTime:  "09:00",
			endTime:    "18:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRoomRepo := new(mocks.MockRoomRepository)
			mockScheduleRepo := new(mocks.MockScheduleRepository)
			mockSlotRepo := new(mocks.MockSlotRepository)

			mockRoomRepo.On("Exists", mock.Anything, roomID).Return(true, nil)
			mockScheduleRepo.On("Exists", mock.Anything, roomID).Return(false, nil)
			mockScheduleRepo.On("Create", mock.Anything, mock.MatchedBy(func(schedule *domain.Schedule) bool {
				return schedule.RoomID == roomID &&
					len(schedule.DaysOfWeek) == len(tt.daysOfWeek) &&
					schedule.StartTime == tt.startTime &&
					schedule.EndTime == tt.endTime
			})).Return(nil)

			scheduleService := service.NewScheduleService(mockScheduleRepo, mockRoomRepo, mockSlotRepo, 30*time.Minute, nil)
			handler := NewScheduleHandler(scheduleService)

			requestBody := CreateScheduleRequest{
				DaysOfWeek: tt.daysOfWeek,
				StartTime:  tt.startTime,
				EndTime:    tt.endTime,
			}

			body, _ := json.Marshal(requestBody)
			req := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID.String()+"/schedule/create", bytes.NewBuffer(body))
			req = mux.SetURLVars(req, map[string]string{"roomId": roomID.String()})

			w := httptest.NewRecorder()
			handler.Create(w, req)

			assert.Equal(t, http.StatusCreated, w.Code)

			mockRoomRepo.AssertExpectations(t)
			mockScheduleRepo.AssertExpectations(t)
		})
	}
}
