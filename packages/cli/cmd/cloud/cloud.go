package cloud

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sentra-lab/cli/internal/utils"
	"github.com/spf13/cobra"
)

type CloudCommand struct {
	logger     *utils.Logger
	authClient *AuthClient
	syncClient *SyncClient
}

func NewCloudCommand(logger *utils.Logger) *cobra.Command {
	cc := &CloudCommand{
		logger: logger,
	}

	cmd := &cobra.Command{
		Use:   "cloud",
		Short: "Sentra Lab Cloud features",
		Long: `Manage cloud features for team collaboration.

Cloud features include:
  • Team sharing and collaboration
  • CI/CD integration
  • Load testing at scale
  • Analytics dashboard
  • Managed mock fixtures

Note: Cloud features require a Sentra Lab account.
Sign up at: https://lab.sentra.dev

Commands:
  • login    - Authenticate with Sentra Lab Cloud
  • logout   - Sign out
  • push     - Upload test runs to cloud
  • pull     - Download shared test runs
  • list     - List cloud runs
  • sync     - Sync local data with cloud`,
	}

	cmd.AddCommand(newLoginCommand(cc))
	cmd.AddCommand(newLogoutCommand(cc))
	cmd.AddCommand(newPushCommand(cc))
	cmd.AddCommand(newPullCommand(cc))
	cmd.AddCommand(newListCommand(cc))
	cmd.AddCommand(newSyncCommand(cc))
	cmd.AddCommand(newStatusCommand(cc))

	return cmd
}

func newLoginCommand(cc *CloudCommand) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Sentra Lab Cloud",
		Long: `Authenticate with Sentra Lab Cloud to enable team features.

This will:
  1. Open browser for OAuth authentication
  2. Save authentication token locally
  3. Enable cloud sync features

Your credentials are stored securely in:
  ~/.sentra-lab/credentials

Example:
  sentra lab cloud login`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			cc.logger.Info("🔐 Authenticating with Sentra Lab Cloud...")
			cc.logger.Info("Opening browser for authentication...")

			cc.authClient = NewAuthClient(cc.logger)

			token, user, err := cc.authClient.Login(ctx)
			if err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}

			if err := cc.saveCredentials(token, user); err != nil {
				return fmt.Errorf("failed to save credentials: %w", err)
			}

			cc.logger.Info("")
			cc.logger.Info(fmt.Sprintf("✅ Logged in as: %s", user.Email))
			if user.Team != "" {
				cc.logger.Info(fmt.Sprintf("   Team: %s", user.Team))
			}
			cc.logger.Info("")
			cc.logger.Info("Cloud features enabled:")
			cc.logger.Info("  • Team sharing: sentra lab cloud push")
			cc.logger.Info("  • Download shared runs: sentra lab cloud pull")
			cc.logger.Info("  • View dashboard: https://lab.sentra.dev")

			return nil
		},
	}
}

func newLogoutCommand(cc *CloudCommand) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Sign out from Sentra Lab Cloud",
		Long:  "Remove local authentication credentials and disable cloud features.",
		RunE: func(cmd *cobra.Command, args []string) error {
			credPath := cc.getCredentialsPath()

			if _, err := os.Stat(credPath); os.IsNotExist(err) {
				cc.logger.Info("Not logged in")
				return nil
			}

			if err := os.Remove(credPath); err != nil {
				return fmt.Errorf("failed to remove credentials: %w", err)
			}

			cc.logger.Info("✅ Logged out successfully")
			return nil
		},
	}
}

