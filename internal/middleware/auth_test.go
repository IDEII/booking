package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"booking-service/internal/domain"
	"booking-service/internal/service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockAuthService struct {
	validateTokenFunc func(token string) (*service.Claims, error)
}

func (m *MockAuthService) ValidateToken(token string) (*service.Claims, error) {
	if m.validateTokenFunc != nil {
		return m.validateTokenFunc(token)
	}
	return nil, nil
}

func (m *MockAuthService) DummyLogin(ctx context.Context, role domain.UserRole) (string, error) {
	return "", nil
}

func (m *MockAuthService) Register(ctx context.Context, email, password string, role domain.UserRole) (*domain.User, error) {
	return nil, nil
}

func (m *MockAuthService) Login(ctx context.Context, email, password string) (string, error) {
	return "", nil
}

func (m *MockAuthService) generateToken(userID uuid.UUID, role domain.UserRole) (string, error) {
	return "", nil
}

type RealAuthServiceAdapter struct {
	authService *service.AuthService
}

func (r *RealAuthServiceAdapter) ValidateToken(token string) (*service.Claims, error) {
	return r.authService.ValidateToken(token)
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r.Context())
	if claims == nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("no claims"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func testRoleHandler(allowedRoles ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r.Context())
		if claims == nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("no claims"))
			return
		}

		for _, role := range allowedRoles {
			if string(claims.Role) == role {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("ok"))
				return
			}
		}
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("forbidden"))
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	userRepo := &MockUserRepository{}
	authService := service.NewAuthService(userRepo, "test-secret", 24*time.Hour)

	token, err := authService.DummyLogin(context.Background(), domain.RoleUser)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	middleware := AuthMiddleware(authService)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()

	handler := middleware(http.HandlerFunc(testHandler))
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Body.String())
}

func TestAuthMiddleware_ValidTokenWithAdminRole(t *testing.T) {
	userRepo := &MockUserRepository{}
	authService := service.NewAuthService(userRepo, "test-secret", 24*time.Hour)

	token, err := authService.DummyLogin(context.Background(), domain.RoleAdmin)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	middleware := AuthMiddleware(authService)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r.Context())
		assert.NotNil(t, claims)
		assert.Equal(t, domain.RoleAdmin, claims.Role)
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_MissingAuthorizationHeader(t *testing.T) {
	userRepo := &MockUserRepository{}
	authService := service.NewAuthService(userRepo, "test-secret", 24*time.Hour)

	middleware := AuthMiddleware(authService)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	w := httptest.NewRecorder()

	handler := middleware(http.HandlerFunc(testHandler))
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var errorResponse map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&errorResponse)
	assert.NoError(t, err)

	errorObj := errorResponse["error"].(map[string]interface{})
	assert.Equal(t, "UNAUTHORIZED", errorObj["code"])
	assert.Equal(t, "missing authorization header", errorObj["message"])
}

