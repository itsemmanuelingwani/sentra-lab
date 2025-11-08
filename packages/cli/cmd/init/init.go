package init

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sentra-lab/cli/internal/utils"
	"github.com/spf13/cobra"
)

type InitCommand struct {
	logger   *utils.Logger
	template string
	force    bool
}

func NewInitCommand(logger *utils.Logger) *cobra.Command {
	ic := &InitCommand{
		logger: logger,
	}

	cmd := &cobra.Command{
		Use:   "init [name]",
		Short: "Initialize a new Sentra Lab project",
		Long: `Create a new Sentra Lab project with scaffolding.

This command creates:
  • Project directory structure
  • lab.yaml (configuration)
  • scenarios/ (example test scenarios)
  • mocks.yaml (mock service configuration)
  • .gitignore
  • README.md

Templates available:
  • default     - Basic agent project
  • python      - Python agent with OpenAI
  • nodejs      - Node.js agent with TypeScript
  • go          - Go agent
  • fullstack   - Complete setup with all mocks

Example:
  sentra lab init my-agent
  sentra lab init my-agent --template=python`,
		Args: cobra.ExactArgs(1),
		RunE: ic.RunE,
	}

	cmd.Flags().StringVar(&ic.template, "template", "default", "Project template (default, python, nodejs, go, fullstack)")
	cmd.Flags().BoolVar(&ic.force, "force", false, "Overwrite existing directory")

	return cmd
}

func (ic *InitCommand) RunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	name := args[0]

	ic.logger.Info("Initializing Sentra Lab project", "name", name, "template", ic.template)

	projectDir := filepath.Join(".", name)

	if _, err := os.Stat(projectDir); err == nil {
		if !ic.force {
			return fmt.Errorf("directory '%s' already exists. Use --force to overwrite", name)
		}
		ic.logger.Warn("Overwriting existing directory", "path", projectDir)
	}

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	if err := ic.scaffoldProject(ctx, projectDir, name); err != nil {
		return fmt.Errorf("failed to scaffold project: %w", err)
	}

	ic.logger.Info("✓ Project initialized successfully", "path", projectDir)
	ic.logger.Info("Next steps:")
	ic.logger.Info("  cd %s", name)
	ic.logger.Info("  sentra lab start    # Start mock services")
	ic.logger.Info("  sentra lab test     # Run test scenarios")

	return nil
}

func (ic *InitCommand) scaffoldProject(ctx context.Context, projectDir, name string) error {
	dirs := []string{
		"scenarios",
		"fixtures",
		"tests",
		".sentra-lab",
	}

	for _, dir := range dirs {
		path := filepath.Join(projectDir, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		ic.logger.Debug("Created directory", "path", path)
	}

	files := map[string]string{
		"lab.yaml":                  generateLabYAML(name),
		"mocks.yaml":                generateMocksYAML(),
		"scenarios/basic-test.yaml": generateBasicScenario(name),
		".gitignore":                generateGitignore(),
		"README.md":                 generateReadme(name),
	}

	switch ic.template {
	case "python":
		files["agent.py"] = generatePythonAgent()
		files["requirements.txt"] = generatePythonRequirements()
		files["scenarios/openai-test.yaml"] = generateOpenAIScenario()
	case "nodejs":
		files["agent.ts"] = generateNodeAgent()
		files["package.json"] = generatePackageJSON(name)
		files["tsconfig.json"] = generateTSConfig()
		files["scenarios/openai-test.yaml"] = generateOpenAIScenario()
	case "go":
		files["agent.go"] = generateGoAgent()
		files["go.mod"] = generateGoMod(name)
		files["scenarios/openai-test.yaml"] = generateOpenAIScenario()
	case "fullstack":
		files["agent.py"] = generatePythonAgent()
		files["requirements.txt"] = generatePythonRequirements()
		files["scenarios/payment-flow.yaml"] = generatePaymentScenario()
		files["scenarios/openai-test.yaml"] = generateOpenAIScenario()
	}

	for filename, content := range files {
		path := filepath.Join(projectDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", filename, err)
		}
		ic.logger.Debug("Created file", "path", path)
	}

	return nil
}
