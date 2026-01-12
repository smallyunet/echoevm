package web

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

//go:embed assets/*
var assetsEmbed embed.FS

type Server struct {
	addr      string
	upgrader  websocket.Upgrader
	hub       *Hub
	assetsDir fs.FS
}

func NewServer(addr string) *Server {
	// Creating a sub-filesystem for the assets directory
	assets, err := fs.Sub(assetsEmbed, "assets")
	if err != nil {
		// Should not happen with embed
		panic(err)
	}

	return &Server{
		addr: addr,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for local debugging
			},
		},
		hub:       NewHub(),
		assetsDir: assets,
	}
}

func (s *Server) Start() error {
	go s.hub.Run()

	// Debug: List files in assets
	fs.WalkDir(s.assetsDir, ".", func(path string, d fs.DirEntry, err error) error {
		if err == nil {
			log.Info().Str("path", path).Msg("Found embedded asset")
		}
		return nil
	})

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
	client.hub.register <- client

	// Allow collection of memory stats by the hub to restart/start execution
	// For now, simpler: just register. The execution loop drives the events.
	// But we might need a way to trigger execution from the UI.

	// We'll spin up the read/write pumps
	go client.writePump()
	go client.readPump()
}

// Broadcast sends a message to all connected clients
func (s *Server) Broadcast(msg []byte) {
	s.hub.broadcast <- msg
}
