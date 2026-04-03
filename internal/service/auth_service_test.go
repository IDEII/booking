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
	"golang.org/x/crypto/bcrypt"
)

func TestAuthService_DummyLogin(t *testing.T) {
	tests := []struct {
		name     string
		role     domain.UserRole
		expected uuid.UUID
		wantErr  bool
	}{
		{
			name:     "admin login",
			role:     domain.RoleAdmin,
			expected: domain.TestAdminID,
			wantErr:  false,
		},
		{
			name:     "user login",
			role:     domain.RoleUser,
			expected: domain.TestUserID,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(mocks.MockUserRepository)
			authService := NewAuthService(mockUserRepo, "test-secret", 24*time.Hour)

			token, err := authService.DummyLogin(context.Background(), tt.role)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)

				claims, err := authService.ValidateToken(token)
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, claims.UserID)
				assert.Equal(t, tt.role, claims.Role)
			}
		})
	}
}

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		password    string
		role        domain.UserRole
		setupMock   func(*mocks.MockUserRepository)
		expectedErr bool
	}{
		{
			name:     "successful registration",
			email:    "test@example.com",
			password: "password123",
			role:     domain.RoleUser,
			setupMock: func(m *mocks.MockUserRepository) {
				m.On("GetByEmail", mock.Anything, "test@example.com").
					Return(nil, nil)
				m.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).
					Return(nil)
			},
			expectedErr: false,
		},
		{
			name:     "email already exists",
			email:    "existing@example.com",
			password: "password123",
			role:     domain.RoleUser,
			setupMock: func(m *mocks.MockUserRepository) {
				existingUser := &domain.User{
					ID:    uuid.New(),
					Email: "existing@example.com",
				}
				m.On("GetByEmail", mock.Anything, "existing@example.com").
					Return(existingUser, nil)
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(mocks.MockUserRepository)
			tt.setupMock(mockUserRepo)

			authService := NewAuthService(mockUserRepo, "test-secret", 24*time.Hour)

			user, err := authService.Register(context.Background(), tt.email, tt.password, tt.role)

			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.email, user.Email)
				assert.Equal(t, tt.role, user.Role)
				assert.NotEmpty(t, user.PasswordHash)

				err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(tt.password))
				assert.NoError(t, err)
			}

			mockUserRepo.AssertExpectations(t)
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	userID := uuid.New()

	tests := []struct {
		name           string
		email          string
		password       string
		setupMock      func(*mocks.MockUserRepository)
		expectedErr    bool
		expectedUserID uuid.UUID
	}{
		{
			name:     "successful login",
			email:    "test@example.com",
			password: password,
			setupMock: func(m *mocks.MockUserRepository) {
				user := &domain.User{
					ID:           userID,
					Email:        "test@example.com",
					PasswordHash: string(hashedPassword),
					Role:         domain.RoleUser,
				}
				m.On("GetByEmail", mock.Anything, "test@example.com").
					Return(user, nil)
			},
			expectedErr:    false,
			expectedUserID: userID,
		},
		{
			name:     "invalid credentials - wrong password",
			email:    "test@example.com",
			password: "wrongpassword",
			setupMock: func(m *mocks.MockUserRepository) {
				user := &domain.User{
					ID:           uuid.New(),
					Email:        "test@example.com",
					PasswordHash: string(hashedPassword),
					Role:         domain.RoleUser,
				}
				m.On("GetByEmail", mock.Anything, "test@example.com").
					Return(user, nil)
			},
			expectedErr: true,
		},
		{
			name:     "user not found",
			email:    "nonexistent@example.com",
			password: password,
			setupMock: func(m *mocks.MockUserRepository) {
				m.On("GetByEmail", mock.Anything, "nonexistent@example.com").
					Return(nil, nil)
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(mocks.MockUserRepository)
			tt.setupMock(mockUserRepo)

			authService := NewAuthService(mockUserRepo, "test-secret", 24*time.Hour)

			token, err := authService.Login(context.Background(), tt.email, tt.password)

			if tt.expectedErr {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)

				claims, err := authService.ValidateToken(token)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedUserID, claims.UserID)
				assert.Equal(t, domain.RoleUser, claims.Role)
			}

			mockUserRepo.AssertExpectations(t)
		})
	}
}

func TestAuthService_TokenValidation(t *testing.T) {
	mockUserRepo := new(mocks.MockUserRepository)
	authService := NewAuthService(mockUserRepo, "test-secret", 24*time.Hour)

	tests := []struct {
		name        string
		setupToken  func() string
		expectedErr bool
	}{
		{
			name: "valid token",
			setupToken: func() string {
				token, _ := authService.generateToken(uuid.New(), domain.RoleUser)
				return token
			},
			expectedErr: false,
		},
		{
			name: "invalid token",
			setupToken: func() string {
				return "invalid.token.string"
			},
			expectedErr: true,
		},
		{
			name: "expired token",
			setupToken: func() string {
				expiredAuth := NewAuthService(mockUserRepo, "test-secret", -1*time.Hour)
				token, _ := expiredAuth.generateToken(uuid.New(), domain.RoleUser)
				return token
			},
			expectedErr: true,
		},
		{
			name: "wrong signature",
			setupToken: func() string {
				wrongSecretAuth := NewAuthService(mockUserRepo, "wrong-secret", 24*time.Hour)
				token, _ := wrongSecretAuth.generateToken(uuid.New(), domain.RoleUser)
				return token
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := tt.setupToken()
			claims, err := authService.ValidateToken(token)

			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
				assert.NotEmpty(t, claims.UserID)
				assert.NotEmpty(t, claims.Role)
			}
		})
	}
}

func TestAuthService_ConcurrentTokenGeneration(t *testing.T) {
	mockUserRepo := new(mocks.MockUserRepository)
	authService := NewAuthService(mockUserRepo, "test-secret", 24*time.Hour)

	userID := uuid.New()
	role := domain.RoleUser

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			token, err := authService.generateToken(userID, role)
			assert.NoError(t, err)
			assert.NotEmpty(t, token)

			claims, err := authService.ValidateToken(token)
			assert.NoError(t, err)
			assert.Equal(t, userID, claims.UserID)
			assert.Equal(t, role, claims.Role)

			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
