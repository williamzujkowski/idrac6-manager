package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// newTestRouter creates a router backed by a mock iDRAC server.
func newTestRouter(t *testing.T) (http.Handler, *httptest.Server) {
	t.Helper()

	// Mock iDRAC server (two-step login flow)
	idracServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/start.html":
			http.SetCookie(w, &http.Cookie{Name: "_appwebSessionId_", Value: "test-session"})
			fmt.Fprint(w, `<html></html>`)
			return
		case "/data/login":
			fmt.Fprint(w, `<root><authResult>0</authResult><forwardUrl>index.html</forwardUrl></root>`)
			return
		}

		get := r.URL.Query().Get("get")
		set := r.URL.Query().Get("set")

		switch {
		case strings.Contains(get, "pwState"):
			fmt.Fprint(w, `<root><pwState>1</pwState></root>`)
		case strings.Contains(get, "temperatures"):
			fmt.Fprint(w, `<root><temperatures>Inlet Temp=23;ok;42;47</temperatures><fans></fans><voltages></voltages></root>`)
		case strings.Contains(get, "hostName"):
			fmt.Fprint(w, `<root><hostName>R710</hostName><sysDesc>PowerEdge R710</sysDesc><sysRev></sysRev><biosVer>6.6.0</biosVer><fwVersion>2.92</fwVersion><LCCfwVersion></LCCfwVersion><osName></osName><svcTag>ABC123</svcTag></root>`)
		case strings.Contains(get, "sel"):
			fmt.Fprint(w, `<root><sel>1|2024-01-01|Normal|Boot</sel></root>`)
		case set != "":
			fmt.Fprint(w, `<root><status>ok</status></root>`)
		default:
			fmt.Fprint(w, `<root></root>`)
		}
	}))

	cfg := &Config{
		Hosts: map[string]*HostConfig{
			"test": {
				Name:     "Test Server",
				Host:     strings.TrimPrefix(idracServer.URL, "https://"),
				Username: "root",
				Password: "calvin",
			},
		},
	}

	return NewRouter(cfg), idracServer
}

func TestHealthEndpoint(t *testing.T) {
	cfg := &Config{Hosts: map[string]*HostConfig{}}
	router := NewRouter(cfg)

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("status = %q, want ok", body["status"])
	}
}

func TestListHosts(t *testing.T) {
	cfg := &Config{
		Hosts: map[string]*HostConfig{
			"server1": {Name: "Server 1", Host: "10.0.0.1", Username: "root", Password: "pass"},
		},
	}
	router := NewRouter(cfg)

	req := httptest.NewRequest("GET", "/api/hosts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var hosts []map[string]string
	json.NewDecoder(w.Body).Decode(&hosts)
	if len(hosts) != 1 {
		t.Fatalf("got %d hosts, want 1", len(hosts))
	}
	// Verify password is not leaked
	if _, ok := hosts[0]["password"]; ok {
		t.Error("password should not be included in host listing")
	}
}

func TestAddHost(t *testing.T) {
	cfg := &Config{Hosts: map[string]*HostConfig{}}
	router := NewRouter(cfg)

	body := `{"id":"new","name":"New Server","host":"10.0.0.2","username":"admin","password":"secret"}`
	req := httptest.NewRequest("POST", "/api/hosts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	if _, ok := cfg.Hosts["new"]; !ok {
		t.Error("host not added to config")
	}
}

func TestAddHost_MissingFields(t *testing.T) {
	cfg := &Config{Hosts: map[string]*HostConfig{}}
	router := NewRouter(cfg)

	body := `{"id":"","host":"10.0.0.2"}`
	req := httptest.NewRequest("POST", "/api/hosts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHostNotFound(t *testing.T) {
	cfg := &Config{Hosts: map[string]*HostConfig{}}
	router := NewRouter(cfg)

	req := httptest.NewRequest("GET", "/api/hosts/nonexistent/power", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestAPIKeyAuth(t *testing.T) {
	cfg := &Config{
		Hosts:  map[string]*HostConfig{},
		APIKey: "secret-key",
	}
	router := NewRouter(cfg)

	// Without key
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("without key: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	// With X-API-Key header
	req = httptest.NewRequest("GET", "/api/health", nil)
	req.Header.Set("X-API-Key", "secret-key")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("with X-API-Key: status = %d, want %d", w.Code, http.StatusOK)
	}

	// With Bearer token
	req = httptest.NewRequest("GET", "/api/health", nil)
	req.Header.Set("Authorization", "Bearer secret-key")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("with Bearer: status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestCORSHeaders(t *testing.T) {
	cfg := &Config{Hosts: map[string]*HostConfig{}}
	router := NewRouter(cfg)

	req := httptest.NewRequest("OPTIONS", "/api/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS origin header missing")
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("CORS methods header missing")
	}
}

func TestHostCtxMiddleware(t *testing.T) {
	h := &Handlers{
		config: &Config{
			Hosts: map[string]*HostConfig{
				"exists": {Name: "Test", Host: "10.0.0.1"},
			},
		},
	}

	// Test existing host
	r := chi.NewRouter()
	r.Route("/hosts/{hostID}", func(r chi.Router) {
		r.Use(h.hostCtx)
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	req := httptest.NewRequest("GET", "/hosts/exists/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("existing host: status = %d, want %d", w.Code, http.StatusOK)
	}

	// Test missing host
	req = httptest.NewRequest("GET", "/hosts/missing/test", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("missing host: status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
