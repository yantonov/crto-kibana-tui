package models

import "time"

// LogEntry is the normalized log record shown in the TUI.
type LogEntry struct {
	Timestamp   time.Time
	Severity    string
	Application string
	TraceID     string
	Message     string
	DataCenter  string
	Environment string
	RawJSON     string // original JSON for detail view / export
}

// Filter holds all user-selected search parameters.
type Filter struct {
	Severity    int    // SeverityAll (-1) = no filter; 0–7 = syslog level
	Application string // empty = all
	TraceID     string // empty = skip
	Query       string // free-text Lucene/KQL query string
	Timeframe   string // e.g. "1h", "30m"
	Environment string // "prod" | "preprod"
}

// SearchResult is returned by one datacenter search goroutine.
type SearchResult struct {
	DataCenter string
	Entries    []LogEntry
	Err        error
}

// CombinedResult is the merged, sorted output of all parallel searches.
type CombinedResult struct {
	Entries   []LogEntry
	DCErrors  map[string]error // per-DC errors (partial success allowed)
	TotalHits int
}
