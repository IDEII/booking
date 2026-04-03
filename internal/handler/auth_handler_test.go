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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAuthHandler_DummyLogin(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    DummyLoginRequest
		setupMock      func(*mocks.MockUserRepository)
		expectedStatus int
	}{
		{
			name:        "successful admin login",
			requestBody: DummyLoginRequest{Role: domain.RoleAdmin},
			setupMock: func(userRepo *mocks.MockUserRepository) {
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "successful user login",
			requestBody: DummyLoginRequest{Role: domain.RoleUser},
			setupMock: func(userRepo *mocks.MockUserRepository) {
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid role",
			requestBody:    DummyLoginRequest{Role: "invalid"},
			setupMock:      func(userRepo *mocks.MockUserRepository) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(mocks.MockUserRepository)
			tt.setupMock(mockUserRepo)

			authService := service.NewAuthService(mockUserRepo, "test-secret", 24*time.Hour)
			authHandler := NewAuthHandler(authService)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/dummyLogin", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			authHandler.DummyLogin(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Contains(t, response, "token")
				assert.NotEmpty(t, response["token"])
			}
		})
	}
}

func TestAuthHandler_Register(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    RegisterRequest
		setupMock      func(*mocks.MockUserRepository)
		expectedStatus int
	}{
		{
			name: "successful registration",
			requestBody: RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
				Role:     domain.RoleUser,
			},
			setupMock: func(userRepo *mocks.MockUserRepository) {
				userRepo.On("GetByEmail", mock.Anything, "test@example.com").
					Return(nil, nil)
				userRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).
					Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "email already exists",
			requestBody: RegisterRequest{
				Email:    "existing@example.com",
				Password: "password123",
				Role:     domain.RoleUser,
			},
			setupMock: func(userRepo *mocks.MockUserRepository) {
				existingUser := &domain.User{
					ID:    uuid.New(),
					Email: "existing@example.com",
				}
				userRepo.On("GetByEmail", mock.Anything, "existing@example.com").
					Return(existingUser, nil)
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(mocks.MockUserRepository)
			tt.setupMock(mockUserRepo)

			authService := service.NewAuthService(mockUserRepo, "test-secret", 24*time.Hour)
			authHandler := NewAuthHandler(authService)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			authHandler.Register(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated {
				var response map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Contains(t, response, "user")
			}

			mockUserRepo.AssertExpectations(t)
		})
	}
}
