package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInfoHandler_Info(t *testing.T) {
	handler := NewInfoHandler()

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "GET request",
			method:         http.MethodGet,
			path:           "/_info",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "POST request",
			method:         http.MethodPost,
			path:           "/_info",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "PUT request",
			method:         http.MethodPut,
			path:           "/_info",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "DELETE request",
			method:         http.MethodDelete,
			path:           "/_info",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "PATCH request",
			method:         http.MethodPatch,
			path:           "/_info",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "OPTIONS request",
			method:         http.MethodOptions,
			path:           "/_info",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "with query parameters",
			method:         http.MethodGet,
			path:           "/_info?param=value&another=test",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			handler.Info(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.expectedBody, w.Body.String())
		})
	}
}

func TestInfoHandler_Info_AlwaysReturnsOK(t *testing.T) {
	handler := NewInfoHandler()

	req := httptest.NewRequest(http.MethodGet, "/_info", nil)
	req.Header.Set("X-Custom-Header", "test")
	req.Header.Set("Authorization", "Bearer token")

	w := httptest.NewRecorder()
	handler.Info(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "OK", w.Body.String())

	contentType := w.Header().Get("Content-Type")
	assert.Empty(t, contentType, "Content-Type header should not be set")
}
