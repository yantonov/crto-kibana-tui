# TUI OpenSearch Log Viewer — Implementation Plan

## Project Structure

```
klt/
├── main.go
├── go.mod / go.sum
├── klt.yaml                    # config file
├── config/
│   ├── schema.go               # Config structs
│   └── config.go               # YAML loading + validation
├── models/
│   └── log.go                  # LogEntry, Filter, SearchResult
├── opensearch/
│   ├── models.go               # HTTP response structs
│   ├── query.go                # DSL query builder
│   ├── client.go               # HTTP wrapper + Basic Auth
│   └── search.go               # Parallel fanout + merge/sort
├── tui/
│   ├── app.go                  # Root Bubble Tea model + screen routing
│   ├── styles.go               # Lip Gloss styles
│   ├── keys.go                 # Key bindings
│   ├── messages.go             # Shared tea.Msg types
│   ├── screens/
│   │   ├── filter.go           # Filter form (entry screen)
│   │   ├── results.go          # Results table
│   │   └── detail.go           # Single log detail/viewport
│   └── components/
│       ├── dropdown.go         # Custom enum selector
│       ├── spinner.go          # Loading overlay
│       └── statusbar.go        # Bottom status bar
└── export/
    ├── clipboard.go
    └── file.go
```

---

## Config File (`klt.yaml`)

```yaml
environments:
  prod:
    data_centers: [ams8, par7, sin5, lax1]
  preprod:
    data_centers: [ams8, par3]

kibana_url_template: "https://kibana.{dc}.{env}.crto.in"
opensearch_url_template: "https://opensearch.{dc}.{env}.crto.in"
index_pattern: "logs-*"
query_timeout_seconds: 10

applications:
  - payments-service
  - auth-service
  - api-gateway

severity_levels: [TRACE, DEBUG, INFO, WARN, ERROR, FATAL]

timeframes:
  - { label: "15 minutes", value: "15m" }
  - { label: "30 minutes", value: "30m" }
  - { label: "1 hour",     value: "1h"  }
  - { label: "3 hours",    value: "3h"  }
  - { label: "6 hours",    value: "6h"  }
  - { label: "12 hours",   value: "12h" }
  - { label: "24 hours",   value: "24h" }
  - { label: "2 days",     value: "2d"  }
  - { label: "7 days",     value: "7d"  }

field_mapping:
  timestamp:   "@timestamp"
  severity:    "severity"
  application: "application"
  trace_id:    "trace_id"
  message:     "message"
```

Credentials come from env vars: `OPENSEARCH_USERNAME` / `OPENSEARCH_PASSWORD`.

---

## Key Data Structures

### `models/log.go`

```go
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
    Severity    string // empty = all
    Application string // empty = all
    TraceID     string // empty = skip
    Query       string // free-text Lucene/KQL query string
    Timeframe   string // e.g. "1h", "30m"
    Environment string // "prod" | "preprod"
}

// SearchResult is returned by one datacenter search.
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
```

### `config/schema.go`

```go
package config

type Config struct {
    Environments          map[string]EnvironmentConfig `yaml:"environments"`
    KibanaURLTemplate     string                       `yaml:"kibana_url_template"`
    OpenSearchURLTemplate string                       `yaml:"opensearch_url_template"`
    IndexPattern          string                       `yaml:"index_pattern"`
    QueryTimeoutSeconds   int                          `yaml:"query_timeout_seconds"`
    Applications          []string                     `yaml:"applications"`
    SeverityLevels        []string                     `yaml:"severity_levels"`
    Timeframes            []TimeframeOption            `yaml:"timeframes"`
    FieldMapping          FieldMapping                 `yaml:"field_mapping"`
}

type EnvironmentConfig struct {
    DataCenters []string `yaml:"data_centers"`
}

type TimeframeOption struct {
    Label string `yaml:"label"`
    Value string `yaml:"value"`
}

type FieldMapping struct {
    Timestamp   string `yaml:"timestamp"`
    Severity    string `yaml:"severity"`
    Application string `yaml:"application"`
    TraceID     string `yaml:"trace_id"`
    Message     string `yaml:"message"`
}
```

---

## Screen Flow

```
FilterScreen ──[search]──► SpinnerOverlay ──[done]──► ResultsScreen ──[enter]──► DetailScreen
     ▲                                                      │                          │
     └──────────────────────────────────[r]─────────────────┘◄─────────[esc/b]────────┘
```

### FilterScreen (`tui/screens/filter.go`)

Tab-navigable form with:
- **Environment** — dropdown (prod / preprod)
- **Severity** — dropdown (TRACE / DEBUG / INFO / WARN / ERROR / FATAL)
- **Application** — dropdown (from config)
- **Timeframe** — dropdown (15m / 30m / 1h / 3h / 6h / 12h / 24h / 2d / 7d)
- **Trace ID** — optional free-text input
- **Query** — optional free-text Lucene/KQL input

