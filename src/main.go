package main

import (
	"flag"
	"log"

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

	client := opensearch.NewClient()

	p := tea.NewProgram(tui.New(cfg, client), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("tui: %v", err)
	}
}
