package main

import (
	"log"
	"sync"
	"time"
)

const dailyLimit = 1440

type QuotaCounter struct {
	mu       sync.Mutex
	count    int
	resetDay int // day of year for reset tracking
	now      func() time.Time
}

func NewQuotaCounter() *QuotaCounter {
	return &QuotaCounter{
		now: time.Now,
	}
}

func (qc *QuotaCounter) Increment() {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	qc.maybeResetLocked()
	qc.count++

	pct := float64(qc.count) / float64(dailyLimit) * 100
	if pct >= 95 {
		log.Printf("quota: CRITICAL - %d/%d calls (%.0f%%)", qc.count, dailyLimit, pct)
	} else if pct >= 80 {
		log.Printf("quota: WARNING - %d/%d calls (%.0f%%)", qc.count, dailyLimit, pct)
	}
}

func (qc *QuotaCounter) IsExhausted() bool {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	qc.maybeResetLocked()
	return float64(qc.count) >= float64(dailyLimit)*0.95
}

func (qc *QuotaCounter) Count() int {
	qc.mu.Lock()
	defer qc.mu.Unlock()
	qc.maybeResetLocked()
	return qc.count
}

func (qc *QuotaCounter) maybeResetLocked() {
	today := qc.now().UTC().YearDay()
	if qc.resetDay != today {
		qc.count = 0
		qc.resetDay = today
	}
}
