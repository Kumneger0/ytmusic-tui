# Building from Source

## Prerequisites

Before building `ytmusic-tui` from source, ensure you have the following installed:

- **[Go](https://go.dev/dl/)** (v1.26 or higher)
- **[Python](https://www.python.org/downloads/)** (v3.13 or higher) & **[uv](https://docs.astral.sh/uv/)** (recommended Python package manager)

- **System Dependencies**:
  - **[yt-dlp](https://github.com/yt-dlp/yt-dlp)**: Required for extracting and fetching YouTube Music audio stream URLs
  - **[ffmpeg](https://ffmpeg.org/)**: Required for audio stream decoding and playback

## Installation Steps

### 1. Clone the Repository

```bash
git clone https://github.com/kumneger0/ytmusic-tui.git
cd ytmusic-tui
```

### 2. Set Up Python Environment

Initialize the Python virtual environment and install backend dependencies:

```bash
uv sync
```

### 3. Build the Go Application

Build the application binary:

```bash
make build
```

> **Note for Linux Users:**
> Ensure `gcc` is installed on your system to support CGO audio bindings:
> ```bash
> CGO_ENABLED=1 make build
> ```

### 4. Install & Launch

To install `ytmusic-tui` to `/usr/local/bin`:

```bash
make install
```

Or run the built binary directly:

```bash
./ytmusic-tui
```

### 5. Authenticate with YouTube Music

To access your playlists, liked songs, and library, extract session cookies from your browser:

```bash
ytmusic-tui extract-cookies
```

---

## Support & Troubleshooting

If you encounter issues during installation or build:
- Ensure `yt-dlp` and `ffmpeg` are in your system `PATH`.
- Check application logs via `ytmusic-tui log`.
- Open an issue on [GitHub Issues](https://github.com/kumneger0/ytmusic-tui/issues).
