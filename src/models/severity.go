package models

import "fmt"

// SeverityAll is the sentinel value for Filter.Severity meaning "no filter".
const SeverityAll = -1

var severityNames = [8]string{
	"Emergency",
	"Alert",
	"Critical",
	"Error",
	"Warning",
	"Notice",
	"Info",
	"Debug",
}

// SeverityName returns the human-readable syslog name for a level code (0–7).
func SeverityName(code int) string {
	if code >= 0 && code < len(severityNames) {
		return severityNames[code]
	}
	return "Unknown"
}

// SeverityLabel returns "Name (N)" for display in the TUI.
func SeverityLabel(code int) string {
	return fmt.Sprintf("%s (%d)", SeverityName(code), code)
}

// AllSeverityOptions returns (label, code) pairs ordered Emergency→Debug.
func AllSeverityOptions() [][2]string {
	opts := make([][2]string, len(severityNames))
	for i, name := range severityNames {
		opts[i] = [2]string{fmt.Sprintf("%s (%d)", name, i), fmt.Sprintf("%d", i)}
	}
	return opts
}
