package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
