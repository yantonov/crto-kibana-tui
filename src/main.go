package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/criteo/klt/src/config"
	"github.com/criteo/klt/src/opensearch"
	"github.com/criteo/klt/src/tui"
)

func main() {
	var cfgPath string
	flag.StringVar(&cfgPath, "config", "", "path to config.yaml (default: config.yaml next to the executable)")
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

	username := os.Getenv("OPENSEARCH_USERNAME")
	password := os.Getenv("OPENSEARCH_PASSWORD")
	if username == "" || password == "" {
		log.Fatal("OPENSEARCH_USERNAME and OPENSEARCH_PASSWORD must be set")
	}

	client := opensearch.NewClient(username, password)

	// Startup credential validation: ping the first available DC to catch
	// bad credentials before the TUI starts.
	if env, dc := firstDC(cfg); env != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Ping(ctx, cfg.KibanaURL(dc, env)); err != nil {
			log.Printf("warning: credential check failed (%s/%s): %v", env, dc, err)
			log.Printf("searches may fail — check OPENSEARCH_USERNAME / OPENSEARCH_PASSWORD")
		}
	}

	p := tea.NewProgram(tui.New(cfg, client), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("tui: %v", err)
	}
}

// firstDC returns the first environment name and DC from the config, used for
// the startup credential ping.
func firstDC(cfg *config.Config) (env, dc string) {
	for e, ecfg := range cfg.Environments {
		if len(ecfg.DataCenters) > 0 {
			return e, ecfg.DataCenters[0]
		}
	}
	return "", ""
}
