package test

import (
	"fmt"
	"time"
)

type TestReporter struct {
	verbose bool
}

func NewTestReporter(verbose bool) *TestReporter {
	return &TestReporter{
		verbose: verbose,
	}
}

func (tr *TestReporter) ReportStart(total int) {
	fmt.Printf("\n🧪 Running %d scenario(s)...\n\n", total)
}

func (tr *TestReporter) ReportScenario(result *TestResult) {
	icon := "✓"
	color := "\033[32m"

	if result.Status == "failed" {
		icon = "✗"
		color = "\033[31m"
	} else if result.Status == "skipped" {
		icon = "⊘"
		color = "\033[33m"
	}

	fmt.Printf("%s%s\033[0m %-50s %6.2fs  $%.4f\n",
		color,
		icon,
		result.Scenario,
		result.Duration.Seconds(),
		result.CostUSD,
	)

	if tr.verbose && result.Status == "failed" {
		for _, failure := range result.Failures {
			fmt.Printf("    └─ %s\n", failure)
		}
	}
}

func (tr *TestReporter) ReportSummary(summary *TestSummary) {
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	passRate := 0.0
	if summary.Total > 0 {
		passRate = float64(summary.Passed) / float64(summary.Total) * 100
	}

	fmt.Printf("Test Results: %d/%d passed (%.1f%%)\n", summary.Passed, summary.Total, passRate)
	fmt.Printf("Duration: %s\n", summary.Duration.Round(time.Millisecond))
	fmt.Printf("Total Cost: $%.4f (simulated)\n", summary.TotalCost)

	if summary.Skipped > 0 {
		fmt.Printf("Skipped: %d\n", summary.Skipped)
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func (tr *TestReporter) ReportFailures(results []*TestResult) {
	failures := 0
	for _, result := range results {
		if result.Status == "failed" {
			failures++
		}
	}

	if failures == 0 {
		return
	}

	fmt.Printf("\n❌ Failed Scenarios (%d):\n\n", failures)

	for _, result := range results {
		if result.Status == "failed" {
			fmt.Printf("  • %s\n", result.Scenario)
			fmt.Printf("    Run ID: %s\n", result.RunID)
			fmt.Printf("    Duration: %s\n", result.Duration.Round(time.Millisecond))

			if len(result.Failures) > 0 {
				fmt.Println("    Failures:")
				for _, failure := range result.Failures {
					fmt.Printf("      - %s\n", failure)
				}
			}

			fmt.Printf("    Replay: sentra lab replay %s\n\n", result.RunID)
		}
	}
}

func (tr *TestReporter) ReportProgress(scenario string, status string, progress float64) {
	if !tr.verbose {
		return
	}

	icon := "⏳"
	if status == "passed" {
		icon = "✓"
	} else if status == "failed" {
		icon = "✗"
	}

	fmt.Printf("%s %-50s %.0f%%\n", icon, scenario, progress*100)
}
