package emitter

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/borderlesshq/restgen/internal/config"
	"github.com/borderlesshq/restgen/internal/schema"
)

// RoutesEmitter generates route handler files.
type RoutesEmitter struct {
	cfg *config.Config
}

// NewRoutesEmitter creates a new routes emitter.
func NewRoutesEmitter(cfg *config.Config) *RoutesEmitter {
	return &RoutesEmitter{cfg: cfg}
}

// Emit generates the routes file content for a schema.
func (e *RoutesEmitter) Emit(s *schema.Schema) (string, error) {
	data := e.buildTemplateData(s)

	tmpl, err := template.New("routes").Funcs(template.FuncMap{
		"lower": strings.ToLower,
		"title": strings.Title,
		"chiMethod": func(method string) string {
			// Convert "POST" -> "Post", "GET" -> "Get", etc.
			return strings.Title(strings.ToLower(method))
		},
	}).Parse(routesTemplate)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}

type templateData struct {
	Package       string
	HandlerName   string
	BasePath      string
	ModelsPackage string
	ModelsAlias   string
	Imports       []importDef
	Calls         []callData
	// IncludeAliases maps namespace -> import alias for included SDLs
	IncludeAliases map[string]string
}

type importDef struct {
	Alias string
	Path  string
}

type callData struct {
	Name           string
	HandlerName    string
	Method         string
	Path           string
	ReturnType     string
	GoReturnType   string // type for ApiResponse generic param (e.g., "models.Contact" or "*models.Contact")
	PathParams     []string
	BodyArg        *argData
	QueryArgs      []argData
	ReturnNullable bool // true if return type is nullable (no !)
}

type argData struct {
	Name      string
	GoName    string
	Type      string
	GoType    string
	IsComplex bool // true if this is a struct type needing schema decoder
}

func (e *RoutesEmitter) buildTemplateData(s *schema.Schema) *templateData {
	// Derive handler name from file name or base path
	handlerName := deriveHandlerName(s)

	// Build imports
	imports := []importDef{
		{Path: "net/http"},
		{Path: "github.com/go-chi/chi/v5"},
		{Path: "github.com/borderlesshq/restgen/shared"},
	}

	modelsAlias := "models"
	if s.Models != "" {
		imports = append(imports, importDef{Alias: modelsAlias, Path: s.Models})
	}

	// Build include aliases and add imports for included SDLs
	includeAliases := make(map[string]string)
	for _, inc := range s.Includes {
		if inc.Models != "" {
			// Use namespace as alias (e.g., geo_models)
			alias := inc.Namespace
			includeAliases[inc.Namespace] = alias
			imports = append(imports, importDef{Alias: alias, Path: inc.Models})
		}
	}

	// Check if we need encoding/json (for body decoding)
	needsJSON := false
	// Check if we need gorilla/schema (for query decoding)
	needsSchema := false

	for _, c := range s.Calls {
		if c.BodyArg() != nil {
			needsJSON = true
		}
		for _, qa := range c.QueryArgs() {
			if e.isComplexType(qa.Type) {
				needsSchema = true
			}
		}
	}

	if needsJSON {
		imports = append([]importDef{{Path: "encoding/json"}}, imports...)
	}
	if needsSchema {
		imports = append(imports, importDef{Path: "github.com/gorilla/schema"})
	}

	// Helper to resolve type to Go type with proper package alias
	resolveGoType := func(typeRef string) string {
		ns, typeName := schema.ParseTypeRef(typeRef)
		if ns != "" {
			// Namespaced type: geo.Location -> geo_models.Location
			if alias, ok := includeAliases[ns]; ok {
				return alias + "." + typeName
			}
			// Fallback if namespace not found (shouldn't happen with validation)
			return ns + "." + typeName
		}
		// Local type
		if _, isScalar := e.cfg.Scalars[typeName]; isScalar {
			return e.cfg.GoType(typeName, true, false)
		}
		return modelsAlias + "." + typeName
	}

	// Build call data
	var calls []callData
	for _, c := range s.Calls {
		// Resolve return type - add pointer if nullable
		goReturnType := resolveGoType(c.ReturnType)
		returnNullable := !c.ReturnRequired
		if returnNullable && !c.ReturnIsList {
			goReturnType = "*" + goReturnType
		}
		if c.ReturnIsList {
			goReturnType = "[]" + resolveGoType(c.ReturnType)
		}

		cd := callData{
			Name:           c.Name,
			HandlerName:    c.HandlerName(),
			Method:         c.Method,
			Path:           c.Path,
			ReturnType:     c.ReturnType,
			GoReturnType:   goReturnType,
			PathParams:     c.PathParams(),
			ReturnNullable: returnNullable,
		}

		if body := c.BodyArg(); body != nil {
			cd.BodyArg = &argData{
				Name:   body.Name,
				GoName: body.Name,
				Type:   body.Type,
				GoType: resolveGoType(body.Type),
			}
		}

		for _, qa := range c.QueryArgs() {
			isComplex := e.isComplexType(qa.Type)
			goType := e.cfg.GoType(qa.Type, qa.Required, qa.IsList)
			if isComplex {
				goType = resolveGoType(qa.Type)
			}

			cd.QueryArgs = append(cd.QueryArgs, argData{
				Name:      qa.Name,
				GoName:    qa.Name,
				Type:      qa.Type,
				GoType:    goType,
				IsComplex: isComplex,
			})
		}

		calls = append(calls, cd)
	}

	return &templateData{
		Package:        e.cfg.Package,
		HandlerName:    handlerName,
		BasePath:       s.Base,
		ModelsPackage:  s.Models,
		ModelsAlias:    modelsAlias,
		Imports:        imports,
		Calls:          calls,
		IncludeAliases: includeAliases,
	}
}

