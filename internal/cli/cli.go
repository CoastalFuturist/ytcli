package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/CoastalFuturist/ytcli/internal/buildinfo"
)

var (
	reMMSS   = regexp.MustCompile(`^([0-5]?\d):([0-5]?\d)$`)
	reHHMMSS = regexp.MustCompile(`^(\d+):([0-5]?\d):([0-5]?\d)$`)
	reNoise  = regexp.MustCompile(`(?i)\s*[\(\[\{][^)\]}]*(official|lyrics?|audio|video|visualizer|mv|hq|hd|4k)[^)\]}]*[\)\]\}]\s*$`)
	reBy     = regexp.MustCompile(`(?i)^(.+?)\s+by\s+(.+)$`)
)

const finalPathPrefix = "__YTCLI_FINAL_PATH__:"

type config struct {
	URL         string
	Start       string
	End         string
	Mode        string
	Output      string
	Artist      string
	Song        string
	AppleMusic  bool
	ShowVersion bool
}

type trackMetadata struct {
	Artist string
	Title  string
}

func normalizeTimestamp(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}

	if m := reMMSS.FindStringSubmatch(value); m != nil {
		minutes, _ := strconv.Atoi(m[1])
		seconds, _ := strconv.Atoi(m[2])
		return fmt.Sprintf("%02d:%02d:%02d", 0, minutes, seconds), nil
	}

	if m := reHHMMSS.FindStringSubmatch(value); m != nil {
		hours, _ := strconv.Atoi(m[1])
		minutes, _ := strconv.Atoi(m[2])
		seconds, _ := strconv.Atoi(m[3])
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds), nil
	}

	return "", fmt.Errorf("invalid timestamp %q; use MM:SS or HH:MM:SS", value)
}

func timestampToSeconds(ts string) int {
	parts := strings.Split(ts, ":")
	if len(parts) != 3 {
		return 0
	}

	hours, _ := strconv.Atoi(parts[0])
	minutes, _ := strconv.Atoi(parts[1])
	seconds, _ := strconv.Atoi(parts[2])
	return hours*3600 + minutes*60 + seconds
}

func sanitizeFilenamePart(value string) string {
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", " -",
		"*", "",
		"?", "",
		"\"", "",
		"<", "",
		">", "",
		"|", "",
	)
	value = replacer.Replace(strings.TrimSpace(value))
	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		return "unknown"
	}
	return value
}

func cleanTitle(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.Trim(s, `"`)
	s = strings.Trim(s, `'`)

	for {
		updated := reNoise.ReplaceAllString(s, "")
		if updated == s {
			break
		}
		s = strings.TrimSpace(updated)
	}

	return strings.Join(strings.Fields(s), " ")
}

func cleanArtist(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimSuffix(s, " - Topic")
	s = strings.TrimSuffix(s, " VEVO")
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return "Unknown Artist"
	}
	return s
}

func parseTrackMetadata(title, uploader string) trackMetadata {
	cleanedTitle := cleanTitle(title)
	cleanedUploader := cleanArtist(uploader)
	separators := []string{" - ", " – ", " — ", " | ", ": "}

	for _, sep := range separators {
		if strings.Contains(cleanedTitle, sep) {
			parts := strings.SplitN(cleanedTitle, sep, 2)
			artist := cleanArtist(parts[0])
			track := cleanTitle(parts[1])
			if artist != "" && track != "" {
				return trackMetadata{Artist: artist, Title: track}
			}
		}
	}

	if m := reBy.FindStringSubmatch(cleanedTitle); m != nil {
		return trackMetadata{
			Artist: cleanArtist(m[2]),
			Title:  cleanTitle(m[1]),
		}
	}

	return trackMetadata{
		Artist: cleanedUploader,
		Title:  cleanedTitle,
	}
}

func outputTemplate(output string, cfg config, meta *trackMetadata) (string, error) {
	defaultTemplate := "%(title)s.%(ext)s"
	if cfg.Mode == "audio" {
		defaultTemplate = "%(artist,uploader)s - %(track,title)s.%(ext)s"
		if meta != nil && strings.TrimSpace(meta.Title) != "" {
			artist := meta.Artist
			if strings.TrimSpace(artist) == "" {
				artist = "Unknown Artist"
			}
			defaultTemplate = fmt.Sprintf(
				"%s - %s.%%(ext)s",
				sanitizeFilenamePart(artist),
				sanitizeFilenamePart(meta.Title),
			)
		}
	}

	if output == "" {
		if cfg.Mode == "audio" {
			return defaultTemplate, nil
		}
		return "", nil
	}

	expanded := output
	if strings.HasPrefix(expanded, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to resolve home directory: %w", err)
		}
		if expanded == "~" {
			expanded = home
		} else if strings.HasPrefix(expanded, "~/") {
			expanded = filepath.Join(home, strings.TrimPrefix(expanded, "~/"))
		}
	}

	info, err := os.Stat(expanded)
	if err == nil && info.IsDir() {
		return filepath.Join(expanded, defaultTemplate), nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("unable to read output path: %w", err)
	}

	if strings.HasSuffix(output, "/") || strings.HasSuffix(output, "\\") {
		return filepath.Join(expanded, defaultTemplate), nil
	}

	return expanded, nil
}

