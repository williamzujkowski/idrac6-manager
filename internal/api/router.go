// Package api provides the HTTP API for the iDRAC6 manager.
package api

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Config holds API server configuration.
type Config struct {
	// Hosts maps host IDs to their iDRAC configurations.
	Hosts map[string]*HostConfig
	// WebFS is the embedded filesystem for static web assets.
	WebFS fs.FS
	// APIKey is the optional API key for authentication.
	APIKey string
}

// HostConfig holds configuration for a single iDRAC host.
type HostConfig struct {
	Name     string `json:"name" yaml:"name"`
	Host     string `json:"host" yaml:"host"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	SSHPort  int    `json:"sshPort,omitempty" yaml:"ssh_port,omitempty"`
}

// NewRouter creates the HTTP router with all API routes.
func NewRouter(cfg *Config) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(corsMiddleware)

	h := &Handlers{config: cfg}

	r.Route("/api", func(r chi.Router) {
		if cfg.APIKey != "" {
			r.Use(apiKeyAuth(cfg.APIKey))
		}

		r.Get("/health", h.Health)

		r.Get("/hosts", h.ListHosts)
		r.Post("/hosts", h.AddHost)

		r.Route("/hosts/{hostID}", func(r chi.Router) {
			r.Use(h.hostCtx)

			r.Get("/power", h.GetPower)
			r.Post("/power", h.SetPower)

			r.Get("/sensors", h.GetSensors)

			r.Get("/info", h.GetSystemInfo)

			r.Get("/sel", h.GetSEL)
			r.Delete("/sel", h.ClearSEL)

			r.Get("/virtualmedia", h.GetVirtualMedia)
			r.Post("/virtualmedia", h.MountVirtualMedia)
			r.Delete("/virtualmedia", h.UnmountVirtualMedia)
		})
	})

	// Serve web UI
	if cfg.WebFS != nil {
		fileServer := http.FileServer(http.FS(cfg.WebFS))
		r.Handle("/*", fileServer)
	}

	return r
}
