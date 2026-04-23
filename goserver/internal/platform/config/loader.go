package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadFromFile(path string) (AppConfig, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return AppConfig{}, err
	}

	config, err := LoadBytes(contents, filepath.Ext(path))
	if err != nil {
		return AppConfig{}, err
	}

	return config, nil
}

func LoadBytes(contents []byte, extension string) (AppConfig, error) {
	config := Default()

	switch strings.ToLower(strings.TrimSpace(extension)) {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(contents, &config); err != nil {
			return AppConfig{}, fmt.Errorf("decode yaml config: %w", err)
		}
	case ".json", "":
		if err := json.Unmarshal(contents, &config); err != nil {
			return AppConfig{}, fmt.Errorf("decode json config: %w", err)
		}
	default:
		return AppConfig{}, fmt.Errorf("unsupported config extension %q", extension)
	}

	return config, config.Validate()
}

func (config AppConfig) MarshalSanitizedJSON() (map[string]any, error) {
	sanitized := config
	sanitized.AsyncAI.APIKey = ""

	payload, err := json.Marshal(sanitized)
	if err != nil {
		return nil, err
	}

	var out map[string]any
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, err
	}

	return out, nil
}
