package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/borderlesshq/restgen/internal/config"
	"github.com/borderlesshq/restgen/internal/emitter"
	"github.com/borderlesshq/restgen/internal/merger"
	"github.com/borderlesshq/restgen/internal/parser"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "generate":
		if err := runGenerate(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "init":
		if err := runInit(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`restgen - Generate REST routes from SDL schemas

Usage:
  restgen generate [-c config.yaml]    Generate routes from schemas
  restgen init                         Initialize with example config and schema

Options:
  -c, --config    Path to config file (default: restgen.yaml)`)
}

func runGenerate(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	configPath := fs.String("c", "restgen.yaml", "config file path")
	fs.StringVar(configPath, "config", "restgen.yaml", "config file path")
	fs.Parse(args)

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Find schema files
	var schemaFiles []string
	for _, pattern := range cfg.Schemas {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return fmt.Errorf("glob pattern %s: %w", pattern, err)
		}
		schemaFiles = append(schemaFiles, matches...)
	}

	if len(schemaFiles) == 0 {
		return fmt.Errorf("no schema files found matching patterns: %v", cfg.Schemas)
	}

	// Process each schema
	p := parser.New()
	routesEmitter := emitter.NewRoutesEmitter(cfg)
	typesEmitter := emitter.NewTypesEmitter(cfg)
	depsEmitter := emitter.NewDependenciesEmitter(cfg.Package)
	m := merger.New()

	// Track directories to format
	dirsToFormat := make(map[string]bool)
	dirsToFormat[cfg.Output] = true

	// Ensure output directory exists
	if err := os.MkdirAll(cfg.Output, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	// Generate dependencies.go once (if it doesn't exist)
	depsFile := filepath.Join(cfg.Output, "dependencies.go")
	if _, err := os.Stat(depsFile); os.IsNotExist(err) {
		depsContent, err := depsEmitter.Emit()
		if err != nil {
			return fmt.Errorf("emitting dependencies: %w", err)
		}
		if err := os.WriteFile(depsFile, []byte(depsContent), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", depsFile, err)
		}
		fmt.Printf("→ %s (new)\n", depsFile)
	}

	for _, schemaFile := range schemaFiles {
		fmt.Printf("Processing %s...\n", schemaFile)

		schema, err := p.ParseFile(schemaFile)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", schemaFile, err)
		}

		// Derive handler name for this schema
		baseName := strings.Split(filepath.Base(schemaFile), ".")[0]

		// Generate routes
		routesContent, err := routesEmitter.Emit(schema)
		if err != nil {
			return fmt.Errorf("emitting routes for %s: %w", schemaFile, err)
		}

		// Output routes file
		routesFile := filepath.Join(cfg.Output, baseName+"_routes.go")

		// Merge with existing if present
		result, err := m.Merge(routesContent, routesFile)
		if err != nil {
			return fmt.Errorf("merging %s: %w", routesFile, err)
		}

		if err := os.WriteFile(routesFile, []byte(result.Content), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", routesFile, err)
		}
		fmt.Printf("  → %s\n", routesFile)

		if len(result.PreservedMethods) > 0 {
			fmt.Printf("    preserved: %v\n", result.PreservedMethods)
		}
		if len(result.RemovedMethods) > 0 {
			fmt.Printf("    removed: %v\n", result.RemovedMethods)
		}

		// Generate types if models path specified
		if schema.Models != "" {
			typesContent, err := typesEmitter.Emit(schema)
			if err != nil {
				return fmt.Errorf("emitting types for %s: %w", schemaFile, err)
			}

			// Derive types output path from models package
			// e.g., github.com/borderlesshq/api/models -> models/
			modelsParts := strings.Split(schema.Models, "/")
			modelsDir := modelsParts[len(modelsParts)-1]
			typesFile := filepath.Join(modelsDir, baseName+"_types.go")

			if err := os.MkdirAll(modelsDir, 0755); err != nil {
				return fmt.Errorf("creating models dir: %w", err)
			}

			if err := os.WriteFile(typesFile, []byte(typesContent), 0644); err != nil {
				return fmt.Errorf("writing %s: %w", typesFile, err)
			}
			fmt.Printf("  → %s\n", typesFile)

			dirsToFormat[modelsDir] = true
		}
	}

	// Format generated files with goimports
	fmt.Println("Formatting generated files...")
	for dir := range dirsToFormat {
		if err := runGoimports(dir); err != nil {
			fmt.Printf("  warning: goimports on %s failed: %v\n", dir, err)
		}
	}

	fmt.Println("Done!")
	return nil
}

// runGoimports runs goimports on the given directory to format code and fix imports.
// Falls back to gofmt if goimports is not available.
func runGoimports(dir string) error {
	// Check if goimports is available
	goimportsPath, err := exec.LookPath("goimports")
	if err != nil {
		// Check in GOPATH/bin
		gopath := os.Getenv("GOPATH")
		if gopath == "" {
			home, _ := os.UserHomeDir()
			gopath = filepath.Join(home, "go")
		}
		goimportsPath = filepath.Join(gopath, "bin", "goimports")
		if _, err := os.Stat(goimportsPath); err != nil {
			// goimports not found, fall back to gofmt
			fmt.Printf("  goimports not found, using gofmt (run 'go install golang.org/x/tools/cmd/goimports@latest' for better formatting)\n")
			return runGofmt(dir)
		}
	}

	// Run goimports -w on the directory
	cmd := exec.Command(goimportsPath, "-w", dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running goimports: %w", err)
	}

	return nil
}

// runGofmt runs gofmt as a fallback when goimports is not available.
func runGofmt(dir string) error {
	cmd := exec.Command("gofmt", "-w", dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running gofmt: %w", err)
	}
	return nil
}

func runInit() error {
	// Create example config
	configContent := `# restgen configuration
package: routes
output: ./routes

helpers:
  package: github.com/yourorg/yourapp/pkg/httputil

scalars:
  Time: time.Time
  ID: string
  Decimal: decimal.Decimal

schemas:
  - ./schemas/*.sdl
`

	// Create example schema
	schemaContent := `# @base("/v1/contacts")
# @models("github.com/yourorg/yourapp/models")

type Calls {
    createContact(input: CreateContactInput!): Contact @post("/")
    getContact(id: ID!): Contact @get("/{id}")
    updateContact(id: ID!, input: UpdateContactInput!): Contact @put("/{id}")
    deleteContact(id: ID!): DeleteResult @delete("/{id}")
    listContacts(filter: ContactFilter): ContactList @get("/")
}

type Contact {
    id: ID!
    name: String!
    email: String!
    createdAt: Time!
}

input CreateContactInput {
    name: String!
    email: String!
}

input UpdateContactInput {
    name: String
    email: String
}

input ContactFilter {
    search: String
    limit: Int
    offset: Int
}

type ContactList {
    items: [Contact!]!
    total: Int!
}

type DeleteResult {
    success: Boolean!
}
`

	// Write config
	if err := os.WriteFile("restgen.yaml", []byte(configContent), 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	fmt.Println("Created restgen.yaml")

	// Create schemas directory and example
	if err := os.MkdirAll("schemas", 0755); err != nil {
		return fmt.Errorf("creating schemas dir: %w", err)
	}

	if err := os.WriteFile("schemas/contacts.sdl", []byte(schemaContent), 0644); err != nil {
		return fmt.Errorf("writing example schema: %w", err)
	}
	fmt.Println("Created schemas/contacts.sdl")

	fmt.Println("\nRun 'restgen generate' to generate routes.")
	return nil
}
