package schema

import "strings"

// Schema represents the intermediate representation of a parsed SDL file.
type Schema struct {
	FileName string    // source file name (e.g., "contacts.sdl")
	Base     string    // base path (e.g., "/v1/contacts")
	Models   string    // models package (e.g., "github.com/borderlesshq/api/models")
	Includes []Include // included SDL files
	Calls    []Call
	Types    []TypeDef
	Inputs   []InputDef
	Enums    []EnumDef
}

// Include represents an imported SDL file.
type Include struct {
	Path      string // relative path to SDL file
	Namespace string // derived namespace (filename without .sdl)
	Models    string // the @models package from included SDL
}

// Call represents a single API endpoint definition.
type Call struct {
	Name           string // function name (e.g., "createContact")
	Method         string // HTTP method (e.g., "POST", "GET")
	Path           string // route path (e.g., "/", "/{id}")
	Args           []Arg
	ReturnType     string // return type (e.g., "Contact", "external.Location")
	ReturnRequired bool   // true if return type is non-nullable (has !)
	ReturnIsList   bool   // true if return type is a list [Type]
}

// Arg represents a function argument.
type Arg struct {
	Name     string // argument name
	Type     string // type name (e.g., "String", "ID", "CreateContactInput", "external.Location")
	Required bool   // true if non-nullable (has !)
	IsList   bool   // true if array type [Type]
}

// TypeDef represents a type definition (output types).
type TypeDef struct {
	Name   string
	Fields []Field
}

// InputDef represents an input definition (input types for mutations).
type InputDef struct {
	Name   string
	Fields []Field
}

// EnumDef represents an enum definition.
type EnumDef struct {
	Name   string
	Values []string
}

// Field represents a field in a type or input.
type Field struct {
	Name     string
	Type     string // can be "TypeName" or "namespace.TypeName"
	Required bool
	IsList   bool
}

// HandlerName returns the exported Go function name for this call.
func (c *Call) HandlerName() string {
	if len(c.Name) == 0 {
		return ""
	}
	// Capitalize first letter
	return string(c.Name[0]-32) + c.Name[1:]
}

// PathParams extracts path parameter names from the path.
// e.g., "/{id}/items/{itemId}" returns ["id", "itemId"]
func (c *Call) PathParams() []string {
	var params []string
	inParam := false
	current := ""
	for _, ch := range c.Path {
		if ch == '{' {
			inParam = true
			current = ""
		} else if ch == '}' {
			inParam = false
			if current != "" {
				params = append(params, current)
			}
		} else if inParam {
			current += string(ch)
		}
	}
	return params
}

// PathParamSet returns a set of path parameter names for quick lookup.
func (c *Call) PathParamSet() map[string]bool {
	set := make(map[string]bool)
	for _, p := range c.PathParams() {
		set[p] = true
	}
	return set
}

// IsBodyMethod returns true if this method typically has a request body.
func (c *Call) IsBodyMethod() bool {
	switch c.Method {
	case "POST", "PUT", "PATCH":
		return true
	default:
		return false
	}
}

// BodyArg returns the single argument that should be decoded from request body.
// For POST/PUT/PATCH: the one arg not in path params.
// For GET/DELETE: returns nil (no body).
func (c *Call) BodyArg() *Arg {
	if !c.IsBodyMethod() {
		return nil
	}

	pathParams := c.PathParamSet()
	var bodyArg *Arg

	for i := range c.Args {
		if !pathParams[c.Args[i].Name] {
			bodyArg = &c.Args[i]
			break // Should only be one
		}
	}

	return bodyArg
}

// QueryArgs returns arguments that should be parsed from query string.
// For GET/DELETE: all args not in path.
// For POST/PUT/PATCH: returns nil (body methods don't use query params for data).
func (c *Call) QueryArgs() []Arg {
	if c.IsBodyMethod() {
		return nil
	}

	pathParams := c.PathParamSet()
	var args []Arg

	for _, arg := range c.Args {
		if !pathParams[arg.Name] {
			args = append(args, arg)
		}
	}

	return args
}

// PathArgNames returns the names of args that are path parameters.
func (c *Call) PathArgNames() []string {
	pathParams := c.PathParamSet()
	var names []string

	for _, arg := range c.Args {
		if pathParams[arg.Name] {
			names = append(names, arg.Name)
		}
	}

	return names
}

// Validate checks the call for semantic errors.
func (c *Call) Validate() error {
	pathParams := c.PathParamSet()

	if c.IsBodyMethod() {
		// Count non-path args - should be exactly 0 or 1
		var nonPathArgs []string
		for _, arg := range c.Args {
			if !pathParams[arg.Name] {
				nonPathArgs = append(nonPathArgs, arg.Name)
			}
		}
		if len(nonPathArgs) > 1 {
			return &ValidationError{
				Call:    c.Name,
				Message: "body methods (POST/PUT/PATCH) can have at most one non-path argument as body, found: " + strings.Join(nonPathArgs, ", "),
			}
		}
	}

	// Check that all path params in URL have matching args
	for param := range pathParams {
		found := false
		for _, arg := range c.Args {
			if arg.Name == param {
				found = true
				break
			}
		}
		if !found {
			return &ValidationError{
				Call:    c.Name,
				Message: "path parameter {" + param + "} has no matching argument",
			}
		}
	}

	return nil
}

// ValidationError represents a schema validation error.
type ValidationError struct {
	Call    string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Call + ": " + e.Message
}

// ParseTypeRef parses a type reference like "Location" or "geo.Location".
// Returns (namespace, typeName). If no namespace, returns ("", typeName).
func ParseTypeRef(typeRef string) (namespace, typeName string) {
	if idx := strings.Index(typeRef, "."); idx != -1 {
		return typeRef[:idx], typeRef[idx+1:]
	}
	return "", typeRef
}

// IsNamespaced returns true if the type reference includes a namespace.
func IsNamespaced(typeRef string) bool {
	return strings.Contains(typeRef, ".")
}
