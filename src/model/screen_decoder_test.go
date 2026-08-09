package model

import (
	"strings"
	"testing"
)

func TestScreenDecoder(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain text",
			in:   "hello",
			want: "hello",
		},
		{
			name: "cursor home then text",
			in:   "\x1b[Hhello",
			want: "hello",
		},
		{
			name: "positioned text",
			in:   "\x1b[2;3Hhi",
			want: "\n  hi",
		},
		{
			name: "overwrite leaves old cells erased",
			in:   "\x1b[Habcdef\x1b[1;1H\x1b[Kxy",
			want: "xy",
		},
		{
			name: "erase to end of line",
			in:   "\x1b[Habcdef\x1b[1;3H\x1b[K",
			want: "ab",
		},
		{
			name: "cursor forward then write",
			in:   "\x1b[Hab\x1b[2Ccd",
			want: "ab  cd",
		},
		{
			name: "erase screen",
			in:   "\x1b[Habcdef\x1b[2Jhello",
			want: "hello",
		},
		{
			name: "multi-row",
			in:   "\x1b[Hrow1\nrow2",
			want: "row1\nrow2",
		},
		{
			name: "mode sequences ignored",
			in:   "\x1b[?1049h\x1b[?25lhi",
			want: "hi",
		},
		{
			name: "scroll region ignored",
			in:   "\x1b[1;40r\x1b[Hok",
			want: "ok",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newScreen(80, 40)
			s.feed(tc.in)
			got := s.String()
			if got != tc.want {
				t.Errorf("screen.String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestScreenFeedIsIncremental(t *testing.T) {
	s := newScreen(80, 40)
	// Feed the stream in arbitrary chunks; the result must match feeding it
	// all at once.
	stream := "\x1b[2J\x1b[H  Groups   \x1b[2;5Hcore\x1b[10;5H● web  nginx\x1b[20;1HPress s to start."
	// Feed byte-by-byte to stress the incremental parser.
	for i := 0; i < len(stream); i++ {
		s.feed(stream[i : i+1])
	}
	want := newScreen(80, 40)
	want.feed(stream)
	if got := s.String(); got != want.String() {
		t.Errorf("incremental feed mismatch:\n got: %q\nwant: %q", got, want.String())
	}
}

func TestScreenTrimsTrailingBlanksAndRows(t *testing.T) {
	s := newScreen(80, 40)
	s.feed("\x1b[Habc   \x1b[2;1H   ")
	got := s.String()
	if got != "abc" {
		t.Errorf("String() = %q, want %q", got, "abc")
	}
}

// TestScreenDoesNotGhostClosedModal is the regression test for the flake: a
// modal's text must disappear from the screen once the renderer overwrites
// those cells, even though the bytes remain in the append-only buffer.
func TestScreenDoesNotGhostClosedModal(t *testing.T) {
	s := newScreen(80, 40)
	// Frame 1: the rename modal is up.
	s.feed("\x1b[2J\x1b[HGroups...\x1b[16;41H╭─────────────╮\x1b[18;41H│ Rename group │\x1b[20;41H│ > core        │\x1b[25;41H╰─────────────╯")
	if !strings.Contains(s.String(), "Rename group") {
		t.Fatalf("modal should be on screen: %q", s.String())
	}
	// Frame 2: the modal closes; the renderer repaints the underlying panel
	// over those cells.
	s.feed("\x1b[16;41H                                            \x1b[18;41H                                            \x1b[20;41H                                            \x1b[5;3Hcore2\x1b[10;43H● web  nginx")
	if strings.Contains(s.String(), "Rename group") {
		t.Fatalf("closed modal ghosted on screen: %q", s.String())
	}
	if !strings.Contains(s.String(), "core2") {
		t.Fatalf("renamed group should be on screen: %q", s.String())
	}
}
