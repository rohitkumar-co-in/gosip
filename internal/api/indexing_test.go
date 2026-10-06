package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rohitkumar-co-in/leadomi-sip/internal/config"
)

func TestInternalSiteNoIndex(t *testing.T) {
	setup := setupTestAPI(t)
	router := NewRouter(&Dependencies{Config: &config.Config{}, DB: setup.DB})
	for _, path := range []string{"/", "/login", "/devices", "/assets/missing.js", "/api/health", "/api/messages", "/robots.txt"} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Header().Get("X-Robots-Tag") != "noindex, nofollow" {
				t.Fatalf("missing noindex on %s response (status %d)", path, response.Code)
			}
			if path == "/api/messages" && response.Code != http.StatusUnauthorized {
				t.Fatalf("protected API status = %d", response.Code)
			}
			if path == "/robots.txt" {
				if response.Code != http.StatusOK || !strings.HasPrefix(response.Header().Get("Content-Type"), "text/plain") {
					t.Fatal("robots.txt must return plain text, not the SPA fallback")
				}
				if !strings.Contains(response.Body.String(), "User-agent: *\nDisallow:\n") {
					t.Fatal("robots policy must allow crawlers to discover noindex headers")
				}
			}
		})
	}
}
