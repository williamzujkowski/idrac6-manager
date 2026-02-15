package idrac

import (
	"fmt"
	"strings"

	racadmssh "github.com/williamzujkowski/idrac6-manager/internal/ssh"
)

// VirtualMediaStatus represents the current virtual media mount state.
type VirtualMediaStatus struct {
	Connected bool   `json:"connected"`
	URL       string `json:"url,omitempty"`
	Type      string `json:"type,omitempty"`
}

// VirtualMedia manages virtual media via RACADM over SSH.
type VirtualMedia struct {
	racadm *racadmssh.RACAdm
}

// NewVirtualMedia creates a new VirtualMedia manager.
func NewVirtualMedia(host string, port int, username, password string) *VirtualMedia {
	return &VirtualMedia{
		racadm: racadmssh.NewRACAdm(host, port, username, password),
	}
}

// GetStatus returns the current virtual media connection status.
func (vm *VirtualMedia) GetStatus() (*VirtualMediaStatus, error) {
	output, err := vm.racadm.Run("remoteimage", "-s")
	if err != nil {
		return nil, fmt.Errorf("checking virtual media status: %w", err)
	}

	status := &VirtualMediaStatus{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Image is") {
			if strings.Contains(line, "connected") {
				status.Connected = true
			}
		}
		if strings.HasPrefix(line, "Image Location") || strings.HasPrefix(line, "Share Name") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				status.URL = strings.TrimSpace(parts[1])
			}
		}
	}

	return status, nil
}

// Mount connects a remote image via NFS, CIFS, or HTTP.
func (vm *VirtualMedia) Mount(imageURL string) error {
	// Disconnect any existing image first
	_ = vm.Unmount()

	// racadm remoteimage -c -l <url>
	_, err := vm.racadm.Run("remoteimage", "-c", "-l", imageURL)
	if err != nil {
		return fmt.Errorf("mounting image %q: %w", imageURL, err)
	}

	return nil
}

// Unmount disconnects the current virtual media image.
func (vm *VirtualMedia) Unmount() error {
	_, err := vm.racadm.Run("remoteimage", "-d")
	if err != nil {
		return fmt.Errorf("unmounting image: %w", err)
	}
	return nil
}
