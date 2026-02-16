package ssh

import "testing"

func TestNewRACAdm(t *testing.T) {
	r := NewRACAdm("10.0.0.1", 0, "root", "pass")

	if r.host != "10.0.0.1" {
		t.Errorf("host = %q, want 10.0.0.1", r.host)
	}
	if r.port != 22 {
		t.Errorf("port = %d, want 22 (default)", r.port)
	}
	if r.username != "root" {
		t.Errorf("username = %q, want root", r.username)
	}
}

func TestNewRACAdm_CustomPort(t *testing.T) {
	r := NewRACAdm("10.0.0.1", 2222, "admin", "secret")

	if r.port != 2222 {
		t.Errorf("port = %d, want 2222", r.port)
	}
}

func TestRun_ConnectionError(t *testing.T) {
	// Use a port that won't have an SSH server
	r := NewRACAdm("127.0.0.1", 19999, "root", "pass")

	_, err := r.Run("getsysinfo")
	if err == nil {
		t.Fatal("Run() should fail with connection error")
	}
}
