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

// SearchAll runs a parallel search across all data centers for the given environment
// and returns a combined, timestamp-sorted result. Per-DC errors do not abort the
// overall search — they are collected in CombinedResult.DCErrors.
func SearchAll(ctx context.Context, filter models.Filter, cfg *config.Config, client *Client) models.CombinedResult {
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
	timeout := time.Duration(cfg.QueryTimeoutSeconds) * time.Second

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
func searchOne(ctx context.Context, dc string, filter models.Filter, cfg *config.Config, client *Client) ([]models.LogEntry, int, error) {
	baseURL := cfg.OpenSearchURL(dc, filter.Environment)
	query := BuildQuery(filter, cfg.FieldMapping, defaultPageSize)

	resp, err := client.Search(ctx, baseURL, cfg.IndexPattern, query)
	if err != nil {
		return nil, 0, fmt.Errorf("search %s: %w", dc, err)
	}

	fm := cfg.FieldMapping
	entries := make([]models.LogEntry, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		entry := mapHit(hit, fm, dc, filter.Environment)
		entries = append(entries, entry)
	}

	return entries, resp.Hits.Total.Value, nil
}

// mapHit converts a raw OpenSearch hit into a LogEntry using the configured field mapping.
func mapHit(hit Hit, fm config.FieldMapping, dc, env string) models.LogEntry {
	src := hit.Source

	raw, _ := json.Marshal(src)

	ts := parseTimestamp(stringField(src, fm.Timestamp))

	return models.LogEntry{
		Timestamp:   ts,
		Severity:    stringField(src, fm.Severity),
		Application: stringField(src, fm.Application),
		TraceID:     stringField(src, fm.TraceID),
		Message:     stringField(src, fm.Message),
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
