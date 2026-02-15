package idrac

import (
	"encoding/xml"
	"fmt"
)

// PowerState represents the server power state.
type PowerState int

const (
	PowerOff     PowerState = 0
	PowerOn      PowerState = 1
	PowerInvalid PowerState = 2
)

func (s PowerState) String() string {
	switch s {
	case PowerOff:
		return "off"
	case PowerOn:
		return "on"
	default:
		return "unknown"
	}
}

// PowerAction represents a power control action.
type PowerAction int

const (
	ActionPowerOff      PowerAction = 0
	ActionPowerOn       PowerAction = 1
	ActionPowerRestart  PowerAction = 2
	ActionPowerReset    PowerAction = 3
	ActionNMI           PowerAction = 4
	ActionGracefulShut  PowerAction = 5
)

// ValidPowerActions maps action names to their numeric values.
var ValidPowerActions = map[string]PowerAction{
	"off":      ActionPowerOff,
	"on":       ActionPowerOn,
	"restart":  ActionPowerRestart,
	"reset":    ActionPowerReset,
	"nmi":      ActionNMI,
	"shutdown": ActionGracefulShut,
}

type powerResponse struct {
	XMLName  xml.Name `xml:"root"`
	PwState  string   `xml:"pwState"`
}

// PowerStatus holds the current power state.
type PowerStatus struct {
	State  PowerState `json:"state"`
	Status string     `json:"status"`
}

// GetPowerState returns the current power state.
func (c *Client) GetPowerState() (*PowerStatus, error) {
	data, err := c.Get("pwState")
	if err != nil {
		return nil, fmt.Errorf("getting power state: %w", err)
	}

	var resp powerResponse
	if err := xml.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing power state: %w", err)
	}

	state := PowerInvalid
	switch resp.PwState {
	case "0":
		state = PowerOff
	case "1":
		state = PowerOn
	}

	return &PowerStatus{
		State:  state,
		Status: state.String(),
	}, nil
}

// SetPower executes a power action.
func (c *Client) SetPower(action PowerAction) error {
	_, err := c.Set(fmt.Sprintf("pwState:%d", action))
	if err != nil {
		return fmt.Errorf("setting power state: %w", err)
	}
	return nil
}

// SetPowerByName executes a power action by name.
func (c *Client) SetPowerByName(name string) error {
	action, ok := ValidPowerActions[name]
	if !ok {
		return fmt.Errorf("unknown power action: %q (valid: off, on, restart, reset, nmi, shutdown)", name)
	}
	return c.SetPower(action)
}
