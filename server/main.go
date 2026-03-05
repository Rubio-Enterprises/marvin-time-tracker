package main

import (
	"log"
	"net/http"
	"time"
)

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	store := NewStateStore(cfg.StateFilePath)
	if err := store.Load(); err != nil {
		log.Fatalf("state load error: %v", err)
	}

	dedup := NewDedupCache(60 * time.Second)

	// Initialize APNs notifier if configured
	var notifier Notifier
	if cfg.APNsKeyID != "" && cfg.APNsTeamID != "" && cfg.APNsPrivateKeyPath != "" {
		apnsClient, err := NewAPNsClient(cfg.APNsPrivateKeyPath, cfg.APNsKeyID, cfg.APNsTeamID, cfg.APNsBundleID)
		if err != nil {
			log.Fatalf("APNs init error: %v", err)
		}
		notifier = apnsClient
		log.Printf("APNs client initialized")
	} else {
		log.Printf("APNs not configured, push notifications disabled")
	}

	// Initialize Marvin client and poller
	marvin := NewMarvinClient(cfg.MarvinAPIToken)
	quota := NewQuotaCounter()
	poller := NewPoller(marvin, store, notifier, cfg.PollIntervalActive, cfg.PollIntervalIdle, quota)
	poller.Start()
	log.Printf("poller started (active=%v, idle=%v)", cfg.PollIntervalActive, cfg.PollIntervalIdle)

	srv := NewServer(store, dedup, notifier)

	log.Printf("listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, srv); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
