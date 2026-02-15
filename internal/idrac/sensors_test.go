package idrac

import (
	"testing"
)

func TestParseTemperatures(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantCount int
		wantFirst string
		wantValue float64
	}{
		{
			name:      "pipe delimited",
			raw:       "Inlet Temp=23;ok;42;47|Exhaust Temp=35;ok;70;75",
			wantCount: 2,
			wantFirst: "Inlet Temp",
			wantValue: 23,
		},
		{
			name:      "single sensor",
			raw:       "CPU Temp=65;ok;90;95",
			wantCount: 1,
			wantFirst: "CPU Temp",
			wantValue: 65,
		},
		{
			name:      "empty string",
			raw:       "",
			wantCount: 0,
		},
		{
			name:      "with warning status",
			raw:       "DIMM Temp=80;warning;85;90",
			wantCount: 1,
			wantFirst: "DIMM Temp",
			wantValue: 80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			readings := parseTemperatures(tt.raw)
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
			if r.Warning != tt.wantWarn {
				t.Errorf("Warning = %f, want %f", r.Warning, tt.wantWarn)
			}
			if r.Critical != tt.wantCrit {
				t.Errorf("Critical = %f, want %f", r.Critical, tt.wantCrit)
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
	}

	for _, tt := range tests {
		got := parseFloat(tt.input)
		if got != tt.want {
			t.Errorf("parseFloat(%q) = %f, want %f", tt.input, got, tt.want)
		}
	}
}
