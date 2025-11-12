package reporter

import (
	"encoding/json"
	"io"
)

type JSONReporter struct{}

func NewJSONReporter() Reporter {
	return &JSONReporter{}
}

func (jr *JSONReporter) Report(w io.Writer, summary interface{}, results interface{}) error {
	report := map[string]interface{}{
		"summary": summary,
		"results": results,
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}