func newPushCommand(cc *CloudCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push [run-id]",
		Short: "Upload test runs to cloud",
		Long: `Upload test run recordings to Sentra Lab Cloud for team sharing.

Features:
  • Share test results with team
  • Persistent storage
  • Web-based visualization
  • Commenting and collaboration

Example:
  sentra lab cloud push                 # Push all recent runs
  sentra lab cloud push run-abc123      # Push specific run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if !cc.isAuthenticated() {
				return fmt.Errorf("not logged in. Run 'sentra lab cloud login' first")
			}

			cc.syncClient = NewSyncClient(cc.logger, cc.loadToken())

			var runIDs []string
			if len(args) > 0 {
				runIDs = []string{args[0]}
			} else {
				var err error
				runIDs, err = cc.getRecentRunIDs()
				if err != nil {
					return fmt.Errorf("failed to get recent runs: %w", err)
				}
			}

			if len(runIDs) == 0 {
				cc.logger.Info("No runs to push")
				return nil
			}

			cc.logger.Info(fmt.Sprintf("📤 Uploading %d run(s) to cloud...", len(runIDs)))

			uploaded := 0
			for _, runID := range runIDs {
				if err := cc.syncClient.PushRun(ctx, runID); err != nil {
					cc.logger.Warn(fmt.Sprintf("Failed to upload %s: %v", runID, err))
					continue
				}
				uploaded++
				cc.logger.Info(fmt.Sprintf("  ✓ %s", runID))
			}

			cc.logger.Info("")
			cc.logger.Info(fmt.Sprintf("✅ Uploaded %d/%d runs", uploaded, len(runIDs)))
			cc.logger.Info("View at: https://lab.sentra.dev/runs")

			return nil
		},
	}

	return cmd
}

func newPullCommand(cc *CloudCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull [run-id]",
		Short: "Download shared test runs",
		Long: `Download test runs shared by your team from Sentra Lab Cloud.

This allows you to:
  • Replay teammate's test runs
  • Debug issues collaboratively
  • Review test results

Example:
  sentra lab cloud pull                 # Pull recent team runs
  sentra lab cloud pull run-abc123      # Pull specific run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if !cc.isAuthenticated() {
				return fmt.Errorf("not logged in. Run 'sentra lab cloud login' first")
			}

			cc.syncClient = NewSyncClient(cc.logger, cc.loadToken())

			if len(args) > 0 {
				runID := args[0]
				cc.logger.Info(fmt.Sprintf("📥 Downloading run: %s", runID))

				if err := cc.syncClient.PullRun(ctx, runID); err != nil {
					return fmt.Errorf("download failed: %w", err)
				}

				cc.logger.Info("✅ Download complete")
				cc.logger.Info(fmt.Sprintf("Replay: sentra lab replay %s", runID))
			} else {
				cc.logger.Info("📥 Downloading recent team runs...")

				runs, err := cc.syncClient.ListTeamRuns(ctx, 10)
				if err != nil {
					return fmt.Errorf("failed to list runs: %w", err)
				}

				if len(runs) == 0 {
					cc.logger.Info("No team runs available")
					return nil
				}

				downloaded := 0
				for _, run := range runs {
					if err := cc.syncClient.PullRun(ctx, run.ID); err != nil {
						cc.logger.Warn(fmt.Sprintf("Failed to download %s: %v", run.ID, err))
						continue
					}
					downloaded++
					cc.logger.Info(fmt.Sprintf("  ✓ %s (%s)", run.ID, run.Scenario))
				}

				cc.logger.Info("")
				cc.logger.Info(fmt.Sprintf("✅ Downloaded %d/%d runs", downloaded, len(runs)))
			}

			return nil
		},
	}

	return cmd
}

func newListCommand(cc *CloudCommand) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List cloud runs",
		Long:  "Display test runs stored in Sentra Lab Cloud for your team.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if !cc.isAuthenticated() {
				return fmt.Errorf("not logged in. Run 'sentra lab cloud login' first")
			}

			cc.syncClient = NewSyncClient(cc.logger, cc.loadToken())

			runs, err := cc.syncClient.ListTeamRuns(ctx, 50)
			if err != nil {
				return fmt.Errorf("failed to list runs: %w", err)
			}

			if len(runs) == 0 {
				cc.logger.Info("No cloud runs found")
				return nil
			}

			cc.logger.Info("☁️  Cloud Runs:")
			cc.logger.Info("")

			for _, run := range runs {
				icon := "✓"
				color := "\033[32m"

				if run.Status == "failed" {
					icon = "✗"
					color = "\033[31m"
				}

				timeAgo := formatTimeAgo(run.UploadedAt)
				uploadedBy := run.UploadedBy
				if uploadedBy == "" {
					uploadedBy = "unknown"
				}

				fmt.Printf("%s%s %s\033[0m  %-30s  %s (by %s)\n",
					color,
					icon,
					run.ID,
					run.Scenario,
					timeAgo,
					uploadedBy,
				)
			}

			cc.logger.Info("")
			cc.logger.Info(fmt.Sprintf("Total: %d runs", len(runs)))
			cc.logger.Info("💡 Download: sentra lab cloud pull <run-id>")

			return nil
		},
	}
}

