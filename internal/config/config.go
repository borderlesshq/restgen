package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the restgen.yaml configuration.
type Config struct {
	Package string            `yaml:"package"` // output package name (e.g., "routes")
	Output  string            `yaml:"output"`  // output directory (e.g., "./routes")
	Models  ModelsConfig      `yaml:"models"`  // default models package config
	Scalars map[string]string `yaml:"scalars"` // scalar type mappings
	Schemas []string          `yaml:"schemas"` // glob patterns for schema files
}

// ModelsConfig specifies the default models package.
type ModelsConfig struct {
	Package string `yaml:"package"` // e.g., "github.com/yourorg/yourapp/models"
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Package: "routes",
		Output:  "./routes",
		Models: ModelsConfig{
			Package: "",
		},
		Scalars: map[string]string{
			"ID":      "string",
			"String":  "string",
			"Int":     "int",
			"Float":   "float64",
			"Boolean": "bool",
			"Time":    "time.Time",
		},
		Schemas: []string{"./schemas/*.sdl"},
	}
}

// Load reads configuration from a YAML file.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // use defaults if no config
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Ensure scalars have defaults
	if cfg.Scalars == nil {
		cfg.Scalars = DefaultConfig().Scalars
	} else {
		defaults := DefaultConfig().Scalars
		for k, v := range defaults {
			if _, ok := cfg.Scalars[k]; !ok {
				cfg.Scalars[k] = v
			}
		}
	}

	return cfg, nil
}

// GoType converts a GraphQL type to a Go type using scalar mappings.
func (c *Config) GoType(gqlType string, required bool, isList bool) string {
	goType := gqlType
	if mapped, ok := c.Scalars[gqlType]; ok {
		goType = mapped
	}

	if isList {
		goType = "[]" + goType
	}

	if !required && !isList {
		goType = "*" + goType
	}

	return goType
}
