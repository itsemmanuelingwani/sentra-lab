package replay

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Exporter struct {
	outputPath string
	format     string
}

func NewExporter(outputPath string) *Exporter {
	format := "json"
	ext := strings.ToLower(filepath.Ext(outputPath))

	switch ext {
	case ".json":
		format = "json"
	case ".html":
		format = "html"
	case ".har":
		format = "har"
	case ".md":
		format = "markdown"
	}

	return &Exporter{
		outputPath: outputPath,
		format:     format,
	}
}

func (e *Exporter) Export(recording *Recording) error {
	switch e.format {
	case "json":
		return e.exportJSON(recording)
	case "html":
		return e.exportHTML(recording)
	case "har":
		return e.exportHAR(recording)
	case "markdown":
		return e.exportMarkdown(recording)
	default:
		return fmt.Errorf("unsupported format: %s", e.format)
	}
}

func (e *Exporter) exportJSON(recording *Recording) error {
	data, err := json.MarshalIndent(recording, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return os.WriteFile(e.outputPath, data, 0644)
}

func (e *Exporter) exportHTML(recording *Recording) error {
	html := generateHTML(recording)
	return os.WriteFile(e.outputPath, []byte(html), 0644)
}

func (e *Exporter) exportHAR(recording *Recording) error {
	har := convertToHAR(recording)

	data, err := json.MarshalIndent(har, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal HAR: %w", err)
	}

	return os.WriteFile(e.outputPath, data, 0644)
}

func (e *Exporter) exportMarkdown(recording *Recording) error {
	md := generateMarkdown(recording)
	return os.WriteFile(e.outputPath, []byte(md), 0644)
}

func generateHTML(recording *Recording) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Replay: %s</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .header {
            background: white;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 20px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        }
        .header h1 {
            margin: 0 0 10px 0;
            color: #333;
        }
        .metadata {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 10px;
            color: #666;
            font-size: 14px;
        }
        .timeline {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        }
        .event {
            padding: 15px;
            border-left: 3px solid #3b82f6;
            margin-bottom: 10px;
            background: #f9fafb;
            border-radius: 4px;
        }
        .event.error {
            border-left-color: #ef4444;
        }
        .event-header {
            display: flex;
            justify-content: space-between;
            font-weight: 500;
            margin-bottom: 5px;
        }
        .event-details {
            font-size: 14px;
            color: #666;
        }
        .timestamp {
            color: #9ca3af;
            font-family: monospace;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>Test Run Replay</h1>
        <div class="metadata">
            <div><strong>Scenario:</strong> %s</div>
            <div><strong>Status:</strong> %s</div>
            <div><strong>Duration:</strong> %s</div>
            <div><strong>Events:</strong> %d</div>
        </div>
    </div>
    <div class="timeline">
        <h2>Event Timeline</h2>
        %s
    </div>
</body>
</html>`,
		recording.Scenario,
		recording.Scenario,
		recording.Status,
		recording.Duration,
		len(recording.Events),
		generateEventTimeline(recording.Events),
	)
}

func generateEventTimeline(events []*Event) string {
	var html strings.Builder

	for _, event := range events {
		errorClass := ""
		if event.Error != nil {
			errorClass = " error"
		}

		html.WriteString(fmt.Sprintf(`
        <div class="event%s">
            <div class="event-header">
                <span>%s</span>
                <span class="timestamp">%s</span>
            </div>
            <div class="event-details">
                Service: %s | Type: %s | Duration: %s
            </div>
        </div>`,
			errorClass,
			event.Summary,
			event.Timestamp.Format("15:04:05.000"),
			event.Service,
			event.Type,
			event.Duration,
		))
	}

	return html.String()
}

func convertToHAR(recording *Recording) map[string]interface{} {
	entries := []map[string]interface{}{}

	for _, event := range recording.Events {
		if event.Type == "http_request" || event.Type == "http_response" {
			entry := map[string]interface{}{
				"startedDateTime": event.Timestamp.Format("2006-01-02T15:04:05.000Z"),
				"time":            event.Duration.Milliseconds(),
				"request": map[string]interface{}{
					"method":      "POST",
					"url":         fmt.Sprintf("http://localhost/%s", event.Service),
					"httpVersion": "HTTP/1.1",
					"headers":     []interface{}{},
					"queryString": []interface{}{},
					"postData":    event.Request,
				},
				"response": map[string]interface{}{
					"status":      200,
					"statusText":  "OK",
					"httpVersion": "HTTP/1.1",
					"headers":     []interface{}{},
					"content":     event.Response,
				},
				"cache":    map[string]interface{}{},
				"timings":  map[string]interface{}{"wait": 0, "receive": event.Duration.Milliseconds()},
			}
			entries = append(entries, entry)
		}
	}

	return map[string]interface{}{
		"log": map[string]interface{}{
			"version": "1.2",
			"creator": map[string]interface{}{
				"name":    "Sentra Lab",
				"version": "1.0.0",
			},
			"entries": entries,
		},
	}
}

func generateMarkdown(recording *Recording) string {
	var md strings.Builder

	md.WriteString(fmt.Sprintf("# Test Run: %s\n\n", recording.Scenario))
	md.WriteString(fmt.Sprintf("**Status:** %s\n", recording.Status))
	md.WriteString(fmt.Sprintf("**Started:** %s\n", recording.StartedAt.Format("2006-01-02 15:04:05")))
	md.WriteString(fmt.Sprintf("**Duration:** %s\n", recording.Duration))
	md.WriteString(fmt.Sprintf("**Events:** %d\n\n", len(recording.Events)))

	md.WriteString("## Event Timeline\n\n")

	for i, event := range recording.Events {
		icon := "•"
		if event.Error != nil {
			icon = "❌"
		}

		md.WriteString(fmt.Sprintf("%d. %s **[%s]** %s\n",
			i+1,
			icon,
			event.Timestamp.Format("15:04:05.000"),
			event.Summary,
		))

		md.WriteString(fmt.Sprintf("   - Service: %s\n", event.Service))
		md.WriteString(fmt.Sprintf("   - Type: %s\n", event.Type))
		md.WriteString(fmt.Sprintf("   - Duration: %s\n", event.Duration))

		if event.TokensUsed > 0 {
			md.WriteString(fmt.Sprintf("   - Tokens: %d\n", event.TokensUsed))
		}

		if event.CostUSD > 0 {
			md.WriteString(fmt.Sprintf("   - Cost: $%.4f\n", event.CostUSD))
		}

		if event.Error != nil {
			md.WriteString(fmt.Sprintf("   - Error: %v\n", event.Error))
		}

		md.WriteString("\n")
	}

	return md.String()
}
