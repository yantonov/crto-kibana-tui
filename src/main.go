package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/criteo/klt/src/config"
	"github.com/criteo/klt/src/models"
	"github.com/criteo/klt/src/opensearch"
	"github.com/criteo/klt/src/tui"
)

func main() {
	var cfgPath string
	var diag bool
	var diagEnv, diagApp, diagTimeframe string

	flag.StringVar(&cfgPath, "config", "", "path to config.yaml (default: config.yaml next to the executable)")
	flag.BoolVar(&diag, "diag", false, "run connectivity and query diagnostics then exit")
	flag.StringVar(&diagEnv, "env", "", "environment for --diag (defaults to first defined environment)")
	flag.StringVar(&diagApp, "app", "", "application filter for --diag")
	flag.StringVar(&diagTimeframe, "timeframe", "15m", "timeframe for --diag (e.g. 15m, 1h, 24h)")
	flag.Parse()

	if cfgPath == "" {
		var err error
		cfgPath, err = config.DefaultConfigPath()
		if err != nil {
			log.Fatalf("resolve config path: %v", err)
		}
		if err := config.WriteTemplate(cfgPath); err != nil {
			log.Fatalf("write config template: %v", err)
		}
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if diag {
		username := os.Getenv("OPENSEARCH_USERNAME")
		password := os.Getenv("OPENSEARCH_PASSWORD")
		if username == "" || password == "" {
			log.Fatal("OPENSEARCH_USERNAME and OPENSEARCH_PASSWORD must be set for --diag")
		}
		client := opensearch.NewClientWithBasicAuth(username, password)
		runDiag(cfg, client, diagEnv, diagApp, diagTimeframe)
		return
	}

	client := opensearch.NewClient()

	p := tea.NewProgram(tui.New(cfg, client), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("tui: %v", err)
	}
}

// runDiag tests connectivity and prints the generated query for the given filter.
func runDiag(cfg *config.Config, client *opensearch.Client, env, app, timeframe string) {
	// Resolve environment.
	if env == "" {
		e, _ := firstDC(cfg)
		env = e
	}
	if env == "" {
		fmt.Fprintln(os.Stderr, "no environments defined in config")
		os.Exit(1)
	}

	dcs, err := cfg.DataCenters(env)
	if err != nil {
		fmt.Fprintf(os.Stderr, "environment %q: %v\n", env, err)
		os.Exit(1)
	}

	filter := models.Filter{
		Environment: env,
		Application: app,
		Timeframe:   timeframe,
		Severity:    models.SeverityAll,
	}

	// Print the query JSON.
	query := opensearch.BuildQuery(filter, 10)
	queryJSON, _ := json.MarshalIndent(query, "", "  ")
	fmt.Printf("=== Query (env=%s app=%q timeframe=%s) ===\n%s\n\n", env, app, timeframe, queryJSON)

	// Ping + search each DC.
	timeout := time.Duration(config.QueryTimeoutSeconds) * time.Second
	sort.Strings(dcs)
	for _, dc := range dcs {
		kibanaURL := cfg.KibanaURL(dc, env)
		fmt.Printf("--- DC: %s  (%s) ---\n", dc, kibanaURL)

		pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := client.Ping(pingCtx, kibanaURL); err != nil {
			fmt.Printf("  ping FAIL: %v\n", err)
			pingCancel()
			continue
		}
		pingCancel()
		fmt.Printf("  ping OK\n")

		searchCtx, searchCancel := context.WithTimeout(context.Background(), timeout)
		resp, err := client.Search(searchCtx, kibanaURL, config.IndexPattern, query)
		searchCancel()
		if err != nil {
			fmt.Printf("  search FAIL: %v\n", err)
			continue
		}
		fmt.Printf("  total hits: %d  (returned: %d)\n", resp.Hits.Total.Value, len(resp.Hits.Hits))
		if len(resp.Hits.Hits) > 0 {
			first := resp.Hits.Hits[0].Source
			if appVal, ok := first["app"]; ok {
				fmt.Printf("  first hit app field: %q\n", fmt.Sprint(appVal))
			} else {
				fmt.Printf("  first hit has no 'app' field — available keys: %v\n", sourceKeys(first))
			}
		}
	}
}

func sourceKeys(src map[string]interface{}) []string {
	keys := make([]string, 0, len(src))
	for k := range src {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// firstDC returns the first environment name and DC from the config, used for
// the startup credential ping.
func firstDC(cfg *config.Config) (env, dc string) {
	for e, ecfg := range config.Environments {
		if len(ecfg.DataCenters) > 0 {
			return e, ecfg.DataCenters[0]
		}
	}
	return "", ""
}
