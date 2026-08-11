package appconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultMaxFiles       = 20
	defaultMaxScanBytes   = 8 << 20
	defaultMaxResults     = 500
	defaultMaxLineBytes   = 8 << 10
	defaultMaxOutputBytes = 512 << 10
)

type Source struct {
	Name     string   `json:"name"`
	Patterns []string `json:"patterns"`
}

type Config struct {
	Sources        []Source `json:"sources"`
	MaxFiles       int      `json:"max_files"`
	MaxScanBytes   int64    `json:"max_scan_bytes"`
	MaxResults     int      `json:"max_results"`
	MaxLineBytes   int      `json:"max_line_bytes"`
	MaxOutputBytes int      `json:"max_output_bytes"`
}

func Load(path string) (Config, error) {
	if !filepath.IsAbs(path) {
		return Config{}, fmt.Errorf("config path must be absolute")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var config Config
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	applyDefaults(&config)
	if err := validate(config); err != nil {
		return Config{}, err
	}
	return config, nil
}

func applyDefaults(config *Config) {
	if config.MaxFiles == 0 {
		config.MaxFiles = defaultMaxFiles
	}
	if config.MaxScanBytes == 0 {
		config.MaxScanBytes = defaultMaxScanBytes
	}
	if config.MaxResults == 0 {
		config.MaxResults = defaultMaxResults
	}
	if config.MaxLineBytes == 0 {
		config.MaxLineBytes = defaultMaxLineBytes
	}
	if config.MaxOutputBytes == 0 {
		config.MaxOutputBytes = defaultMaxOutputBytes
	}
}

func validate(config Config) error {
	if len(config.Sources) == 0 {
		return fmt.Errorf("at least one log source is required")
	}
	if config.MaxFiles < 1 || config.MaxFiles > 100 {
		return fmt.Errorf("max_files must be between 1 and 100")
	}
	if config.MaxScanBytes < 1024 || config.MaxScanBytes > 64<<20 {
		return fmt.Errorf("max_scan_bytes must be between 1024 and 67108864")
	}
	if config.MaxResults < 1 || config.MaxResults > 2000 {
		return fmt.Errorf("max_results must be between 1 and 2000")
	}
	if config.MaxLineBytes < 256 || config.MaxLineBytes > 64<<10 {
		return fmt.Errorf("max_line_bytes must be between 256 and 65536")
	}
	if config.MaxOutputBytes < 1024 || config.MaxOutputBytes > 4<<20 {
		return fmt.Errorf("max_output_bytes must be between 1024 and 4194304")
	}
	seen := make(map[string]struct{}, len(config.Sources))
	for _, source := range config.Sources {
		if source.Name == "" || strings.ContainsAny(source.Name, " /\\") {
			return fmt.Errorf("source name %q is invalid", source.Name)
		}
		if _, exists := seen[source.Name]; exists {
			return fmt.Errorf("source name %q is duplicated", source.Name)
		}
		seen[source.Name] = struct{}{}
		if len(source.Patterns) == 0 {
			return fmt.Errorf("source %q has no patterns", source.Name)
		}
		for _, pattern := range source.Patterns {
			if !filepath.IsAbs(pattern) {
				return fmt.Errorf("source %q pattern must be absolute", source.Name)
			}
			if strings.Contains(pattern, "..") {
				return fmt.Errorf("source %q pattern cannot contain ..", source.Name)
			}
		}
	}
	return nil
}
