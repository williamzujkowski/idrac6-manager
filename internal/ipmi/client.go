// Package ipmi provides an IPMI 2.0 client for iDRAC6 using bougou/go-ipmi.
package ipmi

import (
	"context"
	"fmt"
	"time"

	goipmi "github.com/bougou/go-ipmi"
)

// Client wraps the go-ipmi client for iDRAC6 operations.
type Client struct {
	host     string
	port     int
	username string
	password string
}

// NewClient creates a new IPMI client.
func NewClient(host string, port int, username, password string) *Client {
	if port == 0 {
		port = 623
	}
	return &Client{
		host:     host,
		port:     port,
		username: username,
		password: password,
	}
}

func (c *Client) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

// connect creates an authenticated IPMI connection.
func (c *Client) connect() (*goipmi.Client, error) {
	client, err := goipmi.NewClient(c.host, c.port, c.username, c.password)
	if err != nil {
		return nil, fmt.Errorf("creating IPMI client: %w", err)
	}

	client.WithInterface(goipmi.InterfaceLanplus)

	ctx, cancel := c.ctx()
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		return nil, fmt.Errorf("IPMI connect to %s:%d: %w", c.host, c.port, err)
	}

	return client, nil
}

// GetPowerStatus returns the chassis power status via IPMI.
func (c *Client) GetPowerStatus() (bool, error) {
	client, err := c.connect()
	if err != nil {
		return false, err
	}
	ctx, cancel := c.ctx()
	defer cancel()
	defer client.Close(ctx) //nolint:errcheck

	status, err := client.GetChassisStatus(ctx)
	if err != nil {
		return false, fmt.Errorf("IPMI chassis status: %w", err)
	}

	return status.PowerIsOn, nil
}

// PowerOn turns on the chassis.
func (c *Client) PowerOn() error {
	return c.chassisControl(goipmi.ChassisControlPowerUp)
}

// PowerOff turns off the chassis.
func (c *Client) PowerOff() error {
	return c.chassisControl(goipmi.ChassisControlPowerDown)
}

// PowerCycle power cycles the chassis.
func (c *Client) PowerCycle() error {
	return c.chassisControl(goipmi.ChassisControlPowerCycle)
}

// HardReset hard resets the chassis.
func (c *Client) HardReset() error {
	return c.chassisControl(goipmi.ChassisControlHardReset)
}

func (c *Client) chassisControl(control goipmi.ChassisControl) error {
	client, err := c.connect()
	if err != nil {
		return err
	}
	ctx, cancel := c.ctx()
	defer cancel()
	defer client.Close(ctx) //nolint:errcheck

	if _, err := client.ChassisControl(ctx, control); err != nil {
		return fmt.Errorf("IPMI chassis control: %w", err)
	}
	return nil
}

// GetSEL returns the System Event Log entries via IPMI.
func (c *Client) GetSEL() ([]SELEntry, error) {
	client, err := c.connect()
	if err != nil {
		return nil, err
	}
	ctx, cancel := c.ctx()
	defer cancel()
	defer client.Close(ctx) //nolint:errcheck

	entries, err := client.GetSELEntries(ctx, 0)
	if err != nil {
		return nil, fmt.Errorf("IPMI SEL entries: %w", err)
	}

	var result []SELEntry
	for _, e := range entries {
		entry := SELEntry{
			ID: fmt.Sprintf("%d", e.RecordID),
		}
		if e.Standard != nil {
			entry.Timestamp = e.Standard.Timestamp.Format(time.RFC3339)
			entry.SensorType = e.Standard.SensorType.String()
		}
		result = append(result, entry)
	}

	return result, nil
}

// SELEntry represents an IPMI SEL entry.
type SELEntry struct {
	ID         string `json:"id"`
	Timestamp  string `json:"timestamp,omitempty"`
	SensorType string `json:"sensorType,omitempty"`
}
