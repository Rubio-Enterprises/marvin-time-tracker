package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println("marvin-relay " + version)
		os.Exit(0)
	}

	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	store := NewStateStore(cfg.StateFilePath)
	if err := store.Load(); err != nil {
		log.Fatalf("state load error: %v", err)
	}

	history := NewHistoryStore(cfg.HistoryFilePath)
	if err := history.Load(); err != nil {
		log.Fatalf("history load error: %v", err)
	}

	dedup := NewDedupCache(60 * time.Second)

	// Initialize APNs notifier if configured
	var notifier Notifier
	if cfg.APNsKeyID != "" && cfg.APNsTeamID != "" && cfg.APNsPrivateKeyPath != "" {
		apnsClient, err := NewAPNsClient(cfg.APNsPrivateKeyPath, cfg.APNsKeyID, cfg.APNsTeamID, cfg.APNsBundleID, cfg.APNsEnv)
		if err != nil {
			log.Fatalf("APNs init error: %v", err)
		}
		notifier = apnsClient
		log.Printf("APNs client initialized (%s)", cfg.APNsEnv)
	} else {
		log.Printf("APNs not configured, push notifications disabled")
	}

	broker := NewBroker()

	marvin := NewMarvinClient(cfg.MarvinAPIToken, cfg.MarvinFullAccessToken)

	// Start 8-hour Live Activity renewal
	renewal := NewRenewal(store, notifier, broker)
	renewal.Start()
	log.Printf("renewal monitor started")

	if cfg.APIKey == "" {
		log.Printf("WARNING: API_KEY not set, app endpoints are unprotected")
	}

	srv := NewServer(store, dedup, notifier, WithBroker(broker), WithMarvinClient(marvin), WithHistory(history), WithExternalURL(cfg.ExternalURL), WithAPIKey(cfg.APIKey), WithDebug(cfg.Debug))

	publicServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           srv.PublicHandler(),
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	privateServer := &http.Server{
		Addr:              cfg.PrivateListenAddr,
		Handler:           srv.PrivateHandler(),
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		log.Printf("public listener on %s (webhooks, userscript)", cfg.ListenAddr)
		if err := publicServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})

	g.Go(func() error {
		log.Printf("private listener on %s (app endpoints)", cfg.PrivateListenAddr)
		if err := privateServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})

	g.Go(func() error {
		<-gCtx.Done()
		stop()
		log.Printf("shutting down...")

		renewal.Stop()
		webhookLimiter.Stop()

		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		publicServer.Shutdown(shutCtx)
		privateServer.Shutdown(shutCtx)
		return nil
	})

	if err := g.Wait(); err != nil {
		log.Fatalf("server error: %v", err)
	}

	log.Printf("shutdown complete")
}
