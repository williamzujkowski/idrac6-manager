package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/williamzujkowski/idrac6-manager/internal/idrac"
)

type contextKey string

const hostConfigKey contextKey = "hostConfig"

// Handlers holds API handler dependencies.
type Handlers struct {
	config  *Config
	clients sync.Map // map[string]*idrac.Client
	vmedia  sync.Map // map[string]*idrac.VirtualMedia
}

// getClient returns or creates an iDRAC client for the given host.
func (h *Handlers) getClient(hostID string) (*idrac.Client, error) {
	if cached, ok := h.clients.Load(hostID); ok {
		return cached.(*idrac.Client), nil
	}

	hostCfg, ok := h.config.Hosts[hostID]
	if !ok {
		return nil, fmt.Errorf("host %q not found", hostID)
	}

	client := idrac.NewClient(hostCfg.Host, hostCfg.Username, hostCfg.Password)
	if err := client.Login(); err != nil {
		return nil, fmt.Errorf("login to %s failed: %w", hostCfg.Host, err)
	}

	h.clients.Store(hostID, client)
	return client, nil
}

// hostCtx middleware extracts the host ID and validates it exists.
func (h *Handlers) hostCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hostID := chi.URLParam(r, "hostID")
		hostCfg, ok := h.config.Hosts[hostID]
		if !ok {
			writeError(w, http.StatusNotFound, "host not found: "+hostID)
			return
		}

		ctx := context.WithValue(r.Context(), hostConfigKey, hostCfg)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Health returns service health status.
func (h *Handlers) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "idrac6-manager",
	})
}

// ListHosts returns all configured hosts (without credentials).
func (h *Handlers) ListHosts(w http.ResponseWriter, _ *http.Request) {
	type hostInfo struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Host string `json:"host"`
	}

	var hosts []hostInfo
	for id, cfg := range h.config.Hosts {
		hosts = append(hosts, hostInfo{
			ID:   id,
			Name: cfg.Name,
			Host: cfg.Host,
		})
	}

	writeJSON(w, http.StatusOK, hosts)
}

// AddHost adds a new host configuration at runtime.
func (h *Handlers) AddHost(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Host     string `json:"host"`
		Username string `json:"username"`
		Password string `json:"password"`
		SSHPort  int    `json:"sshPort,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ID == "" || req.Host == "" || req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "id, host, username, and password are required")
		return
	}

	h.config.Hosts[req.ID] = &HostConfig{
		Name:     req.Name,
		Host:     req.Host,
		Username: req.Username,
		Password: req.Password,
		SSHPort:  req.SSHPort,
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "added", "id": req.ID})
}

// GetPower returns the current power state.
func (h *Handlers) GetPower(w http.ResponseWriter, r *http.Request) {
	hostID := chi.URLParam(r, "hostID")
	client, err := h.getClient(hostID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	status, err := client.GetPowerState()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, status)
}

// SetPower executes a power action.
func (h *Handlers) SetPower(w http.ResponseWriter, r *http.Request) {
	hostID := chi.URLParam(r, "hostID")

	var req struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Action == "" {
		writeError(w, http.StatusBadRequest, "action is required (on, off, restart, reset, nmi, shutdown)")
		return
	}

	client, err := h.getClient(hostID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := client.SetPowerByName(req.Action); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "action": req.Action})
}

// GetSensors returns all sensor readings.
func (h *Handlers) GetSensors(w http.ResponseWriter, r *http.Request) {
	hostID := chi.URLParam(r, "hostID")
	client, err := h.getClient(hostID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sensors, err := client.GetSensors()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, sensors)
}

// GetSystemInfo returns system identification info.
func (h *Handlers) GetSystemInfo(w http.ResponseWriter, r *http.Request) {
	hostID := chi.URLParam(r, "hostID")
	client, err := h.getClient(hostID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	info, err := client.GetSystemInfo()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, info)
}

// GetSEL returns the System Event Log.
func (h *Handlers) GetSEL(w http.ResponseWriter, r *http.Request) {
	hostID := chi.URLParam(r, "hostID")
	client, err := h.getClient(hostID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sel, err := client.GetSEL()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, sel)
}

// ClearSEL clears the System Event Log.
func (h *Handlers) ClearSEL(w http.ResponseWriter, r *http.Request) {
	hostID := chi.URLParam(r, "hostID")
	client, err := h.getClient(hostID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := client.ClearSEL(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

// getVMedia returns or creates a VirtualMedia manager for the given host.
func (h *Handlers) getVMedia(hostID string) (*idrac.VirtualMedia, error) {
	if cached, ok := h.vmedia.Load(hostID); ok {
		return cached.(*idrac.VirtualMedia), nil
	}

	hostCfg, ok := h.config.Hosts[hostID]
	if !ok {
		return nil, fmt.Errorf("host %q not found", hostID)
	}

	sshPort := hostCfg.SSHPort
	if sshPort == 0 {
		sshPort = 22
	}

	vm := idrac.NewVirtualMedia(hostCfg.Host, sshPort, hostCfg.Username, hostCfg.Password)
	h.vmedia.Store(hostID, vm)
	return vm, nil
}

// GetVirtualMedia returns the current virtual media mount status.
func (h *Handlers) GetVirtualMedia(w http.ResponseWriter, r *http.Request) {
	hostID := chi.URLParam(r, "hostID")
	vm, err := h.getVMedia(hostID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	status, err := vm.GetStatus()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, status)
}

// MountVirtualMedia mounts an ISO/IMG via RACADM.
func (h *Handlers) MountVirtualMedia(w http.ResponseWriter, r *http.Request) {
	hostID := chi.URLParam(r, "hostID")

	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}

	vm, err := h.getVMedia(hostID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := vm.Mount(req.URL); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "mounted", "url": req.URL})
}

// UnmountVirtualMedia unmounts the current virtual media.
func (h *Handlers) UnmountVirtualMedia(w http.ResponseWriter, r *http.Request) {
	hostID := chi.URLParam(r, "hostID")
	vm, err := h.getVMedia(hostID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := vm.Unmount(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "unmounted"})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
