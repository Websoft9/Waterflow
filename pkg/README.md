# pkg/

This directory contains reusable Go packages for the Waterflow project.

## Organization

- Place shared, reusable code here that can be imported by other parts of the application
- Each subdirectory should represent a distinct package with its own responsibilities
- Follow Go naming conventions (lowercase, no underscores)

## Example Structure

```
pkg/
├── workflow/     # Core workflow execution logic
├── parser/       # YAML parsing utilities
├── container/    # Container runtime interfaces
└── plugin/       # Plugin system components
```

## Guidelines

- Keep packages focused and minimal
- Use `internal/` for private packages that should not be imported externally
- Include comprehensive tests for each package
- Document public APIs with comments