Keys: `Tab` / `Shift+Tab` cycle focus, `Enter` open/confirm dropdown, `Ctrl+S` trigger search.

### ResultsScreen (`tui/screens/results.go`)

Table columns: **Timestamp | Severity | DC | Application | Message**

Keys:
- `↑`/`↓` or `j`/`k` — navigate rows
- `Enter` — open DetailScreen
- `e` — export results to file
- `c` — copy selected row to clipboard
- `r` — back to FilterScreen
- `/` — filter within results

### DetailScreen (`tui/screens/detail.go`)

Scrollable viewport for a single log entry.

Keys:
- `↑`/`↓` or `j`/`k` — scroll
- `r` — toggle raw JSON / formatted view
- `c` — copy full entry JSON to clipboard
- `o` — open Kibana URL in browser (constructed from template + trace ID)
- `Esc` / `b` — back to ResultsScreen

### Components

- **`dropdown.go`** — collapsed `[ Option ▾ ]` / expanded inline list toggle; wraps `bubbles/list`
- **`spinner.go`** — loading overlay shown while searching; wraps `bubbles/spinner`
- **`statusbar.go`** — one-line bottom bar: screen name, DC health indicators (red = error), result count, key hints

---

## OpenSearch Query DSL

Single `bool` query — clauses only added when the corresponding filter is non-empty:

```json
{
  "size": 500,
  "sort": [{ "@timestamp": { "order": "desc" } }],
  "query": {
    "bool": {
      "must": [
        { "query_string": { "query": "<free text>", "default_field": "message" } }
      ],
      "filter": [
        { "range": { "@timestamp": { "gte": "now-1h" } } },
        { "term":  { "severity":    "ERROR"       } },
        { "term":  { "application": "api-gateway" } },
        { "term":  { "trace_id":    "abc-xyz"     } }
      ]
    }
  }
}
```

Timeframe-to-DSL mapping:

| Config value | DSL `gte` |
|---|---|
| `15m` | `now-15m` |
| `30m` | `now-30m` |
| `1h`  | `now-1h`  |
| `3h`  | `now-3h`  |
| `6h`  | `now-6h`  |
| `12h` | `now-12h` |
| `24h` | `now-24h` |
| `2d`  | `now-2d/d`|
| `7d`  | `now-7d/d`|

Field names are read from `config.FieldMapping` — nothing hardcoded in the query builder.

---

## Parallel Search

`opensearch/search.go` — one goroutine per DC using `errgroup` + context with timeout:

1. Launch one goroutine per DC in the selected environment
2. Each goroutine: build URL from template, build DSL body, POST to `/<index_pattern>/_search` with Basic Auth
3. Collect results with per-request context timeout (`query_timeout_seconds`)
4. Merge all `Entries` slices, stable-sort by `Timestamp` descending
5. Collect per-DC errors into `DCErrors` map (partial success allowed)
6. Return `CombinedResult`

Plain `net/http` is used — no OpenSearch SDK. Single `_search` endpoint with JSON DSL is sufficient.

---

## Key Packages

| Purpose | Package |
|---|---|
| TUI framework | `github.com/charmbracelet/bubbletea` |
| Styling | `github.com/charmbracelet/lipgloss` |
| Widgets (table, viewport, textinput, spinner, list, key) | `github.com/charmbracelet/bubbles` |
| YAML parsing | `gopkg.in/yaml.v3` |
| HTTP client | `net/http` (stdlib) |
| JSON encode/decode | `encoding/json` (stdlib) |
| Parallel search with error collection | `golang.org/x/sync/errgroup` |
| Clipboard | `github.com/atotto/clipboard` |
| Browser open | `os/exec` + platform detection |

---

## Implementation Phases

| Phase | Scope |
|---|---|
| **1 — Foundation** | Config loading (`gopkg.in/yaml.v3`), domain models, DSL query builder, HTTP client, parallel search with `errgroup` |
| **2 — Smoke test** | `main.go` CLI mode: load config, run hardcoded search, print JSON to stdout — validates full stack before UI |
| **3 — TUI skeleton** | `app.go` root model + screen enum routing, `styles.go`, `keys.go`, `messages.go` |
| **4 — FilterScreen** | Custom dropdown component, text inputs, tab navigation, validation |
| **5 — Search wiring** | `SearchStartedMsg` → spinner overlay → `SearchDoneMsg` → switch to ResultsScreen |
| **6 — ResultsScreen** | Table with 5 columns, status bar with DC health indicators |
| **7 — DetailScreen** | Viewport, raw/formatted toggle |
| **8 — Export** | Clipboard copy, file export (NDJSON), Kibana URL browser open |
| **9 — Polish** | Terminal resize handling, help screen (`?`), `--config` CLI flag, startup credential validation |
