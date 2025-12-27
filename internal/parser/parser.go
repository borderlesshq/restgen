package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/borderlesshq/restgen/internal/schema"
)

// Parser parses SDL files into schema IR.
type Parser struct {
	// baseDir is used to resolve relative @include paths
	baseDir string
	// cache prevents re-parsing the same file
	cache map[string]*schema.Schema
}

// New creates a new parser.
func New() *Parser {
	return &Parser{
		cache: make(map[string]*schema.Schema),
	}
}

// ParseFile parses a single SDL file.
func (p *Parser) ParseFile(path string) (*schema.Schema, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	// Check cache
	if cached, ok := p.cache[absPath]; ok {
		return cached, nil
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	// Set base dir for resolving includes
	p.baseDir = filepath.Dir(absPath)

	s, err := p.Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	s.FileName = filepath.Base(path)

	// Cache result
	p.cache[absPath] = s

	return s, nil
}

// Parse parses SDL content into a Schema.
func (p *Parser) Parse(content string) (*schema.Schema, error) {
	s := &schema.Schema{}

	// Parse directives from comments at top
	// # @base("/v1/contacts")
	// # @models("github.com/borderlesshq/api/models")
	// # @include("path/to/other.sdl")
	baseRe := regexp.MustCompile(`#\s*@base\s*\(\s*"([^"]+)"\s*\)`)
	modelsRe := regexp.MustCompile(`#\s*@models\s*\(\s*"([^"]+)"\s*\)`)
	includeRe := regexp.MustCompile(`#\s*@include\s*\(\s*"([^"]+)"\s*\)`)

	if m := baseRe.FindStringSubmatch(content); len(m) > 1 {
		s.Base = m[1]
	}
	if m := modelsRe.FindStringSubmatch(content); len(m) > 1 {
		s.Models = m[1]
	}

	// Parse all includes
	includeMatches := includeRe.FindAllStringSubmatch(content, -1)
	for _, m := range includeMatches {
		includePath := m[1]

		inc, err := p.parseInclude(includePath)
		if err != nil {
			return nil, fmt.Errorf("parsing include %s: %w", includePath, err)
		}
		s.Includes = append(s.Includes, *inc)
	}

	// Parse type blocks using a proper brace-matching approach
	blocks := extractBlocks(content)

	for _, block := range blocks {
		if block.name == "Calls" {
			calls, err := p.parseCalls(block.body)
			if err != nil {
				return nil, fmt.Errorf("parsing Calls block: %w", err)
			}
			s.Calls = calls
		} else if block.kind == "type" {
			typeDef, err := p.parseTypeDef(block.name, block.body)
			if err != nil {
				return nil, fmt.Errorf("parsing type %s: %w", block.name, err)
			}
			s.Types = append(s.Types, *typeDef)
		} else if block.kind == "input" {
			inputDef, err := p.parseInputDef(block.name, block.body)
			if err != nil {
				return nil, fmt.Errorf("parsing input %s: %w", block.name, err)
			}
			s.Inputs = append(s.Inputs, *inputDef)
		}
	}

	return s, nil
}

// parseInclude parses an included SDL file and extracts its metadata.
func (p *Parser) parseInclude(includePath string) (*schema.Include, error) {
	// Resolve path relative to current SDL file
	fullPath := includePath
	if !filepath.IsAbs(includePath) && p.baseDir != "" {
		fullPath = filepath.Join(p.baseDir, includePath)
	}

	// Parse the included file (will use cache if already parsed)
	includedSchema, err := p.ParseFile(fullPath)
	if err != nil {
		return nil, err
	}

	// Derive namespace from filename
	// e.g., "geo_models.sdl" -> "geo_models"
	namespace := strings.TrimSuffix(filepath.Base(includePath), ".sdl")
	// Replace hyphens with underscores for valid Go identifiers
	namespace = strings.ReplaceAll(namespace, "-", "_")

	return &schema.Include{
		Path:      includePath,
		Namespace: namespace,
		Models:    includedSchema.Models,
	}, nil
}

type block struct {
	kind string
	name string
	body string
}

// extractBlocks extracts type/input blocks handling nested braces.
func extractBlocks(content string) []block {
	var blocks []block

	// Find "type Name {" or "input Name {"
	re := regexp.MustCompile(`(type|input)\s+(\w+)\s*\{`)
	matches := re.FindAllStringSubmatchIndex(content, -1)

	for _, match := range matches {
		kind := content[match[2]:match[3]]
		name := content[match[4]:match[5]]
		braceStart := match[1] - 1 // position of opening {

		// Find matching closing brace
		depth := 1
		bodyStart := braceStart + 1
		bodyEnd := bodyStart

		for i := bodyStart; i < len(content) && depth > 0; i++ {
			if content[i] == '{' {
				depth++
			} else if content[i] == '}' {
				depth--
				if depth == 0 {
					bodyEnd = i
				}
			}
		}

		blocks = append(blocks, block{
			kind: kind,
			name: name,
			body: content[bodyStart:bodyEnd],
		})
	}

	return blocks
}

// parseCalls parses the Calls block content.
func (p *Parser) parseCalls(body string) ([]schema.Call, error) {
	var calls []schema.Call

	// Match: createContact(input: CreateContactInput!): Contact! @post("/")
	// Also handles namespaced types: geo.Location, [geo.Location!]!
	callRe := regexp.MustCompile(`(\w+)\s*\(([^)]*)\)\s*:\s*(\[?[\w.]+!?\]?!?)\s*@(get|post|put|patch|delete)\s*\(\s*"([^"]+)"\s*\)`)

	matches := callRe.FindAllStringSubmatch(body, -1)
	for _, m := range matches {
		name := m[1]
		argsStr := m[2]
		returnTypeRaw := m[3]
		method := strings.ToUpper(m[4])
		path := m[5]

		args, err := p.parseArgs(argsStr)
		if err != nil {
			return nil, fmt.Errorf("parsing args for %s: %w", name, err)
		}

		// Parse return type for nullability and list
		returnRequired := false
		returnIsList := false
		returnType := returnTypeRaw

		// Check for outer required: [Type!]! or Type!
		if strings.HasSuffix(returnType, "!") {
			returnRequired = true
			returnType = strings.TrimSuffix(returnType, "!")
		}

		// Check for list type [Type] or [Type!]
		if strings.HasPrefix(returnType, "[") && strings.HasSuffix(returnType, "]") {
			returnIsList = true
			returnType = returnType[1 : len(returnType)-1] // Remove [ and ]
		}

		// Remove inner ! for list items like [Type!]
		returnType = strings.TrimSuffix(returnType, "!")

		call := schema.Call{
			Name:           name,
			Method:         method,
			Path:           path,
			Args:           args,
			ReturnType:     returnType,
			ReturnRequired: returnRequired,
			ReturnIsList:   returnIsList,
		}

		// Validate the call
		if err := call.Validate(); err != nil {
			return nil, err
		}

		calls = append(calls, call)
	}

	return calls, nil
}

// parseArgs parses function arguments like "id: ID!, input: CreateContactInput"
// Also handles namespaced types: geo.Location, [geo.Location!]!
func (p *Parser) parseArgs(argsStr string) ([]schema.Arg, error) {
	if strings.TrimSpace(argsStr) == "" {
		return nil, nil
	}

	var args []schema.Arg

	// Split by comma, handling nested brackets
	parts := splitArgs(argsStr)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Parse: name: Type! or name: [Type!]! or name: geo.Type!
		colonIdx := strings.Index(part, ":")
		if colonIdx == -1 {
			return nil, fmt.Errorf("invalid arg: %s", part)
		}

		name := strings.TrimSpace(part[:colonIdx])
		typeStr := strings.TrimSpace(part[colonIdx+1:])

		arg := schema.Arg{Name: name}

		// Check for outer required: [Type!]! or Type!
		if strings.HasSuffix(typeStr, "!") {
			arg.Required = true
			typeStr = strings.TrimSuffix(typeStr, "!")
		}

		// Check for list type [Type] or [Type!]
		if strings.HasPrefix(typeStr, "[") && strings.HasSuffix(typeStr, "]") {
			arg.IsList = true
			typeStr = typeStr[1 : len(typeStr)-1] // Remove [ and ]
		}

		// Remove inner ! for list items like [Type!]
		typeStr = strings.TrimSuffix(typeStr, "!")

		arg.Type = typeStr
		args = append(args, arg)
	}

	return args, nil
}

