# Release Process

This document describes how to create releases for `go-acme-store` using GoReleaser.

## Prerequisites

- [mise](https://mise.jdx.dev/) installed (for tool management)
- Git repository with proper tags
- GitHub access (for creating releases)

## Quick Start

```bash
# Install tools
mise install

# Check configuration
mise exec -- make release-check

# Build locally for testing
mise exec -- make release-build
```

## Version Information

The build system automatically injects version information into the binaries:

- **Version**: From `git describe --tags --always --dirty`
- **Commit SHA**: From `git rev-parse --short HEAD`
- **Build Date**: Current timestamp

View current version:
```bash
make version
```

## Release Types

### 1. Local Build (Development)

Build for your current platform only:

```bash
mise exec -- make release-build
```

Artifacts will be in `dist/` directory.

### 2. Snapshot Release (Testing)

Create a full release without a git tag (for testing):

```bash
mise exec -- make release-snapshot
```

This builds for all platforms and creates archives, but doesn't publish to GitHub.

### 3. Official Release

Create an official release with a git tag:

```bash
# Create and push a tag
git tag v1.0.0
git push origin v1.0.0

# Create the release
mise exec -- make release
```

This will:
- Build binaries for all platforms (Linux, macOS, Windows)
- Build for all architectures (amd64, arm64, arm/v7)
- Create tar.gz and zip archives
- Generate checksums
- Build multi-arch Docker images
- Push Docker images to ghcr.io
- Create a GitHub release with changelog
- Upload all artifacts

## Supported Platforms

### Binary Releases

#### acme-store (daemon)
- Linux: amd64, arm64, arm/v7
- macOS: amd64, arm64
- Windows: amd64

#### acme-store-fetcher (utility)
- Linux: amd64, arm64, arm/v7
- macOS: amd64, arm64
- Windows: amd64

### Docker Images

Multi-arch Docker images are automatically built and published:
- `ghcr.io/gbolo/acme-store:latest`
- `ghcr.io/gbolo/acme-store:v1.0.0`
- `ghcr.io/gbolo/acme-store:1.0`

Supported architectures:
- linux/amd64
- linux/arm64
- linux/arm/v7

The images include:
- Pre-built `acme-store` binary
- Web UI files
- Example configuration file

## Changelog

GoReleaser automatically generates a changelog from git commits. For best results, use conventional commits:

- `feat:` - New features (🚀 Features section)
- `fix:` - Bug fixes (🐛 Bug Fixes section)
- `docs:` - Documentation (📝 Documentation section)
- `perf:` - Performance improvements (⚡ Performance section)

Example:
```bash
git commit -m "feat: add ability to delete unmanaged certificates"
git commit -m "fix: hard delete now works for unmanaged certs"
```

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make version` | Show current version information |
| `make release-check` | Validate GoReleaser configuration |
| `make release-build` | Build locally for current platform |
| `make release-snapshot` | Create snapshot release (no tag) |
| `make release` | Create official release (requires tag) |

## Build Artifacts

After a successful release, you'll find:

```
dist/
├── acme-store_1.0.0_Linux_x86_64.tar.gz
├── acme-store_1.0.0_Darwin_x86_64.tar.gz
├── acme-store_1.0.0_Windows_x86_64.zip
├── checksums.txt
└── ...
```

Each archive contains:
- `acme-store` binary
- `acme-store-fetcher` binary
- `web/` directory (UI files)
- Documentation files

## Version Injection

The following variables are injected at build time:

```go
// pkg/meta/version.go
var (
    Name      = "acme-store"           // Binary name
    Version   = "v1.0.0"               // Git tag
    CommitSHA = "8200bb9"              // Short commit hash
    BuildDate = "2025-12-13T11:40:00Z" // Build timestamp
)
```

## Troubleshooting

### No tags found

If you see "git doesn't contain any tags", create an initial tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

### Dirty working directory

If version shows `-dirty`, commit your changes:

```bash
git add .
git commit -m "chore: prepare release"
```

### GoReleaser not found

Install tools via mise:

```bash
mise install
```

## CI/CD Integration

To automate releases in CI/CD:

```yaml
# Example GitHub Actions workflow
name: Release
on:
  push:
    tags:
      - 'v*'

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: jdx/mise-action@v2
      - name: Run GoReleaser
        run: mise exec -- goreleaser release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Docker Images

### Using Published Images

Pull and run the latest image:

```bash
docker pull ghcr.io/gbolo/acme-store:latest
docker run -p 8080:8080 ghcr.io/gbolo/acme-store:latest
```

Or use a specific version:

```bash
docker pull ghcr.io/gbolo/acme-store:v1.0.0
```

### Testing Docker Builds Locally

Test the Docker build in snapshot mode:

```bash
# This builds separate images for each platform
mise exec -- goreleaser release --snapshot --clean

# Test the amd64 image
docker run -p 8080:8080 ghcr.io/gbolo/acme-store:0.0.1-next-amd64
```

### How Docker Builds Work

GoReleaser's `dockers_v2` feature:

1. **Builds binaries first** - All platforms and architectures
2. **Creates build context** - Organizes binaries by platform:
   ```
   temp-context/
   ├── linux/amd64/acme-store
   ├── linux/arm64/acme-store
   ├── linux/arm/v7/acme-store
   └── web/
   ```
3. **Builds with buildx** - Creates multi-arch manifests
4. **Pushes to registry** - Single command pulls correct image for your platform

### Setting up buildx (if needed)

On Linux, you may need to set up a multi-platform builder:

```bash
docker buildx create --name=goreleaser --use
docker run --privileged --rm tonistiigi/binfmt --install all
```

## Manual Build (without GoReleaser)

If you prefer to build manually:

```bash
# Build with version info
make build

# Or build a specific binary
make build-daemon
make build-fetcher
```

The Makefile automatically injects the same version information.

