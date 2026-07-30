#!/usr/bin/env sh

set -e

usage() {
    echo "Usage: install.sh [tag] [--root] [-h|--help]"
    echo "  tag: The version to install (e.g. v1.0.0). Defaults to latest."
    echo "  --root: Allow installation as root (not recommended)."
    echo "  -h, --help: Show this help message."
    exit 0
}

# Parse arguments
for arg in "$@"; do
    shift
    case "$arg" in
        "--root") override_root=1 ;;
        "-h"|"--help") usage ;;
        *)
        if echo "$arg" | grep -qv "^-"; then
            tag="$arg"
        else
            echo "Invalid option $arg" >&2
            exit 1
        fi
    esac
done

is_root() {
    [ "$(id -u)" -eq 0 ]
}

if is_root && [ "${override_root:-0}" -eq 0 ]; then
    echo "The script was run as root or with sudo. This is not recommended."
    echo "If you want to install as root, pass the '--root' parameter."
    exit 1
fi

log() {
    echo "$1"
}
OS=$(uname -s)
ARCH=$(uname -m)

case "$OS" in
    Darwin)
        if [ "$ARCH" = "x86_64" ]; then
            target="Darwin_x86_64"
        elif [ "$ARCH" = "arm64" ]; then
            target="Darwin_arm64"
        else
            log "Unsupported architecture: $ARCH on Darwin"
            exit 1
        fi
        ext="tar.gz"
        ;;
    Linux)
        if [ "$ARCH" = "x86_64" ]; then
            target="Linux_x86_64"
        else
            log "Unsupported architecture: $ARCH on Linux. Currently only x86_64 is supported."
            exit 1
        fi
        ext="tar.gz"
        ;;
    MINGW*|MSYS*|CYGWIN*)
        target="Windows_x86_64"
        ext="zip"
        ;;
    *)
        log "Unsupported platform: $OS $ARCH"
        exit 1
        ;;
esac

command -v curl >/dev/null || { log "curl is required for installation." >&2; exit 1; }
if [ "$ext" = "tar.gz" ]; then
    command -v tar >/dev/null || { log "tar is required for installation." >&2; exit 1; }
else
    command -v unzip >/dev/null || { log "unzip is required for installation." >&2; exit 1; }
fi

repo="kumneger0/ytmusic-tui"
releases_uri="https://github.com/$repo/releases"

if [ -z "$tag" ]; then
    log "Fetching latest version info..."
    tag=$(curl -LsH 'Accept: application/json' "$releases_uri/latest" | sed 's/.*"tag_name":"\([^"]*\)".*/\1/')
fi

version=${tag#v}

log "Installing ytmusic-tui v$version for $target..."

download_uri="$releases_uri/download/v$version/ytmusic-tui_${target}.${ext}"

ytmusic_tui_dir="$HOME/.ytmusic-tui"
bin_dir="$ytmusic_tui_dir/bin"
exe="$bin_dir/ytmusic-tui"
archive="$ytmusic_tui_dir/ytmusic-tui.$ext"
mkdir -p "$bin_dir"

curl --fail --location --progress-bar --output "$archive" "$download_uri"
if [ "$ext" = "tar.gz" ]; then
    tar -xzf "$archive" -C "$bin_dir"
else
    unzip -o "$archive" -d "$bin_dir"
fi

chmod +x "$exe"

rm "$archive"

notfound() {
    cat << EOINFO

Manually add the directory to your PATH:
export PATH="\$PATH:$bin_dir"
EOINFO
}

endswith_newline() {
    [ "$(tail -c1 "$1" | wc -l)" -gt 0 ]
}

check_shell() {
    shell_rc="$HOME/$1"
    path_export="export PATH=\"\$PATH:$bin_dir\""

    if [ "$1" = ".zshrc" ] && [ -n "${ZDOTDIR}" ]; then
        shell_rc="$ZDOTDIR/$1"
    fi

    if [ ! -f "$shell_rc" ]; then
        log "Creating $shell_rc..."
        touch "$shell_rc"
    fi

    if ! grep -q "$bin_dir" "$shell_rc"; then
        log "Adding $bin_dir to PATH in $shell_rc..."
        if [ -s "$shell_rc" ] && ! endswith_newline "$shell_rc"; then
            echo >> "$shell_rc"
        fi
        echo "$path_export" >> "$shell_rc"
    else
        log "PATH already contains $bin_dir in $shell_rc"
    fi
}

current_shell=$(basename "$SHELL")
case "$current_shell" in
    zsh) check_shell ".zshrc" ;;
    bash)
        [ -f "$HOME/.bashrc" ] && check_shell ".bashrc"
        [ -f "$HOME/.bash_profile" ] && check_shell ".bash_profile"
        ;;
    fish)
        fish_config="$HOME/.config/fish/config.fish"
        mkdir -p "$(dirname "$fish_config")"
        if ! grep -q "$bin_dir" "$fish_config" 2>/dev/null; then
            log "Adding $bin_dir to PATH in $fish_config..."
            echo "fish_add_path $bin_dir" >> "$fish_config"
        fi
        ;;
    *)
        notfound
        ;;
esac

log "Successfully installed ytmusic-tui to $exe"
log "Please restart your terminal or source your shell config to apply changes."
log "Run 'ytmusic-tui --help' to get started!"
