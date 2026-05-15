package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/rs/cors"
)

type Server struct {
	publicHandler  http.Handler
	privateHandler http.Handler
	store          *StateStore
}

func NewServer(store *StateStore, dedup *DedupCache, notifier Notifier, opts ...ServerOption) *Server {
	so := &serverOptions{}
	for _, o := range opts {
		o(so)
	}

	// Convert concrete nil pointers to proper nil interfaces to avoid
	// non-nil interface wrapping nil pointer issues.
	var broker BrokerPublisher
	if so.broker != nil {
		broker = so.broker
	}
	var history SessionRecorder
	if so.history != nil {
		history = so.history
	}

	wh := NewWebhookHandler(store, dedup, notifier, broker, history, so.debug)
	rh := NewRegisterHandler(store, notifier, broker)

	auth := func(h http.HandlerFunc) http.HandlerFunc {
		return requireAPIKey(so.apiKey, h)
	}

	// Public mux: webhooks and userscript (exposed via Tailscale Funnel)
	publicMux := http.NewServeMux()
	publicMux.HandleFunc("POST /webhook/start", rateLimitMiddleware(wh.HandleStart))
	publicMux.HandleFunc("POST /webhook/stop", rateLimitMiddleware(wh.HandleStop))
	publicMux.HandleFunc("GET /userscript/marvin-relay-tracker.user.js", userscriptHandler(so.externalURL))

	c := cors.New(cors.Options{
		AllowedOrigins:       []string{"*"},
		AllowedMethods:       []string{"GET", "POST", "PUT", "OPTIONS"},
		AllowedHeaders:       []string{"Content-Type", "Authorization"},
		OptionsSuccessStatus: 200,
	})

	// Private mux: app-facing endpoints (Tailscale network only)
	privateMux := http.NewServeMux()
	privateMux.HandleFunc("GET /status", auth(statusHandler(store)))
	privateMux.HandleFunc("POST /register", auth(rh.Handle))

	if so.history != nil {
		privateMux.HandleFunc("GET /history", auth(historyHandler(so.history)))
	}

	if so.broker != nil {
		privateMux.HandleFunc("GET /events", auth(sseHandler(store, so.broker)))
	}

	if so.marvin != nil {
		th := NewTrackHandler(store, so.marvin, notifier, broker, history)
		privateMux.HandleFunc("POST /start", auth(th.HandleStart))
		privateMux.HandleFunc("POST /stop", auth(th.HandleStop))
		privateMux.HandleFunc("GET /tasks", auth(tasksHandler(so.marvin)))
	}

	return &Server{
		publicHandler:  c.Handler(publicMux),
		privateHandler: privateMux,
		store:          store,
	}
}

type serverOptions struct {
	marvin      MarvinAPIClient
	broker      *Broker // concrete type needed for SSE handler's Subscribe()
	history     *HistoryStore
	externalURL string
	apiKey      string
	debug       bool
}

type ServerOption func(*serverOptions)

func WithBroker(b *Broker) ServerOption {
	return func(so *serverOptions) {
		so.broker = b
	}
}

func WithMarvinClient(mc MarvinAPIClient) ServerOption {
	return func(so *serverOptions) {
		so.marvin = mc
	}
}

func WithHistory(h *HistoryStore) ServerOption {
	return func(so *serverOptions) {
		so.history = h
	}
}

func WithExternalURL(url string) ServerOption {
	return func(so *serverOptions) {
		so.externalURL = url
	}
}

func WithAPIKey(key string) ServerOption {
	return func(so *serverOptions) {
		so.apiKey = key
	}
}

func WithDebug(debug bool) ServerOption {
	return func(so *serverOptions) {
		so.debug = debug
	}
}

func (s *Server) PublicHandler() http.Handler  { return s.publicHandler }
func (s *Server) PrivateHandler() http.Handler { return s.privateHandler }

func historyHandler(history *HistoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 10
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = n
			}
		}
		if limit > 200 {
			limit = 200
		}

		sessions := history.Recent(limit)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sessions)
	}
}

func statusHandler(store *StateStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := store.Get()

		resp := map[string]interface{}{
			"status":              "ok",
			"tracking":            state.IsTracking(),
			"hasPushToStartToken": state.PushToStartToken != "",
			"hasUpdateToken":      state.UpdateToken != "",
			"hasDeviceToken":      state.DeviceToken != "",
		}

		if state.IsTracking() {
			resp["taskId"] = state.TrackingTaskID
			resp["taskTitle"] = state.TaskTitle
			resp["startedAt"] = state.StartedAt
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
