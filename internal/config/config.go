// Package config loads service configuration from an optional YAML file with
// environment variable overrides (GOSOUL_ prefix).
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level service configuration.
type Config struct {
	Gateway Gateway `yaml:"gateway"`
	Admin   Admin   `yaml:"admin"`
	Storage Storage `yaml:"storage"`

	// Lobby/Game/Resource are the local Core servers the gateway and clients
	// talk to. Resource serves config/bundles; Lobby owns login/rooms/match;
	// Game owns running matches.
	Lobby    ServerAddr `yaml:"lobby"`
	Game     ServerAddr `yaml:"game"`
	Resource ServerAddr `yaml:"resource"`
}

// Gateway holds the MITM proxy settings.
type Gateway struct {
	Listen string   `yaml:"listen"` // e.g. "127.0.0.1:8080"
	CA     CAConfig `yaml:"ca"`
}

// CAConfig points at the root CA key pair.
type CAConfig struct {
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
}

// Storage holds persistence settings.
type Storage struct {
	Path string `yaml:"path"`
}

// Admin holds the GM API HTTP settings.
type Admin struct {
	Listen string `yaml:"listen"`
}

// ServerAddr is a host:port pair.
type ServerAddr struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// Addr renders host:port.
func (s ServerAddr) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// Default returns the development defaults.
func Default() *Config {
	return &Config{
		Gateway: Gateway{
			Listen: "127.0.0.1:8080",
			CA:     CAConfig{Cert: "data/ca/cert.pem", Key: "data/ca/key.pem"},
		},
		Admin:    Admin{Listen: "127.0.0.1:9090"},
		Lobby:    ServerAddr{Host: "127.0.0.1", Port: 8441},
		Game:     ServerAddr{Host: "127.0.0.1", Port: 8443},
		Resource: ServerAddr{Host: "127.0.0.1", Port: 8440},
	}
}

// Load reads the YAML file if present, then folds in env overrides.
func Load(path string) (*Config, error) {
	cfg := Default()
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("config: read %s: %w", path, err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("config: parse %s: %w", path, err)
		}
	}
	applyEnv(cfg)
	return cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("GOSOUL_GATEWAY_LISTEN"); v != "" {
		cfg.Gateway.Listen = v
	}
	if v := os.Getenv("GOSOUL_ADMIN_LISTEN"); v != "" {
		cfg.Admin.Listen = v
	}
	if v := os.Getenv("GOSOUL_DB"); v != "" {
		cfg.Storage.Path = v
	}
}
