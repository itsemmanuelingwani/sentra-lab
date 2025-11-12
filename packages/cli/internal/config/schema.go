package config

type Schema struct {
	Version string
	Fields  []FieldSchema
}

type FieldSchema struct {
	Name        string
	Type        string
	Required    bool
	Default     interface{}
	Description string
	Validation  ValidationRule
}

type ValidationRule struct {
	MinValue    interface{}
	MaxValue    interface{}
	AllowedValues []interface{}
	Pattern     string
}

func GetSchema(version string) *Schema {
	switch version {
	case "1.0", "1.1":
		return getV1Schema()
	default:
		return getV1Schema()
	}
}

func getV1Schema() *Schema {
	return &Schema{
		Version: "1.1",
		Fields: []FieldSchema{
			{
				Name:        "name",
				Type:        "string",
				Required:    true,
				Description: "Project name",
			},
			{
				Name:        "version",
				Type:        "string",
				Required:    true,
				Description: "Configuration schema version",
			},
			{
				Name:        "agent.runtime",
				Type:        "string",
				Required:    true,
				Description: "Agent runtime (python, nodejs, go)",
				Validation: ValidationRule{
					AllowedValues: []interface{}{"python", "nodejs", "go"},
				},
			},
			{
				Name:        "agent.entry_point",
				Type:        "string",
				Required:    true,
				Description: "Main agent file",
			},
			{
				Name:        "agent.timeout",
				Type:        "duration",
				Required:    false,
				Default:     "30s",
				Description: "Agent execution timeout",
			},
			{
				Name:        "simulation.record_full_trace",
				Type:        "boolean",
				Required:    false,
				Default:     true,
				Description: "Enable full execution recording",
			},
			{
				Name:        "simulation.enable_cost_tracking",
				Type:        "boolean",
				Required:    false,
				Default:     true,
				Description: "Track API costs",
			},
			{
				Name:        "simulation.max_concurrent_scenarios",
				Type:        "integer",
				Required:    false,
				Default:     10,
				Description: "Maximum parallel scenarios",
				Validation: ValidationRule{
					MinValue: 1,
					MaxValue: 100,
				},
			},
			{
				Name:        "storage.recordings_dir",
				Type:        "string",
				Required:    false,
				Default:     ".sentra-lab/recordings",
				Description: "Directory for recordings",
			},
			{
				Name:        "storage.database",
				Type:        "string",
				Required:    false,
				Default:     ".sentra-lab/sentra.db",
				Description: "SQLite database file",
			},
		},
	}
}

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) Validate(config *Config) error {
	return config.Validate()
}

type Migrator struct{}

func NewMigrator() *Migrator {
	return &Migrator{}
}

func (m *Migrator) Migrate(config *Config) (bool, error) {
	return false, nil
}