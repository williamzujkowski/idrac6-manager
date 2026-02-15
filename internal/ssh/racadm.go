// Package ssh provides RACADM command execution over SSH.
package ssh

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// RACAdm executes RACADM commands over SSH on an iDRAC6.
type RACAdm struct {
	host     string
	port     int
	username string
	password string
}

// NewRACAdm creates a new RACADM SSH executor.
func NewRACAdm(host string, port int, username, password string) *RACAdm {
	if port == 0 {
		port = 22
	}
	return &RACAdm{
		host:     host,
		port:     port,
		username: username,
		password: password,
	}
}

// Run executes a RACADM command and returns stdout.
func (r *RACAdm) Run(args ...string) (string, error) {
	cmd := "racadm " + strings.Join(args, " ")

	config := &ssh.ClientConfig{
		User: r.username,
		Auth: []ssh.AuthMethod{
			ssh.Password(r.password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // iDRAC6 has no CA
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", r.host, r.port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return "", fmt.Errorf("SSH connect to %s: %w", addr, err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("SSH session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(cmd); err != nil {
		return "", fmt.Errorf("RACADM command %q: %w (stderr: %s)", cmd, err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}
