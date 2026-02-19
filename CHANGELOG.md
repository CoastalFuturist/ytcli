# Changelog

All notable changes to this project are documented in this file.

## [Unreleased]

### Added
- Audio metadata override flags: `--artist` and `--song` (`--mode audio` only).
- Tests covering manual metadata overrides, output template fallback, and flag validation.

### Changed
- Audio mode now keeps yt-dlp's artist/title output template as the default fallback when parsed metadata is incomplete.
- Downloaded audio files are tagged after download with resolved metadata, including filename-based inference when needed.
- README usage/docs updated with the new flags and an explicit "flags first, URL last" note.

## [0.1.0] - 2026-02-18

### Added
- Initial public release with `audio`, `video`, and `full` download modes.
- Clip support via `--start` and `--end`.
- Optional Apple Music import on macOS via `--apple-music` in audio mode.
- CI and GoReleaser setup for cross-platform binaries.
