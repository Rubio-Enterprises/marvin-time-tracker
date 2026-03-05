package main

import (
	"log"
	"time"
)

const renewalThreshold = 7*time.Hour + 45*time.Minute

type Renewal struct {
	store    *StateStore
	notifier Notifier
	stop     chan struct{}
	now      func() time.Time // for testing
}

func NewRenewal(store *StateStore, notifier Notifier) *Renewal {
	return &Renewal{
		store:    store,
		notifier: notifier,
		stop:     make(chan struct{}),
		now:      time.Now,
	}
}

func (rn *Renewal) Start() {
	go rn.run()
}

func (rn *Renewal) Stop() {
	close(rn.stop)
}

func (rn *Renewal) run() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-rn.stop:
			return
		case <-ticker.C:
			rn.check()
		}
	}
}

func (rn *Renewal) check() {
	state := rn.store.Get()
	if !state.IsTracking() {
		return
	}

	if state.LiveActivityStartedAt.IsZero() {
		return
	}

	elapsed := rn.now().Sub(state.LiveActivityStartedAt)
	if elapsed < renewalThreshold {
		return
	}

	if rn.notifier == nil {
		return
	}

	log.Printf("renewal: Live Activity at %v, triggering renewal", elapsed.Round(time.Second))

	// End current Live Activity
	if state.UpdateToken != "" {
		if err := rn.notifier.EndActivity(state.UpdateToken); err != nil {
			log.Printf("renewal: end error: %v", err)
			return
		}
	}

	// Brief pause for APNs processing
	time.Sleep(500 * time.Millisecond)

	// Start new Live Activity with original startedAt
	rn.store.Update(func(s *State) {
		s.LiveActivityStartedAt = rn.now()
		s.UpdateToken = "" // Will be re-registered by iOS app
	})

	updatedState := rn.store.Get()
	if updatedState.PushToStartToken != "" {
		if err := rn.notifier.StartActivity(updatedState.PushToStartToken, state.TaskTitle, state.StartedAt); err != nil {
			log.Printf("renewal: start error: %v", err)
		} else {
			log.Printf("renewal: new Live Activity started, preserving original startedAt")
		}
	} else {
		log.Printf("renewal: no pushToStartToken available for restart")
	}
}
