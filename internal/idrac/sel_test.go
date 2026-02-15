package idrac

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseSEL(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantCount int
	}{
		{"pipe delimited", "1|2024-01-01 12:00:00|Normal|System Boot\n2|2024-01-01 12:05:00|Warning|Temperature above threshold", 2},
		{"empty", "", 0},
		{"semicolon delimited", "1;2024-01-01;Critical;Disk failure", 1},
		{"fallback single line", "Unknown event data", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := parseSEL(tt.raw)
			if len(entries) != tt.wantCount {
				t.Errorf("got %d entries, want %d", len(entries), tt.wantCount)
			}
		})
	}
}

func TestParseSELLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantID   string
		wantSev  string
		wantDesc string
	}{
		{"pipe format", "42|2024-06-15 10:30:00|Normal|System powered on", "42", "Normal", "System powered on"},
		{"semicolon format", "7;2024-06-15;Critical;PSU failure", "7", "Critical", "PSU failure"},
		{"fallback", "raw event text", "0", "Unknown", "raw event text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := parseSELLine(tt.line)
			if e.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", e.ID, tt.wantID)
			}
			if e.Severity != tt.wantSev {
				t.Errorf("Severity = %q, want %q", e.Severity, tt.wantSev)
			}
			if e.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", e.Description, tt.wantDesc)
			}
		})
	}
}

func TestGetSEL(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/start.html":
			http.SetCookie(w, &http.Cookie{Name: "_appwebSessionId_", Value: "sess"})
			fmt.Fprint(w, `<html></html>`)
		case "/data/login":
			fmt.Fprint(w, `<root><authResult>0</authResult><forwardUrl>index.html</forwardUrl></root>`)
		case "/data":
			fmt.Fprint(w, `<root><sel>1|2024-01-01 12:00:00|Normal|Boot
2|2024-01-01 12:05:00|Warning|Temp high</sel></root>`)
		}
	}))
	defer server.Close()

	c := NewClient("localhost", "root", "calvin")
	c.baseURL = server.URL
	c.http = server.Client()
	_ = c.Login()

	sel, err := c.GetSEL()
	if err != nil {
		t.Fatalf("GetSEL() error = %v", err)
	}

	if sel.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", sel.TotalCount)
	}
	if sel.Entries[0].Description != "Boot" {
		t.Errorf("first entry description = %q, want Boot", sel.Entries[0].Description)
	}
}
