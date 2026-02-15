package idrac

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// SELEntry represents a System Event Log entry.
type SELEntry struct {
	ID          string `json:"id"`
	Timestamp   string `json:"timestamp"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Entity      string `json:"entity,omitempty"`
}

// SELData holds the system event log.
type SELData struct {
	Entries    []SELEntry `json:"entries"`
	TotalCount int        `json:"totalCount"`
}

type selResponse struct {
	XMLName xml.Name `xml:"root"`
	SEL     string   `xml:"sel"`
}

// GetSEL returns the System Event Log entries.
func (c *Client) GetSEL() (*SELData, error) {
	data, err := c.Get("sel")
	if err != nil {
		return nil, fmt.Errorf("getting SEL: %w", err)
	}

	var resp selResponse
	if err := xml.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing SEL: %w", err)
	}

	entries := parseSEL(resp.SEL)

	return &SELData{
		Entries:    entries,
		TotalCount: len(entries),
	}, nil
}

// ClearSEL clears the System Event Log.
func (c *Client) ClearSEL() error {
	_, err := c.Set("selClr:1")
	if err != nil {
		return fmt.Errorf("clearing SEL: %w", err)
	}
	return nil
}

// parseSEL parses the raw SEL string into structured entries.
// Format varies by firmware but typically: "id|timestamp|severity|description\n..."
func parseSEL(raw string) []SELEntry {
	var entries []SELEntry

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return entries
	}

	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		entry := parseSELLine(line)
		if entry.ID != "" {
			entries = append(entries, entry)
		}
	}

	return entries
}

// parseSELLine parses a single SEL entry line.
func parseSELLine(line string) SELEntry {
	// Try pipe-delimited format: "1|2024-01-01 12:00:00|Normal|System Boot"
	parts := strings.SplitN(line, "|", 4)
	if len(parts) >= 4 {
		return SELEntry{
			ID:          strings.TrimSpace(parts[0]),
			Timestamp:   strings.TrimSpace(parts[1]),
			Severity:    strings.TrimSpace(parts[2]),
			Description: strings.TrimSpace(parts[3]),
		}
	}

	// Try semicolon-delimited
	parts = strings.SplitN(line, ";", 4)
	if len(parts) >= 4 {
		return SELEntry{
			ID:          strings.TrimSpace(parts[0]),
			Timestamp:   strings.TrimSpace(parts[1]),
			Severity:    strings.TrimSpace(parts[2]),
			Description: strings.TrimSpace(parts[3]),
		}
	}

	// Fallback: treat entire line as description
	return SELEntry{
		ID:          "0",
		Description: line,
		Severity:    "Unknown",
	}
}
