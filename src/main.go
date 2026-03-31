package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/criteo/klt/src/config"
	"github.com/criteo/klt/src/models"
	"github.com/criteo/klt/src/opensearch"
)

func main() {
	cfgPath := config.DefaultConfigPath
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	username := os.Getenv("OPENSEARCH_USERNAME")
	password := os.Getenv("OPENSEARCH_PASSWORD")
	if username == "" || password == "" {
		log.Fatal("OPENSEARCH_USERNAME and OPENSEARCH_PASSWORD must be set")
	}

	filter := models.Filter{
		Environment: "prod",
		Timeframe:   "15m",
		Severity:    "ERROR",
	}

	client := opensearch.NewClient(username, password)

	result := opensearch.SearchAll(context.Background(), filter, cfg, client)

	if len(result.DCErrors) > 0 {
		for dc, dcErr := range result.DCErrors {
			fmt.Fprintf(os.Stderr, "DC %s error: %v\n", dc, dcErr)
		}
	}

	fmt.Fprintf(os.Stderr, "total hits: %d, returned: %d\n", result.TotalHits, len(result.Entries))

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result.Entries); err != nil {
		log.Fatalf("encode: %v", err)
	}
}