// isComplexType returns true if the type is a struct (not a scalar).
func (e *RoutesEmitter) isComplexType(typeName string) bool {
	_, isScalar := e.cfg.Scalars[typeName]
	return !isScalar
}

func deriveHandlerName(s *schema.Schema) string {
	// From file name: contacts.sdl -> Contacts, business_locations.sdl -> BusinessLocations
	if s.FileName != "" {
		name := strings.TrimSuffix(s.FileName, ".sdl")
		if len(name) > 0 {
			return toPascalCase(name)
		}
	}

	// From base path: /v1/contacts -> Contacts
	if s.Base != "" {
		parts := strings.Split(strings.Trim(s.Base, "/"), "/")
		if len(parts) > 0 {
			name := parts[len(parts)-1]
			return toPascalCase(name)
		}
	}

	return "Handler"
}

// toPascalCase converts snake_case or kebab-case to PascalCase.
// e.g., "business_locations" -> "BusinessLocations", "my-handler" -> "MyHandler"
func toPascalCase(s string) string {
	// Replace hyphens with underscores for uniform handling
	s = strings.ReplaceAll(s, "-", "_")

	parts := strings.Split(s, "_")
	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			result.WriteString(strings.ToUpper(part[:1]))
			if len(part) > 1 {
				result.WriteString(part[1:])
			}
		}
	}
	return result.String()
}

var routesTemplate = `// Code generated by restgen. DO NOT EDIT ABOVE THE MARKER.

package {{.Package}}

import (
{{- range .Imports}}
	{{if .Alias}}{{.Alias}} {{end}}"{{.Path}}"
{{- end}}
)

// ============================================================================
// HANDLER
// ============================================================================

type {{.HandlerName}}Handler struct {
	// add dependencies here
}

type {{.HandlerName}}Param func(*{{.HandlerName}}Handler)

func New{{.HandlerName}}Handler(params ...{{.HandlerName}}Param) *{{.HandlerName}}Handler {
	h := &{{.HandlerName}}Handler{}
	for _, param := range params {
		param(h)
	}
	shared.AssertDependencies(*h, "New{{.HandlerName}}Handler")
	return h
}

// ============================================================================
// ROUTES
// ============================================================================

func (h *{{.HandlerName}}Handler) BasePath() string {
	return "{{.BasePath}}"
}

func (h *{{.HandlerName}}Handler) Routes() chi.Router {
	r := chi.NewRouter()
	h.applyMiddleware(r)

{{- range .Calls}}
	r.{{.Method | chiMethod}}("{{.Path}}", h.{{.HandlerName}})
{{- end}}

	return r
}

// ============================================================================
// MIDDLEWARE (add your middleware here)
// ============================================================================

func (h *{{.HandlerName}}Handler) applyMiddleware(r chi.Router) {
	// Example:
	// r.Use(middleware.RequestID)
	// r.Use(middleware.Logger)
	//
	// Per-route middleware can be applied in Routes() using r.With(...)
}

// RouteMiddleware returns middleware for specific routes.
// This is for documentation/introspection; apply via r.With() in Routes().
func (h *{{.HandlerName}}Handler) RouteMiddleware() map[string][]func(http.Handler) http.Handler {
	return map[string][]func(http.Handler) http.Handler{
		// "POST /": {rateLimiter},
		// "GET /{id}": {cacheMiddleware},
	}
}

// --- RESTGEN MARKER (do not edit above) ---

// ============================================================================
// HANDLER IMPLEMENTATIONS
// ============================================================================
{{range .Calls}}

func (h *{{$.HandlerName}}Handler) {{.HandlerName}}(w http.ResponseWriter, r *http.Request) {
{{- if .PathParams}}
	// Path parameters:
{{- range .PathParams}}
	 // {{.}} := chi.URLParam(r, "{{.}}")
{{- end}}
{{- end}}
{{- if .BodyArg}}
	 var {{.BodyArg.GoName}} {{.BodyArg.GoType}}
	 if err := json.NewDecoder(r.Body).Decode(&{{.BodyArg.GoName}}); err != nil {
	     shared.WriteResponse(w, http.StatusBadRequest, &shared.ApiResponse[{{.GoReturnType}}]{
	         Message: err.Error(),
	     })
	     return
    }
{{- end}}
{{- if .QueryArgs}}
	// Query parameters:
{{- $returnType := .GoReturnType}}
{{- range .QueryArgs}}
{{- if .IsComplex}}
	 var {{.GoName}} {{.GoType}}
	 decoder := schema.NewDecoder()
	 decoder.IgnoreUnknownKeys(true)
	 if err := decoder.Decode(&{{.GoName}}, r.URL.Query()); err != nil {
	     shared.WriteResponse(w, http.StatusBadRequest, &shared.ApiResponse[{{$returnType}}]{
	         Message: err.Error(),
	     })
	     return
	 }
{{- else}}
	// {{.GoName}} := r.URL.Query().Get("{{.Name}}")
{{- end}}
{{- end}}
{{- end}}

	// TODO: implement {{.HandlerName}}
	shared.WriteResponse(w, http.StatusNotImplemented, &shared.ApiResponse[{{.GoReturnType}}]{
		Message: "{{.HandlerName}} not implemented",
	})
}
{{- end}}

// --- REMOVED HANDLERS ---
`
