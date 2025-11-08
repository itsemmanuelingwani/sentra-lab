package replay

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sentra-lab/cli/internal/config"
	"github.com/sentra-lab/cli/internal/grpc"
	"github.com/sentra-lab/cli/internal/ui"
	"github.com/sentra-lab/cli/internal/utils"
	"github.com/spf13/cobra"
)

type ReplayCommand struct {
	logger       *utils.Logger
	configLoader *config.Loader
	engineClient *grpc.EngineClient
	list         bool
	speed        float64
	breakpoint   string
	compare      string
	export       string
	stepByStep   bool
}

func NewReplayCommand(logger *utils.Logger) *cobra.Command {
	rc := &ReplayCommand{
		logger: logger,
	}

	cmd := &cobra.Command{
		Use:   "replay [run-id]",
		Short: "Replay and debug test executions",
		Long: `Interactive time-travel debugger for test runs.

Features:
  • Step-by-step execution replay
  • Event timeline visualization
  • State inspection at any point
  • Breakpoint support
  • Side-by-side comparison of runs
  • Export to various formats

The replay command provides an interactive TUI for debugging.
Use arrow keys to navigate, Space to play/pause, and 'q' to quit.

Example:
  sentra lab replay                     # Replay last failed run
  sentra lab replay run-abc123          # Replay specific run
  sentra lab replay --list              # List recent runs
  sentra lab replay run-abc123 --step   # Step-by-step mode
  sentra lab replay run-abc123 --compare run-def456  # Compare two runs
  sentra lab replay run-abc123 --export report.json  # Export to JSON`,
		PreRunE: rc.PreRunE,
		RunE:    rc.RunE,
	}

	cmd.Flags().BoolVar(&rc.list, "list", false, "List recent runs")
	cmd.Flags().Float64Var(&rc.speed, "speed", 1.0, "Playback speed (0.1 to 10.0)")
	cmd.Flags().StringVar(&rc.breakpoint, "breakpoint", "", "Pause at specific event ID")
	cmd.Flags().StringVar(&rc.compare, "compare", "", "Compare with another run")
	cmd.Flags().StringVar(&rc.export, "export", "", "Export to file (json, html, har)")
	cmd.Flags().BoolVar(&rc.stepByStep, "step", false, "Step-by-step mode")

	return cmd
}

func (rc *ReplayCommand) PreRunE(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")
	if configPath == "" {
		configPath = "lab.yaml"
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s", configPath)
	}

	var err error
	rc.configLoader, err = config.NewLoader(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg, err := rc.configLoader.Load()
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	rc.engineClient, err = grpc.NewEngineClient(cfg.GetEngineAddress())
	if err != nil {
		return fmt.Errorf("failed to create engine client: %w", err)
	}

	return nil
}

func (rc *ReplayCommand) RunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	if rc.list {
		return rc.listRuns(ctx)
	}

	var runID string
	if len(args) > 0 {
		runID = args[0]
	} else {
		lastFailed, err := rc.getLastFailedRun(ctx)
		if err != nil {
			return fmt.Errorf("no run ID specified and no failed runs found: %w", err)
		}
		runID = lastFailed
		rc.logger.Info(fmt.Sprintf("Replaying last failed run: %s", runID))
	}

	if rc.export != "" {
		return rc.exportRun(ctx, runID)
	}

	if rc.compare != "" {
		return rc.compareRuns(ctx, runID, rc.compare)
	}

	return rc.replayInteractive(ctx, runID)
}

func (rc *ReplayCommand) listRuns(ctx context.Context) error {
	rc.logger.Info("📋 Recent test runs:")
	rc.logger.Info("")

	runs, err := rc.engineClient.ListRuns(ctx, 20)
	if err != nil {
		return fmt.Errorf("failed to list runs: %w", err)
	}

	if len(runs) == 0 {
		rc.logger.Info("No runs found. Run 'sentra lab test' first.")
		return nil
	}

	for _, run := range runs {
		icon := "✓"
		color := "\033[32m"

		if run.Status == "failed" {
			icon = "✗"
			color = "\033[31m"
		}

		timeAgo := formatTimeAgo(run.CompletedAt)

		fmt.Printf("%s%s %s\033[0m  %-30s  %s\n",
			color,
			icon,
			run.ID,
			run.Scenario,
			timeAgo,
		)
	}

	rc.logger.Info("")
	rc.logger.Info("💡 Replay a run: sentra lab replay <run-id>")

	return nil
}

func (rc *ReplayCommand) getLastFailedRun(ctx context.Context) (string, error) {
	runs, err := rc.engineClient.ListRuns(ctx, 50)
	if err != nil {
		return "", err
	}

	for _, run := range runs {
		if run.Status == "failed" {
			return run.ID, nil
		}
	}

	return "", fmt.Errorf("no failed runs found")
}

func (rc *ReplayCommand) replayInteractive(ctx context.Context, runID string) error {
	rc.logger.Info(fmt.Sprintf("🔄 Loading replay for run: %s", runID))

	recording, err := rc.engineClient.GetRecording(ctx, runID)
	if err != nil {
		return fmt.Errorf("failed to load recording: %w", err)
	}

	rc.logger.Info(fmt.Sprintf("  Scenario: %s", recording.Scenario))
	rc.logger.Info(fmt.Sprintf("  Started: %s", recording.StartedAt.Format(time.RFC3339)))
	rc.logger.Info(fmt.Sprintf("  Duration: %s", recording.Duration))
	rc.logger.Info(fmt.Sprintf("  Events: %d", len(recording.Events)))
	rc.logger.Info("")
	rc.logger.Info("🎮 Starting interactive replay...")
	rc.logger.Info("   [←/→] Step  [Space] Play/Pause  [B] Breakpoint  [Q] Quit")
	rc.logger.Info("")

	model := ui.NewReplayModel(recording, rc.speed, rc.stepByStep, rc.breakpoint)

	return ui.RunReplayUI(model)
}

func (rc *ReplayCommand) compareRuns(ctx context.Context, runID1, runID2 string) error {
	rc.logger.Info(fmt.Sprintf("🔄 Comparing runs: %s vs %s", runID1, runID2))

	recording1, err := rc.engineClient.GetRecording(ctx, runID1)
	if err != nil {
		return fmt.Errorf("failed to load first recording: %w", err)
	}

	recording2, err := rc.engineClient.GetRecording(ctx, runID2)
	if err != nil {
		return fmt.Errorf("failed to load second recording: %w", err)
	}

	model := ui.NewComparisonModel(recording1, recording2)

	return ui.RunComparisonUI(model)
}

func (rc *ReplayCommand) exportRun(ctx context.Context, runID string) error {
	rc.logger.Info(fmt.Sprintf("📤 Exporting run: %s", runID))

	recording, err := rc.engineClient.GetRecording(ctx, runID)
	if err != nil {
		return fmt.Errorf("failed to load recording: %w", err)
	}

	exporter := NewExporter(rc.export)

	if err := exporter.Export(recording); err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	rc.logger.Info(fmt.Sprintf("✅ Exported to: %s", rc.export))

	return nil
}

func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return fmt.Sprintf("%d seconds ago", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%d minutes ago", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(duration.Hours()))
	}

	return fmt.Sprintf("%d days ago", int(duration.Hours()/24))
}