func buildArgs(cfg config, meta *trackMetadata) ([]string, error) {
	args := []string{}

	switch cfg.Mode {
	case "audio":
		args = append(args, "-x", "--audio-format", "mp3", "--audio-quality", "0")
	case "video":
		args = append(args, "-f", "bv*[ext=mp4]/bv*", "--recode-video", "mp4")
	case "full":
		args = append(args, "-f", "bv*+ba/b", "--merge-output-format", "mp4")
	default:
		return nil, fmt.Errorf("invalid mode %q; expected audio, video, or full", cfg.Mode)
	}

	if cfg.Start != "" || cfg.End != "" {
		start := cfg.Start
		if start == "" {
			start = "00:00:00"
		}
		section := "*" + start + "-" + cfg.End
		args = append(args, "--download-sections", section)
	}

	if cfg.Output != "" {
		template, err := outputTemplate(cfg.Output, cfg, meta)
		if err != nil {
			return nil, err
		}
		args = append(args, "-o", template)
	} else {
		template, err := outputTemplate("", cfg, meta)
		if err != nil {
			return nil, err
		}
		if template != "" {
			args = append(args, "-o", template)
		}
	}

	args = append(args, cfg.URL)
	return args, nil
}

func newFlagSet(cfg *config, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet("ytcli", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&cfg.Start, "start", "", "clip start timestamp (MM:SS or HH:MM:SS)")
	fs.StringVar(&cfg.End, "end", "", "clip end timestamp (MM:SS or HH:MM:SS)")
	fs.StringVar(&cfg.Mode, "mode", "full", "download mode: audio, video, or full")
	fs.StringVar(&cfg.Output, "output", "", "destination file path or directory")
	fs.StringVar(&cfg.Artist, "artist", "", "manual artist tag override for audio mode")
	fs.StringVar(&cfg.Song, "song", "", "manual song title tag override for audio mode")
	fs.BoolVar(&cfg.AppleMusic, "apple-music", false, "when mode=audio, import downloaded track into Apple Music library (macOS)")
	fs.BoolVar(&cfg.ShowVersion, "version", false, "print version and build metadata, then exit")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage:\n  ytcli [--start MM:SS|HH:MM:SS] [--end MM:SS|HH:MM:SS] [--mode audio|video|full] [--output PATH] [--artist NAME] [--song TITLE] [--apple-music] [--version] <url>\n  ytcli version\n")
		fs.PrintDefaults()
	}
	return fs
}

func parseConfig(args []string, stderr io.Writer) (config, *flag.FlagSet, error) {
	var cfg config
	if len(args) == 1 && args[0] == "version" {
		cfg.ShowVersion = true
		return cfg, nil, nil
	}

	fs := newFlagSet(&cfg, stderr)
	if err := fs.Parse(args); err != nil {
		return cfg, fs, err
	}

	if cfg.ShowVersion {
		return cfg, fs, nil
	}

	if fs.NArg() != 1 {
		return cfg, fs, fmt.Errorf("missing required url argument")
	}
	cfg.URL = fs.Arg(0)

	start, err := normalizeTimestamp(cfg.Start)
	if err != nil {
		return cfg, fs, err
	}
	end, err := normalizeTimestamp(cfg.End)
	if err != nil {
		return cfg, fs, err
	}
	cfg.Start = start
	cfg.End = end

	if cfg.Start != "" && cfg.End != "" && timestampToSeconds(cfg.End) <= timestampToSeconds(cfg.Start) {
		return cfg, fs, fmt.Errorf("--end must be greater than --start")
	}
	if cfg.AppleMusic && cfg.Mode != "audio" {
		return cfg, fs, fmt.Errorf("--apple-music is only supported with --mode audio")
	}
	if (strings.TrimSpace(cfg.Artist) != "" || strings.TrimSpace(cfg.Song) != "") && cfg.Mode != "audio" {
		return cfg, fs, fmt.Errorf("--artist and --song are only supported with --mode audio")
	}
	if cfg.Song != "" && cleanTitle(cfg.Song) == "" {
		return cfg, fs, fmt.Errorf("--song must not be empty")
	}

	return cfg, fs, nil
}

