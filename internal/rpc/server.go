package rpc

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rs/zerolog"
	"github.com/smallyunet/echoevm/internal/config"
)

// Server represents the RPC server for echoevm
type Server struct {
	endpoint    string
	apis        []rpc.API
	httpServer  *http.Server
	rpcServer   *rpc.Server
	listener    net.Listener
	logger      zerolog.Logger
	shutdownCtx context.Context
}

// NewServer creates a new RPC server with the specified configuration
func NewServer(endpoint string, logger zerolog.Logger) *Server {
	return &Server{
		endpoint: endpoint,
		logger:   logger,
	}
}

// Start initializes and starts the RPC server
func (s *Server) Start() error {
	// Create a new RPC server
	s.rpcServer = rpc.NewServer()

	// Register APIs
	apis := s.GetAPIs()
	for _, api := range apis {
		if err := s.rpcServer.RegisterName(api.Namespace, api.Service); err != nil {
			s.logger.Error().
				Str("api_namespace", api.Namespace).
				Err(err).
				Msg("Failed to register API")
			return fmt.Errorf("error registering API %s: %w", api.Namespace, err)
		}
		s.logger.Debug().
			Str("api_namespace", api.Namespace).
			Str("api_version", api.Version).
			Bool("api_public", api.Public).
			Msg("API registered successfully")
	}

	// Create HTTP server
	s.httpServer = &http.Server{
		Handler: s.rpcHandler(),
	}

	// Start listening
	s.logger.Info().
		Str("endpoint", s.endpoint).
		Msg("Starting RPC server")

	listener, err := net.Listen("tcp", s.endpoint)
	if err != nil {
		s.logger.Error().
			Str("endpoint", s.endpoint).
			Err(err).
			Msg("Failed to start RPC server")
		return fmt.Errorf("failed to start RPC server: %w", err)
	}
	s.listener = listener

	// Start HTTP server in a separate goroutine
	go func() {
		err := s.httpServer.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			s.logger.Error().
				Str("endpoint", s.endpoint).
				Err(err).
				Msg("HTTP server error")
		}
	}()

	s.logger.Info().
		Str("endpoint", s.endpoint).
		Msg("RPC server started successfully")

	return nil
}

// Stop gracefully shuts down the RPC server
func (s *Server) Stop() error {
	s.logger.Info().Msg("Shutting down RPC server")

	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(context.Background()); err != nil {
			s.logger.Error().
				Err(err).
				Msg("Failed to shutdown HTTP server")
			return err
		}
		s.logger.Debug().Msg("HTTP server shutdown completed")
	}

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			s.logger.Error().
				Err(err).
				Msg("Failed to close listener")
			return err
		}
		s.logger.Debug().Msg("Listener closed successfully")
	}

	s.logger.Info().Msg("RPC server shutdown completed")
	return nil
}

// GetAPIs returns all the APIs that this server provides
func (s *Server) GetAPIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: config.DefaultAPINamespace,
			Version:   config.DefaultAPIVersion,
			Service:   NewEthAPI(s),
			Public:    config.DefaultAPIPublic,
		},
		{
			Namespace: "web3",
			Version:   config.DefaultAPIVersion,
			Service:   NewWeb3API(s),
			Public:    config.DefaultAPIPublic,
		},
		{
			Namespace: "net",
			Version:   config.DefaultAPIVersion,
			Service:   NewNetAPI(s),
			Public:    config.DefaultAPIPublic,
		},
	}
}

// rpcHandler creates an HTTP handler for the RPC server
func (s *Server) rpcHandler() http.Handler {
	// Create a new handler for the RPC server
	// Note: This is simplified as go-ethereum has more complex handlers
	return s.rpcServer
}
