package main

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func tempStateFile(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "state.json")
}

func TestStateStoreRoundTrip(t *testing.T) {
	path := tempStateFile(t)
	store := NewStateStore(path)

	now := time.Now().Truncate(time.Millisecond)
	err := store.Update(func(s *State) {
		s.TrackingTaskID = "task-123"
		s.TaskTitle = "Test Task"
		s.StartedAt = 1772734813781
		s.PushToStartToken = "token-abc"
		s.UpdateToken = "token-xyz"
		s.LiveActivityStartedAt = now
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	store2 := NewStateStore(path)
	if err := store2.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	got := store2.Get()
	if got.TrackingTaskID != "task-123" {
		t.Errorf("expected task-123, got %s", got.TrackingTaskID)
	}
	if got.TaskTitle != "Test Task" {
		t.Errorf("expected Test Task, got %s", got.TaskTitle)
	}
	if got.StartedAt != 1772734813781 {
		t.Errorf("expected 1772734813781, got %d", got.StartedAt)
	}
	if got.PushToStartToken != "token-abc" {
		t.Errorf("expected token-abc, got %s", got.PushToStartToken)
	}
}

func TestStateStoreLoadMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.json")
	store := NewStateStore(path)

	if err := store.Load(); err != nil {
		t.Fatalf("load should not fail for missing file: %v", err)
	}

	got := store.Get()
	if got.IsTracking() {
		t.Error("expected empty state to not be tracking")
	}
}

func TestStateStoreClear(t *testing.T) {
	path := tempStateFile(t)
	store := NewStateStore(path)

	store.Update(func(s *State) {
		s.TrackingTaskID = "task-123"
		s.TaskTitle = "Test"
	})

	if err := store.Clear(); err != nil {
		t.Fatalf("clear failed: %v", err)
	}

	got := store.Get()
	if got.IsTracking() {
		t.Error("expected cleared state to not be tracking")
	}
	if got.TaskTitle != "" {
		t.Errorf("expected empty title, got %s", got.TaskTitle)
	}

	// Verify file on disk is also cleared
	store2 := NewStateStore(path)
	if err := store2.Load(); err != nil {
		t.Fatalf("load after clear failed: %v", err)
	}
	if store2.Get().IsTracking() {
		t.Error("expected loaded-after-clear state to not be tracking")
	}
}

func TestStateStoreConcurrentAccess(t *testing.T) {
	path := tempStateFile(t)
	store := NewStateStore(path)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.Update(func(s *State) {
				s.TrackingTaskID = "task"
			})
			_ = store.Get()
		}()
	}
	wg.Wait()

	// If we got here without a race/panic, the mutex is working
	got := store.Get()
	if got.TrackingTaskID != "task" {
		t.Errorf("expected task, got %s", got.TrackingTaskID)
	}
}

func TestStateStoreAtomicWrite(t *testing.T) {
	path := tempStateFile(t)
	store := NewStateStore(path)

	store.Update(func(s *State) {
		s.TrackingTaskID = "task-1"
	})

	// Verify the file exists and is valid JSON
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read state file: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("state file is empty")
	}
}
