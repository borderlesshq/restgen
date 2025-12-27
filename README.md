# restgen

A schema-first REST API code generator for Go. Define your API in a GraphQL-like SDL and generate idiomatic Chi router handlers, request/response types, and boilerplate.

## Features

- **Schema-first development** — Define endpoints and types in `.sdl` or `.graphql` files
- **Chi router generation** — Produces clean, idiomatic Go handlers
- **Type generation** — Generates request/response structs with proper JSON tags
- **Merge on regeneration** — Preserves your handler implementations when regenerating
- **Include system** — Share types across schemas with namespaced imports
- **Nullable semantics** — Follows GraphQL conventions (`Type` = nullable, `Type!` = required)

## Installation

```bash
go install github.com/borderlesshq/restgen@latest
```

## Quick Start

```bash
# Initialize a new project
restgen init

# Edit schemas/example.sdl, then generate
restgen generate
```

## Schema Definition Language

### Basic Structure

```graphql
# @base("/v1/contacts")
# @models("github.com/yourorg/yourapp/models")

type Calls {
    createContact(input: CreateContactInput!): Contact! @post("/")
    getContact(id: ID!): Contact @get("/{id}")
    updateContact(id: ID!, input: UpdateContactInput!): Contact! @put("/{id}")
    deleteContact(id: ID!): DeleteResult! @delete("/{id}")
    listContacts(filter: ContactFilter): ContactList! @get("/")
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
```

### Directives

| Directive | Description |
|-----------|-------------|
| `@base("/path")` | Base path for all routes in this schema |
| `@models("pkg/path")` | Go package path for generated types |
| `@include("other.sdl")` | Import types from another schema |
| `@get`, `@post`, `@put`, `@patch`, `@delete` | HTTP method + path |

### Type System

**Scalars** (configured in `restgen.yaml`):

| SDL Type | Go Type |
|----------|---------|
| `String` | `string` |
| `Int` | `int` |
| `Float` | `float64` |
| `Boolean` | `bool` |
| `ID` | `string` |
| `Time` | `time.Time` |

**Nullability** (follows GraphQL semantics):

```graphql
name: String!    # required → string
name: String     # nullable → *string

items: [Item!]!  # required list of required items → []Item
items: [Item]    # nullable list of nullable items → *[]*Item
```

### Parameter Routing

Parameters are automatically routed based on HTTP method:

**POST/PUT/PATCH** (body methods):
- `{param}` in path → path parameter
- Single remaining arg → JSON request body
- Multiple non-path args → validation error

**GET/DELETE** (query methods):
- `{param}` in path → path parameter  
- Remaining args → query parameters
- Complex types decoded with `gorilla/schema`

```graphql
# Path: id, Body: input
updateContact(id: ID!, input: UpdateContactInput!): Contact! @put("/{id}")

# Path: iso2, stateCode, Body: location
updateLocation(iso2: String!, stateCode: String!, location: LocationInput!): Location! 
    @put("/locations/{iso2}/states/{stateCode}")

# Path: none, Query: filter (complex type, uses gorilla/schema)
listContacts(filter: ContactFilter): ContactList! @get("/")

# Path: id, Query: format (scalar)
getContact(id: ID!, format: String): Contact @get("/{id}")
```

### Include System

Share types across schemas using protobuf-style imports:

```graphql
# geo.sdl
# @models("github.com/yourorg/yourapp/models/geo")

type Location {
    lat: Float!
    lng: Float!
}

input LocationInput {
    lat: Float!
    lng: Float!
}
```

```graphql
# contacts.sdl
# @base("/v1/contacts")
# @models("github.com/yourorg/yourapp/models")
# @include("geo.sdl")

type Contact {
    id: ID!
    name: String!
    location: geo.Location!    # namespaced reference
    backupLocation: geo.Location  # nullable
}

type Calls {
    updateLocation(id: ID!, loc: geo.LocationInput!): geo.Location! @put("/{id}/location")
}
```

Generated imports:

```go
import (
    models "github.com/yourorg/yourapp/models"
    geo "github.com/yourorg/yourapp/models/geo"
)
```

## Configuration

`restgen.yaml`:

