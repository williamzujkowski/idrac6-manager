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

// XML structures for iDRAC6 sensor responses
// The actual format uses nested XML elements:
// <root><sensortype><thresholdSensorList><sensor>...</sensor></thresholdSensorList></sensortype></root>

type sensorXMLRoot struct {
	XMLName   xml.Name       `xml:"root"`
	Sensors   sensorTypeWrap `xml:"sensortype"`
	PowerOn   string         `xml:"powerOn"`
	RawTemps  string         `xml:"temperatures"`
	RawFans   string         `xml:"fans"`
	RawVolts  string         `xml:"voltages"`
}

type sensorTypeWrap struct {
	SensorID  string        `xml:"sensorid"`
	Threshold sensorListXML `xml:"thresholdSensorList"`
}

type sensorListXML struct {
	Sensors []sensorXML `xml:"sensor"`
}

type sensorXML struct {
	Status     string `xml:"sensorStatus"`
	Name       string `xml:"name"`
	Reading    string `xml:"reading"`
	Units      string `xml:"units"`
	MinWarning string `xml:"minWarning"`
	MaxWarning string `xml:"maxWarning"`
	MinFailure string `xml:"minFailure"`
	MaxFailure string `xml:"maxFailure"`
}

// GetSensors returns all sensor readings (temperatures, fans, voltages).
// Makes separate requests for each sensor type since iDRAC6 returns
// different XML structures per type.
func (c *Client) GetSensors() (*SensorData, error) {
	result := &SensorData{}

	// Get temperatures (sensorid=1)
	temps, err := c.getSensorType("temperatures")
	if err == nil {
		result.Temperatures = temps
	}

	// Get fans (sensorid=4)
	fans, err := c.getSensorType("fans")
	if err == nil {
		result.Fans = fans
	}

	// Get voltages (sensorid=2)
	volts, err := c.getSensorType("voltages")
	if err == nil {
		result.Voltages = volts
	}

	return result, nil
}

// getSensorType fetches and parses a single sensor type.
func (c *Client) getSensorType(sensorType string) ([]SensorReading, error) {
	data, err := c.Get(sensorType)
	if err != nil {
		return nil, fmt.Errorf("getting %s: %w", sensorType, err)
	}

	// Try XML element format first (proper iDRAC6 response)
	var root sensorXMLRoot
	if err := xml.Unmarshal(data, &root); err == nil {
		if len(root.Sensors.Threshold.Sensors) > 0 {
			return parseXMLSensors(root.Sensors.Threshold.Sensors), nil
		}
	}

	// Fallback: try legacy pipe-delimited string format
	raw := extractRawSensorString(data, sensorType)
	if raw != "" {
		unit := sensorTypeUnit(sensorType)
		return parseLegacySensors(raw, unit), nil
	}

	return nil, nil
}

// GetTemperatures returns temperature sensor readings.
func (c *Client) GetTemperatures() ([]SensorReading, error) {
	return c.getSensorType("temperatures")
}

// parseXMLSensors converts XML sensor elements to SensorReadings.
func parseXMLSensors(sensors []sensorXML) []SensorReading {
	var readings []SensorReading
	for _, s := range sensors {
		r := SensorReading{
			Name:   s.Name,
			Value:  parseFloat(s.Reading),
			Unit:   s.Units,
			Status: strings.ToLower(s.Status),
		}

		// Use maxWarning/maxFailure as thresholds
		if w := parseFloat(s.MaxWarning); w > 0 {
			r.Warning = w
		}
		if c := parseFloat(s.MaxFailure); c > 0 {
			r.Critical = c
		}

		readings = append(readings, r)
	}
	return readings
}

// extractRawSensorString extracts a raw sensor string from XML for legacy format.
func extractRawSensorString(data []byte, sensorType string) string {
	type genericRoot struct {
		XMLName      xml.Name `xml:"root"`
		Temperatures string   `xml:"temperatures"`
		Fans         string   `xml:"fans"`
		Voltages     string   `xml:"voltages"`
	}

	var gr genericRoot
	if err := xml.Unmarshal(data, &gr); err != nil {
		return ""
	}

	switch sensorType {
	case "temperatures":
		return gr.Temperatures
	case "fans":
		return gr.Fans
	case "voltages":
		return gr.Voltages
	}
	return ""
}

func sensorTypeUnit(sensorType string) string {
	switch sensorType {
	case "temperatures":
		return "C"
	case "fans":
		return "RPM"
	case "voltages":
		return "V"
	default:
		return ""
	}
}

// parseLegacySensors handles the pipe-delimited format: "Name=value;status;warn;crit|..."
func parseLegacySensors(raw string, unit string) []SensorReading {
	var readings []SensorReading
	for _, entry := range splitSensors(raw) {
		r := parseSensorEntry(entry, unit)
		if r.Name != "" {
			readings = append(readings, r)
		}
	}
	return readings
}

// splitSensors splits a raw sensor string into individual entries.
func splitSensors(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if strings.Contains(raw, "|") {
		return strings.Split(raw, "|")
	}
	if strings.Contains(raw, "\n") {
		return strings.Split(raw, "\n")
	}
	return []string{raw}
}

// parseSensorEntry parses a single legacy sensor entry like "Inlet Temp=23;ok;42;47"
func parseSensorEntry(entry string, unit string) SensorReading {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return SensorReading{}
	}

	r := SensorReading{Unit: unit, Status: "ok"}

	parts := strings.SplitN(entry, "=", 2)
	if len(parts) == 2 {
		r.Name = strings.TrimSpace(parts[0])
		fields := strings.Split(parts[1], ";")
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
		r.Name = entry
	}

	return r
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "N/A" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
