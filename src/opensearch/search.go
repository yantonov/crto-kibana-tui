package opensearch

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/criteo/klt/src/config"
	"github.com/criteo/klt/src/models"
	"golang.org/x/sync/errgroup"
)

const defaultPageSize = 500

// SearchAll implements Searcher.SearchAll. It runs a parallel search across all
// data centers for the given environment and returns a combined, timestamp-sorted
// result. Per-DC errors do not abort the overall search — they are collected in
// CombinedResult.DCErrors.
func (c *Client) SearchAll(ctx context.Context, cfg config.Provider, filter models.Filter) models.CombinedResult {
	return searchAll(ctx, filter, cfg, c)
}

// searchAll is the internal implementation shared by (*Client).SearchAll.
func searchAll(ctx context.Context, filter models.Filter, cfg config.Provider, client *Client) models.CombinedResult {
	dcs, err := cfg.DataCenters(filter.Environment)
	if err != nil {
		return models.CombinedResult{
			DCErrors: map[string]error{"*": err},
		}
	}

	type dcResult struct {
		dc      string
		entries []models.LogEntry
		total   int
		err     error
	}

	resultCh := make(chan dcResult, len(dcs))

	g, gctx := errgroup.WithContext(ctx)
	timeout := cfg.QueryTimeout()

	for _, dc := range dcs {
		dc := dc // capture
		g.Go(func() error {
			dcCtx, cancel := context.WithTimeout(gctx, timeout)
			defer cancel()

			entries, total, err := searchOne(dcCtx, dc, filter, cfg, client)
			resultCh <- dcResult{dc: dc, entries: entries, total: total, err: err}
			return nil // always nil — errors go into resultCh
		})
	}

	// Wait then close; errors are per-DC inside resultCh.
	_ = g.Wait()
	close(resultCh)

	combined := models.CombinedResult{
		DCErrors: make(map[string]error),
	}
	for r := range resultCh {
		if r.err != nil {
			combined.DCErrors[r.dc] = r.err
			continue
		}
		combined.Entries = append(combined.Entries, r.entries...)
		combined.TotalHits += r.total
	}

	// Stable sort descending by timestamp across all DCs.
	sort.SliceStable(combined.Entries, func(i, j int) bool {
		return combined.Entries[i].Timestamp.After(combined.Entries[j].Timestamp)
	})

	return combined
}

// searchOne performs a single _search against one data center and maps hits to LogEntry.
func searchOne(ctx context.Context, dc string, filter models.Filter, cfg config.Provider, client *Client) ([]models.LogEntry, int, error) {
	baseURL := cfg.KibanaURL(dc, filter.Environment)
	query := buildQuery(filter, defaultPageSize)

	resp, err := client.search(ctx, baseURL, cfg.IndexPattern(), query)
	if err != nil {
		return nil, 0, fmt.Errorf("search %s: %w", dc, err)
	}

	entries := make([]models.LogEntry, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		entry := mapHit(hit, dc, filter.Environment)
		entries = append(entries, entry)
	}

	return entries, resp.Hits.Total.Value, nil
}

// mapHit converts a raw OpenSearch hit into a LogEntry.
func mapHit(hit Hit, dc, env string) models.LogEntry {
	src := hit.Source

	raw, _ := json.Marshal(src)

	var severity string
	if code, ok := intField(src, fieldSeverity); ok {
		severity = models.SeverityLabel(code)
	} else {
		severity = stringField(src, fieldSeverity)
	}

	return models.LogEntry{
		Timestamp:   parseTimestamp(stringField(src, fieldTimestamp)),
		Severity:    severity,
		Application: stringField(src, fieldApplication),
		TraceID:     stringField(src, fieldTraceID),
		Message:     stringField(src, fieldMessage),
		DataCenter:  dc,
		Environment: env,
		RawJSON:     string(raw),
	}
}

func stringField(src map[string]interface{}, key string) string {
	if v, ok := src[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// intField reads a numeric field from an OpenSearch source map.
// JSON numbers unmarshal as float64, so both float64 and int are handled.
func intField(src map[string]interface{}, key string) (int, bool) {
	v, ok := src[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	}
	return 0, false
}

func parseTimestamp(s string) time.Time {
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
