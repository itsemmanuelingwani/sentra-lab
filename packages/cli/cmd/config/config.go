package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/sentra-lab/cli/internal/config"
	"github.com/sentra-lab/cli/internal/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type ConfigCommand struct {
	logger       *utils.Logger
	configLoader *config.Loader
	global       bool
}

func NewConfigCommand(logger *utils.Logger) *cobra.Command {
	cc := &ConfigCommand{
		logger: logger,
	}

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage Sentra Lab configuration",
		Long: `View and modify Sentra Lab configuration.

Configuration is stored in:
  • Local: ./lab.yaml (project-specific)
  • Global: ~/.sentra-lab/config.yaml (user-wide)

Commands:
  • get <key>           - Get configuration value
  • set <key> <value>   - Set configuration value
  • list                - List all configuration
  • validate            - Validate configuration file
  • migrate             - Migrate config to latest version

Examples:
  sentra lab config get simulation.parallel
  sentra lab config set simulation.parallel 5
  sentra lab config list
  sentra lab config validate`,
	}

	cmd.AddCommand(newGetCommand(cc))
	cmd.AddCommand(newSetCommand(cc))
	cmd.AddCommand(newListCommand(cc))
	cmd.AddCommand(newValidateCommand(cc))
	cmd.AddCommand(newMigrateCommand(cc))

	cmd.PersistentFlags().BoolVar(&cc.global, "global", false, "Use global config")

	return cmd
}

func newGetCommand(cc *ConfigCommand) *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get configuration value",
		Long: `Get a configuration value by key.

Use dot notation for nested keys:
  sentra lab config get simulation.parallel
  sentra lab config get mocks.openai.enabled

Example:
  sentra lab config get simulation.parallel`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			configPath := cc.getConfigPath()
			loader, err := config.NewLoader(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			cfg, err := loader.Load()
			if err != nil {
				return fmt.Errorf("failed to parse config: %w", err)
			}

			value, err := cfg.Get(key)
			if err != nil {
				return fmt.Errorf("key not found: %s", key)
			}

			fmt.Println(formatValue(value))
			return nil
		},
	}
}

func newSetCommand(cc *ConfigCommand) *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set configuration value",
		Long: `Set a configuration value.

Use dot notation for nested keys:
  sentra lab config set simulation.parallel 5
  sentra lab config set mocks.openai.enabled true

Value types are auto-detected:
  • Numbers: 5, 3.14
  • Booleans: true, false
  • Strings: "hello", hello

Example:
  sentra lab config set simulation.parallel 5
  sentra lab config set mocks.openai.latency_ms 1000`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]

			configPath := cc.getConfigPath()
			loader, err := config.NewLoader(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			cfg, err := loader.Load()
			if err != nil {
				return fmt.Errorf("failed to parse config: %w", err)
			}

			parsedValue := parseValue(value)

			if err := cfg.Set(key, parsedValue); err != nil {
				return fmt.Errorf("failed to set value: %w", err)
			}

			if err := loader.Save(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			cc.logger.Info(fmt.Sprintf("✅ Set %s = %v", key, parsedValue))
			return nil
		},
	}
}

func newListCommand(cc *ConfigCommand) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configuration",
		Long:  "Display all configuration keys and values in a readable format.",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := cc.getConfigPath()
			loader, err := config.NewLoader(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			cfg, err := loader.Load()
			if err != nil {
				return fmt.Errorf("failed to parse config: %w", err)
			}

			cc.logger.Info(fmt.Sprintf("Configuration from: %s", configPath))
			cc.logger.Info("")

			data := cfg.Raw()
			cc.printConfigTree(data, 0)

			return nil
		},
	}
}

func newValidateCommand(cc *ConfigCommand) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		Long: `Validate the configuration file for syntax and schema errors.

This checks:
  • YAML syntax
  • Required fields
  • Field types
  • Value ranges

Example:
  sentra lab config validate`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := cc.getConfigPath()
			loader, err := config.NewLoader(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			cfg, err := loader.Load()
			if err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}

			validator := config.NewValidator()
			if err := validator.Validate(cfg); err != nil {
				return fmt.Errorf("validation failed:\n%w", err)
			}

			cc.logger.Info("✅ Configuration is valid")
			return nil
		},
	}
}

func newMigrateCommand(cc *ConfigCommand) *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Migrate config to latest version",
		Long: `Migrate configuration file to the latest schema version.

This will:
  • Backup existing config
  • Apply schema migrations
  • Validate migrated config

The original file is backed up to lab.yaml.backup

Example:
  sentra lab config migrate`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := cc.getConfigPath()
			loader, err := config.NewLoader(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			cfg, err := loader.Load()
			if err != nil {
				return fmt.Errorf("failed to parse config: %w", err)
			}

			backupPath := configPath + ".backup"
			if err := copyFile(configPath, backupPath); err != nil {
				return fmt.Errorf("failed to backup config: %w", err)
			}

			cc.logger.Info(fmt.Sprintf("📦 Backed up config to: %s", backupPath))

			migrator := config.NewMigrator()
			migrated, err := migrator.Migrate(cfg)
			if err != nil {
				return fmt.Errorf("migration failed: %w", err)
			}

			if !migrated {
				cc.logger.Info("✅ Config is already up to date")
				os.Remove(backupPath)
				return nil
			}

			if err := loader.Save(cfg); err != nil {
				return fmt.Errorf("failed to save migrated config: %w", err)
			}

			cc.logger.Info("✅ Config migrated successfully")
			cc.logger.Info(fmt.Sprintf("Backup available at: %s", backupPath))

			return nil
		},
	}
}

func (cc *ConfigCommand) getConfigPath() string {
	if cc.global {
		homeDir, _ := os.UserHomeDir()
		return fmt.Sprintf("%s/.sentra-lab/config.yaml", homeDir)
	}
	return "lab.yaml"
}

func (cc *ConfigCommand) printConfigTree(data map[string]interface{}, indent int) {
	for key, value := range data {
		prefix := strings.Repeat("  ", indent)

		switch v := value.(type) {
		case map[string]interface{}:
			cc.logger.Info(fmt.Sprintf("%s%s:", prefix, key))
			cc.printConfigTree(v, indent+1)
		default:
			cc.logger.Info(fmt.Sprintf("%s%s: %v", prefix, key, formatValue(v)))
		}
	}
}

func formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case int, int64, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case []interface{}:
		items := make([]string, len(v))
		for i, item := range v {
			items[i] = formatValue(item)
		}
		return "[" + strings.Join(items, ", ") + "]"
	case map[string]interface{}:
		data, _ := yaml.Marshal(v)
		return string(data)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func parseValue(value string) interface{} {
	value = strings.TrimSpace(value)

	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}

	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		inner := strings.Trim(value, "[]")
		parts := strings.Split(inner, ",")
		result := make([]interface{}, len(parts))
		for i, part := range parts {
			result[i] = parseValue(strings.TrimSpace(part))
		}
		return result
	}

	if strings.Contains(value, ".") {
		var f float64
		if _, err := fmt.Sscanf(value, "%f", &f); err == nil {
			return f
		}
	}

	var i int
	if _, err := fmt.Sscanf(value, "%d", &i); err == nil {
		return i
	}

	return value
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
