package cmds

import "testing"

// TestWaitForLogBatchesBufferedLines is the fix for the logs modal opening
// with a visible scroll-to-bottom: `docker logs --tail N` replays its
// history in a tight burst, and WaitForLog must drain everything already
// buffered into one LogLinesMsg rather than returning after the first line,
// or the modal would still render (and follow-scroll) once per line.
func TestWaitForLogBatchesBufferedLines(t *testing.T) {
	ch := make(chan string, 3)
	ch <- "line1"
	ch <- "line2"
	ch <- "line3"

	msg := WaitForLog(ch)()

	lines, ok := msg.(LogLinesMsg)
	if !ok {
		t.Fatalf("expected LogLinesMsg, got %T", msg)
	}
	want := []string{"line1", "line2", "line3"}
	if len(lines) != len(want) {
		t.Fatalf("got %d lines, want %d: %v", len(lines), len(want), lines)
	}
	for i, w := range want {
		if lines[i] != w {
			t.Errorf("line %d = %q, want %q", i, lines[i], w)
		}
	}
}

// TestWaitForLogStopsAtWhatsBuffered checks the drain does not block waiting
// for more lines than are already available - a line sent after the call
// starts must wait for the next WaitForLog instead of being folded in.
func TestWaitForLogStopsAtWhatsBuffered(t *testing.T) {
	ch := make(chan string, 2)
	ch <- "line1"

	msg := WaitForLog(ch)()

	lines, ok := msg.(LogLinesMsg)
	if !ok {
		t.Fatalf("expected LogLinesMsg, got %T", msg)
	}
	if len(lines) != 1 || lines[0] != "line1" {
		t.Fatalf("got %v, want [line1]", lines)
	}
}

func TestWaitForLogOnClosedEmptyChannel(t *testing.T) {
	ch := make(chan string)
	close(ch)

	msg := WaitForLog(ch)()

	if _, ok := msg.(LogStreamEndedMsg); !ok {
		t.Fatalf("expected LogStreamEndedMsg, got %T", msg)
	}
}

// TestWaitForLogDrainsThenSeesClose covers a channel that has buffered lines
// and is then closed: the batch already sent should come back intact, with
// the close only surfacing on the following call (WaitForLog's caller
// re-issues it after every LogLinesMsg).
func TestWaitForLogDrainsThenSeesClose(t *testing.T) {
	ch := make(chan string, 2)
	ch <- "line1"
	ch <- "line2"
	close(ch)

	msg := WaitForLog(ch)()
	lines, ok := msg.(LogLinesMsg)
	if !ok {
		t.Fatalf("expected LogLinesMsg, got %T", msg)
	}
	if len(lines) != 2 || lines[0] != "line1" || lines[1] != "line2" {
		t.Fatalf("got %v, want [line1 line2]", lines)
	}

	msg = WaitForLog(ch)()
	if _, ok := msg.(LogStreamEndedMsg); !ok {
		t.Fatalf("expected LogStreamEndedMsg on the following call, got %T", msg)
	}
}
