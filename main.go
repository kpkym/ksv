package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	confPath := findConfigPath()
	baseDir := filepath.Dir(confPath)

	conf, err := loadConfig(confPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		fmt.Fprintf(os.Stderr, "Expected config at: %s\n", confPath)
		os.Exit(1)
	}

	if len(conf.Services) == 0 {
		fmt.Fprintf(os.Stderr, "No services configured in %s\n", confPath)
		os.Exit(1)
	}

	var services []*Service
	for _, cfg := range conf.Services {
		services = append(services, NewService(cfg, baseDir, conf.LogMaxSize))
	}

	p := tea.NewProgram(
		newModel(services),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
