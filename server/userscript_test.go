package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUserscriptHandler_Headers(t *testing.T) {
	handler := userscriptHandler("")

	req := httptest.NewRequest(http.MethodGet, "/userscript/marvin-relay-tracker.user.js", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/javascript" {
		t.Errorf("expected Content-Type text/javascript, got %s", ct)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("expected Cache-Control no-cache, got %s", cc)
	}
	if xcto := w.Header().Get("X-Content-Type-Options"); xcto != "nosniff" {
		t.Errorf("expected X-Content-Type-Options nosniff, got %s", xcto)
	}
}

func TestUserscriptHandler_ContainsMetadata(t *testing.T) {
	handler := userscriptHandler("")

	req := httptest.NewRequest(http.MethodGet, "/userscript/marvin-relay-tracker.user.js", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "@name") {
		t.Error("response should contain @name metadata")
	}
}

func TestUserscriptHandler_ReplacesPlaceholder(t *testing.T) {
	handler := userscriptHandler("https://relay.example.com")

	req := httptest.NewRequest(http.MethodGet, "/userscript/marvin-relay-tracker.user.js", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()
	if strings.Contains(body, "__RELAY_URL__") {
		t.Error("placeholder __RELAY_URL__ should be replaced when ExternalURL is set")
	}
	if !strings.Contains(body, "https://relay.example.com/userscript/marvin-relay-tracker.user.js") {
		t.Error("response should contain rewritten URL")
	}
}

func TestUserscriptHandler_PlaceholderKeptWhenEmpty(t *testing.T) {
	handler := userscriptHandler("")

	req := httptest.NewRequest(http.MethodGet, "/userscript/marvin-relay-tracker.user.js", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "__RELAY_URL__") {
		t.Error("placeholder __RELAY_URL__ should be kept when ExternalURL is empty")
	}
}

func TestUserscriptHandler_TrailingSlashStripped(t *testing.T) {
	handler := userscriptHandler("https://relay.example.com/")

	req := httptest.NewRequest(http.MethodGet, "/userscript/marvin-relay-tracker.user.js", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()
	if strings.Contains(body, "https://relay.example.com//") {
		t.Error("trailing slash should be stripped to avoid double-slash in URLs")
	}
	if !strings.Contains(body, "https://relay.example.com/userscript/marvin-relay-tracker.user.js") {
		t.Error("response should contain correctly rewritten URL without double slash")
	}
}
