package ipmi

import "testing"

func TestNewClient(t *testing.T) {
	c := NewClient("10.0.0.1", 0, "root", "pass")

	if c.host != "10.0.0.1" {
		t.Errorf("host = %q, want 10.0.0.1", c.host)
	}
	if c.port != 623 {
		t.Errorf("port = %d, want 623 (default)", c.port)
	}
	if c.username != "root" {
		t.Errorf("username = %q, want root", c.username)
	}
}

func TestNewClient_CustomPort(t *testing.T) {
	c := NewClient("10.0.0.1", 624, "admin", "secret")

	if c.port != 624 {
		t.Errorf("port = %d, want 624", c.port)
	}
}
