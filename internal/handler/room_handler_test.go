package handler

import (
	"booking-service/internal/domain"
	"booking-service/internal/mocks"
	"booking-service/internal/service"

	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupRoomHandler() (*RoomHandler, *mocks.MockRoomRepository) {
	mockRoomRepo := new(mocks.MockRoomRepository)
	roomService := service.NewRoomService(mockRoomRepo)
	roomHandler := NewRoomHandler(roomService)

	return roomHandler, mockRoomRepo
}

func TestRoomHandler_Create(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    CreateRoomRequest
		setupMock      func(*mocks.MockRoomRepository)
		expectedStatus int
	}{
		{
			name: "successful room creation",
			requestBody: CreateRoomRequest{
				Name:        "Conference Room A",
				Description: stringPtr("Large conference room"),
				Capacity:    intPtr(10),
			},
			setupMock: func(roomRepo *mocks.MockRoomRepository) {
				roomRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Room")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "room creation without optional fields",
			requestBody: CreateRoomRequest{
				Name: "Small Meeting Room",
			},
			setupMock: func(roomRepo *mocks.MockRoomRepository) {
				roomRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Room")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockRoomRepo := setupRoomHandler()
			tt.setupMock(mockRoomRepo)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/rooms/create", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			handler.Create(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated {
				var response map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Contains(t, response, "room")
			}

			mockRoomRepo.AssertExpectations(t)
		})
	}
}

func TestRoomHandler_List(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mocks.MockRoomRepository)
		expectedStatus int
		expectedCount  int
	}{
		{
			name: "successful list rooms",
			setupMock: func(roomRepo *mocks.MockRoomRepository) {
				rooms := []domain.Room{
					{ID: uuid.New(), Name: "Room 1"},
					{ID: uuid.New(), Name: "Room 2"},
				}
				roomRepo.On("List", mock.Anything).Return(rooms, nil)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name: "empty list",
			setupMock: func(roomRepo *mocks.MockRoomRepository) {
				roomRepo.On("List", mock.Anything).Return([]domain.Room{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockRoomRepo := setupRoomHandler()
			tt.setupMock(mockRoomRepo)

			req := httptest.NewRequest(http.MethodGet, "/rooms/list", nil)
			w := httptest.NewRecorder()

			handler.List(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.NewDecoder(w.Body).Decode(&response)
			assert.NoError(t, err)

			rooms := response["rooms"].([]interface{})
			assert.Len(t, rooms, tt.expectedCount)

			mockRoomRepo.AssertExpectations(t)
		})
	}
}
