package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	store := NewStateStore(tempStateFile(t))
	dedup := NewDedupCache(60 * time.Second)
	return NewServer(store, dedup, nil)
}

func TestServerStatus(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %v", resp["status"])
	}
	if resp["tracking"] != false {
		t.Errorf("expected tracking false, got %v", resp["tracking"])
	}
}

func TestServerWebhookLifecycle(t *testing.T) {
	srv := newTestServer(t)

	// Start tracking
	body, _ := json.Marshal(webhookPayload{
		TaskID:    "task-1",
		Title:     "Test Task",
		Timestamp: 1772734813781,
	})
	req := httptest.NewRequest(http.MethodPost, "/webhook/start", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("start: expected 200, got %d", w.Code)
	}

	// Check status
	req = httptest.NewRequest(http.MethodGet, "/status", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["tracking"] != true {
		t.Errorf("expected tracking true after start")
	}
	if resp["taskId"] != "task-1" {
		t.Errorf("expected taskId task-1, got %v", resp["taskId"])
	}

	// Stop tracking
	stopBody, _ := json.Marshal(webhookPayload{
		TaskID:    "task-1",
		Timestamp: 1772734823781,
	})
	req = httptest.NewRequest(http.MethodPost, "/webhook/stop", bytes.NewReader(stopBody))
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Check status again
	req = httptest.NewRequest(http.MethodGet, "/status", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	json.NewDecoder(w.Body).Decode(&resp)
	if resp["tracking"] != false {
		t.Errorf("expected tracking false after stop")
	}
}

func TestServerCORSHeaders(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodOptions, "/webhook/start", nil)
	req.Header.Set("Origin", "https://app.amazingmarvin.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("CORS preflight: expected 200, got %d", w.Code)
	}

	acao := w.Header().Get("Access-Control-Allow-Origin")
	if acao != "*" {
		t.Errorf("expected ACAO *, got %s", acao)
	}
}
