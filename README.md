# ytcli

`ytcli` is a lightweight CLI wrapper around `yt-dlp` for downloading YouTube media.

## Features

- Download full video (`mp4` with audio)
- Download video-only (`mp4`)
- Download audio-only (`mp3`)
- Clip by time range (`--start` / `--end`)
- Optional Apple Music import after audio download on macOS (`--apple-music`)
- Version output via `--version` or `ytcli version`

## Requirements

- `yt-dlp` in `PATH`
- `ffmpeg` in `PATH`
- macOS + Music.app only for `--apple-music`

Example installs:

```bash
# macOS
brew install yt-dlp ffmpeg

# Ubuntu/Debian
sudo apt update
sudo apt install -y yt-dlp ffmpeg
```

## Install

### 1) Download a release binary (recommended)

1. Go to: `https://github.com/CoastalFuturist/ytcli/releases/latest`
2. Download the archive for your OS/architecture.
3. Extract and place `ytcli` (`ytcli.exe` on Windows) somewhere on your `PATH`.

### 2) Install with Go

```bash
go install github.com/CoastalFuturist/ytcli@latest
```

### 3) Build from source

```bash
git clone https://github.com/CoastalFuturist/ytcli.git
cd ytcli
make build
```

## Usage

```bash
ytcli [--start MM:SS|HH:MM:SS] [--end MM:SS|HH:MM:SS] [--mode audio|video|full] [--output PATH] [--apple-music] <url>

# also supported
ytcli --version
ytcli version
```

## Flags

- `--mode`: `audio`, `video`, or `full` (default: `full`)
- `--start`: clip start timestamp (`MM:SS` or `HH:MM:SS`)
- `--end`: clip end timestamp (`MM:SS` or `HH:MM:SS`)
- `--output`: output path (file or directory)
- `--apple-music`: import downloaded audio into Apple Music (macOS, `--mode audio` only)
- `--version`: print build version/commit/date and exit

## Quick Examples

```bash
# Full video (default mode)
ytcli "https://youtu.be/u9oxz7AQg5c"

# Audio-only to Downloads
ytcli --mode audio --output "$HOME/Downloads" "https://youtu.be/u9oxz7AQg5c"

# Audio-only and import to Apple Music (macOS)
ytcli --mode audio --apple-music --output "$HOME/Downloads" "https://youtu.be/u9oxz7AQg5c"

# Clip from 00:30 to 01:00
ytcli --start 00:30 --end 01:00 --mode full "https://youtu.be/u9oxz7AQg5c"

# Version info
ytcli --version
ytcli version
```

## Development

```bash
make fmt
make test
make build
```

## Releases

- CI runs tests/build on push and PR.
- Pushing a semver tag (for example `v0.1.0`) triggers GoReleaser to publish release binaries for:
  - macOS: `amd64`, `arm64`
  - Linux: `amd64`, `arm64`
  - Windows: `amd64`, `arm64`

Create a release tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

## License

MIT
