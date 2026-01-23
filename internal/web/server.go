package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"github.com/smallyunet/echoevm/internal/config"
)

//go:embed assets/*
var assetsEmbed embed.FS

type Server struct {
	addr      string
	upgrader  websocket.Upgrader
	hub       *Hub
	assetsDir fs.FS
	control   chan ControlMessage
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
		hub:       NewHub(),
		assetsDir: assets,
		control:   make(chan ControlMessage, 16),
	}
}

func (s *Server) Start() error {
	go s.hub.Run()

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(s.assetsDir)))
	mux.HandleFunc("/ws", s.serveWs)

	log.Info().Str("addr", s.addr).Msg("Starting Web Debugger UI")
	return http.ListenAndServe(s.addr, mux)
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
