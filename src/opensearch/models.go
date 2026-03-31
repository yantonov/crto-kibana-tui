package opensearch

// SearchResponse represents the top-level OpenSearch _search API response.
type SearchResponse struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []Hit `json:"hits"`
	} `json:"hits"`
}

// Hit is a single document returned by OpenSearch.
type Hit struct {
	Index  string                 `json:"_index"`
	Source map[string]interface{} `json:"_source"`
}
