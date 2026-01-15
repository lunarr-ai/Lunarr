# CLAUDE.md - Lunarr

Lunarr is an agent mesh that connects team knowledge, routes questions to the right people, and surfaces context before you need it.

## Project Structure

Lunarr is a monorepo:

```
lunarr/
├── desktop/ # Tauri desktop app
└── api/     # FastAPI backend
```

## Commands

Use [Task](https://taskfile.dev) to run commands. Run `task` to see all available tasks.

```bash
# Install all dependencies
task install

# Development servers (run separately)
task desktop:dev  # Tauri desktop app
task api:dev      # FastAPI backend

# Build
task build        # Build all components
task desktop:build

# Code quality
task lint         # Lint all
task format       # Format all

# Testing
task test         # Run all tests
task api:test     # Run API tests only
```
