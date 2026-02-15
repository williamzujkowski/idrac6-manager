package idrac

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockIDRAC creates a test server that mimics the iDRAC6 two-step login flow.
func mockIDRAC(t *testing.T, authResult int, forwardURL string) *httptest.Server {
	t.Helper()
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/start.html":
			http.SetCookie(w, &http.Cookie{
				Name:  "_appwebSessionId_",
				Value: "test-session-123",
			})
			fmt.Fprint(w, `<html><body>start</body></html>`)

		case "/data/login":
			if r.Method != "POST" {
				t.Errorf("login: expected POST, got %s", r.Method)
			}
			fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
				<root>
					<authResult>%d</authResult>
					<forwardUrl>%s</forwardUrl>
					<errorMsg></errorMsg>
				</root>`, authResult, forwardURL)

		case "/data":
			// Verify session cookie
			cookie, err := r.Cookie("_appwebSessionId_")
			if err != nil || cookie.Value == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			get := r.URL.Query().Get("get")
			set := r.URL.Query().Get("set")
			if get != "" {
				handleDataGet(w, get)
			} else if set != "" {
				fmt.Fprint(w, `<root><status>ok</status></root>`)
			}

		case "/data/logout":
			fmt.Fprint(w, `<root><status>ok</status></root>`)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func handleDataGet(w http.ResponseWriter, keys string) {
	if strings.Contains(keys, "pwState") {
		fmt.Fprint(w, `<root><pwState>1</pwState></root>`)
	} else if strings.Contains(keys, "temperatures") {
		fmt.Fprint(w, `<root><temperatures>Inlet Temp=23;ok;42;47</temperatures><fans></fans><voltages></voltages></root>`)
	} else if strings.Contains(keys, "hostName") {
		fmt.Fprint(w, `<root><hostName>R710</hostName><sysDesc>PowerEdge R710</sysDesc></root>`)
	} else {
		fmt.Fprint(w, `<root></root>`)
	}
}

func TestNewClient(t *testing.T) {
	c := NewClient("192.168.1.172", "root", "calvin")

	if c.host != "192.168.1.172" {
		t.Errorf("host = %q, want 192.168.1.172", c.host)
	}
	if c.username != "root" {
		t.Errorf("username = %q, want root", c.username)
	}
	if c.baseURL != "https://192.168.1.172" {
		t.Errorf("baseURL = %q, want https://192.168.1.172", c.baseURL)
	}
}

func TestLogin_Success(t *testing.T) {
	server := mockIDRAC(t, 0, "index.html")
	defer server.Close()

	c := NewClient("localhost", "root", "calvin")
	c.baseURL = server.URL
	c.http = server.Client()

	if err := c.Login(); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if c.sessionID != "test-session-123" {
		t.Errorf("sessionID = %q, want test-session-123", c.sessionID)
	}
}

func TestLogin_WithNewAuth(t *testing.T) {
	server := mockIDRAC(t, 0, "index.html?ST1=token1abc,ST2=token2def")
	defer server.Close()

	c := NewClient("localhost", "root", "calvin")
	c.baseURL = server.URL
	c.http = server.Client()

	if err := c.Login(); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if !c.newAuth {
		t.Error("newAuth should be true")
	}
	if c.st1 != "token1abc" {
		t.Errorf("st1 = %q, want token1abc", c.st1)
	}
	if c.st2 != "token2def" {
		t.Errorf("st2 = %q, want token2def", c.st2)
	}
}

func TestLogin_Failure(t *testing.T) {
	server := mockIDRAC(t, 1, "")
	defer server.Close()

	c := NewClient("localhost", "root", "wrong")
	c.baseURL = server.URL
	c.http = server.Client()

	err := c.Login()
	if err == nil {
		t.Fatal("Login() should have failed")
	}
	if !strings.Contains(err.Error(), "login failed") {
		t.Errorf("error = %v, want to contain 'login failed'", err)
	}
}

func TestGet_WithSession(t *testing.T) {
	server := mockIDRAC(t, 0, "index.html")
	defer server.Close()

	c := NewClient("localhost", "root", "calvin")
	c.baseURL = server.URL
	c.http = server.Client()

	if err := c.Login(); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	data, err := c.Get("pwState")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if !strings.Contains(string(data), "pwState") {
		t.Errorf("response should contain pwState, got: %s", string(data))
	}
}

func TestGet_RetryOn401(t *testing.T) {
	callCount := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/start.html":
			http.SetCookie(w, &http.Cookie{
				Name:  "_appwebSessionId_",
				Value: "sess",
			})
			fmt.Fprint(w, `<html></html>`)
		case "/data/login":
			fmt.Fprint(w, `<root><authResult>0</authResult><forwardUrl>index.html</forwardUrl></root>`)
		case "/data":
			callCount++
			if callCount == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			fmt.Fprint(w, `<root><pwState>1</pwState></root>`)
		}
	}))
	defer server.Close()

	c := NewClient("localhost", "root", "calvin")
	c.baseURL = server.URL
	c.http = server.Client()

	if err := c.Login(); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	data, err := c.Get("pwState")
	if err != nil {
		t.Fatalf("Get() should succeed after retry, got error = %v", err)
	}

	if !strings.Contains(string(data), "pwState") {
		t.Errorf("retry response should contain pwState")
	}
}

func TestExtractTokens(t *testing.T) {
	tests := []struct {
		name       string
		forwardURL string
		wantST1    string
		wantST2    string
		wantNew    bool
	}{
		{
			name:       "standard tokens",
			forwardURL: "index.html?ST1=abc123,ST2=def456",
			wantST1:    "abc123",
			wantST2:    "def456",
			wantNew:    true,
		},
		{
			name:       "no query string",
			forwardURL: "index.html",
			wantST1:    "",
			wantST2:    "",
			wantNew:    false,
		},
		{
			name:       "only ST1",
			forwardURL: "index.html?ST1=only",
			wantST1:    "only",
			wantST2:    "",
			wantNew:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{}
			c.extractTokens(tt.forwardURL)

			if c.st1 != tt.wantST1 {
				t.Errorf("st1 = %q, want %q", c.st1, tt.wantST1)
			}
			if c.st2 != tt.wantST2 {
				t.Errorf("st2 = %q, want %q", c.st2, tt.wantST2)
			}
			if c.newAuth != tt.wantNew {
				t.Errorf("newAuth = %v, want %v", c.newAuth, tt.wantNew)
			}
		})
	}
}

func TestLogout(t *testing.T) {
	server := mockIDRAC(t, 0, "index.html")
	defer server.Close()

	c := NewClient("localhost", "root", "calvin")
	c.baseURL = server.URL
	c.http = server.Client()

	_ = c.Login()
	if err := c.Logout(); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if c.sessionID != "" {
		t.Error("sessionID should be empty after logout")
	}
}

func TestHost(t *testing.T) {
	c := NewClient("10.0.0.1", "admin", "pass")
	if c.Host() != "10.0.0.1" {
		t.Errorf("Host() = %q, want 10.0.0.1", c.Host())
	}
	if c.BaseURL() != "https://10.0.0.1" {
		t.Errorf("BaseURL() = %q, want https://10.0.0.1", c.BaseURL())
	}
}
