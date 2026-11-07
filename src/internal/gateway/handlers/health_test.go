package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deepankar96/distributed-rate-limiter/internal/gateway/handlers"
)

func TestHealth(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	handlers.Health(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}
