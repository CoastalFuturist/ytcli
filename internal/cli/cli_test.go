package cli

import "testing"

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
