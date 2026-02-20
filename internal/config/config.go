package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Browsers []BrowserEntry `yaml:"browsers"`
}

type BrowserEntry struct {
	Browser         string `yaml:"browser"`
	Profile         string `yaml:"profile"`
	CookieStorePath string `yaml:"cookie_store_path"`
}

func DefaultConfigFile() string {
	if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
		return filepath.Join(xdg, "gh", "attach.yml")
	}

	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "gh", "attach.yml")
	}

	return "attach.yml"
}

func LoadConfig(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config file: %w", err)
	}

	for i := range cfg.Browsers {
		cfg.Browsers[i].Browser = strings.TrimSpace(cfg.Browsers[i].Browser)
		cfg.Browsers[i].Profile = strings.TrimSpace(cfg.Browsers[i].Profile)
		cfg.Browsers[i].CookieStorePath = strings.TrimSpace(cfg.Browsers[i].CookieStorePath)
	}

	return cfg, nil
}
