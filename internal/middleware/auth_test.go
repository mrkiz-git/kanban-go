package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mrkiz-git/kanba-go/internal/auth"
	"github.com/mrkiz-git/kanba-go/internal/middleware"
)

func TestAuthRejectsMalformedAuthorizationHeader(t *testing.T) {
	tokens := auth.NewTokenService([]byte("test-secret-at-least-32-bytes-long"))
	handler := middleware.Auth(middleware.AuthConfig{
		Tokens: tokens,
		Users:  nil,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/boards", nil)
	req.Header.Set("Authorization", "Token not-a-bearer-token")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}
