package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"goserver/internal/web"
)

func TestHandlerDelegatesPlatformAPI(t *testing.T) {
	frontend := web.NewFrontend(t.TempDir())
	handler := NewHandler(
		frontend,
		nil,
		nil,
		nil,
		http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusNoContent)
		}),
		nil,
	)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/config/current", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected platform API delegation status %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

func TestHandlerServesPlatformRoutesFromFrontend(t *testing.T) {
	frontend := web.NewFrontend(t.TempDir())
	handler := NewHandler(frontend, nil, nil, nil, nil, nil)

	request := httptest.NewRequest(http.MethodGet, "/platform/companies", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected platform route to serve index with %d, got %d", http.StatusOK, recorder.Code)
	}
}
