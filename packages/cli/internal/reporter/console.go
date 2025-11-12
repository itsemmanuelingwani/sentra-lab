package reporter

import (
	"fmt"
	"io"
	"strings"
	"time"
)

type ConsoleReporter struct {
	verbose bool
}

func NewConsoleReporter(logger interface{}) Reporter {
	return &ConsoleReporter{
		verbose: false,
	}
}

func (cr *ConsoleReporter) Report(w io.Writer, summary interface{}, results interface{}) error {
	fmt.Fprintln(w, "\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(w, "Test Results")
	fmt.Fprintln(w, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	return nil
}

type Reporter interface {
	Report(w io.Writer, summary interface{}, results interface{}) error
}