func TestAuthMiddleware_InvalidAuthorizationHeaderFormat(t *testing.T) {
	userRepo := &MockUserRepository{}
	authService := service.NewAuthService(userRepo, "test-secret", 24*time.Hour)

	middleware := AuthMiddleware(authService)

	tests := []struct {
		name        string
		authHeader  string
		expectedMsg string
	}{
		{
			name:        "missing Bearer prefix",
			authHeader:  "token123",
			expectedMsg: "invalid authorization header format",
		},
		{
			name:        "wrong prefix",
			authHeader:  "Basic token123",
			expectedMsg: "invalid authorization header format",
		},
		{
			name:        "empty token",
			authHeader:  "Bearer ",
			expectedMsg: "invalid token",
		},
		{
			name:        "too many parts",
			authHeader:  "Bearer token123 extra",
			expectedMsg: "invalid authorization header format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", tt.authHeader)

			w := httptest.NewRecorder()

			handler := middleware(http.HandlerFunc(testHandler))
			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)

			var errorResponse map[string]interface{}
			err := json.NewDecoder(w.Body).Decode(&errorResponse)
			assert.NoError(t, err)

			errorObj := errorResponse["error"].(map[string]interface{})
			assert.Equal(t, "UNAUTHORIZED", errorObj["code"])
			assert.Equal(t, tt.expectedMsg, errorObj["message"])
		})
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	userRepo := &MockUserRepository{}
	authService := service.NewAuthService(userRepo, "test-secret", 24*time.Hour)

	middleware := AuthMiddleware(authService)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.string")

	w := httptest.NewRecorder()

	handler := middleware(http.HandlerFunc(testHandler))
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var errorResponse map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&errorResponse)
	assert.NoError(t, err)

	errorObj := errorResponse["error"].(map[string]interface{})
	assert.Equal(t, "UNAUTHORIZED", errorObj["code"])
	assert.Equal(t, "invalid token", errorObj["message"])
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	userRepo := &MockUserRepository{}
	expiredAuthService := service.NewAuthService(userRepo, "test-secret", -1*time.Hour)

	token, err := expiredAuthService.DummyLogin(context.Background(), domain.RoleUser)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	authService := service.NewAuthService(userRepo, "test-secret", 24*time.Hour)
	middleware := AuthMiddleware(authService)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()

	handler := middleware(http.HandlerFunc(testHandler))
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_WrongSignature(t *testing.T) {
	userRepo := &MockUserRepository{}
	signerAuthService := service.NewAuthService(userRepo, "signer-secret", 24*time.Hour)

	token, err := signerAuthService.DummyLogin(context.Background(), domain.RoleUser)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	validatorAuthService := service.NewAuthService(userRepo, "different-secret", 24*time.Hour)
	middleware := AuthMiddleware(validatorAuthService)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()

	handler := middleware(http.HandlerFunc(testHandler))
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_ContextPropagation(t *testing.T) {
	userRepo := &MockUserRepository{}
	authService := service.NewAuthService(userRepo, "test-secret", 24*time.Hour)

	token, err := authService.DummyLogin(context.Background(), domain.RoleUser)
	require.NoError(t, err)

	middleware := AuthMiddleware(authService)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r.Context())
		assert.NotNil(t, claims)
		assert.NotEmpty(t, claims.UserID)
		assert.Equal(t, domain.RoleUser, claims.Role)
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRoleMiddleware_AllowedRole(t *testing.T) {
	claims := &service.Claims{
		UserID: uuid.New(),
		Role:   domain.RoleAdmin,
	}
	ctx := context.WithValue(context.Background(), ClaimsKey, claims)

	req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	middleware := RoleMiddleware("admin")
	handler := middleware(http.HandlerFunc(testHandler))
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRoleMiddleware_AllowedRoleMultiple(t *testing.T) {
	claims := &service.Claims{
		UserID: uuid.New(),
		Role:   domain.RoleUser,
	}
	ctx := context.WithValue(context.Background(), ClaimsKey, claims)

	req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	middleware := RoleMiddleware("admin", "user", "moderator")
	handler := middleware(http.HandlerFunc(testHandler))
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRoleMiddleware_ForbiddenRole(t *testing.T) {
	claims := &service.Claims{
		UserID: uuid.New(),
		Role:   domain.RoleUser,
	}
	ctx := context.WithValue(context.Background(), ClaimsKey, claims)

	req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	middleware := RoleMiddleware("admin")
	handler := middleware(http.HandlerFunc(testHandler))
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var errorResponse map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&errorResponse)
	assert.NoError(t, err)

	errorObj := errorResponse["error"].(map[string]interface{})
	assert.Equal(t, "FORBIDDEN", errorObj["code"])
	assert.Equal(t, "access denied", errorObj["message"])
}

func TestRoleMiddleware_MissingClaims(t *testing.T) {
	ctx := context.Background()
	req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	middleware := RoleMiddleware("admin")
	handler := middleware(http.HandlerFunc(testHandler))
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var errorResponse map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&errorResponse)
	assert.NoError(t, err)

	errorObj := errorResponse["error"].(map[string]interface{})
	assert.Equal(t, "UNAUTHORIZED", errorObj["code"])
	assert.Equal(t, "unauthorized", errorObj["message"])
}

func TestRoleMiddleware_EmptyAllowedRoles(t *testing.T) {
	claims := &service.Claims{
		UserID: uuid.New(),
		Role:   domain.RoleUser,
	}
	ctx := context.WithValue(context.Background(), ClaimsKey, claims)

	req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	middleware := RoleMiddleware()
	handler := middleware(http.HandlerFunc(testHandler))
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRoleMiddleware_ChainedWithAuthMiddleware(t *testing.T) {
	userRepo := &MockUserRepository{}
	authService := service.NewAuthService(userRepo, "test-secret", 24*time.Hour)

	token, err := authService.DummyLogin(context.Background(), domain.RoleUser)
	require.NoError(t, err)

	authMiddleware := AuthMiddleware(authService)
	roleMiddleware := RoleMiddleware("user")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()

	handler := authMiddleware(roleMiddleware(http.HandlerFunc(testHandler)))
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRoleMiddleware_ChainedWithAuthMiddlewareWrongRole(t *testing.T) {
	userRepo := &MockUserRepository{}
	authService := service.NewAuthService(userRepo, "test-secret", 24*time.Hour)

	token, err := authService.DummyLogin(context.Background(), domain.RoleUser)
	require.NoError(t, err)

	authMiddleware := AuthMiddleware(authService)
	roleMiddleware := RoleMiddleware("admin")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()

	handler := authMiddleware(roleMiddleware(http.HandlerFunc(testHandler)))
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetClaims_WithValidClaims(t *testing.T) {
	expectedClaims := &service.Claims{
		UserID: uuid.New(),
		Role:   domain.RoleAdmin,
	}
	ctx := context.WithValue(context.Background(), ClaimsKey, expectedClaims)

	claims := GetClaims(ctx)
	assert.NotNil(t, claims)
	assert.Equal(t, expectedClaims.UserID, claims.UserID)
	assert.Equal(t, expectedClaims.Role, claims.Role)
}

func TestGetClaims_WithNilClaims(t *testing.T) {
	ctx := context.Background()
	claims := GetClaims(ctx)
	assert.Nil(t, claims)
}

func TestGetClaims_WithWrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), ClaimsKey, "not a claims object")

	claims := GetClaims(ctx)
	assert.Nil(t, claims)
}

func TestGetClaims_WithNilContext(t *testing.T) {
	claims := GetClaims(nil)
	assert.Nil(t, claims)
}

func TestAuthMiddleware_Integration_WithRealService(t *testing.T) {
	userRepo := &MockUserRepository{}
	authService := service.NewAuthService(userRepo, "integration-secret", 24*time.Hour)

	token, err := authService.DummyLogin(context.Background(), domain.RoleUser)
	require.NoError(t, err)

	middleware := AuthMiddleware(authService)

	mux := http.NewServeMux()
	mux.Handle("/protected", middleware(http.HandlerFunc(testHandler)))

	server := httptest.NewServer(mux)
	defer server.Close()

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, server.URL+"/protected", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRoleMiddleware_Integration_WithRealService(t *testing.T) {
	userRepo := &MockUserRepository{}
	authService := service.NewAuthService(userRepo, "integration-secret", 24*time.Hour)

	token, err := authService.DummyLogin(context.Background(), domain.RoleUser)
	require.NoError(t, err)

	authMiddleware := AuthMiddleware(authService)
	roleMiddleware := RoleMiddleware("user")

	mux := http.NewServeMux()
	mux.Handle("/user-only", authMiddleware(roleMiddleware(http.HandlerFunc(testHandler))))

	server := httptest.NewServer(mux)
	defer server.Close()

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, server.URL+"/user-only", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func BenchmarkAuthMiddleware_ValidToken(b *testing.B) {
	userRepo := &MockUserRepository{}
	authService := service.NewAuthService(userRepo, "bench-secret", 24*time.Hour)
	token, _ := authService.DummyLogin(context.Background(), domain.RoleUser)

	middleware := AuthMiddleware(authService)
	handler := middleware(http.HandlerFunc(testHandler))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkRoleMiddleware_Allowed(b *testing.B) {
	claims := &service.Claims{
		UserID: uuid.New(),
		Role:   domain.RoleAdmin,
	}
	ctx := context.WithValue(context.Background(), ClaimsKey, claims)

	middleware := RoleMiddleware("admin")
	handler := middleware(http.HandlerFunc(testHandler))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

type MockUserRepository struct{}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	return nil
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return nil, nil
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return nil, nil
}
