# Contributing to ytmusic-tui

Thank you for your interest in contributing to **ytmusic-tui**! This document provides guidelines and setup instructions to help you get started.

---

## Architecture Overview

`ytmusic-tui` consists of two main components:
- **Go TUI Frontend**: Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), Lipgloss, and Cobra.
- **Python Backend Service**: A gRPC / Connect RPC server managing YouTube Music API interactions, authentication, and metadata.

---

## Prerequisites

Before starting, ensure you have the following installed:

- **Go**: `1.26` or higher
- **Python**: `3.13` or higher
- **uv**: Fast Python package manager ([install guide](https://docs.astral.sh/uv/getting-started/installation/))
- **Protobuf Compiler (`protoc`)**: Required if modifying `.proto` files
  - Go plugins: `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest` and `go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest`
- **System Dependencies**: `ffmpeg` (for audio playback support)

---

## Development Setup

1. **Fork & Clone**:
   ```bash
   git clone https://github.com/YOUR_USERNAME/ytmusic-tui.git
   cd ytmusic-tui
   ```

2. **Install Python Environment**:
   ```bash
   uv sync
   ```

3. **Install Git Hooks** *(Enforces commit message formatting)*:
   ```bash
   make hooks
   ```

---

## Common Makefile Commands

| Command | Description |
| --- | --- |
| `make run` | Build and run the TUI locally |
| `make proto` | Generate Protobuf & gRPC code for both Go and Python |
| `make proto-go` | Generate Go protobuf code (`gen/`) |
| `make proto-python` | Generate Python protobuf code (`grpc_server/gen/`) |
| `make hooks` | Set up local git commit hooks |

---

## Development Workflow

### Modifying Protobuf Schemas
If you update `proto/music.proto`:
```bash
make proto
```

### Running Tests
Run the Go unit test suite:
```bash
go test -v ./...
```

---

## Pull Request Guidelines

1. **Conventional Commits**: Commit messages must follow Conventional Commit rules (e.g., `feat: ...`, `fix: ...`, `docs: ...`, `test: ...`).
2. **Code Formatting**: Ensure Go code is formatted (`go fmt ./...`) and tests pass (`go test ./...`).
3. **Keep PRs Focused**: Limit PRs to a single feature or bugfix for fast review.

---

## Reporting Issues

Use [GitHub Issues](https://github.com/kumneger0/ytmusic-tui/issues) for bug reports and feature requests. Include your OS, Go version, and log output (`ytmusic-tui log` or `~/.config/ytmusic-tui/ytmusic-tui.log`) when reporting bugs.
