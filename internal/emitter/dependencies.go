package emitter

import (
	"bytes"
	"text/template"
)

// DependenciesEmitter generates the dependencies file (only once, never overwritten).
type DependenciesEmitter struct {
	pkg string
}

// NewDependenciesEmitter creates a new dependencies emitter.
func NewDependenciesEmitter(pkg string) *DependenciesEmitter {
	return &DependenciesEmitter{pkg: pkg}
}

// Emit generates the dependencies file content.
func (e *DependenciesEmitter) Emit() (string, error) {
	data := &depsTemplateData{
		Package: e.pkg,
	}

	tmpl, err := template.New("dependencies").Parse(depsTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

type depsTemplateData struct {
	Package string
}

var depsTemplate = `package {{.Package}}

// This file is NOT regenerated. Add your With* param functions and helpers here.
//
// Handler structs are in their respective *_routes.go files - add your
// dependency fields there (they will be preserved during regeneration).
//
// Example struct fields (add to the relevant *_routes.go):
//
//   type ContactsHandler struct {
//       db     *sql.DB
//       logger *slog.Logger
//   }
//
// Example param functions:
//
// func WithDB(db *sql.DB) ContactsParam {
//     return func(h *ContactsHandler) {
//         h.db = db
//     }
// }
//
// func WithLogger(logger *slog.Logger) ContactsParam {
//     return func(h *ContactsHandler) {
//         h.logger = logger
//     }
// }
`
