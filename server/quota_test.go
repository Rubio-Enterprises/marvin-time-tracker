package main

import (
	"testing"
	"time"
)

func TestQuotaIncrement(t *testing.T) {
	qc := NewQuotaCounter()
	qc.Increment()
	qc.Increment()
	qc.Increment()

	if qc.Count() != 3 {
		t.Errorf("expected 3, got %d", qc.Count())
	}
}

func TestQuotaWarningThreshold(t *testing.T) {
	qc := NewQuotaCounter()
	// Manually set count near warning
	qc.mu.Lock()
	qc.count = 1151 // 80% of 1440
	qc.resetDay = time.Now().UTC().YearDay()
	qc.mu.Unlock()

	// This should trigger a warning log (not crash)
	qc.Increment()

	if qc.Count() != 1152 {
		t.Errorf("expected 1152, got %d", qc.Count())
	}
}

func TestQuotaExhausted(t *testing.T) {
	qc := NewQuotaCounter()
	qc.mu.Lock()
	qc.count = 1368 // 95% of 1440
	qc.resetDay = time.Now().UTC().YearDay()
	qc.mu.Unlock()

	if !qc.IsExhausted() {
		t.Error("expected exhausted at 95%")
	}
}

func TestQuotaNotExhausted(t *testing.T) {
	qc := NewQuotaCounter()
	qc.Increment()

	if qc.IsExhausted() {
		t.Error("should not be exhausted after 1 call")
	}
}

func TestQuotaDailyReset(t *testing.T) {
	now := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	qc := NewQuotaCounter()
	qc.now = func() time.Time { return now }

	qc.Increment()
	qc.Increment()

	if qc.Count() != 2 {
		t.Errorf("expected 2, got %d", qc.Count())
	}

	// Advance to next day
	tomorrow := now.Add(24 * time.Hour)
	qc.now = func() time.Time { return tomorrow }

	if qc.Count() != 0 {
		t.Errorf("expected 0 after day change, got %d", qc.Count())
	}
}
