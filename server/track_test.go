package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTrackHandlerStart(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.PushToStartToken = "pts-token"
	})
	mc := &mockMarvinClient{}
	notifier := &mockNotifier{}
	th := NewTrackHandler(store, mc, notifier)

	body, _ := json.Marshal(startRequest{TaskID: "task-1", Title: "Test Task"})
	req := httptest.NewRequest(http.MethodPost, "/start", bytes.NewReader(body))
	w := httptest.NewRecorder()
	th.HandleStart(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if len(mc.trackCalls) != 1 || mc.trackCalls[0].Action != "START" {
		t.Errorf("expected 1 START call, got %+v", mc.trackCalls)
	}

	state := store.Get()
	if !state.IsTracking() {
		t.Error("expected tracking after start")
	}
	if notifier.startCalls != 1 {
		t.Errorf("expected 1 start notification, got %d", notifier.startCalls)
	}
}

func TestTrackHandlerStop(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.TrackingTaskID = "task-1"
		s.TaskTitle = "Running"
		s.StartedAt = 12345
		s.UpdateToken = "upd-token"
	})
	mc := &mockMarvinClient{}
	notifier := &mockNotifier{}
	th := NewTrackHandler(store, mc, notifier)

	body, _ := json.Marshal(stopRequest{})
	req := httptest.NewRequest(http.MethodPost, "/stop", bytes.NewReader(body))
	w := httptest.NewRecorder()
	th.HandleStop(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if len(mc.trackCalls) != 1 || mc.trackCalls[0].Action != "STOP" {
		t.Errorf("expected 1 STOP call, got %+v", mc.trackCalls)
	}

	state := store.Get()
	if state.IsTracking() {
		t.Error("expected not tracking after stop")
	}
	if notifier.endCalls != 1 {
		t.Errorf("expected 1 end notification, got %d", notifier.endCalls)
	}
}

func TestTrackHandlerStopWithTaskID(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.TrackingTaskID = "task-1"
	})
	mc := &mockMarvinClient{}
	th := NewTrackHandler(store, mc, nil)

	body, _ := json.Marshal(stopRequest{TaskID: "task-1"})
	req := httptest.NewRequest(http.MethodPost, "/stop", bytes.NewReader(body))
	w := httptest.NewRecorder()
	th.HandleStop(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if mc.trackCalls[0].TaskID != "task-1" {
		t.Errorf("expected task-1, got %s", mc.trackCalls[0].TaskID)
	}
}

func TestTrackHandlerStopNoTask(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	mc := &mockMarvinClient{}
	th := NewTrackHandler(store, mc, nil)

	body, _ := json.Marshal(stopRequest{})
	req := httptest.NewRequest(http.MethodPost, "/stop", bytes.NewReader(body))
	w := httptest.NewRecorder()
	th.HandleStop(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTrackHandlerStartMissingTaskID(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	mc := &mockMarvinClient{}
	th := NewTrackHandler(store, mc, nil)

	body, _ := json.Marshal(startRequest{Title: "No ID"})
	req := httptest.NewRequest(http.MethodPost, "/start", bytes.NewReader(body))
	w := httptest.NewRecorder()
	th.HandleStart(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
