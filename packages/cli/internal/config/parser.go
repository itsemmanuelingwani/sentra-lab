package config

import (
	"fmt"
	"strings"
	"time"
)

type Config struct {
	Name       string                 `yaml:"name"`
	Version    string                 `yaml:"version"`
	Agent      AgentConfig            `yaml:"agent"`
	Mocks      map[string]MockConfig  `yaml:"mocks"`
	Simulation SimulationConfig       `yaml:"simulation"`
	Storage    StorageConfig          `yaml:"storage"`
	raw        map[string]interface{}
}

type AgentConfig struct {
	Runtime    string `yaml:"runtime"`
	EntryPoint string `yaml:"entry_point"`
	Timeout    string `yaml:"timeout"`
}

type MockConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Port      int    `yaml:"port"`
	LatencyMS int    `yaml:"latency_ms"`
	RateLimit int    `yaml:"rate_limit"`
	ErrorRate float64 `yaml:"error_rate"`
}

type SimulationConfig struct {
	RecordFullTrace       bool `yaml:"record_full_trace"`
	EnableCostTracking    bool `yaml:"enable_cost_tracking"`
	MaxConcurrentScenarios int  `yaml:"max_concurrent_scenarios"`
}

type StorageConfig struct {
	RecordingsDir string `yaml:"recordings_dir"`
	Database      string `yaml:"database"`
}

func (c *Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}

	if c.Version == "" {
		return fmt.Errorf("version is required")
	}

	if c.Agent.Runtime == "" {
		return fmt.Errorf("agent.runtime is required")
	}

	validRuntimes := []string{"python", "nodejs", "go"}
	if !contains(validRuntimes, c.Agent.Runtime) {
		return fmt.Errorf("invalid agent.runtime: %s (must be one of: %s)", c.Agent.Runtime, strings.Join(validRuntimes, ", "))
	}

	if c.Agent.EntryPoint == "" {
		return fmt.Errorf("agent.entry_point is required")
	}

	return nil
}

func (c *Config) ApplyDefaults() {
	if c.Agent.Timeout == "" {
		c.Agent.Timeout = "30s"
	}

	if c.Simulation.MaxConcurrentScenarios == 0 {
		c.Simulation.MaxConcurrentScenarios = 10
	}

	if c.Storage.RecordingsDir == "" {
		c.Storage.RecordingsDir = ".sentra-lab/recordings"
	}

	if c.Storage.Database == "" {
		c.Storage.Database = ".sentra-lab/sentra.db"
	}

	for name, mock := range c.Mocks {
		if mock.Port == 0 {
			switch name {
			case "openai":
				mock.Port = 8080
			case "stripe":
				mock.Port = 8081
			case "coreledger":
				mock.Port = 8082
			default:
				mock.Port = 8000
			}
			c.Mocks[name] = mock
		}
	}
}

func (c *Config) Get(key string) (interface{}, error) {
	parts := strings.Split(key, ".")

	var current interface{} = c.raw
	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			if val, exists := m[part]; exists {
				current = val
			} else {
				return nil, fmt.Errorf("key not found: %s", key)
			}
		} else {
			return nil, fmt.Errorf("invalid path: %s", key)
		}
	}

	return current, nil
}

func (c *Config) Set(key string, value interface{}) error {
	parts := strings.Split(key, ".")

	if c.raw == nil {
		c.raw = make(map[string]interface{})
	}

	current := c.raw
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
		} else {
			if _, exists := current[part]; !exists {
				current[part] = make(map[string]interface{})
			}

			if m, ok := current[part].(map[string]interface{}); ok {
				current = m
			} else {
				return fmt.Errorf("cannot set value at path: %s", key)
			}
		}
	}

	return nil
}

func (c *Config) Raw() map[string]interface{} {
	return c.raw
}

func (c *Config) GetEngineAddress() string {
	return "localhost:50051"
}

func (c *Config) GetAgentTimeout() time.Duration {
	duration, err := time.ParseDuration(c.Agent.Timeout)
	if err != nil {
		return 30 * time.Second
	}
	return duration
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}