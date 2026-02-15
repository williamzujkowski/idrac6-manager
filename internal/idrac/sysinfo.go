package idrac

import (
	"encoding/xml"
	"fmt"
)

// SystemInfo holds system identification data.
type SystemInfo struct {
	Hostname    string `json:"hostname"`
	Model       string `json:"model"`
	ServiceTag  string `json:"serviceTag"`
	BIOSVersion string `json:"biosVersion"`
	FWVersion   string `json:"fwVersion"`
	LCCVersion  string `json:"lccVersion"`
	OSName      string `json:"osName,omitempty"`
}

type sysInfoResponse struct {
	XMLName      xml.Name `xml:"root"`
	HostName     string   `xml:"hostName"`
	SysDesc      string   `xml:"sysDesc"`
	SysRev       string   `xml:"sysRev"`
	BiosVer      string   `xml:"biosVer"`
	FwVersion    string   `xml:"fwVersion"`
	LCCfwVersion string   `xml:"LCCfwVersion"`
	OSName       string   `xml:"osName"`
	SvcTag       string   `xml:"svcTag"`
}

// GetSystemInfo returns system identification and firmware info.
func (c *Client) GetSystemInfo() (*SystemInfo, error) {
	data, err := c.Get("hostName", "sysDesc", "sysRev", "biosVer", "fwVersion", "LCCfwVersion", "osName", "svcTag")
	if err != nil {
		return nil, fmt.Errorf("getting system info: %w", err)
	}

	var resp sysInfoResponse
	if err := xml.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing system info: %w", err)
	}

	return &SystemInfo{
		Hostname:    resp.HostName,
		Model:       resp.SysDesc,
		ServiceTag:  resp.SvcTag,
		BIOSVersion: resp.BiosVer,
		FWVersion:   resp.FwVersion,
		LCCVersion:  resp.LCCfwVersion,
		OSName:      resp.OSName,
	}, nil
}
