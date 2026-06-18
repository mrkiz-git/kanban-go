package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatic(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "index.html"), "<html>home</html>")
	writeFile(t, filepath.Join(root, "login.html"), "<html>login</html>")
	writeFile(t, filepath.Join(root, "_next", "static", "app.js"), "console.log('app')")

	handler := Static(root)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "root",
			path:       "/",
			wantStatus: http.StatusOK,
			wantBody:   "home",
		},
		{
			name:       "html route",
			path:       "/login",
			wantStatus: http.StatusOK,
			wantBody:   "login",
		},
		{
			name:       "asset",
			path:       "/_next/static/app.js",
			wantStatus: http.StatusOK,
			wantBody:   "console.log",
		},
		{
			name:       "spa fallback",
			path:       "/boards/demo",
			wantStatus: http.StatusOK,
			wantBody:   "home",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Fatalf("body = %q, want substring %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestStaticStatError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permission checks")
	}

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "index.html"), "<html>home</html>")
	if err := os.Chmod(root, 0o000); err != nil {
		t.Fatalf("chmod root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(root, 0o755)
	})

	handler := Static(root)
	req := httptest.NewRequest(http.MethodGet, "/boards/demo", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestStaticMissingIndex(t *testing.T) {
	root := t.TempDir()
	handler := Static(root)

	req := httptest.NewRequest(http.MethodGet, "/boards/demo", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
