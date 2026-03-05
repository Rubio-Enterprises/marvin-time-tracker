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
	srv := NewServer(store, dedup, nil)

	log.Printf("listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, srv); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
