package idrac

import (
	"testing"
)

func TestParseXMLSensors(t *testing.T) {
	sensors := []sensorXML{
		{
			Status:     "Normal",
			Name:       "system board ambient",
			Reading:    "20",
			Units:      "C",
			MinWarning: "8",
			MaxWarning: "42",
			MinFailure: "3",
			MaxFailure: "47",
		},
		{
			Status:     "Normal",
			Name:       "cpu1",
			Reading:    "45",
			Units:      "C",
			MaxWarning: "85",
			MaxFailure: "90",
		},
	}

	readings := parseXMLSensors(sensors)
	if len(readings) != 2 {
		t.Fatalf("got %d readings, want 2", len(readings))
	}

	if readings[0].Name != "system board ambient" {
		t.Errorf("name = %q, want system board ambient", readings[0].Name)
	}
	if readings[0].Value != 20 {
		t.Errorf("value = %f, want 20", readings[0].Value)
	}
	if readings[0].Unit != "C" {
		t.Errorf("unit = %q, want C", readings[0].Unit)
	}
	if readings[0].Status != "normal" {
		t.Errorf("status = %q, want normal", readings[0].Status)
	}
	if readings[0].Warning != 42 {
		t.Errorf("warning = %f, want 42", readings[0].Warning)
	}
	if readings[0].Critical != 47 {
		t.Errorf("critical = %f, want 47", readings[0].Critical)
	}
}

func TestParseXMLSensors_Fans(t *testing.T) {
	sensors := []sensorXML{
		{
			Status:     "Normal",
			Name:       "system board 1",
			Reading:    "1440",
			Units:      "RPM",
			MinWarning: "N/A",
			MaxWarning: "N/A",
			MinFailure: "720",
			MaxFailure: "N/A",
		},
	}

	readings := parseXMLSensors(sensors)
	if len(readings) != 1 {
		t.Fatalf("got %d readings, want 1", len(readings))
	}
	if readings[0].Value != 1440 {
		t.Errorf("value = %f, want 1440", readings[0].Value)
	}
	// N/A should parse to 0
	if readings[0].Warning != 0 {
		t.Errorf("warning = %f, want 0 (N/A)", readings[0].Warning)
	}
}

func TestParseLegacySensors(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		unit      string
		wantCount int
		wantFirst string
		wantValue float64
	}{
		{
			name:      "pipe delimited",
			raw:       "Inlet Temp=23;ok;42;47|Exhaust Temp=35;ok;70;75",
			unit:      "C",
			wantCount: 2,
			wantFirst: "Inlet Temp",
			wantValue: 23,
		},
		{
			name:      "single sensor",
			raw:       "CPU Temp=65;ok;90;95",
			unit:      "C",
			wantCount: 1,
			wantFirst: "CPU Temp",
			wantValue: 65,
		},
		{
			name:      "empty string",
			raw:       "",
			unit:      "C",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			readings := parseLegacySensors(tt.raw, tt.unit)
			if len(readings) != tt.wantCount {
				t.Errorf("got %d readings, want %d", len(readings), tt.wantCount)
			}
			if tt.wantCount > 0 && readings[0].Name != tt.wantFirst {
				t.Errorf("first name = %q, want %q", readings[0].Name, tt.wantFirst)
			}
			if tt.wantCount > 0 && readings[0].Value != tt.wantValue {
				t.Errorf("first value = %f, want %f", readings[0].Value, tt.wantValue)
			}
		})
	}
}

func TestParseSensorEntry(t *testing.T) {
	tests := []struct {
		name     string
		entry    string
		unit     string
		wantName string
		wantVal  float64
		wantStat string
		wantWarn float64
		wantCrit float64
	}{
		{
			name:     "full entry",
			entry:    "Inlet Temp=23;ok;42;47",
			unit:     "C",
			wantName: "Inlet Temp",
			wantVal:  23,
			wantStat: "ok",
			wantWarn: 42,
			wantCrit: 47,
		},
		{
			name:     "value only",
			entry:    "Fan1=5400",
			unit:     "RPM",
			wantName: "Fan1",
			wantVal:  5400,
			wantStat: "ok",
		},
		{
			name:  "empty entry",
			entry: "",
			unit:  "C",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := parseSensorEntry(tt.entry, tt.unit)
			if r.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", r.Name, tt.wantName)
			}
			if r.Value != tt.wantVal {
				t.Errorf("Value = %f, want %f", r.Value, tt.wantVal)
			}
			if r.Status != tt.wantStat && tt.wantStat != "" {
				t.Errorf("Status = %q, want %q", r.Status, tt.wantStat)
			}
		})
	}
}

func TestSplitSensors(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantCount int
	}{
		{"pipe delimited", "a|b|c", 3},
		{"newline delimited", "a\nb\nc", 3},
		{"single", "abc", 1},
		{"empty", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitSensors(tt.raw)
			if len(result) != tt.wantCount {
				t.Errorf("got %d entries, want %d", len(result), tt.wantCount)
			}
		})
	}
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"42", 42},
		{"3.14", 3.14},
		{" 23 ", 23},
		{"", 0},
		{"abc", 0},
		{"N/A", 0},
	}

	for _, tt := range tests {
		got := parseFloat(tt.input)
		if got != tt.want {
			t.Errorf("parseFloat(%q) = %f, want %f", tt.input, got, tt.want)
		}
	}
}