func newSyncCommand(cc *CloudCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync local data with cloud",
		Long: `Bidirectional sync between local and cloud storage.

This will:
  • Upload new local runs
  • Download new team runs
  • Sync scenarios and fixtures

Example:
  sentra lab cloud sync`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if !cc.isAuthenticated() {
				return fmt.Errorf("not logged in. Run 'sentra lab cloud login' first")
			}

			cc.syncClient = NewSyncClient(cc.logger, cc.loadToken())

			cc.logger.Info("🔄 Syncing with cloud...")

			stats, err := cc.syncClient.Sync(ctx)
			if err != nil {
				return fmt.Errorf("sync failed: %w", err)
			}

			cc.logger.Info("")
			cc.logger.Info("✅ Sync complete:")
			cc.logger.Info(fmt.Sprintf("  ↑ Uploaded: %d runs", stats.Uploaded))
			cc.logger.Info(fmt.Sprintf("  ↓ Downloaded: %d runs", stats.Downloaded))
			cc.logger.Info(fmt.Sprintf("  ⚠️  Conflicts: %d", stats.Conflicts))

			return nil
		},
	}

	return cmd
}

func newStatusCommand(cc *CloudCommand) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show cloud authentication status",
		Long:  "Display current authentication status and account information.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cc.isAuthenticated() {
				cc.logger.Info("❌ Not logged in")
				cc.logger.Info("Run 'sentra lab cloud login' to authenticate")
				return nil
			}

			user, err := cc.loadUser()
			if err != nil {
				return fmt.Errorf("failed to load user info: %w", err)
			}

			cc.logger.Info("✅ Logged in")
			cc.logger.Info(fmt.Sprintf("Email: %s", user.Email))
			if user.Team != "" {
				cc.logger.Info(fmt.Sprintf("Team: %s", user.Team))
			}
			cc.logger.Info(fmt.Sprintf("Plan: %s", user.Plan))

			return nil
		},
	}
}

func (cc *CloudCommand) isAuthenticated() bool {
	credPath := cc.getCredentialsPath()
	_, err := os.Stat(credPath)
	return err == nil
}

func (cc *CloudCommand) getCredentialsPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".sentra-lab", "credentials")
}

func (cc *CloudCommand) saveCredentials(token string, user *User) error {
	homeDir, _ := os.UserHomeDir()
	credDir := filepath.Join(homeDir, ".sentra-lab")

	if err := os.MkdirAll(credDir, 0700); err != nil {
		return err
	}

	credPath := filepath.Join(credDir, "credentials")
	data := fmt.Sprintf("token=%s\nemail=%s\nteam=%s\nplan=%s\n",
		token, user.Email, user.Team, user.Plan)

	return os.WriteFile(credPath, []byte(data), 0600)
}

func (cc *CloudCommand) loadToken() string {
	data, err := os.ReadFile(cc.getCredentialsPath())
	if err != nil {
		return ""
	}

	lines := string(data)
	for _, line := range strings.Split(lines, "\n") {
		if strings.HasPrefix(line, "token=") {
			return strings.TrimPrefix(line, "token=")
		}
	}

	return ""
}

func (cc *CloudCommand) loadUser() (*User, error) {
	data, err := os.ReadFile(cc.getCredentialsPath())
	if err != nil {
		return nil, err
	}

	user := &User{}
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]
		switch key {
		case "email":
			user.Email = value
		case "team":
			user.Team = value
		case "plan":
			user.Plan = value
		}
	}

	return user, nil
}

func (cc *CloudCommand) getRecentRunIDs() ([]string, error) {
	recordingsDir := ".sentra-lab/recordings"

	entries, err := os.ReadDir(recordingsDir)
	if err != nil {
		return nil, err
	}

	var runIDs []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "run-") {
			runIDs = append(runIDs, entry.Name())
		}
	}

	return runIDs, nil
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

type User struct {
	Email string
	Team  string
	Plan  string
}
