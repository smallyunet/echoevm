package web

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"github.com/smallyunet/echoevm/internal/config"
	"github.com/smallyunet/echoevm/internal/differential"
	"github.com/smallyunet/echoevm/internal/replay"
)

//go:embed assets/*
var assetsEmbed embed.FS

type Server struct {
	addr         string
	upgrader     websocket.Upgrader
	hub          *Hub
	assetsDir    fs.FS
	assetVersion string
	control      chan ControlMessage
	differential *differential.Engine
	diffSlots    chan struct{}
	replay       *replay.Service
	replaySlots  chan struct{}
}

func NewDifferentialServer(addr string, engine *differential.Engine) *Server {
	s := NewServer(addr)
	s.differential = engine
	s.diffSlots = make(chan struct{}, 1)
	s.replaySlots = make(chan struct{}, 1)
	service, err := replay.NewService(context.Background(), config.GetRuntimeConfig().EthereumRPC)
	if err != nil {
		log.Warn().Err(err).Msg("Transaction replay is unavailable")
	} else {
		s.replay = service
	}
	return s
}

func NewServer(addr string) *Server {
	// Creating a sub-filesystem for the assets directory
	assets, err := fs.Sub(assetsEmbed, "assets")
	if err != nil {
		// Should not happen with embed
		panic(err)
	}

	runtimeConfig := config.GetRuntimeConfig()
	allowedOrigins, allowAll := parseAllowedOrigins(runtimeConfig.WebOrigins)

	return &Server{
		addr: addr,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return isOriginAllowed(r, allowedOrigins, allowAll)
			},
		},
		hub:          NewHub(),
		assetsDir:    assets,
		assetVersion: fingerprintAssets(assets, "diff.css", "diff.js"),
		control:      make(chan ControlMessage, 16),
	}
}

func fingerprintAssets(assets fs.FS, names ...string) string {
	hash := sha256.New()
	for _, name := range names {
		data, err := fs.ReadFile(assets, name)
		if err != nil {
			panic(fmt.Sprintf("read embedded asset %s: %v", name, err))
		}
		_, _ = hash.Write(data)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))[:16]
}

func cacheVersionedAsset(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("v") == "" {
			w.Header().Set("Cache-Control", "no-cache")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) Start() error {
	go s.hub.Run()

	mux := http.NewServeMux()
	if s.differential != nil {
		mux.HandleFunc("/", s.serveDifferentialIndex)
		assets := http.StripPrefix("/assets/", http.FileServer(http.FS(s.assetsDir)))
		mux.Handle("/assets/", cacheVersionedAsset(assets))
		mux.HandleFunc("/api/diff", s.serveDiff)
		mux.HandleFunc("/api/replay", s.serveReplay)
		mux.HandleFunc("/healthz", s.serveHealth)
	} else {
		mux.Handle("/", http.FileServer(http.FS(s.assetsDir)))
		mux.HandleFunc("/ws", s.serveWs)
	}

	name := "Web Debugger"
	if s.differential != nil {
		name = "Differential Explorer"
	}
	log.Info().Str("addr", s.addr).Str("mode", name).Msg("Starting EchoEVM web UI")
	return http.ListenAndServe(s.addr, mux)
}

func (s *Server) serveReplay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.replay == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "transaction replay is unavailable: configure ECHOEVM_ETHEREUM_RPC with a trace-capable RPC endpoint")
		return
	}
	select {
	case s.replaySlots <- struct{}{}:
		defer func() { <-s.replaySlots }()
	default:
		writeJSONError(w, http.StatusTooManyRequests, "another transaction replay is already running")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var req replay.Request
	if err := decoder.Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeJSONError(w, http.StatusBadRequest, "invalid request: expected one JSON object")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	result, err := s.replay.Replay(ctx, req)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (s *Server) serveHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "echoevm-differential-explorer"})
}

func (s *Server) serveDifferentialIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := fs.ReadFile(s.assetsDir, "diff.html")
	if err != nil {
		http.Error(w, "Explorer asset unavailable", http.StatusInternalServerError)
		return
	}
	data = []byte(strings.ReplaceAll(string(data), "{{ASSET_VERSION}}", s.assetVersion))
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func (s *Server) serveDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	select {
	case s.diffSlots <- struct{}{}:
		defer func() { <-s.diffSlots }()
	default:
		writeJSONError(w, http.StatusTooManyRequests, "too many concurrent comparisons")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 512*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var req differential.Request
	if err := decoder.Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeJSONError(w, http.StatusBadRequest, "invalid request: expected one JSON object")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	result, err := s.differential.Compare(ctx, req)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (s *Server) serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade to websocket")
		return
	}

	client := &Client{hub: s.hub, conn: conn, send: make(chan []byte, 256)}
	client.control = s.control
	client.hub.register <- client

	// Allow collection of memory stats by the hub to restart/start execution
	// For now, simpler: just register. The execution loop drives the events.
	// But we might need a way to trigger execution from the UI.

	// We'll spin up the read/write pumps
	go client.writePump()
	go client.readPump()
}

func (s *Server) Control() <-chan ControlMessage {
	return s.control
}

func parseAllowedOrigins(raw string) ([]string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, false
	} else {
		if trimmed == "*" {
			return nil, true
		} else {
			parts := strings.Split(trimmed, ",")
			origins := make([]string, 0, len(parts))
			for _, part := range parts {
				origin := strings.TrimSpace(part)
				if origin == "" {
					_ = origin
				} else {
					origins = append(origins, origin)
				}
			}
			return origins, false
		}
	}
}

func isOriginAllowed(r *http.Request, allowed []string, allowAll bool) bool {
	origin := r.Header.Get("Origin")
	if allowAll {
		return true
	} else {
		if origin == "" {
			return true
		} else {
			if len(allowed) == 0 {
				return isSameOrigin(r, origin)
			} else {
				return containsOrigin(allowed, origin)
			}
		}
	}
}

func isSameOrigin(r *http.Request, origin string) bool {
	httpOrigin := "http://" + r.Host
	httpsOrigin := "https://" + r.Host
	if origin == httpOrigin {
		return true
	} else {
		if origin == httpsOrigin {
			return true
		} else {
			return false
		}
	}
}

func containsOrigin(allowed []string, origin string) bool {
	for _, item := range allowed {
		if item == origin {
			return true
		} else {
			_ = item
		}
	}
	return false
}

// Broadcast sends a message to all connected clients
func (s *Server) Broadcast(msg []byte) {
	s.hub.broadcast <- msg
}
