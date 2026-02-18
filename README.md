# ytcli

A lightweight CLI wrapper around `yt-dlp` for downloading YouTube media as:
- full MP4 (video + audio)
- video-only MP4
- audio-only MP3
- optional clipped ranges (`--start` / `--end`)
- optional Apple Music import for audio downloads on macOS (`--apple-music`)

## Features

- Fast single-command downloads with sensible defaults
- Clean audio filename parsing (`Artist - Title.mp3`) when possible
- Clip extraction from timestamps
- Custom output path or output directory
- Apple Music library import after download (macOS only)
- Automated CI + tag-based release binaries via GitHub Actions

## Requirements

- `yt-dlp` installed and available in `PATH`
- `ffmpeg` installed (required by `yt-dlp` for conversion/recode/extraction)
- macOS + Music.app only if using `--apple-music`

Install helper tools:

```bash
# macOS (Homebrew)
brew install yt-dlp ffmpeg

# Ubuntu/Debian
sudo apt update
sudo apt install -y yt-dlp ffmpeg
```

## Install

### Option 1: Download a prebuilt binary (recommended for non-developers)

1. Open the latest release: `https://github.com/CoastalFuturist/ytcli/releases/latest`
2. Download the archive for your OS/CPU.
3. Extract and move `ytcli` (`ytcli.exe` on Windows) into your `PATH`.

### Option 2: Install with Go

```bash
go install github.com/CoastalFuturist/ytcli@latest
```

If you publish this under a different GitHub repo path, replace the module path accordingly.

### Option 3: Build from source

```bash
git clone https://github.com/CoastalFuturist/ytcli.git
cd ytcli
make build
```

or:

```bash
go build -trimpath -o ytcli .
```

## Usage

```bash
ytcli [--start MM:SS|HH:MM:SS] [--end MM:SS|HH:MM:SS] [--mode audio|video|full] [--output PATH] [--apple-music] [--version] <url>
```

## Flags

| Flag | Description | Default |
|---|---|---|
| `--mode` | Download mode: `audio`, `video`, `full` | `full` |
| `--start` | Clip start time (`MM:SS` or `HH:MM:SS`) | none |
| `--end` | Clip end time (`MM:SS` or `HH:MM:SS`) | none |
| `--output` | Output file path or directory | yt-dlp default |
| `--apple-music` | Import downloaded audio track into Apple Music (macOS, audio mode only) | `false` |
| `--version` | Print version and exit | `false` |

## Examples

```bash
# Full video (best quality) as MP4
ytcli "https://youtu.be/u9oxz7AQg5c"

# Audio-only MP3 to a folder
ytcli --mode audio --output "$HOME/Downloads" "https://youtu.be/u9oxz7AQg5c"

# Audio-only MP3 + import into Apple Music (macOS)
ytcli --mode audio --apple-music --output "$HOME/Downloads" "https://youtu.be/u9oxz7AQg5c"

# Download a 30-second clip
ytcli --start 00:30 --end 01:00 --mode full "https://youtu.be/u9oxz7AQg5c"

# Video-only MP4
ytcli --mode video "https://youtu.be/u9oxz7AQg5c"
```

## Apple Music Notes

- `--apple-music` requires `--mode audio`.
- On first use, macOS may prompt for Automation permissions (Terminal/iTerm -> Music).
- The file is downloaded first, then imported to your library.

## Development

```bash
make fmt
make test
make build
```

### Versioned builds

```bash
make build VERSION=v1.0.0
```

## GitHub Release Flow

This repo includes:
- `.github/workflows/ci.yml`: runs tests + build on pushes/PRs
- `.github/workflows/release.yml`: builds cross-platform binaries and attaches them to GitHub Releases on `v*` tags

To publish a release:

```bash
git tag v1.0.0
git push origin v1.0.0
```

The release workflow will generate downloadable archives for Linux, macOS, and Windows.

## License

MIT - see `LICENSE`.
