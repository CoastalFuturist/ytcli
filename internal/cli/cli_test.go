package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeTimestamp(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "mmss", input: "3:05", want: "00:03:05"},
		{name: "hhmmss", input: "1:02:03", want: "01:02:03"},
		{name: "invalid", input: "99", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeTimestamp(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got value %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseTrackMetadata(t *testing.T) {
	tests := []struct {
		name       string
		title      string
		uploader   string
		wantArtist string
		wantTrack  string
	}{
		{
			name:       "artist dash title",
			title:      "Daft Punk - Harder Better Faster Stronger (Official Video)",
			uploader:   "Random Channel",
			wantArtist: "Daft Punk",
			wantTrack:  "Harder Better Faster Stronger",
		},
		{
			name:       "title by artist",
			title:      "Believer by Imagine Dragons",
			uploader:   "Imagine Dragons - Topic",
			wantArtist: "Imagine Dragons",
			wantTrack:  "Believer",
		},
		{
			name:       "fallback to uploader",
			title:      "Mystery Song",
			uploader:   "Unknown Uploader VEVO",
			wantArtist: "Unknown Uploader",
			wantTrack:  "Mystery Song",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			meta := parseTrackMetadata(tc.title, tc.uploader)
			if meta.Artist != tc.wantArtist {
				t.Fatalf("artist got %q, want %q", meta.Artist, tc.wantArtist)
			}
			if meta.Title != tc.wantTrack {
				t.Fatalf("title got %q, want %q", meta.Title, tc.wantTrack)
			}
		})
	}
}

func TestParseFinalPathLine(t *testing.T) {
	path, ok := parseFinalPathLine("__YTCLI_FINAL_PATH__:/tmp/song.mp3")
	if !ok {
		t.Fatal("expected prefixed line to parse")
	}
	if path != "/tmp/song.mp3" {
		t.Fatalf("got %q, want %q", path, "/tmp/song.mp3")
	}

	_, ok = parseFinalPathLine("[download] 100%")
	if ok {
		t.Fatal("did not expect non-prefixed line to parse")
	}
}

func TestOutputTemplateAudioWithoutMetadata(t *testing.T) {
	cfg := config{Mode: "audio"}

	got, err := outputTemplate("", cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "%(artist,uploader)s - %(track,title)s.%(ext)s"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildArgsAudioUsesFallbackTemplateWhenMetadataUnavailable(t *testing.T) {
	cfg := config{
		Mode: "audio",
		URL:  "https://youtu.be/example",
	}

	args, err := buildArgs(cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantOutput := "%(artist,uploader)s - %(track,title)s.%(ext)s"
	foundOutput := false
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "-o" {
			foundOutput = true
			if args[i+1] != wantOutput {
				t.Fatalf("got output template %q, want %q", args[i+1], wantOutput)
			}
			break
		}
	}
	if !foundOutput {
		t.Fatalf("expected -o argument, got %v", args)
	}
}

func TestOutputTemplateAudioDirectoryWithoutMetadata(t *testing.T) {
	cfg := config{Mode: "audio"}

	got, err := outputTemplate("/tmp", cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join("/tmp", "%(artist,uploader)s - %(track,title)s.%(ext)s")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestOutputTemplateAudioWithIncompleteMetadataUsesFallbackTemplate(t *testing.T) {
	cfg := config{Mode: "audio"}
	meta := &trackMetadata{Artist: "Daft Punk", Title: ""}

	got, err := outputTemplate("", cfg, meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "%(artist,uploader)s - %(track,title)s.%(ext)s"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestApplyManualMetadata(t *testing.T) {
	base := &trackMetadata{Artist: "Original Artist", Title: "Original Title"}

	got, applied := applyManualMetadata(base, "Override Artist", "")
	if !applied {
		t.Fatal("expected manual metadata to be applied")
	}
	if got.Artist != "Override Artist" {
		t.Fatalf("artist got %q, want %q", got.Artist, "Override Artist")
	}
	if got.Title != "Original Title" {
		t.Fatalf("title got %q, want %q", got.Title, "Original Title")
	}

	got, applied = applyManualMetadata(nil, "Custom Artist", "Custom Song")
	if !applied {
		t.Fatal("expected manual metadata to be applied")
	}
	if got.Artist != "Custom Artist" {
		t.Fatalf("artist got %q, want %q", got.Artist, "Custom Artist")
	}
	if got.Title != "Custom Song" {
		t.Fatalf("title got %q, want %q", got.Title, "Custom Song")
	}

	got, applied = applyManualMetadata(base, "", "")
	if applied {
		t.Fatal("did not expect manual metadata to be applied")
	}
	if got != base {
		t.Fatal("expected original metadata pointer to be returned")
	}
}

func TestParseConfigRejectsMetadataOverridesOutsideAudioMode(t *testing.T) {
	_, _, err := parseConfig(
		[]string{"--mode", "full", "--artist", "Daft Punk", "https://youtu.be/example"},
		&bytes.Buffer{},
	)
	if err == nil {
		t.Fatal("expected parseConfig to return an error")
	}
	if !strings.Contains(err.Error(), "only supported with --mode audio") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseConfigAcceptsMetadataOverridesInAudioMode(t *testing.T) {
	cfg, _, err := parseConfig(
		[]string{"--mode", "audio", "--artist", "Daft Punk", "--song", "One More Time", "https://youtu.be/example"},
		&bytes.Buffer{},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Artist != "Daft Punk" {
		t.Fatalf("artist got %q, want %q", cfg.Artist, "Daft Punk")
	}
	if cfg.Song != "One More Time" {
		t.Fatalf("song got %q, want %q", cfg.Song, "One More Time")
	}
}

func TestParseConfigRejectsEmptySongOverride(t *testing.T) {
	_, _, err := parseConfig(
		[]string{"--mode", "audio", "--song", "   ", "https://youtu.be/example"},
		&bytes.Buffer{},
	)
	if err == nil {
		t.Fatal("expected parseConfig to return an error")
	}
	if !strings.Contains(err.Error(), "--song must not be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInferTrackMetadataFromPath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantOK     bool
		wantArtist string
		wantTitle  string
	}{
		{
			name:       "artist and title from filename",
			path:       "/tmp/Daft Punk - One More Time.mp3",
			wantOK:     true,
			wantArtist: "Daft Punk",
			wantTitle:  "One More Time",
		},
		{
			name:       "title only fallback",
			path:       "/tmp/Unknown Song.mp3",
			wantOK:     true,
			wantArtist: "Unknown Artist",
			wantTitle:  "Unknown Song",
		},
		{
			name:   "empty filename",
			path:   "/tmp/.mp3",
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := inferTrackMetadataFromPath(tc.path)
			if ok != tc.wantOK {
				t.Fatalf("ok got %v, want %v", ok, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if got.Artist != tc.wantArtist {
				t.Fatalf("artist got %q, want %q", got.Artist, tc.wantArtist)
			}
			if got.Title != tc.wantTitle {
				t.Fatalf("title got %q, want %q", got.Title, tc.wantTitle)
			}
		})
	}
}
