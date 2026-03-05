package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMarvinClientGetTrackedItem(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/trackedItem" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-API-Token") != "test-token" {
			t.Errorf("missing or wrong X-API-Token header")
		}
		json.NewEncoder(w).Encode(TrackedItemResponse{
			TaskID:    "task-1",
			Title:     "Test Task",
			StartedAt: 1772734813781,
		})
	}))
	defer srv.Close()

	mc := &marvinClient{
		httpClient: srv.Client(),
		token:      "test-token",
		baseURL:    srv.URL,
	}

	item, err := mc.GetTrackedItem()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item == nil {
		t.Fatal("expected item, got nil")
	}
	if item.TaskID != "task-1" {
		t.Errorf("expected task-1, got %s", item.TaskID)
	}
	if item.Title != "Test Task" {
		t.Errorf("expected Test Task, got %s", item.Title)
	}
}

func TestMarvinClientGetTrackedItemEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	mc := &marvinClient{
		httpClient: srv.Client(),
		token:      "test-token",
		baseURL:    srv.URL,
	}

	item, err := mc.GetTrackedItem()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item != nil {
		t.Errorf("expected nil for no tracked item, got %+v", item)
	}
}

func TestMarvinClientTrack(t *testing.T) {
	var gotTaskID, gotAction string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/track" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var payload struct {
			TaskID string `json:"taskId"`
			Action string `json:"action"`
		}
		json.NewDecoder(r.Body).Decode(&payload)
		gotTaskID = payload.TaskID
		gotAction = payload.Action
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	mc := &marvinClient{
		httpClient: srv.Client(),
		token:      "test-token",
		baseURL:    srv.URL,
	}

	if err := mc.Track("task-1", "START"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotTaskID != "task-1" || gotAction != "START" {
		t.Errorf("expected task-1/START, got %s/%s", gotTaskID, gotAction)
	}
}

func TestMarvinClientGetTrackedItemError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer srv.Close()

	mc := &marvinClient{
		httpClient: srv.Client(),
		token:      "test-token",
		baseURL:    srv.URL,
	}

	_, err := mc.GetTrackedItem()
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}
