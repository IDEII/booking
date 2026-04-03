package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"booking-service/internal/domain"
	"booking-service/internal/middleware"
	"booking-service/internal/service"

	"github.com/google/uuid"
)

type TestResponse struct {
	StatusCode int
	Body       map[string]interface{}
}

func ExecuteRequest(handler http.HandlerFunc, method, url string, body interface{}, claims *service.Claims) *TestResponse {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}

	req := httptest.NewRequest(method, url, nil)
	if reqBody != nil {
		req = httptest.NewRequest(method, url, bytes.NewBuffer(reqBody))
	}

	if claims != nil {
		ctx := context.WithValue(req.Context(), middleware.ClaimsKey, claims)
		req = req.WithContext(ctx)
	}

	w := httptest.NewRecorder()
	handler(w, req)

	var responseBody map[string]interface{}
	json.NewDecoder(w.Body).Decode(&responseBody)

	return &TestResponse{
		StatusCode: w.Code,
		Body:       responseBody,
	}
}

func addClaimsToContext(ctx context.Context, userID uuid.UUID, role domain.UserRole) context.Context {
	claims := &service.Claims{
		UserID: userID,
		Role:   role,
	}
	return context.WithValue(ctx, middleware.ClaimsKey, claims)
}

func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}