func applyManualMetadata(base *trackMetadata, artistOverride, songOverride string) (*trackMetadata, bool) {
	if strings.TrimSpace(artistOverride) == "" && strings.TrimSpace(songOverride) == "" {
		return base, false
	}

	combined := trackMetadata{}
	if base != nil {
		combined = *base
	}

	if strings.TrimSpace(artistOverride) != "" {
		combined.Artist = cleanArtist(artistOverride)
	}
	if strings.TrimSpace(songOverride) != "" {
		combined.Title = cleanTitle(songOverride)
	}

	if strings.TrimSpace(combined.Artist) == "" {
		combined.Artist = "Unknown Artist"
	}
	return &combined, true
}

func importIntoAppleMusic(path string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("--apple-music is only supported on macOS")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve downloaded file path: %w", err)
	}
	if _, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("downloaded file not found for Apple Music import: %w", err)
	}

	script := `
on run argv
	set targetPath to POSIX file (item 1 of argv)
	tell application "Music"
		add targetPath
	end tell
end run
`
	cmd := exec.Command("osascript", "-e", script, absPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("failed to import into Apple Music: %s", message)
	}

	return nil
}

func parseFinalPathLine(line string) (string, bool) {
	if !strings.HasPrefix(line, finalPathPrefix) {
		return "", false
	}

	path := strings.TrimSpace(strings.TrimPrefix(line, finalPathPrefix))
	if path == "" {
		return "", false
	}
	return path, true
}

func writeAudioMetadata(path string, meta trackMetadata) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve downloaded file path: %w", err)
	}
	if _, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("downloaded file not found for metadata tagging: %w", err)
	}

	ext := filepath.Ext(absPath)
	base := strings.TrimSuffix(filepath.Base(absPath), ext)
	tmpPath := filepath.Join(filepath.Dir(absPath), base+".ytcli-tagging"+ext)
	defer os.Remove(tmpPath)

	cmd := exec.Command(
		"ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-nostdin",
		"-y",
		"-i", absPath,
		"-map", "0",
		"-c", "copy",
		"-metadata", "artist="+meta.Artist,
		"-metadata", "title="+meta.Title,
		tmpPath,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("ffmpeg metadata write failed: %s", message)
	}

	if err := os.Rename(tmpPath, absPath); err != nil {
		return fmt.Errorf("failed to finalize tagged audio file: %w", err)
	}
	return nil
}

func inferTrackMetadataFromPath(path string) (trackMetadata, bool) {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	base = strings.TrimSpace(base)
	if base == "" {
		return trackMetadata{}, false
	}

	parts := strings.SplitN(base, " - ", 2)
	if len(parts) == 2 {
		artist := cleanArtist(parts[0])
		title := cleanTitle(parts[1])
		if title != "" {
			return trackMetadata{Artist: artist, Title: title}, true
		}
	}

	title := cleanTitle(base)
	if title == "" {
		return trackMetadata{}, false
	}
	return trackMetadata{Artist: "Unknown Artist", Title: title}, true
}

func resolveYtDlpBinary() (string, error) {
	if p, err := exec.LookPath("yt-dlp"); err == nil {
		return p, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to determine current directory: %w", err)
	}

	localVenv := filepath.Join(cwd, ".venv", "bin", "yt-dlp")
	if info, err := os.Stat(localVenv); err == nil && !info.IsDir() {
		return localVenv, nil
	}

	return "", fmt.Errorf("yt-dlp is not installed or not available in PATH (and .venv/bin/yt-dlp was not found)")
}