```yaml
# Package name for generated routes
package: routes

# Output directory for routes
output: ./routes

# Scalar type mappings
scalars:
  Time: time.Time
  ID: string
  Decimal: decimal.Decimal

# Schema file patterns
schemas:
  - schemas/*.sdl
```

## Generated Files

| File | Regenerated | Purpose |
|------|-------------|---------|
| `routes/dependencies.go` | No (created once) | Your `With*` param functions, helpers |
| `routes/*_routes.go` | Yes (merged) | Handlers, routes, middleware |
| `models/*_types.go` | Yes | Request/response structs |

### Handler Structure

```go
// Generated handler with functional options pattern
type ContactsHandler struct {
    // add dependencies here (preserved on regen), always remove this whole comment line when you add dependencies to your handler struct.
}

type ContactsParam func(*ContactsHandler)

func NewContactsHandler(params ...ContactsParam) *ContactsHandler {
    h := &ContactsHandler{}
    for _, param := range params {
        param(h)
    }
    shared.AssertDependencies(*h, "NewContactsHandler")
    return h
}

func (h *ContactsHandler) Routes() chi.Router {
    r := chi.NewRouter()
    h.applyMiddleware(r)
    r.Post("/", h.CreateContact)
    r.Get("/{id}", h.GetContact)
    // ...
    return r
}
```

### Adding Dependencies

1. Add fields to the handler struct in `*_routes.go`:

```go
type ContactsHandler struct {
    db     *sql.DB
    logger *slog.Logger
}
```

2. Add param functions in `dependencies.go`:

```go
func WithDB(db *sql.DB) ContactsParam {
    return func(h *ContactsHandler) {
        h.db = db
    }
}

func WithLogger(logger *slog.Logger) ContactsParam {
    return func(h *ContactsHandler) {
        h.logger = logger
    }
}
```

3. Use in your application:

```go
handler := routes.NewContactsHandler(
    routes.WithDB(db),
    routes.WithLogger(logger),
)

r := chi.NewRouter()
r.Mount("/v1/contacts", handler.Routes())
```

### Implementing Handlers

Handler stubs are generated below the marker. Implement them and they'll be preserved:

```go
// --- RESTGEN MARKER (do not edit above) ---

func (h *ContactsHandler) CreateContact(w http.ResponseWriter, r *http.Request) {
    var input models.CreateContactInput
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        shared.WriteResponse(w, http.StatusBadRequest, &shared.ApiResponse[models.Contact]{
            Message: err.Error(),
        })
        return
    }

    contact, err := h.db.CreateContact(r.Context(), input)
    if err != nil {
        shared.WriteResponse(w, http.StatusInternalServerError, &shared.ApiResponse[models.Contact]{
            Message: err.Error(),
        })
        return
    }

    shared.WriteResponse(w, http.StatusCreated, &shared.ApiResponse[models.Contact]{
        Data:    contact,
        Success: true,
    })
}
```

### Response Types

Return types follow nullability rules:

```graphql
getContact(id: ID!): Contact    # nullable return
createContact(input: CreateContactInput!): Contact!  # required return
```

Generated response types:

```go
// Nullable return → pointer generic param
shared.ApiResponse[*models.Contact]

// Required return → value generic param  
shared.ApiResponse[models.Contact]
```

## Shared Package

The `shared` package provides common utilities:

```go
// Generic API response wrapper
type ApiResponse[T any] struct {
    Data    T      `json:"data,omitempty"`
    Message string `json:"message,omitempty"`
    Success bool   `json:"success"`
}

// Write JSON response
func WriteResponse[T any](w http.ResponseWriter, statusCode int, response *ApiResponse[T])

// Validate all exported pointer/interface fields are non-nil
func AssertDependencies(h any, constructor string)
```

## CLI Commands

```bash
# Initialize new project with example config and schema
restgen init

# Generate code from schemas
restgen generate
restgen generate -c custom-config.yaml

# Show version
restgen version
```

## Merge Behavior

When regenerating, restgen preserves:

- ✅ Handler struct fields (in `*_routes.go`)
- ✅ Handler method implementations (below the marker)
- ✅ `applyMiddleware` customizations
- ✅ `RouteMiddleware` customizations
- ✅ Everything in `dependencies.go`

Removed endpoints are moved to a commented "REMOVED HANDLERS" section.

## License

MIT
