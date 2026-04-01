package opensearch

// searchRequest is the top-level body sent to the OpenSearch _search endpoint.
// It is unexported — the only entry point is buildQuery(); the result is consumed
// exclusively inside this package.
type searchRequest struct {
	Size  int          `json:"size"`
	Sort  []sortClause `json:"sort"`
	Query boolWrapper  `json:"query"`
}

// sortClause maps a field name to a sort direction, e.g. {"@timestamp": {"order": "desc"}}.
type sortClause map[string]sortOrder

type sortOrder struct {
	Order string `json:"order"`
}

// boolWrapper is the {"bool": ...} envelope required by the OpenSearch bool query.
type boolWrapper struct {
	Bool boolClauses `json:"bool"`
}

type boolClauses struct {
	Must   []queryClause `json:"must,omitempty"`
	Filter []queryClause `json:"filter,omitempty"`
}

// queryClause is a leaf DSL node. Leaf nodes stay as flexible maps because the
// OpenSearch DSL has many clause shapes (range, term, query_string, …) and
// adding a new clause type should not require a new Go type.
type queryClause map[string]interface{}