func fetchTrackMetadata(ytDlpBinary, url string) (*trackMetadata, error) {
	cmd := exec.Command(
		ytDlpBinary,
		"--skip-download",
		"--no-warnings",
		"--print", "%(artist,uploader)s",
		"--print", "%(track,title)s",
		url,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := []string{}
	for _, line := range strings.Split(string(out), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	if len(lines) < 2 {
		return nil, fmt.Errorf("failed to fetch artist/title metadata")
	}

	meta := trackMetadata{
		Artist: cleanArtist(lines[0]),
		Title:  cleanTitle(lines[1]),
	}
	if meta.Title == "" {
		return nil, fmt.Errorf("missing track title metadata")
	}
	return &meta, nil
}

func run(cfg config, stdout, stderr io.Writer) error {
	ytDlpBinary, err := resolveYtDlpBinary()
	if err != nil {
		return err
	}

	var meta *trackMetadata
	if cfg.Mode == "audio" {
		fetchedMeta, fetchErr := fetchTrackMetadata(ytDlpBinary, cfg.URL)
		if fetchErr == nil {
			meta = fetchedMeta
			fmt.Fprintf(stdout, "Parsed audio metadata: %s - %s\n", meta.Artist, meta.Title)
		} else {
			fmt.Fprintf(stderr, "Warning: metadata parsing failed, using yt-dlp artist/title fallback template (%v)\n", fetchErr)
		}

		if updatedMeta, applied := applyManualMetadata(meta, cfg.Artist, cfg.Song); applied {
			meta = updatedMeta
			fmt.Fprintf(stdout, "Using manual metadata override: %s - %s\n", meta.Artist, meta.Title)
		}
	}

	args, err := buildArgs(cfg, meta)
	if err != nil {
		return err
	}

	var downloadedPath string
	captureFinalPath := cfg.Mode == "audio" || cfg.AppleMusic
	if captureFinalPath {
		args = append(args, "--print", "after_move:"+finalPathPrefix+"%(filepath)s")
	}

	cmd := exec.Command(ytDlpBinary, args...)
	cmd.Stderr = stderr
	if captureFinalPath {
		cmdStdout, err := cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("failed to capture yt-dlp output: %w", err)
		}
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start download: %w", err)
		}

		scanner := bufio.NewScanner(cmdStdout)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if parsedPath, ok := parseFinalPathLine(line); ok {
				downloadedPath = parsedPath
				continue
			}
			fmt.Fprintln(stdout, line)
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("failed to read yt-dlp output: %w", err)
		}
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("download failed: %w", err)
		}
	} else {
		cmd.Stdout = stdout
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("download failed: %w", err)
		}
	}

	if cfg.Mode == "audio" {
		if strings.TrimSpace(downloadedPath) == "" {
			fmt.Fprintln(stderr, "Warning: download completed but output path was unavailable, skipping metadata tagging")
		} else {
			needsInference := meta == nil ||
				strings.TrimSpace(meta.Title) == "" ||
				strings.TrimSpace(meta.Artist) == "" ||
				(meta.Artist == "Unknown Artist" && strings.TrimSpace(cfg.Artist) == "")
			if needsInference {
				inferredMeta, ok := inferTrackMetadataFromPath(downloadedPath)
				if !ok {
					fmt.Fprintln(stderr, "Warning: metadata unavailable and could not infer tags from file name")
				} else if meta == nil {
					meta = &inferredMeta
					fmt.Fprintf(stderr, "Warning: metadata fetch failed, inferred tags from filename: %s - %s\n", meta.Artist, meta.Title)
				} else {
					if strings.TrimSpace(meta.Title) == "" {
						meta.Title = inferredMeta.Title
					}
					if strings.TrimSpace(meta.Artist) == "" || (meta.Artist == "Unknown Artist" && strings.TrimSpace(cfg.Artist) == "") {
						meta.Artist = inferredMeta.Artist
					}
				}
			}

			if meta != nil && strings.TrimSpace(meta.Title) != "" {
				if err := writeAudioMetadata(downloadedPath, *meta); err != nil {
					fmt.Fprintf(stderr, "Warning: failed to write audio metadata tags (%v)\n", err)
				} else {
					fmt.Fprintf(stdout, "Tagged audio metadata: %s - %s\n", meta.Artist, meta.Title)
				}
			} else if meta != nil {
				fmt.Fprintln(stderr, "Warning: metadata title is empty, skipping audio metadata tagging")
			}
		}
	}

	if cfg.AppleMusic {
		if strings.TrimSpace(downloadedPath) == "" {
			return fmt.Errorf("download completed but could not determine output path for Apple Music import")
		}
		if err := importIntoAppleMusic(downloadedPath); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Imported into Apple Music: %s\n", downloadedPath)
	}

	fmt.Fprintln(stdout, "Download completed successfully.")
	return nil
}

func Main(args []string, stdout, stderr io.Writer) int {
	cfg, fs, err := parseConfig(args, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintf(stderr, "Error: %v\n", err)
		if fs != nil {
			fs.Usage()
		}
		return 2
	}

	if cfg.ShowVersion {
		fmt.Fprintln(stdout, buildinfo.String())
		return 0
	}

	if err := run(cfg, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}
