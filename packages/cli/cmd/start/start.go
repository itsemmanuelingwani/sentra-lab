package start

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sentra-lab/cli/internal/config"
	"github.com/sentra-lab/cli/internal/docker"
	"github.com/sentra-lab/cli/internal/utils"
	"github.com/spf13/cobra"
)

type StartCommand struct {
	logger        *utils.Logger
	dockerManager *docker.Manager
	configLoader  *config.Loader
	detach        bool
	pull          bool
	rebuild       bool
}

func NewStartCommand(logger *utils.Logger) *cobra.Command {
	sc := &StartCommand{
		logger: logger,
	}

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start Sentra Lab services",
		Long: `Start all mock services and simulation engine using Docker.

Services started:
  • Simulation Engine (gRPC server)
  • Mock OpenAI API
  • Mock Stripe API
  • Mock CoreLedger API
  • Mock Database services (if configured)

All services run in Docker containers with health checks.
Use --detach to run in background.

Example:
  sentra lab start              # Start and show logs
  sentra lab start --detach     # Start in background
  sentra lab start --pull       # Pull latest images first`,
		PreRunE: sc.PreRunE,
		RunE:    sc.RunE,
	}

	cmd.Flags().BoolVarP(&sc.detach, "detach", "d", false, "Run in background")
	cmd.Flags().BoolVar(&sc.pull, "pull", false, "Pull latest Docker images")
	cmd.Flags().BoolVar(&sc.rebuild, "rebuild", false, "Rebuild containers")

	return cmd
}

func (sc *StartCommand) PreRunE(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")
	if configPath == "" {
		configPath = "lab.yaml"
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s\nRun 'sentra lab init' to create a new project", configPath)
	}

	var err error
	sc.configLoader, err = config.NewLoader(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg, err := sc.configLoader.Load()
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	sc.dockerManager, err = docker.NewManager(sc.logger, cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize Docker manager: %w", err)
	}

	return nil
}

func (sc *StartCommand) RunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	sc.logger.Info("🚀 Starting Sentra Lab services...")

	if err := sc.dockerManager.CheckDockerRunning(ctx); err != nil {
		return fmt.Errorf("Docker check failed: %w\n\nMake sure Docker is installed and running:\n  macOS: open /Applications/Docker.app\n  Linux: sudo systemctl start docker\n  Windows: Start Docker Desktop", err)
	}

	if sc.pull {
		sc.logger.Info("📥 Pulling latest Docker images...")
		if err := sc.dockerManager.PullImages(ctx); err != nil {
			return fmt.Errorf("failed to pull images: %w", err)
		}
	}

	if sc.rebuild {
		sc.logger.Info("🔨 Rebuilding containers...")
		if err := sc.dockerManager.Stop(ctx); err != nil {
			sc.logger.Warn("Failed to stop existing containers", "error", err)
		}
	}

	sc.logger.Info("🔧 Starting Docker containers...")
	if err := sc.dockerManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start services: %w", err)
	}

	sc.logger.Info("⏳ Waiting for services to be healthy...")
	if err := sc.dockerManager.WaitHealthy(ctx, 60*time.Second); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	services := sc.dockerManager.GetServiceURLs()
	
	sc.logger.Info("✅ All services running:")
	for _, svc := range services {
		healthTime := sc.dockerManager.GetHealthCheckTime(svc.Name)
		sc.logger.Info(fmt.Sprintf("  ✓ %-20s %s (%dms)", svc.Name, svc.URL, healthTime.Milliseconds()))
	}

	sc.logger.Info("")
	sc.logger.Info("💡 Configure your agent to use these endpoints:")
	for _, svc := range services {
		if svc.EnvVar != "" {
			sc.logger.Info(fmt.Sprintf("   export %s=%s", svc.EnvVar, svc.URL))
		}
	}

	sc.logger.Info("")
	sc.logger.Info("Next steps:")
	sc.logger.Info("  • Run 'sentra lab test' to test your agent")
	sc.logger.Info("  • Run 'sentra lab logs' to view service logs")
	sc.logger.Info("  • Run 'sentra lab stop' to stop services")

	if sc.detach {
		sc.logger.Info("")
		sc.logger.Info("Services running in background. Use 'sentra lab logs -f' to follow logs.")
		return nil
	}

	sc.logger.Info("")
	sc.logger.Info("📋 Streaming logs (Ctrl+C to stop)...")
	sc.logger.Info("")

	return sc.streamLogs(ctx)
}

func (sc *StartCommand) streamLogs(ctx context.Context) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	logCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- sc.dockerManager.StreamLogs(logCtx, "all")
	}()

	select {
	case <-sigChan:
		sc.logger.Info("")
		sc.logger.Info("⚠️  Received interrupt signal")
		sc.logger.Info("Services are still running. Use 'sentra lab stop' to stop them.")
		cancel()
		return nil
	case err := <-errChan:
		return err
	}
}

func (sc *StartCommand) Stop(ctx context.Context) error {
	if sc.dockerManager == nil {
		configPath := "lab.yaml"
		var err error
		sc.configLoader, err = config.NewLoader(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		cfg, err := sc.configLoader.Load()
		if err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}

		sc.dockerManager, err = docker.NewManager(sc.logger, cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize Docker manager: %w", err)
		}
	}

	sc.logger.Info("🛑 Stopping Sentra Lab services...")

	if err := sc.dockerManager.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop services: %w", err)
	}

	sc.logger.Info("✅ All services stopped")
	return nil
}

func (sc *StartCommand) Logs(ctx context.Context, service string, follow bool, tail int) error {
	if sc.dockerManager == nil {
		configPath := "lab.yaml"
		var err error
		sc.configLoader, err = config.NewLoader(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		cfg, err := sc.configLoader.Load()
		if err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}

		sc.dockerManager, err = docker.NewManager(sc.logger, cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize Docker manager: %w", err)
		}
	}

	if follow {
		return sc.dockerManager.StreamLogs(ctx, service)
	}

	return sc.dockerManager.GetLogs(ctx, service, tail)
}

func (sc *StartCommand) Status(ctx context.Context) error {
	if sc.dockerManager == nil {
		configPath := "lab.yaml"
		var err error
		sc.configLoader, err = config.NewLoader(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		cfg, err := sc.configLoader.Load()
		if err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}

		sc.dockerManager, err = docker.NewManager(sc.logger, cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize Docker manager: %w", err)
		}
	}

	status, err := sc.dockerManager.GetStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	sc.logger.Info("Sentra Lab Services:")
	sc.logger.Info("")

	for _, svc := range status {
		statusIcon := "✓"
		statusColor := "\033[32m"
		if svc.Status != "healthy" {
			statusIcon = "✗"
			statusColor = "\033[31m"
		}

		sc.logger.Info(fmt.Sprintf("%s%s %-20s\033[0m %s", statusColor, statusIcon, svc.Name, svc.URL))
		sc.logger.Info(fmt.Sprintf("    Status: %s", svc.Status))
		if svc.Uptime > 0 {
			sc.logger.Info(fmt.Sprintf("    Uptime: %s", time.Duration(svc.Uptime).String()))
		}
		if svc.Memory > 0 {
			sc.logger.Info(fmt.Sprintf("    Memory: %s", formatBytes(svc.Memory)))
		}
		sc.logger.Info("")
	}

	return nil
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
