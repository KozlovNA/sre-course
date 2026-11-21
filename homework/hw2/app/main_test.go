package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewGreeting(t *testing.T) {
	handler := newHandler("test", "test-pod")
	const oldMessage = "hello version=test pod=test-pod\n"
	const newMessage = "hello from the new greeting version=test pod=test-pod\n"

	checkResponse(t, handler, "/feature", http.StatusOK, "new-greeting=false\n")
	checkResponse(t, handler, "/message", http.StatusOK, oldMessage)
	checkResponse(t, handler, "/admin/feature?enabled=true", http.StatusOK, "new-greeting=true\n")
	checkResponse(t, handler, "/feature", http.StatusOK, "new-greeting=true\n")
	checkResponse(t, handler, "/message", http.StatusOK, newMessage)
	checkResponse(t, handler, "/admin/feature?enabled=false", http.StatusOK, "new-greeting=false\n")
	checkResponse(t, handler, "/feature", http.StatusOK, "new-greeting=false\n")
	checkResponse(t, handler, "/message", http.StatusOK, oldMessage)
}

func TestInvalidFeatureValue(t *testing.T) {
	handler := newHandler("test", "test-pod")
	checkResponse(t, handler, "/admin/feature?enabled=true", http.StatusOK, "new-greeting=true\n")
	for _, path := range []string{
		"/admin/feature",
		"/admin/feature?enabled=0",
		"/admin/feature?enabled=false&enabled=true",
	} {
		checkResponse(t, handler, path, http.StatusBadRequest, "use enabled=true or enabled=false\n")
		checkResponse(t, handler, "/feature", http.StatusOK, "new-greeting=true\n")
	}
}

func checkResponse(t *testing.T, handler http.Handler, path string, status int, body string) {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	if response.Code != status || response.Body.String() != body {
		t.Fatalf("GET %s = %d %q, want %d %q", path, response.Code, response.Body.String(), status, body)
	}
}
