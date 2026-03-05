package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type webhookPayload struct {
	TaskID    string `json:"taskId"`
	Title     string `json:"title"`
	Timestamp int64  `json:"timestamp"`
}

type WebhookHandler struct {
	store    *StateStore
	dedup    *DedupCache
	notifier Notifier
}

func NewWebhookHandler(store *StateStore, dedup *DedupCache, notifier Notifier) *WebhookHandler {
	return &WebhookHandler{
		store:    store,
		dedup:    dedup,
		notifier: notifier,
	}
}

func (wh *WebhookHandler) HandleStart(w http.ResponseWriter, r *http.Request) {
	// Acknowledge immediately
	w.WriteHeader(http.StatusOK)

	var payload webhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("webhook/start: invalid JSON: %v", err)
		return
	}

	if payload.TaskID == "" {
		log.Printf("webhook/start: missing taskId")
		return
	}

	if payload.Timestamp == 0 {
		payload.Timestamp = time.Now().UnixMilli()
	}

	key := DedupKey(payload.TaskID, payload.Timestamp)
	if wh.dedup.IsDuplicate(key) {
		log.Printf("webhook/start: dedup hit for %s", payload.TaskID)
		return
	}

	now := time.Now()
	wh.store.Update(func(s *State) {
		s.TrackingTaskID = payload.TaskID
		s.TaskTitle = payload.Title
		s.StartedAt = payload.Timestamp
		s.LastWebhookAt = now
		s.LiveActivityStartedAt = now
	})

	log.Printf("webhook/start: tracking %s (%s)", payload.TaskID, payload.Title)

	if wh.notifier == nil {
		return
	}

	state := wh.store.Get()
	if state.UpdateToken != "" {
		if err := wh.notifier.UpdateActivity(state.UpdateToken, payload.Title, payload.Timestamp); err != nil {
			log.Printf("webhook/start: update activity error: %v", err)
		}
	} else if state.PushToStartToken != "" {
		if err := wh.notifier.StartActivity(state.PushToStartToken, payload.Title, payload.Timestamp); err != nil {
			log.Printf("webhook/start: start activity error: %v", err)
		}
	} else {
		log.Printf("webhook/start: no push tokens available")
	}
}

func (wh *WebhookHandler) HandleStop(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	var payload webhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("webhook/stop: invalid JSON: %v", err)
		return
	}

	if payload.Timestamp == 0 {
		payload.Timestamp = time.Now().UnixMilli()
	}

	if payload.TaskID != "" {
		key := DedupKey(payload.TaskID, payload.Timestamp)
		if wh.dedup.IsDuplicate(key) {
			log.Printf("webhook/stop: dedup hit for %s", payload.TaskID)
			return
		}
	}

	state := wh.store.Get()
	updateToken := state.UpdateToken

	wh.store.Update(func(s *State) {
		s.TrackingTaskID = ""
		s.TaskTitle = ""
		s.StartedAt = 0
		s.LastWebhookAt = time.Now()
		s.LiveActivityStartedAt = time.Time{}
	})

	log.Printf("webhook/stop: stopped tracking")

	if wh.notifier != nil && updateToken != "" {
		if err := wh.notifier.EndActivity(updateToken); err != nil {
			log.Printf("webhook/stop: end activity error: %v", err)
		}
	}
}
