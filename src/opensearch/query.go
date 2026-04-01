package opensearch

import (
	"fmt"
	"strings"

	"github.com/criteo/klt/src/models"
)

const (
	fieldTimestamp   = "@timestamp"
	fieldSeverity    = "severity"
	fieldApplication = "application"
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

// buildQuery constructs a typed OpenSearch DSL query body from the given filter.
func buildQuery(f models.Filter, size int) searchRequest {
	filters := []queryClause{
		{
			"range": queryClause{
				fieldTimestamp: queryClause{
					"gte": timeframeDSL(f.Timeframe),
					"lte": "now",
				},
			},
		},
	}

	if f.Severity >= 0 {
		filters = append(filters, queryClause{
			"term": queryClause{fieldSeverity: f.Severity},
		})
	}
	if f.Application != "" {
		filters = append(filters, queryClause{
			"term": queryClause{fieldApplication: f.Application},
		})
	}
	if f.TraceID != "" {
		filters = append(filters, queryClause{
			"term": queryClause{fieldTraceID: f.TraceID},
		})
	}

	clauses := boolClauses{Filter: filters}

	if q := strings.TrimSpace(f.Query); q != "" {
		clauses.Must = []queryClause{
			{
				"query_string": queryClause{
					"query":         q,
					"default_field": fieldMessage,
				},
			},
		}
	}

	return searchRequest{
		Size: size,
		Sort: []sortClause{
			{fieldTimestamp: sortOrder{Order: "desc"}},
		},
		Query: boolWrapper{Bool: clauses},
	}
}
