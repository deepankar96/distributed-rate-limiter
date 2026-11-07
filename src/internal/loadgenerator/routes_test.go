package loadgenerator_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deepankar96/distributed-rate-limiter/internal/loadgenerator"
)

func TestRoutes(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{"health", http.MethodGet, "/healthz", http.StatusOK},
		{"head health", http.MethodHead, "/healthz", http.StatusOK},
		{"unknown path", http.MethodGet, "/unknown", http.StatusNotFound},
		{"unsupported health method", http.MethodPost, "/healthz", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := loadgenerator.NewHandler()
			request := httptest.NewRequest(tt.method, tt.path, nil)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != tt.want {
				t.Fatalf("%s %s: status = %d, want %d", tt.method, tt.path, response.Code, tt.want)
			}
		})
	}
}
