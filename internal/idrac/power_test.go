package idrac

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func mockIDRACWithPower(t *testing.T, pwState string) *httptest.Server {
	t.Helper()
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/start.html":
			http.SetCookie(w, &http.Cookie{Name: "_appwebSessionId_", Value: "sess"})
			fmt.Fprint(w, `<html></html>`)
		case "/data/login":
			fmt.Fprint(w, `<root><authResult>0</authResult><forwardUrl>index.html</forwardUrl></root>`)
		case "/data":
			if r.URL.Query().Get("set") != "" {
				fmt.Fprint(w, `<root><status>ok</status></root>`)
			} else {
				fmt.Fprintf(w, `<root><pwState>%s</pwState></root>`, pwState)
			}
		}
	}))
}

func TestGetPowerState(t *testing.T) {
	tests := []struct {
		name       string
		pwState    string
		wantState  PowerState
		wantStatus string
	}{
		{"power on", "1", PowerOn, "on"},
		{"power off", "0", PowerOff, "off"},
		{"invalid", "2", PowerInvalid, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := mockIDRACWithPower(t, tt.pwState)
			defer server.Close()

			c := NewClient("localhost", "root", "calvin")
			c.baseURL = server.URL
			c.http = server.Client()
			_ = c.Login()

			status, err := c.GetPowerState()
			if err != nil {
				t.Fatalf("GetPowerState() error = %v", err)
			}
			if status.State != tt.wantState {
				t.Errorf("State = %v, want %v", status.State, tt.wantState)
			}
			if status.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", status.Status, tt.wantStatus)
			}
		})
	}
}

func TestSetPowerByName(t *testing.T) {
	server := mockIDRACWithPower(t, "1")
	defer server.Close()

	c := NewClient("localhost", "root", "calvin")
	c.baseURL = server.URL
	c.http = server.Client()
	_ = c.Login()

	for _, action := range []string{"on", "off", "restart", "reset", "nmi", "shutdown"} {
		if err := c.SetPowerByName(action); err != nil {
			t.Errorf("SetPowerByName(%q) error = %v", action, err)
		}
	}

	if err := c.SetPowerByName("invalid"); err == nil {
		t.Error("SetPowerByName(invalid) should fail")
	}
}

func TestPowerStateString(t *testing.T) {
	if PowerOn.String() != "on" {
		t.Errorf("PowerOn.String() = %q, want on", PowerOn.String())
	}
	if PowerOff.String() != "off" {
		t.Errorf("PowerOff.String() = %q, want off", PowerOff.String())
	}
	if PowerInvalid.String() != "unknown" {
		t.Errorf("PowerInvalid.String() = %q, want unknown", PowerInvalid.String())
	}
}
