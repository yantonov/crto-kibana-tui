package main

import (
	"flag"
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/yantonov/crtokt/src/config"
	"github.com/yantonov/crtokt/src/credentials"
	"github.com/yantonov/crtokt/src/opensearch"
	"github.com/yantonov/crtokt/src/tui"
)

func main() {
	var cfgPath string
	var logout bool

	flag.StringVar(&cfgPath, "config", "", "path to config.yaml (default: config.yaml next to the executable)")
	flag.BoolVar(&logout, "logout", false, "remove the stored credentials from the keychain and exit")
	flag.Parse()

	store := credentials.Keychain{}

	if logout {
		if err := store.Clear(); err != nil {
			log.Fatalf("clear keychain: %v", err)
		}
		fmt.Printf("credentials removed from the keychain (service %q)\n", credentials.Service())
		return
	}

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

	p := tea.NewProgram(tui.New(cfg, client, store), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("tui: %v", err)
	}
}
