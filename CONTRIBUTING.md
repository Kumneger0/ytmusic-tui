# Contributing to ytmusic-tui

Thank you for your interest in contributing to the YTMusic TUI!

## Development Setup

1. **Fork and Clone**:
   - Fork the [ytmusic-tui](https://github.com/kumneger0/ytmusic-tui) repository.
   - Clone your fork locally:
     ```bash
     git clone https://github.com/YOUR_USERNAME/ytmusic-tui.git
     cd ytmusic-tui
     ```

2. **Prerequisites**:
   - Go 1.25 or higher.
   - `yt-dlp` and `ffmpeg` installed in your PATH.
   - **Spotify Developer Credentials**: You must have a Spotify Client ID and Client Secret exported as environment variables (`SPOTIFY_CLIENT_ID` and `SPOTIFY_CLIENT_SECRET`).

3. **Clone and Run**:
   ```bash
   go mod download
   go run main.go
   ```

## Pull Request Guidelines

- Follow standard Go formatting (`go fmt`).
- Provide a clear description of the changes in your PR.
- If you are adding a new feature, please update the documentation in the [ytmusic-tui_docs](https://github.com/Kumneger0/ytmusic-tui_docs) repository if applicable.

## Reporting Issues

Use the GitHub Issues tab to report bugs or suggest features. Please provide environment details (OS, Go version) when reporting bugs.
