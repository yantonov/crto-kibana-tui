package opensearch

import (
	"fmt"
	"strings"

	"github.com/criteo/klt/src/models"
)

const (
	fieldTimestamp   = "@timestamp"
	fieldSeverity    = "level"
	fieldApplication = "app"
	fieldTraceID     = "trace_id"
	fieldMessage     = "message"
)

// timeframeDSL converts a timeframe string (e.g. "1h", "2d") to an OpenSearch range gte value.
func timeframeDSL(tf string) string {
	switch tf {
	case "2d", "7d":
		return fmt.Sprintf("now-%s/d", tf)
	default:
		return fmt.Sprintf("now-%s", tf)
	}
}

// BuildQuery constructs an OpenSearch DSL query body from the given filter.
func BuildQuery(f models.Filter, size int) map[string]interface{} {
	filters := []interface{}{
		map[string]interface{}{
			"range": map[string]interface{}{
				fieldTimestamp: map[string]interface{}{
					"gte": timeframeDSL(f.Timeframe),
					"lte": "now",
				},
			},
		},
	}

	if f.Severity != "" {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{fieldSeverity: f.Severity},
		})
	}
	if f.Application != "" {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{fieldApplication: f.Application},
		})
	}
	if f.TraceID != "" {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{fieldTraceID: f.TraceID},
		})
	}

	boolQuery := map[string]interface{}{
		"filter": filters,
	}

	if q := strings.TrimSpace(f.Query); q != "" {
		boolQuery["must"] = []interface{}{
			map[string]interface{}{
				"query_string": map[string]interface{}{
					"query":         q,
					"default_field": fieldMessage,
				},
			},
		}
	}

	return map[string]interface{}{
		"size": size,
		"sort": []interface{}{
			map[string]interface{}{
				fieldTimestamp: map[string]interface{}{"order": "desc"},
			},
		},
		"query": map[string]interface{}{
			"bool": boolQuery,
		},
	}
}
