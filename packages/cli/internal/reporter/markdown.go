package reporter

import (
	"fmt"
	"io"
)

type MarkdownReporter struct{}

func NewMarkdownReporter() Reporter {
	return &MarkdownReporter{}
}

func (mr *MarkdownReporter) Report(w io.Writer, summary interface{}, results interface{}) error {
	fmt.Fprintln(w, "# Test Report")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "## Summary")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "## Results")
	fmt.Fprintln(w, "")
	
	return nil
}

type HTMLReporter struct{}

func NewHTMLReporter() Reporter {
	return &HTMLReporter{}
}

func (hr *HTMLReporter) Report(w io.Writer, summary interface{}, results interface{}) error {
	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Test Report</title>
    <style>
        body { font-family: system-ui; max-width: 1200px; margin: 0 auto; padding: 20px; }
        .header { background: #7D56F4; color: white; padding: 20px; border-radius: 8px; }
        .summary { background: #f5f5f5; padding: 20px; margin: 20px 0; border-radius: 8px; }
        .result { padding: 10px; margin: 10px 0; border-left: 4px solid #7D56F4; }
        .passed { border-left-color: #04B575; }
        .failed { border-left-color: #FF0000; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Test Report</h1>
    </div>
    <div class="summary">
        <h2>Summary</h2>
    </div>
    <div class="results">
        <h2>Results</h2>
    </div>
</body>
</html>`

	_, err := w.Write([]byte(html))
	return err
}