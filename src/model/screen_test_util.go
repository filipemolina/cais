package model

import (
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

// screen is a minimal ANSI cell-buffer decoder for the subset of escape
// sequences the Bubble Tea cursed renderer emits into the rig's output
// buffer. It reconstructs the current visible screen from the append-only
// delta stream, so waiters can ask "is this substring on screen right now?"
// instead of "did it ever appear in the byte history?".
//
// Supported sequences (everything the renderer emits with the rig's setup):
//
//	ESC [ H            cursor home (1,1)
//	ESC [ r ; c H      cursor to row r, col c (1-based)
//	ESC [ n C          cursor forward n (default 1)
//	ESC [ K            erase to end of line
//	ESC [ 1 K          erase start of line
//	ESC [ 2 J          erase entire screen (cursor home)
//	ESC [ r ; c r      set scroll region (ignored: no scrolling used)
//	ESC [ ? n h/l/u    mode set/reset (ignored)
//	ESC [ > n m        modifyOtherKeys (ignored)
//	ESC [ = n ; m u    kitty keyboard (ignored)
//
// Plain text writes cells at the cursor, overwriting what was there. The
// screen is a grid of runes; String() renders it row by row, trimming
// trailing blanks, which is what substring matching should see.
type screen struct {
	mu      sync.Mutex
	rows    [][]rune
	curR    int    // 0-based
	curC    int    // 0-based
	pending string // buffered bytes of a split escape sequence
}

func newScreen(w, h int) *screen {
	s := &screen{}
	s.resize(w, h)
	return s
}

func (s *screen) resize(w, h int) {
	s.rows = make([][]rune, h)
	for i := range s.rows {
		s.rows[i] = make([]rune, w)
		for j := range s.rows[i] {
			s.rows[i][j] = ' '
		}
	}
}

func (s *screen) home() { s.curR, s.curC = 0, 0 }

// newline moves the cursor to the start of the next line. The renderer
// enables newline mapping (SetMapNewline) when the output is not a TTY,
// meaning the terminal translates a bare LF into CRLF, so column resets to
// 0.
func (s *screen) newline() {
	if s.curR < len(s.rows)-1 {
		s.curR++
	}
	s.curC = 0
}
func (s *screen) move(r, c int) { s.curR, s.curC = r, c }
func (s *screen) forward(n int) { s.curC += n }

func (s *screen) eraseLine(mode int) {
	row := s.rows[s.curR]
	switch mode {
	case 0: // to end of line
		for c := s.curC; c < len(row); c++ {
			row[c] = ' '
		}
	case 1: // start of line
		for c := 0; c <= s.curC && c < len(row); c++ {
			row[c] = ' '
		}
	case 2: // entire line
		for c := range row {
			row[c] = ' '
		}
	}
}

func (s *screen) eraseScreen() {
	for i := range s.rows {
		for j := range s.rows[i] {
			s.rows[i][j] = ' '
		}
	}
	s.home()
}

func (s *screen) write(text string) {
	s.writeRunes([]rune(text))
}

func (s *screen) writeRunes(runes []rune) {
	row := s.rows[s.curR]
	for _, r := range runes {
		if s.curC >= len(row) {
			break
		}
		row[s.curC] = r
		s.curC++
	}
}

// String renders the current screen, trimming trailing blank columns and
// dropping fully-blank rows at the bottom (so a modal that closed leaves no
// ghost cells). Safe for concurrent use.
func (s *screen) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var b strings.Builder
	lastNonBlank := -1
	for i, row := range s.rows {
		trimmed := strings.TrimRight(string(row), " ")
		if strings.TrimSpace(trimmed) != "" {
			lastNonBlank = i
		}
	}
	for i, row := range s.rows {
		if i > lastNonBlank {
			break
		}
		b.WriteString(strings.TrimRight(string(row), " "))
		if i < lastNonBlank {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// feed decodes a chunk of the render stream into the screen state. It is
// safe to call with arbitrary chunk boundaries: an escape sequence split
// across calls is buffered until it completes.
func (s *screen) feed(data string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.feedLocked(data)
}

func (s *screen) feedLocked(data string) {
	s.pending += data
	data = s.pending
	s.pending = ""

	i := 0
	for i < len(data) {
		if data[i] != '\x1b' {
			// Plain text run up to the next escape, decoded as UTF-8 runes.
			// Newline and carriage return are cursor movement, not cells.
			j := i
			for j < len(data) && data[j] != '\x1b' && data[j] != '\n' && data[j] != '\r' {
				j++
			}
			run := data[i:j]
			// If the run ends mid-UTF-8-rune (bytes split across chunks),
			// hold the incomplete suffix in pending and process the rest.
			if !utf8.FullRune([]byte(run)) {
				k := len(run)
				for k > 0 && !utf8.RuneStart(run[k-1]) {
					k--
				}
				if k > 0 && !utf8.FullRune([]byte(run[k-1:])) {
					s.pending = run[k-1:] + s.pending
					run = run[:k-1]
				}
			}
			s.writeRunes([]rune(run))
			i = j
			if i < len(data) && data[i] == '\n' {
				s.newline()
				i++
			} else if i < len(data) && data[i] == '\r' {
				s.curC = 0
				i++
			}
			continue
		}
		if i+1 >= len(data) || data[i+1] != '[' {
			// A lone ESC or a non-CSI escape; if it's the last byte we may
			// be mid-sequence, so buffer it. Otherwise skip it.
			if i == len(data)-1 {
				s.pending = data[i:]
				return
			}
			i++
			continue
		}
		// Parse the CSI sequence: ESC [ params... final-byte
		j := i + 2
		for j < len(data) && !isFinalByte(data[j]) {
			j++
		}
		if j >= len(data) {
			// Truncated sequence: buffer the remainder and stop.
			s.pending = data[i:]
			return
		}
		paramStr := data[i+2 : j]
		final := data[j]
		params := parseParams(paramStr)
		s.applyCSI(params, final)
		i = j + 1
	}
}

func isFinalByte(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func parseParams(s string) []int {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ";")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimPrefix(p, "?")
		if p == "" {
			out = append(out, 0)
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			out = append(out, 0)
			continue
		}
		out = append(out, n)
	}
	return out
}

func (s *screen) applyCSI(params []int, final byte) {
	get := func(idx int, def int) int {
		if idx < len(params) && params[idx] != 0 {
			return params[idx]
		}
		return def
	}
	switch final {
	case 'H', 'f':
		r := get(0, 1) - 1
		c := get(1, 1) - 1
		if r < 0 {
			r = 0
		}
		if c < 0 {
			c = 0
		}
		if r >= len(s.rows) {
			return
		}
		s.move(r, c)
	case 'C':
		s.forward(get(0, 1))
	case 'K':
		s.eraseLine(get(0, 0))
	case 'J':
		mode := get(0, 0)
		if mode == 2 {
			s.eraseScreen()
		} else if mode == 0 {
			// erase from cursor to end of screen
			s.eraseLine(0)
		}
	case 'r', 'h', 'l', 'u', 'm':
		// scroll region / modes / colors: ignored
	default:
		// unknown: ignored
	}
}
