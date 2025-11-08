package config

import (
	"fmt"
	"strings"
)

type ValidationError struct {
	Field   string
	Message string
	Hint    string
}

func (ve *ValidationError) Error() string {
	msg := fmt.Sprintf("  • %s: %s", ve.Field, ve.Message)
	if ve.Hint != "" {
		msg += fmt.Sprintf("\n    💡 %s", ve.Hint)
	}
	return msg
}

type Validator struct {
	errors []ValidationError
}

func NewValidator() *Validator {
	return &Validator{
		errors: []ValidationError{},
	}
}

func (v *Validator) Validate(cfg interface{}) error {
	data, ok := cfg.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid config format")
	}

	v.validateName(data)
	v.validateVersion(data)
	v.validateAgent(data)
	v.validateMocks(data)
	v.validateSimulation(data)
	v.validateStorage(data)

	if len(v.errors) > 0 {
		return v.formatErrors()
	}

	return nil
}

func (v *Validator) validateName(data map[string]interface{}) {
	name, ok := data["name"].(string)
	if !ok || name == "" {
		v.addError("name", "is required", "Add a project name: name: my-agent")
	}
}

func (v *Validator) validateVersion(data map[string]interface{}) {
	version, ok := data["version"].(string)
	if !ok || version == "" {
		v.addError("version", "is required", "Add version: version: \"1.0\"")
	}
}

func (v *Validator) validateAgent(data map[string]interface{}) {
	agent, ok := data["agent"].(map[string]interface{})
	if !ok {
		v.addError("agent", "section is required", "Add agent configuration")
		return
	}

	runtime, ok := agent["runtime"].(string)
	if !ok || runtime == "" {
		v.addError("agent.runtime", "is required", "Specify: python, nodejs, or go")
	} else {
		validRuntimes := []string{"python", "nodejs", "go"}
		if !contains(validRuntimes, runtime) {
			v.addError("agent.runtime", fmt.Sprintf("invalid value: %s", runtime),
				"Must be one of: python, nodejs, go")
		}
	}

	entryPoint, ok := agent["entry_point"].(string)
	if !ok || entryPoint == "" {
		v.addError("agent.entry_point", "is required", "Specify the main file: agent.py, agent.ts, agent.go")
	}

	if timeout, ok := agent["timeout"].(string); ok {
		if !isValidDuration(timeout) {
			v.addError("agent.timeout", fmt.Sprintf("invalid duration: %s", timeout),
				"Use format: 30s, 5m, 1h")
		}
	}
}

func (v *Validator) validateMocks(data map[string]interface{}) {
	mocks, ok := data["mocks"].(map[string]interface{})
	if !ok {
		return
	}

	for mockName, mockConfig := range mocks {
		mockData, ok := mockConfig.(map[string]interface{})
		if !ok {
			continue
		}

		if enabled, ok := mockData["enabled"].(bool); ok && enabled {
			if port, ok := mockData["port"].(int); ok {
				if port < 1024 || port > 65535 {
					v.addError(fmt.Sprintf("mocks.%s.port", mockName),
						fmt.Sprintf("invalid port: %d", port),
						"Use port between 1024 and 65535")
				}
			}

			if latency, ok := mockData["latency_ms"].(int); ok {
				if latency < 0 {
					v.addError(fmt.Sprintf("mocks.%s.latency_ms", mockName),
						"cannot be negative",
						"Use positive value or 0")
				}
			}

			if rateLimit, ok := mockData["rate_limit"].(int); ok {
				if rateLimit < 0 {
					v.addError(fmt.Sprintf("mocks.%s.rate_limit", mockName),
						"cannot be negative",
						"Use positive value or 0 for unlimited")
				}
			}

			if errorRate, ok := mockData["error_rate"].(float64); ok {
				if errorRate < 0 || errorRate > 1 {
					v.addError(fmt.Sprintf("mocks.%s.error_rate", mockName),
						fmt.Sprintf("invalid value: %f", errorRate),
						"Use value between 0.0 and 1.0")
				}
			}
		}
	}
}

func (v *Validator) validateSimulation(data map[string]interface{}) {
	simulation, ok := data["simulation"].(map[string]interface{})
	if !ok {
		return
	}

	if maxConcurrent, ok := simulation["max_concurrent_scenarios"].(int); ok {
		if maxConcurrent < 1 {
			v.addError("simulation.max_concurrent_scenarios",
				"must be at least 1",
				"Use value >= 1")
		}
		if maxConcurrent > 100 {
			v.addError("simulation.max_concurrent_scenarios",
				"too high (max 100)",
				"Reduce to avoid resource exhaustion")
		}
	}
}

func (v *Validator) validateStorage(data map[string]interface{}) {
	storage, ok := data["storage"].(map[string]interface{})
	if !ok {
		return
	}

	if recordingsDir, ok := storage["recordings_dir"].(string); ok {
		if recordingsDir == "" {
			v.addError("storage.recordings_dir",
				"cannot be empty",
				"Specify directory path")
		}
	}

	if database, ok := storage["database"].(string); ok {
		if database == "" {
			v.addError("storage.database",
				"cannot be empty",
				"Specify database file path")
		}
	}
}

func (v *Validator) addError(field, message, hint string) {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Message: message,
		Hint:    hint,
	})
}

func (v *Validator) formatErrors() error {
	var builder strings.Builder

	builder.WriteString("❌ Configuration validation failed:\n\n")

	for _, err := range v.errors {
		builder.WriteString(err.Error())
		builder.WriteString("\n")
	}

	builder.WriteString("\n📖 See: https://docs.sentra.dev/configuration\n")

	return fmt.Errorf(builder.String())
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func isValidDuration(duration string) bool {
	validSuffixes := []string{"s", "m", "h", "ms", "us", "ns"}

	for _, suffix := range validSuffixes {
		if strings.HasSuffix(duration, suffix) {
			return true
		}
	}

	return false
}
