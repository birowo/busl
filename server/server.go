package server

import (
	"log"
	"net/http"
	"time"

	"github.com/braintree/manners"
	"github.com/gorilla/mux"
)

// Config holds all the server options
type Config struct {
	EnforceHTTPS      bool
	Credentials       string
	HeartbeatDuration time.Duration
	StorageBaseURL    string
}

// Server is a launchable api listener
type Server struct {
	*manners.GracefulServer
	*Config
}

// NewServer creates a new server instance
func NewServer(config *Config) *Server {
	return &Server{
		GracefulServer: manners.NewServer(),
		Config:         config,
	}
}

// Start starts the server instance
func (s *Server) Start(port string, shutdown <-chan struct{}) {
	log.Printf("http.start.port=%s\n", port)
	s.Handler = s.router()
	go s.listenForShutdown(shutdown)

	s.Addr = ":" + port
	if err := s.ListenAndServe(); err != nil {
		log.Fatalf("server.server error=%v", err)
	}
}

func (s *Server) listenForShutdown(shutdown <-chan struct{}) {
	log.Println("http.graceful.await")
	<-shutdown
	log.Println("http.graceful.shutdown")
	s.Close()
}

func (s *Server) router() http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/health", s.addDefaultHeaders(s.health))

	r.HandleFunc("/streams/{key:.+}", s.addDefaultHeaders(s.subscribe)).Methods("GET")
	r.HandleFunc("/streams/{key:.+}", s.addDefaultHeaders(s.publish)).Methods("POST")
	r.HandleFunc("/streams/{key:.+}", s.auth(s.addDefaultHeaders(s.createStream))).Methods("PUT")

	return logRequest(s.enforceHTTPS(r.ServeHTTP))
}
