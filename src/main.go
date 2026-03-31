package main

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/criteo/klt/src/config"
	"github.com/criteo/klt/src/opensearch"
	"github.com/criteo/klt/src/tui"
)

func main() {
	var cfgPath string
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	} else {
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

	p := tea.NewProgram(tui.New(cfg, client), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("tui: %v", err)
	}
}
