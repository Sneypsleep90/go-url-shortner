package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sneypsleep90/go-url-shortner/internal/domain"
	"github.com/Sneypsleep90/go-url-shortner/internal/handler"
	"github.com/Sneypsleep90/go-url-shortner/internal/service"
	"github.com/Sneypsleep90/go-url-shortner/internal/storage/memory"
	"github.com/gin-gonic/gin"
)

func TestCreateAndGetLink(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
	}{
		{name: "https", url: "https://example.com"},
		{name: "http", url: "http://example.org/articles/42"},
		{name: "query and fragment", url: "https://example.com:8443/search?q=go&lang=en#results"},
		{name: "encoded space", url: "https://example.com/a%20b"},
		{name: "localhost", url: "http://localhost:8080/path"},
		{name: "IPv4", url: "http://127.0.0.1:8080/path"},
		{name: "IPv6", url: "https://[::1]:8443/path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := newTestRouter()
			request := httptest.NewRequest(
				http.MethodPost,
				"/links",
				strings.NewReader(`{"original_url":"`+tt.url+`"}`),
			)
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusCreated {
				t.Fatalf("POST status = %d, body = %s", response.Code, response.Body.String())
			}
			created := responseJSON(t, response)
			if created["short_code"] == "" || created["original_url"] != tt.url {
				t.Fatalf("unexpected created link: %v", created)
			}

			request = httptest.NewRequest(http.MethodGet, "/"+created["short_code"], nil)
			response = httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("GET status = %d, want 200", response.Code)
			}
			found := responseJSON(t, response)
			if found["short_code"] != created["short_code"] || found["original_url"] != tt.url {
				t.Fatalf("unexpected retrieved link: %v", found)
			}
		})
	}
}

func TestCreateInvalidURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		wantError string
	}{
		{
			name:      "empty body",
			body:      "",
			wantError: "request body is empty",
		},
		{
			name:      "invalid JSON",
			body:      `{"original_url":`,
			wantError: "request body must be valid JSON",
		},
		{
			name:      "array instead of object",
			body:      `[]`,
			wantError: "JSON object",
		},
		{
			name:      "null URL",
			body:      `{"original_url":null}`,
			wantError: "original_url is required",
		},
		{
			name:      "body too large",
			body:      `{"original_url":"https://example.com/` + strings.Repeat("a", 1<<20) + `"}`,
			wantError: "request body exceeds",
		},
		{
			name:      "missing URL",
			body:      `{}`,
			wantError: "original_url is required",
		},
		{
			name:      "empty URL",
			body:      `{"original_url":""}`,
			wantError: "original_url is required",
		},
		{
			name:      "invalid URL",
			body:      `{"original_url":"not-a-url"}`,
			wantError: "HTTP or HTTPS URL",
		},
		{
			name:      "HTTPS without hostname",
			body:      `{"original_url":"https://:443/path"}`,
			wantError: "HTTP or HTTPS URL",
		},
		{
			name:      "HTTP without hostname",
			body:      `{"original_url":"http://:80/path"}`,
			wantError: "HTTP or HTTPS URL",
		},
		{
			name:      "space in path",
			body:      `{"original_url":"https://example.com/a b"}`,
			wantError: "original_url must not contain spaces",
		},
		{
			name:      "leading space",
			body:      `{"original_url":" https://example.com"}`,
			wantError: "original_url must not contain spaces",
		},
		{
			name:      "trailing space",
			body:      `{"original_url":"https://example.com "}`,
			wantError: "original_url must not contain spaces",
		},
		{
			name:      "space in host",
			body:      `{"original_url":"https://exa mple.com/path"}`,
			wantError: "original_url must not contain spaces",
		},
		{
			name:      "space in query",
			body:      `{"original_url":"https://example.com/?q=hello world"}`,
			wantError: "original_url must not contain spaces",
		},
		{
			name:      "unsupported scheme",
			body:      `{"original_url":"ftp://example.com/file"}`,
			wantError: "HTTP or HTTPS URL",
		},
		{
			name:      "number instead of URL",
			body:      `{"original_url":123}`,
			wantError: "original_url must be string",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := newTestRouter()
			request := httptest.NewRequest(http.MethodPost, "/links", strings.NewReader(tt.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", response.Code)
			}
			body := responseJSON(t, response)
			if !strings.Contains(body["error"], tt.wantError) {
				t.Fatalf("error = %q, want %q", body["error"], tt.wantError)
			}
		})
	}
}

func TestGetMissingLink(t *testing.T) {
	t.Parallel()

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/missing", nil)
	newTestRouter().ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", response.Code)
	}
	if body := responseJSON(t, response); body["error"] != "link not found" {
		t.Fatalf("unexpected error: %v", body)
	}
}

func TestInternalServerError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "create", method: http.MethodPost, path: "/links", body: `{"original_url":"https://example.com"}`},
		{name: "get", method: http.MethodGet, path: "/abcdefghij"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := gin.New()
			handler.NewHandler(failingService{}).RegisterRoutes(router)
			request := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500", response.Code)
			}
			if body := responseJSON(t, response); body["error"] != "internal server error" {
				t.Fatalf("unexpected error: %v", body)
			}
		})
	}
}

type failingService struct{}

func (failingService) CreateCode(context.Context, domain.Link) (domain.Link, error) {
	return domain.Link{}, errors.New("database unavailable")
}

func (failingService) GetURLByCode(context.Context, string) (domain.Link, error) {
	return domain.Link{}, errors.New("database unavailable")
}

func newTestRouter() *gin.Engine {
	repo := memory.New()
	svc := service.NewService(repo)
	h := handler.NewHandler(svc)
	router := gin.New()
	h.RegisterRoutes(router)
	return router
}

func responseJSON(t *testing.T, response *httptest.ResponseRecorder) map[string]string {
	t.Helper()

	body := make(map[string]string)
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}
