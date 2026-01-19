# CLAUDE.md - A2A Sidecar

A lightweight proxy that wraps non-A2A-compliant agents, translating between the A2A protocol and the agent's native HTTP API.

## Commands

```bash
task build   # Build binary to ./bin/sidecar
task run     # Build and run
task lint    # Run golangci-lint
task fmt     # Format with gofumpt
```

## Code Style

This project follows Kubernetes-style Go conventions.

### Functional Options

Use functional options for configurable constructors instead of config structs:

```go
// Good: Flexible, extensible, readable at call site
srv := server.New(handler,
    server.WithPort(8080),
    server.WithLogger(logger),
)

// Avoid: Config struct with many optional fields
srv := server.New(handler, server.Config{Port: 8080, Logger: logger})
```

Components:

- `Options` struct holds all configurable values
- `DefaultOptions()` returns sensible defaults
- `Option` type is `func(*Options)`
- `WithXxx()` functions return `Option`

### Struct Field Comments

All struct fields must have comments:

```go
// Config holds application configuration.
type Config struct {
    // Port is the HTTP server port.
    Port int
    // LogLevel is the minimum log level for logging.
    LogLevel slog.Level
}
```

### Other Patterns

- **Context First**: `func Get(ctx context.Context, id string)`
- **Error Sentinels**: `var ErrNotFound = errors.New("...")`
- **Interface First**: Define interfaces where they're used, not where implemented

### Naming

| Element    | Convention                     | Example            |
| ---------- | ------------------------------ | ------------------ |
| Packages   | lowercase, single word         | `config`, `server` |
| Interfaces | verb-er suffix                 | `HealthChecker`    |
| Options    | `Option` type, `WithXxx` funcs | `WithPort(8080)`   |
| Errors     | `ErrXxx` variables             | `ErrNotFound`      |
