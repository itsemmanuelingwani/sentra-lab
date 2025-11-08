package main

import (
	"fmt"
	"os"

	"github.com/sentra-lab/cli/cmd/cloud"
	"github.com/sentra-lab/cli/cmd/config"
	"github.com/sentra-lab/cli/cmd/init"
	"github.com/sentra-lab/cli/cmd/replay"
	"github.com/sentra-lab/cli/cmd/start"
	"github.com/sentra-lab/cli/cmd/test"
	"github.com/sentra-lab/cli/internal/utils"
	"github.com/spf13/cobra"
)

var (
	version = "1.0.0"
	commit  = "dev"
	date    = "unknown"
)

func main() {
	logger := utils.NewLogger("sentra-lab", "info")

	rootCmd := &cobra.Command{
		Use:   "sentra",
		Short: "Sentra Lab - Local-first simulation platform for AI agents",
		Long: `Sentra Lab enables developers to test AI agents locally without API costs.

Features:
  • Local-first simulation (works offline)
  • Production-parity mocks (OpenAI, Stripe, etc.)
  • Time-travel debugging (replay any execution)
  • Cost estimation (predict production costs)
  • CI/CD integration (GitHub Actions, GitLab CI)

Get started:
  sentra lab init my-agent    # Create new project
  sentra lab start            # Start mock services
  sentra lab test             # Run test scenarios`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			if verbose {
				logger.SetLevel("debug")
			}
			return nil
		},
	}

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().String("config", "", "Config file (default: ./lab.yaml)")

	labCmd := &cobra.Command{
		Use:   "lab",
		Short: "Sentra Lab commands",
		Long:  "Core commands for managing Sentra Lab simulations",
	}

	labCmd.AddCommand(
		init.NewInitCommand(logger),
		start.NewStartCommand(logger),
		test.NewTestCommand(logger),
		replay.NewReplayCommand(logger),
		config.NewConfigCommand(logger),
		cloud.NewCloudCommand(logger),
	)

	labCmd.AddCommand(newStopCommand(logger))
	labCmd.AddCommand(newLogsCommand(logger))
	labCmd.AddCommand(newStatusCommand(logger))
	labCmd.AddCommand(newQuickstartCommand(logger))

	rootCmd.AddCommand(labCmd)

	if err := rootCmd.Execute(); err != nil {
		logger.Error("command failed", "error", err)
		os.Exit(1)
	}
}

func newStopCommand(logger *utils.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop all Sentra Lab services",
		Long:  "Gracefully shutdown all Docker containers and cleanup resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			return start.NewStartCommand(logger).Stop(cmd.Context())
		},
	}
	return cmd
}

func newLogsCommand(logger *utils.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs [service]",
		Short: "View service logs",
		Long:  "Stream logs from simulation engine and mock services",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			follow, _ := cmd.Flags().GetBool("follow")
			tail, _ := cmd.Flags().GetInt("tail")
			
			service := "all"
			if len(args) > 0 {
				service = args[0]
			}
			
			return start.NewStartCommand(logger).Logs(cmd.Context(), service, follow, tail)
		},
	}
	
	cmd.Flags().BoolP("follow", "f", false, "Follow log output")
	cmd.Flags().IntP("tail", "n", 100, "Number of lines to show")
	
	return cmd
}

func newStatusCommand(logger *utils.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show service status",
		Long:  "Display health and status of all Sentra Lab services",
		RunE: func(cmd *cobra.Command, args []string) error {
			return start.NewStartCommand(logger).Status(cmd.Context())
		},
	}
	return cmd
}

func newQuickstartCommand(logger *utils.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quickstart [name]",
		Short: "Quick setup and test (convenience wrapper)",
		Long: `Convenience command that runs: init + start + test

This is a shortcut for first-time users. For production use,
prefer explicit commands: init, start, test separately.

Example:
  sentra lab quickstart my-agent`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			name := args[0]
			
			logger.Info("🚀 Quickstart: initializing project...", "name", name)
			
			initCmd := init.NewInitCommand(logger)
			if err := initCmd.RunE(cmd, []string{name}); err != nil {
				return fmt.Errorf("init failed: %w", err)
			}
			
			if err := os.Chdir(name); err != nil {
				return fmt.Errorf("failed to change directory: %w", err)
			}
			
			logger.Info("🔧 Quickstart: starting services...")
			
			startCmd := start.NewStartCommand(logger)
			if err := startCmd.RunE(cmd, []string{}); err != nil {
				return fmt.Errorf("start failed: %w", err)
			}
			
			logger.Info("🧪 Quickstart: running tests...")
			
			testCmd := test.NewTestCommand(logger)
			if err := testCmd.RunE(cmd, []string{}); err != nil {
				return fmt.Errorf("test failed: %w", err)
			}
			
			logger.Info("✅ Quickstart complete!")
			logger.Info("Next steps:")
			logger.Info("  • Edit scenarios/ to add more tests")
			logger.Info("  • Run 'sentra lab test' to re-run tests")
			logger.Info("  • Run 'sentra lab replay' to debug failures")
			
			return nil
		},
	}
	
	return cmd
}
