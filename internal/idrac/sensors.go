package idrac

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// SensorReading represents a single sensor value.
type SensorReading struct {
	Name     string  `json:"name"`
	Value    float64 `json:"value"`
	Unit     string  `json:"unit"`
	Status   string  `json:"status"`
	Warning  float64 `json:"warning,omitempty"`
	Critical float64 `json:"critical,omitempty"`
}

// SensorData holds all sensor readings grouped by type.
type SensorData struct {
	Temperatures []SensorReading `json:"temperatures"`
	Fans         []SensorReading `json:"fans"`
	Voltages     []SensorReading `json:"voltages"`
}

// Temperature XML structures from iDRAC6
type temperaturesResponse struct {
	XMLName      xml.Name `xml:"root"`
	Temperatures string   `xml:"temperatures"`
}

type fansResponse struct {
	XMLName xml.Name `xml:"root"`
	Fans    string   `xml:"fans"`
}

type voltagesResponse struct {
	XMLName  xml.Name `xml:"root"`
	Voltages string   `xml:"voltages"`
}

// GetSensors returns all sensor readings (temperatures, fans, voltages).
func (c *Client) GetSensors() (*SensorData, error) {
	data, err := c.Get("temperatures", "fans", "voltages")
	if err != nil {
		return nil, fmt.Errorf("getting sensor data: %w", err)
	}

	result := &SensorData{}

	// Parse temperatures
	var tempResp temperaturesResponse
	if err := xml.Unmarshal(data, &tempResp); err == nil && tempResp.Temperatures != "" {
		result.Temperatures = parseTemperatures(tempResp.Temperatures)
	}

	// Parse fans
	var fanResp fansResponse
	if err := xml.Unmarshal(data, &fanResp); err == nil && fanResp.Fans != "" {
		result.Fans = parseFans(fanResp.Fans)
	}

	// Parse voltages
	var voltResp voltagesResponse
	if err := xml.Unmarshal(data, &voltResp); err == nil && voltResp.Voltages != "" {
		result.Voltages = parseVoltages(voltResp.Voltages)
	}

	return result, nil
}

// GetTemperatures returns temperature sensor readings.
func (c *Client) GetTemperatures() ([]SensorReading, error) {
	data, err := c.Get("temperatures")
	if err != nil {
		return nil, fmt.Errorf("getting temperatures: %w", err)
	}

	var resp temperaturesResponse
	if err := xml.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing temperatures: %w", err)
	}

	return parseTemperatures(resp.Temperatures), nil
}

// parseTemperatures parses the iDRAC6 temperature format.
// Format: "SensorName1=value1;status1;warn1;crit1|SensorName2=..."
// The exact format varies by firmware, so we handle multiple variants.
func parseTemperatures(raw string) []SensorReading {
	var readings []SensorReading

	// iDRAC6 sends sensor data in various formats depending on firmware
	// Common format: each sensor separated by some delimiter
	sensors := splitSensors(raw)
	for _, sensor := range sensors {
		r := parseSensorEntry(sensor, "C")
		if r.Name != "" {
			readings = append(readings, r)
		}
	}

	return readings
}

// parseFans parses fan sensor data.
func parseFans(raw string) []SensorReading {
	var readings []SensorReading

	sensors := splitSensors(raw)
	for _, sensor := range sensors {
		r := parseSensorEntry(sensor, "RPM")
		if r.Name != "" {
			readings = append(readings, r)
		}
	}

	return readings
}

// parseVoltages parses voltage sensor data.
func parseVoltages(raw string) []SensorReading {
	var readings []SensorReading

	sensors := splitSensors(raw)
	for _, sensor := range sensors {
		r := parseSensorEntry(sensor, "V")
		if r.Name != "" {
			readings = append(readings, r)
		}
	}

	return readings
}

// splitSensors splits a raw sensor string into individual entries.
// Handles both "|" and newline delimiters.
func splitSensors(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	// Try pipe-delimited first (most common)
	if strings.Contains(raw, "|") {
		return strings.Split(raw, "|")
	}
	// Try newline-delimited
	if strings.Contains(raw, "\n") {
		return strings.Split(raw, "\n")
	}
	// Single sensor
	return []string{raw}
}

// parseSensorEntry parses a single sensor entry like "Inlet Temp=23;ok;42;47"
func parseSensorEntry(entry string, unit string) SensorReading {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return SensorReading{}
	}

	r := SensorReading{Unit: unit, Status: "ok"}

	// Split name=value
	parts := strings.SplitN(entry, "=", 2)
	if len(parts) == 2 {
		r.Name = strings.TrimSpace(parts[0])
		valuePart := parts[1]

		// Split value;status;warning;critical
		fields := strings.Split(valuePart, ";")
		if len(fields) >= 1 {
			r.Value = parseFloat(fields[0])
		}
		if len(fields) >= 2 {
			r.Status = strings.TrimSpace(fields[1])
		}
		if len(fields) >= 3 {
			r.Warning = parseFloat(fields[2])
		}
		if len(fields) >= 4 {
			r.Critical = parseFloat(fields[3])
		}
	} else {
		// No = sign, try as just a name:value or raw value
		r.Name = entry
	}

	return r
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
