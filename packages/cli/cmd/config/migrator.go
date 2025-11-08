package config

import (
	"fmt"
)

type Migrator struct {
	migrations []Migration
}

type Migration struct {
	FromVersion string
	ToVersion   string
	Description string
	Migrate     func(data map[string]interface{}) error
}

func NewMigrator() *Migrator {
	return &Migrator{
		migrations: []Migration{
			{
				FromVersion: "0.9",
				ToVersion:   "1.0",
				Description: "Add simulation settings and update mock structure",
				Migrate:     migrateV09ToV10,
			},
			{
				FromVersion: "1.0",
				ToVersion:   "1.1",
				Description: "Add storage configuration",
				Migrate:     migrateV10ToV11,
			},
		},
	}
}

func (m *Migrator) Migrate(cfg interface{}) (bool, error) {
	data, ok := cfg.(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("invalid config format")
	}

	currentVersion := m.getCurrentVersion(data)
	targetVersion := "1.1"

	if currentVersion == targetVersion {
		return false, nil
	}

	migrated := false

	for _, migration := range m.migrations {
		if currentVersion == migration.FromVersion {
			fmt.Printf("📦 Applying migration: %s → %s\n", migration.FromVersion, migration.ToVersion)
			fmt.Printf("   %s\n", migration.Description)

			if err := migration.Migrate(data); err != nil {
				return false, fmt.Errorf("migration failed: %w", err)
			}

			data["version"] = migration.ToVersion
			currentVersion = migration.ToVersion
			migrated = true
		}
	}

	return migrated, nil
}

func (m *Migrator) getCurrentVersion(data map[string]interface{}) string {
	if version, ok := data["version"].(string); ok {
		return version
	}
	return "0.9"
}

func migrateV09ToV10(data map[string]interface{}) error {
	if _, ok := data["simulation"]; !ok {
		data["simulation"] = map[string]interface{}{
			"record_full_trace":         true,
			"enable_cost_tracking":      true,
			"max_concurrent_scenarios":  10,
		}
	}

	if mocks, ok := data["mocks"].(map[string]interface{}); ok {
		for mockName, mockConfig := range mocks {
			if mockData, ok := mockConfig.(map[string]interface{}); ok {
				if _, hasEnabled := mockData["enabled"]; !hasEnabled {
					mockData["enabled"] = true
				}

				if mockName == "openai" {
					if _, hasLatency := mockData["latency_ms"]; !hasLatency {
						mockData["latency_ms"] = 1000
					}
					if _, hasRateLimit := mockData["rate_limit"]; !hasRateLimit {
						mockData["rate_limit"] = 3500
					}
					if _, hasErrorRate := mockData["error_rate"]; !hasErrorRate {
						mockData["error_rate"] = 0.01
					}
				}
			}
		}
	}

	return nil
}

func migrateV10ToV11(data map[string]interface{}) error {
	if _, ok := data["storage"]; !ok {
		data["storage"] = map[string]interface{}{
			"recordings_dir": ".sentra-lab/recordings",
			"database":       ".sentra-lab/sentra.db",
		}
	}

	if agent, ok := data["agent"].(map[string]interface{}); ok {
		if _, hasTimeout := agent["timeout"]; !hasTimeout {
			agent["timeout"] = "30s"
		}
	}

	return nil
}
