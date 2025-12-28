# RestGen SDL IntelliJ Plugin

A JetBrains IDE plugin that provides language support for RestGen SDL (Schema Definition Language) files.

## Features

- **Syntax Highlighting** for `.sdl` files
  - Keywords: `type`, `input`, `enum`, `Calls`
  - Scalar types: `String`, `Int`, `Float`, `Boolean`, `ID`, `Time`
  - HTTP directives: `@get`, `@post`, `@put`, `@patch`, `@delete`
  - Config directives: `@base`, `@models`, `@include`
  - Strings, numbers, identifiers
  - Comments (`#` and `//`)

- **Brace Matching** for `{}`, `[]`, `()`

- **Code Commenting** with `#`

- **Color Settings Page** to customize highlighting colors

## Installation

### From JetBrains Marketplace

1. Open your JetBrains IDE (IntelliJ IDEA, GoLand, WebStorm, etc.)
2. Go to `Settings/Preferences` → `Plugins` → `Marketplace`
3. Search for "RestGen SDL"
4. Click `Install`

### From Disk

1. Download the latest release `.zip` file
2. Go to `Settings/Preferences` → `Plugins` → `⚙️` → `Install Plugin from Disk...`
3. Select the downloaded `.zip` file

## Building from Source

### Prerequisites

- JDK 17+
- Gradle 8.0+

### Build

```bash
./gradlew build
```

### Run in Development IDE

```bash
./gradlew runIde
```

### Package Plugin

```bash
./gradlew buildPlugin
```

The plugin zip will be in `build/distributions/`.

## SDL Syntax Example

```graphql
# @base("/v1/contacts")
# @models("github.com/yourorg/app/models")
# @include("common.sdl")

enum ContactStatus {
    active
    inactive
    pending
}

type Contact {
    id: ID!
    name: String!
    email: String
    status: ContactStatus!
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

type Calls {
    // Get a contact by ID
    getContact(id: ID!): Contact @get("/{id}")
    
    // Create a new contact
    createContact(input: CreateContactInput!): Contact! @post("/")
    
    // Update an existing contact
    updateContact(id: ID!, input: UpdateContactInput!): Contact! @put("/{id}")
    
    // Delete a contact
    deleteContact(id: ID!): DeleteResult! @delete("/{id}")
    
    // List all contacts with filtering
    listContacts(filter: ContactFilter): [Contact!]! @get("/")
}
```

## Compatibility

- IntelliJ IDEA 2023.3+
- GoLand 2023.3+
- WebStorm 2023.3+
- Other JetBrains IDEs 2023.3+

## License

MIT License

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## Related

- [RestGen](https://github.com/borderlesshq/restgen) - The schema-first REST API code generator for Go
