package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRedirect(t *testing.T) {
	tests := []struct {
		name           string
		destination    string
		statusCode     int
		requestPath    string
		requestQuery   string
		wantLocation   string
		wantStatusCode int
	}{
		{
			name:           "basic redirect 302",
			destination:    "https://example.com",
			statusCode:     302,
			requestPath:    "/foo/bar",
			wantLocation:   "https://example.com/foo/bar",
			wantStatusCode: 302,
		},
		{
			name:           "with query string",
			destination:    "https://example.com",
			statusCode:     301,
			requestPath:    "/foo/bar",
			requestQuery:   "q=hello&page=1",
			wantLocation:   "https://example.com/foo/bar?q=hello&page=1",
			wantStatusCode: 301,
		},
		{
			name:           "destination with path",
			destination:    "https://example.com/api",
			statusCode:     302,
			requestPath:    "/v1/users",
			wantLocation:   "https://example.com/api/v1/users",
			wantStatusCode: 302,
		},
		{
			name:           "destination with trailing slash and path",
			destination:    "https://example.com/api/",
			statusCode:     302,
			requestPath:    "/v1/users",
			wantLocation:   "https://example.com/api/v1/users",
			wantStatusCode: 302,
		},
		{
			name:           "root path",
			destination:    "https://example.com",
			statusCode:     307,
			requestPath:    "/",
			wantLocation:   "https://example.com/",
			wantStatusCode: 307,
		},
		{
			name:           "308 permanent redirect",
			destination:    "https://new.example.com",
			statusCode:     308,
			requestPath:    "/old-page",
			wantLocation:   "https://new.example.com/old-page",
			wantStatusCode: 308,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := newRedirectHandler(tt.destination, tt.statusCode)
			if err != nil {
				t.Fatalf("newRedirectHandler: %v", err)
			}

			target := tt.requestPath
			if tt.requestQuery != "" {
				target += "?" + tt.requestQuery
			}

			req := httptest.NewRequest(http.MethodGet, target, nil)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatusCode)
			}

			if loc := rec.Header().Get("Location"); loc != tt.wantLocation {
				t.Errorf("Location = %q, want %q", loc, tt.wantLocation)
			}
		})
	}
}

func TestHealth(t *testing.T) {
	h, err := newRedirectHandler("https://example.com", 302)
	if err != nil {
		t.Fatalf("newRedirectHandler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("health status = %d, want %d", rec.Code, http.StatusOK)
	}

	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Errorf("health body = %q, want 'status:ok'", rec.Body.String())
	}
}

func TestReady(t *testing.T) {
	h, err := newRedirectHandler("https://example.com", 302)
	if err != nil {
		t.Fatalf("newRedirectHandler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("ready status = %d, want %d", rec.Code, http.StatusOK)
	}

	if !strings.Contains(rec.Body.String(), `"status":"ready"`) {
		t.Errorf("ready body = %q, want 'status:ready'", rec.Body.String())
	}
}

func TestMetrics(t *testing.T) {
	h, err := newRedirectHandler("https://example.com", 302)
	if err != nil {
		t.Fatalf("newRedirectHandler: %v", err)
	}

	// First make a redirect request to populate metrics
	req := httptest.NewRequest(http.MethodGet, "/some-path", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	// Now check /metrics
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("metrics status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "redirect_requests_total") {
		t.Errorf("metrics should contain redirect_requests_total, got: %s", body)
	}
}

func TestConfig(t *testing.T) {
	t.Run("destination with trailing slash preserves path correctly", func(t *testing.T) {
		h, err := newRedirectHandler("https://example.com/app/", 302)
		if err != nil {
			t.Fatalf("newRedirectHandler: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/users", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		want := "https://example.com/app/users"
		if loc := rec.Header().Get("Location"); loc != want {
			t.Errorf("Location = %q, want %q", loc, want)
		}
	})
}