// parseTypeDef parses a type block into a TypeDef.
func (p *Parser) parseTypeDef(name, body string) (*schema.TypeDef, error) {
	fields, err := p.parseFields(body)
	if err != nil {
		return nil, err
	}
	return &schema.TypeDef{Name: name, Fields: fields}, nil
}

// parseInputDef parses an input block into an InputDef.
func (p *Parser) parseInputDef(name, body string) (*schema.InputDef, error) {
	fields, err := p.parseFields(body)
	if err != nil {
		return nil, err
	}
	return &schema.InputDef{Name: name, Fields: fields}, nil
}

// parseFields parses field definitions like "id: ID!" or "items: [Contact!]!"
func (p *Parser) parseFields(body string) ([]schema.Field, error) {
	var fields []schema.Field

	lines := strings.Split(body, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}

		name := strings.TrimSpace(line[:colonIdx])
		typeStr := strings.TrimSpace(line[colonIdx+1:])

		field := schema.Field{Name: name}

		// Check for outer required: [Type!]! or Type!
		if strings.HasSuffix(typeStr, "!") {
			field.Required = true
			typeStr = strings.TrimSuffix(typeStr, "!")
		}

		// Check for list type [Type] or [Type!]
		if strings.HasPrefix(typeStr, "[") && strings.HasSuffix(typeStr, "]") {
			field.IsList = true
			typeStr = typeStr[1 : len(typeStr)-1] // Remove [ and ]
		}

		// Remove inner ! for list items like [Contact!]
		typeStr = strings.TrimSuffix(typeStr, "!")

		field.Type = typeStr
		fields = append(fields, field)
	}

	return fields, nil
}

// splitArgs splits comma-separated args, respecting nested brackets.
func splitArgs(s string) []string {
	var parts []string
	var current strings.Builder
	depth := 0

	for _, ch := range s {
		switch ch {
		case '[':
			depth++
			current.WriteRune(ch)
		case ']':
			depth--
			current.WriteRune(ch)
		case ',':
			if depth == 0 {
				parts = append(parts, current.String())
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}
