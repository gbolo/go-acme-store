# Web UI Development

The web UI files are located in `pkg/webui/static/` and are embedded into the Go binary at build time.

## File Structure

```
pkg/webui/
├── embed.go          # Embeds static files using //go:embed directive
└── static/           # Web UI source files (EDIT THESE)
    ├── app.js
    ├── favicon.svg
    ├── index.html
    └── style.css
```

## Development Workflow

1. **Edit files**: Modify files directly in `pkg/webui/static/`
2. **Build**: Run `make build` to compile with embedded files
3. **Deploy**: Single binary includes all web assets

## Why This Location?

Go's `//go:embed` directive can only embed files in the same package or subdirectories. Since we embed in the `pkg/webui` package, the files must be at `pkg/webui/static/`.

