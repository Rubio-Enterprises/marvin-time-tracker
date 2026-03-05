package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type TrackHandler struct {
	store    *StateStore
	marvin   MarvinAPIClient
	notifier Notifier
}

func NewTrackHandler(store *StateStore, marvin MarvinAPIClient, notifier Notifier) *TrackHandler {
	return &TrackHandler{
		store:    store,
		marvin:   marvin,
		notifier: notifier,
	}
}

type startRequest struct {
	TaskID string `json:"taskId"`
	Title  string `json:"title"`
}

type stopRequest struct {
	TaskID string `json:"taskId,omitempty"`
}

func (th *TrackHandler) HandleStart(w http.ResponseWriter, r *http.Request) {
	var req startRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.TaskID == "" {
		http.Error(w, `{"error":"taskId required"}`, http.StatusBadRequest)
		return
	}

	if err := th.marvin.Track(req.TaskID, "START"); err != nil {
		log.Printf("track/start: marvin error: %v", err)
		http.Error(w, `{"error":"failed to start tracking"}`, http.StatusBadGateway)
		return
	}

	now := time.Now()
	startedAt := now.UnixMilli()

	th.store.Update(func(s *State) {
		s.TrackingTaskID = req.TaskID
		s.TaskTitle = req.Title
		s.StartedAt = startedAt
		s.LiveActivityStartedAt = now
	})

	log.Printf("track/start: started %s (%s)", req.TaskID, req.Title)

	if th.notifier != nil {
		state := th.store.Get()
		if state.UpdateToken != "" {
			th.notifier.UpdateActivity(state.UpdateToken, req.Title, startedAt)
		} else if state.PushToStartToken != "" {
			th.notifier.StartActivity(state.PushToStartToken, req.Title, startedAt)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (th *TrackHandler) HandleStop(w http.ResponseWriter, r *http.Request) {
	var req stopRequest
	// Body is optional for stop
	json.NewDecoder(r.Body).Decode(&req)

	state := th.store.Get()
	taskID := req.TaskID
	if taskID == "" {
		taskID = state.TrackingTaskID
	}

	if taskID == "" {
		http.Error(w, `{"error":"no task to stop"}`, http.StatusBadRequest)
		return
	}

	if err := th.marvin.Track(taskID, "STOP"); err != nil {
		log.Printf("track/stop: marvin error: %v", err)
		http.Error(w, `{"error":"failed to stop tracking"}`, http.StatusBadGateway)
		return
	}

	updateToken := state.UpdateToken
	th.store.Update(func(s *State) {
		s.TrackingTaskID = ""
		s.TaskTitle = ""
		s.StartedAt = 0
		s.LiveActivityStartedAt = time.Time{}
	})

	log.Printf("track/stop: stopped %s", taskID)

	if th.notifier != nil && updateToken != "" {
		th.notifier.EndActivity(updateToken)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
