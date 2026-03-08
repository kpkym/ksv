package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const defaultLogMaxSize int64 = 10 * 1024 * 1024 // 10MB

type Config struct {
	LogMaxSize int64 // bytes, 0 = unlimited
	Services   []ServiceConfig
}

type ServiceConfig struct {
	Name string
	Dir  string
	Cmd  string
}

var sectionRe = regexp.MustCompile(`^\[(.+)\]$`)

func findConfigPath() string {
	if dir := os.Getenv("KPK_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "ksv", "services.conf")
	}
	home, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(home, ".config", "ksv", "services.conf")
	}
	return "services.conf"
}

func loadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open config: %w", err)
	}
	defer f.Close()

	cfg := &Config{LogMaxSize: defaultLogMaxSize}
	var current *ServiceConfig

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if line == "" {
			continue
		}

		if m := sectionRe.FindStringSubmatch(line); m != nil {
			if current != nil && current.Dir != "" && current.Cmd != "" {
				cfg.Services = append(cfg.Services, *current)
			}
			current = &ServiceConfig{Name: m[1]}
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		if current == nil {
			// Global config
			if key == "log_max_size" {
				if size, err := parseSize(val); err == nil {
					cfg.LogMaxSize = size
				}
			}
			continue
		}

		switch key {
		case "dir":
			current.Dir = val
		case "cmd":
			current.Cmd = val
		}
	}

	if current != nil && current.Dir != "" && current.Cmd != "" {
		cfg.Services = append(cfg.Services, *current)
	}

	return cfg, scanner.Err()
}

// parseSize parses a size string like "10M", "500K", "1G" into bytes.
// Plain numbers are treated as bytes.
func parseSize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "0" {
		return 0, nil
	}

	multiplier := int64(1)
	if len(s) > 1 {
		switch strings.ToUpper(s[len(s)-1:]) {
		case "K":
			multiplier = 1024
			s = s[:len(s)-1]
		case "M":
			multiplier = 1024 * 1024
			s = s[:len(s)-1]
		case "G":
			multiplier = 1024 * 1024 * 1024
			s = s[:len(s)-1]
		}
	}

	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0, err
	}
	return n * multiplier, nil
}
