package idrac

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetPowerState(t *testing.T) {
	tests := []struct {
		name       string
		xmlResp    string
		wantState  PowerState
		wantStatus string
	}{
		{
			name:       "power on",
			xmlResp:    `<root><pwState>1</pwState></root>`,
			wantState:  PowerOn,
			wantStatus: "on",
		},
		{
			name:       "power off",
			xmlResp:    `<root><pwState>0</pwState></root>`,
			wantState:  PowerOff,
			wantStatus: "off",
		},
		{
			name:       "invalid state",
			xmlResp:    `<root><pwState>2</pwState></root>`,
			wantState:  PowerInvalid,
			wantStatus: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/data/login" {
					http.SetCookie(w, &http.Cookie{Name: "_appwebSessionId_", Value: "sess"})
					fmt.Fprint(w, `<root><authResult>0</authResult><forwardUrl>index.html</forwardUrl></root>`)
					return
				}
				fmt.Fprint(w, tt.xmlResp)
			}))
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
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/data/login" {
			http.SetCookie(w, &http.Cookie{Name: "_appwebSessionId_", Value: "sess"})
			fmt.Fprint(w, `<root><authResult>0</authResult><forwardUrl>index.html</forwardUrl></root>`)
			return
		}
		fmt.Fprint(w, `<root><status>ok</status></root>`)
	}))
	defer server.Close()

	c := NewClient("localhost", "root", "calvin")
	c.baseURL = server.URL
	c.http = server.Client()
	_ = c.Login()

	// Valid actions
	for _, action := range []string{"on", "off", "restart", "reset", "nmi", "shutdown"} {
		if err := c.SetPowerByName(action); err != nil {
			t.Errorf("SetPowerByName(%q) error = %v", action, err)
		}
	}

	// Invalid action
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
