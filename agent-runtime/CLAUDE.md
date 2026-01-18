# API

## Commands

Always use uv instead of python or pip.

```bash
# Formatting
uv run ruff format
# Linting
uv run ruff check --fix
# Testing
uv run pytest
```

## Code Quality

- After making changes, run formatting and linting commands.

## Python Style Guide

- Do not use f-strings for logging. Use %s instead.
- Add docstrings for public methods with Args, Returns, and Raises sections (if applicable).
- Do not add docstrings for private methods.

## Testing

- Each case should test a single functionality.
- Keep tests minimal instead of many granular tests.
