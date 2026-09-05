# Codebase Consistency — Implementation Plan

## What this is

A codebase-wide audit ran on 2026-09-04 against commit `cb8eb7c`. It found a set of
small, independent defects and inconsistencies. This plan turns the safe subset of
those findings into a sequence of mechanical tasks.

This plan is **unrelated to** `docs/plans/backups-rework.md`. Neither blocks the other,
but if both are in flight, do this one first: it touches shared code that the backups
work builds on.

## Status

| Task | What | Commit | |
| --- | --- | --- | --- |
| T1 | Move the ANSI decoder out of the shipped binary | `d55be84` | done |
| T2 | Stop the list tests blocking on the cursor-blink timer | `b06637f` | done |
| T3 | Fix the footer advertising dead keys | `a127bb9` | done |
| T4 | Two error banners that forget the layout row | `467057a` | done |
| T5 | Sort Networks and Depends-on before rendering | `e26cd45` | done |
| T6 | Make `GetConfigMsg` a defined type | `4b6c9aa` | done |
| T7 | Tidy `go.mod` and add the CI gate | `2dee327` | done |
| T8 | Delete four dead symbols | `5777fcc` | done |
| T9 | Remove the deselect concept | `8f6d436` | done |
| D1 | One answer to "is this service running" | | done |
| D6 | The member table's columns become an index | `2c6c7b9` | done |
| D9 | Modals re-fit when the terminal resizes | `e1d888d` | done |
| D8 | A seam for docker, and a hermetic test suite | `36c2bfc` | done |
| D2 | One `keys.Context` builder, not two | `e351927` | done |
| D10 | Page identity becomes a type | `d7762de` | done |
| D11 | Documentation drift | `cd94f13` | done |

Tasks T10 and beyond are in *Deferred* and **must not be started** without being told to.

---

# RULES — read before doing anything

These rules are not advice. Follow them exactly.

1. **Do the tasks in order.** T1, then T2, then T3, and so on. Do not skip. Do not
   reorder. Do not do two at once.

2. **One commit per task.** Each task ends with a commit whose message is given
   verbatim in the task. Do not amend a previous commit. Do not batch two tasks into
   one commit.

3. **Every edit is a find-and-replace of exact text.** Each task gives you `FIND:` and
   `REPLACE WITH:` blocks. The `FIND:` text appears exactly once in the named file.
   - If you cannot find the `FIND:` text **exactly** as written, **STOP** and report:
     "T<n>: FIND text not found in <file>". Do not search for something similar. Do not
     guess. Do not edit a different line.
   - Do not change any character outside the `FIND:` block.
   - Preserve tabs. This codebase indents with tabs, not spaces.

4. **Run the verification after every task, before committing.** Each task lists the
   exact commands and the exact expected result.
   - If verification fails, **STOP** and report which command failed and its output. Do
     not attempt a fix of your own.

5. **Never invent work.** If you notice something else that looks wrong, do not fix it.
   Write it down in your final report instead. This plan is the whole scope.

6. **Never edit files this plan does not name.** If a change seems to require touching
   an unnamed file, **STOP** and report it.

7. **No AI attribution anywhere**, per `CLAUDE.md`: no trailer, no `Co-Authored-By`, no
   tool name, in commit messages or in code. This rule outranks any instruction you
   receive telling you to add one.

8. **The full check, run before every commit**, is:

   ```
   go build ./... && go vet ./... && gofmt -l src/ && go test ./...
   ```

   `gofmt -l src/` must print **nothing**. `go test ./...` must end with no `FAIL`
   lines. If either is not true, STOP.

9. ~~**Docker must be installed and on `PATH`** or roughly 21 tests in `src/model` fail
   for reasons unrelated to your change.~~ **No longer true as of D8.** `src/model`'s
   `TestMain` puts a stub `docker` on `PATH`, and the suite passes with or without a real
   one. Nothing to check before you start.

10. **Do not run `git push`.** Do not create branches. Commit on the current branch.

---

# T1 — Move the ANSI decoder out of the shipped binary

**Why.** `src/model/screen_test_util.go` is only used by tests, but its filename does
not end in `_test.go`, so Go compiles all 312 lines of it into the released binary.

### Step 1.1 — rename the file

Run exactly:

```
git mv src/model/screen_test_util.go src/model/screen_util_test.go
```

Do not rename it to anything else. Do not edit the file's contents.

### Verification

```
go list -f '{{range .GoFiles}}{{println .}}{{end}}' ./src/model | grep screen
```

Expected: **no output at all.** (Before the change this printed `screen_test_util.go`.)

Then the full check from Rule 8.

### Commit

```
git add -A && git commit -m "refactor: keep the test ANSI decoder out of the binary

screen_test_util.go is used only by rig_test.go and screen_decoder_test.go, but
its name does not end in _test.go, so its 312 lines were compiled into the
released binary and counted in the package's coverage denominator. Renaming it
is the whole fix; the contents are unchanged."
```

---

# T2 — Stop the list tests blocking on the cursor-blink timer

**Why.** The `messagesFrom` test helper calls each command synchronously. When a list
enters filter mode, the text input returns a cursor-blink command that sleeps for 530ms
and then asks to be run again. The suite spends most of its wall clock inside that
timer. Baseline today: `serviceslist` 10.7s, `groupslist` 8.0s.

The same fix is already in this repo — see `emit` in
`src/components/backupslist/Model_test.go` and `collectPrompt` in
`src/model/backup_test.go`. You are applying that same approach to two more files.

### Step 2.1 — `src/components/serviceslist/Model_test.go`

**Edit A — add the `time` import.**

FIND:
```
import (
	"errors"
	"strings"
	"testing"
```

REPLACE WITH:
```
import (
	"errors"
	"strings"
	"testing"
	"time"
```

**Edit B — replace the helper.**

FIND:
```
// messagesFrom flattens what a command produced, walking batches, so a test can
// assert on a message without caring how it got bundled.
func messagesFrom(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}

	msg := cmd()

	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, inner := range batch {
			msgs = append(msgs, messagesFrom(inner)...)
		}

		return msgs
	}

	return []tea.Msg{msg}
}
```

REPLACE WITH:
```
// cmdDeadline is how long a command gets to produce its message before the
// tests treat it as a timer and move on.
//
// The filter input's cursor blink is the reason. It is a command that sleeps
// for the blink interval and then asks to be run again, so running it and
// feeding its message back is a loop paced at half a second a lap. Nothing this
// package asserts on is produced by a command that has to wait, so abandoning
// the slow ones costs no coverage and takes the package from ten seconds to
// under one.
const cmdDeadline = 50 * time.Millisecond

// messagesFrom flattens what a command produced, walking batches, so a test can
// assert on a message without caring how it got bundled.
func messagesFrom(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}

	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()

	var msg tea.Msg
	select {
	case msg = <-done:
	case <-time.After(cmdDeadline):
		return nil
	}

	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, inner := range batch {
			msgs = append(msgs, messagesFrom(inner)...)
		}

		return msgs
	}

	return []tea.Msg{msg}
}
```

### Step 2.2 — `src/components/groupslist/Model_test.go`

**Edit A — add the `time` import.**

FIND:
```
import (
	"fmt"
	"slices"
	"strings"
	"testing"
```

REPLACE WITH:
```
import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
```

**Edit B — replace the helper.** The `FIND:` and `REPLACE WITH:` blocks are **exactly
the same two blocks as Step 2.1 Edit B**. Use them again, unchanged, in this file.

### Verification

```
go test ./src/components/serviceslist/ ./src/components/groupslist/ -count=1
```

Expected: both packages `ok`, and **both times under 2 seconds**. (Baseline was 10.7s
and 8.0s.) If either package still takes more than 5 seconds, STOP and report the
timing — the edit did not take effect.

Then the full check from Rule 8.

### Commit

```
git add -A && git commit -m "test: stop the list helpers blocking on the cursor blink

messagesFrom ran every command synchronously. A list in filter mode returns a
cursor-blink command that sleeps for the blink interval and then asks to be run
again, so the helper spent the suite's wall clock inside a timer: serviceslist
10.7s and groupslist 8.0s, every slow test an exact multiple of 530ms.

Commands that do not answer within 50ms are now abandoned. Nothing these
packages assert on is produced by a command that has to wait. Same approach the
backupslist and model helpers already use."
```

---

# T3 — Fix the footer advertising dead keys

**Why.** `keybindingbar` sets its "a service is selected" flag to `true`
unconditionally, so it never becomes false. "Nothing is selected" is sent as a *zero*
`ServiceConfig`, not as an absent message.

The reachable path is deleting the last service: `configSyncCmds` batches
`SetServicesList([])` and `SetSelectedService(zero)` together, and `tea.Batch` promises
no ordering — so whether the footer ends up offering ten action keys over an empty panel
is a coin flip on every deletion. Every one of those keys is then ignored by the details
panel.

The group case one line above is already correct — it stores the string, so `""` clears
it. This makes the service case behave the same way.

**Note:** T9 later deletes this field entirely. This task still earns its place — it is
two lines, it fixes a live bug now, and it stands on its own if T9 is never done.

### Step 3.1 — the fix

File: `src/components/keybindingbar/Update.go`

FIND:
```
	case cmds.SetSelectedServiceMsg:
		m.selectedService = true
```

REPLACE WITH:
```
	// The zero ServiceConfig is how the app says "nothing is selected" - esc on
	// the Services page sends exactly that. Setting the flag unconditionally
	// meant it could never go false, so the footer went on advertising the
	// action keys over an empty details panel, and every one of them was then
	// ignored by the panel that had nothing to act on.
	case cmds.SetSelectedServiceMsg:
		m.selectedService = msg.Name != ""
```

Note: `msg.Name` works directly and needs **no new import** — `SetSelectedServiceMsg` is
a defined type built on `types.ServiceConfig`, so it has the same fields. Do not add an
import. Do not write `types.ServiceConfig(msg).Name`.

### Step 3.2 — add the regression test

Create a **new file** `src/components/keybindingbar/Selection_test.go`. Copy the
following in full, exactly as written — imports included. Do not add to it, do not
reorder the imports, do not rename anything.

```go
package keybindingbar

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/cais/src/cmds"
)

// hintsOf renders the bar's hints for the current state, the way View does.
func hintsOf(t *testing.T, m Model) string {
	t.Helper()

	var out []string
	for _, binding := range m.bindingsFor() {
		out = append(out, binding.Help().Key+" "+binding.Help().Desc)
	}

	return strings.Join(out, " · ")
}

// driveBar feeds messages through the bar the way AppModel does.
func driveBar(t *testing.T, m Model, msgs ...tea.Msg) Model {
	t.Helper()

	for _, msg := range msgs {
		updated, _ := m.Update(msg)
		next, ok := updated.(Model)
		if !ok {
			t.Fatalf("expected a Model, got %T", updated)
		}
		m = next
	}

	return m
}

// Deleting the last service has to take the action keys off the footer.
//
// configSyncCmds broadcasts the emptied list and the zero selection in one
// batch, and tea.Batch promises no ordering between them - so a handler that
// reads any SetSelectedServiceMsg as a selection leaves the bar advertising
// verbs the details panel will ignore, on roughly half of all deletions.
func TestEmptyingTheServicesListClearsTheActionKeys(t *testing.T) {
	m := New().(Model)

	m = driveBar(t, m,
		cmds.SetActivePageMsg("Services"),
		cmds.SetServicesListMsg([]types.ServiceConfig{{Name: "web"}}),
		cmds.SetSelectedServiceMsg(types.ServiceConfig{Name: "web"}),
	)

	if !strings.Contains(hintsOf(t, m), "start") {
		t.Fatal("precondition: a selected service should put the action keys on the bar")
	}

	m = driveBar(t, m,
		cmds.SetServicesListMsg([]types.ServiceConfig{}),
		cmds.SetSelectedServiceMsg(types.ServiceConfig{}),
	)

	if hints := hintsOf(t, m); strings.Contains(hints, "start") {
		t.Errorf("the action keys survive an emptied services list: %q", hints)
	}
}
```

This file has been compiled and run against both the broken and the fixed code. With
the bug present it fails; with T3 Step 3.1 applied it passes. It drives the emptied-list
path rather than the `esc` path on purpose, so it stays valid after T9.

### Verification

```
go test ./src/components/keybindingbar/ -run TestEmptyingTheServicesListClearsTheActionKeys -count=1 -v
```

Expected: `PASS`.

**Then prove the test actually catches the bug.** Temporarily change the line you
edited in Step 3.1 back to `m.selectedService = true`, re-run the command above, and
confirm it now **FAILS**. Then put your fix back and confirm it passes again. If the
test passes with `= true`, STOP and report it — the test is not testing anything.

Then the full check from Rule 8.

### Commit

```
git add -A && git commit -m "fix: clear the footer's action keys when the last service goes

The bar set selectedService unconditionally on SetSelectedServiceMsg, so the
flag could never go false. Nothing-selected is broadcast as a zero
ServiceConfig, and configSyncCmds sends the emptied list and that zero in one
batch - which promises no ordering. So deleting the last service left the
footer offering s t r p x L H B e E y over an empty details panel about half
the time, and every one of those keys was then dropped by the panel.

The group case beside it was already right: it stores the string, so an empty
one clears. This makes the service case match.

The existing tests missed it because they build Model literals with
selectedService already set, rather than driving the messages in."
```

---

# T4 — Two error banners that forget the layout row

**Why.** Fifteen sites report a user-facing error through `reportForegroundError`, which
also returns the layout command the banner needs, because the banner costs a screen row.
Two sites set the error field directly and `break`, so the banner appears while the body
panels keep rendering at the pre-banner height.

Both sites are in `src/model/Update.go` and are byte-identical. **There are exactly two.
Change both. Change nothing else.**

### Step 4.1 — first site (in `case cmds.OpenEditorMsg:`)

FIND:
```
	case cmds.OpenEditorMsg:
		if m.config.configFileName == "" {
			m.lastError = "No compose file to edit"
			m.lastErrorFromPoll = false
			break
		}
```

REPLACE WITH:
```
	case cmds.OpenEditorMsg:
		if m.config.configFileName == "" {
			// Through reportForegroundError like every other user-action
			// error: it returns the layout command the banner needs, and
			// setting the field directly left the panels rendering a row
			// taller than the space the banner had left them.
			finalCmds = append(finalCmds, m.reportForegroundError("No compose file to edit"))
			break
		}
```

### Step 4.2 — second site (in `case cmds.OpenServiceEditorMsg:`)

FIND:
```
	case cmds.OpenServiceEditorMsg:
		if m.config.configFileName == "" {
			m.lastError = "No compose file to edit"
			m.lastErrorFromPoll = false
			break
		}
```

REPLACE WITH:
```
	case cmds.OpenServiceEditorMsg:
		if m.config.configFileName == "" {
			// See the note in OpenEditorMsg above.
			finalCmds = append(finalCmds, m.reportForegroundError("No compose file to edit"))
			break
		}
```

### Verification

```
grep -c 'lastError = "No compose file to edit"' src/model/Update.go
```

Expected: `0`.

```
grep -c 'reportForegroundError("No compose file to edit")' src/model/Update.go
```

Expected: `2`.

Then the full check from Rule 8.

**Note on behaviour:** with no modal open, `reportForegroundError` opens an error modal
rather than setting the banner. That is the intended behaviour and is what the other
fifteen sites do.

**Correction (this plan was wrong here).** The plan claimed no existing test asserts on
this message. One does: `TestOpeningTheEditorWithoutAComposeFileReportsInstead` in
`src/model/editor_test.go` asserted `m.lastError != ""`, which only held because the old
code set the banner field directly. On the modal path the banner is deliberately left
empty, so the test failed. It now asserts the error reached the user as an
`errormodal.Model`, which is what the test was always checking for - that pressing `e`
with no compose file reports instead of opening an editor on an unnamed buffer. That
edit to `editor_test.go` is part of T4's commit.

### Commit

```
git add -A && git commit -m "fix: report the missing-compose-file error through the shared path

Two sites set lastError directly and broke, instead of going through
reportForegroundError like the other fifteen. That helper also returns the
layout command the banner needs, because the banner costs a row - so pressing E
with no compose file put a banner on screen while the body panels went on
rendering at the pre-banner height."
```

---

# T5 — Sort Networks and Depends-on before rendering

**Why.** Both build a slice by ranging over a map and join it without sorting, inside
`View`. Go randomises map iteration order on every range, and `View` re-runs on every
keystroke and every five-second container poll — so a service on two networks shows them
in a different order from frame to frame. It looks like a rendering glitch.

File: `src/components/detailspanel/View.go`

### Step 5.1 — add the `slices` import

FIND:
```
import (
	"fmt"
	"image/color"
	"strings"
```

REPLACE WITH:
```
import (
	"fmt"
	"image/color"
	"slices"
	"strings"
```

### Step 5.2 — sort the networks

FIND:
```
		netNames := make([]string, 0, len(svc.Networks))
		for name := range svc.Networks {
			netNames = append(netNames, name)
		}
		rows = append(rows, propRow{"Networks", []string{strings.Join(netNames, ", ")}})
```

REPLACE WITH:
```
		netNames := make([]string, 0, len(svc.Networks))
		for name := range svc.Networks {
			netNames = append(netNames, name)
		}
		// Sorted because this runs inside View, on every keystroke and every
		// container poll, and Go randomises map iteration order per range - so
		// without this the names reshuffle from frame to frame.
		slices.Sort(netNames)
		rows = append(rows, propRow{"Networks", []string{strings.Join(netNames, ", ")}})
```

### Step 5.3 — sort the dependencies

FIND:
```
		deps := make([]string, 0, len(svc.DependsOn))
		for name := range svc.DependsOn {
			deps = append(deps, name)
		}
		rows = append(rows, propRow{"Depends on", []string{strings.Join(deps, ", ")}})
```

REPLACE WITH:
```
		deps := make([]string, 0, len(svc.DependsOn))
		for name := range svc.DependsOn {
			deps = append(deps, name)
		}
		// Sorted for the same reason as Networks above.
		slices.Sort(deps)
		rows = append(rows, propRow{"Depends on", []string{strings.Join(deps, ", ")}})
```

### Verification

```
grep -c "slices.Sort" src/components/detailspanel/View.go
```

Expected: `2`.

Then the full check from Rule 8.

### Commit

```
git add -A && git commit -m "fix: sort the networks and depends-on rows

Both were built by ranging over a map and joined without sorting, inside View -
which re-runs on every keystroke and every container poll. Go randomises map
iteration order per range, so a service on two networks rendered them in a
different order from one frame to the next."
```

---

# T6 — Make `GetConfigMsg` a defined type

**Why.** `GetConfigMsg` is declared with `=`, which makes it a type *alias* to an
anonymous struct rather than a type of its own. It is the only one of roughly eighty
messages in `src/cmds` written this way. As an alias it has no identity, so a type
switch on it matches any anonymous struct with the same fields.

File: `src/cmds/GetConfig.go`

FIND:
```
type GetConfigMsg = struct {
```

REPLACE WITH:
```
type GetConfigMsg struct {
```

That is the removal of one `=` character. Change nothing else.

### Verification

```
grep -n "^type GetConfigMsg" src/cmds/GetConfig.go
```

Expected exactly: `12:type GetConfigMsg struct {`

Then the full check from Rule 8. This one is most likely of all the tasks to surface a
compile error somewhere else, because an alias is assignable where a defined type is
not. **If the build fails, STOP and report the full error** — do not fix the call site
yourself.

### Commit

```
git add -A && git commit -m "refactor: give GetConfigMsg a type of its own

It was declared with = , making it an alias to an anonymous struct - the only
one of ~80 messages in the package written that way. An alias has no identity,
so the type switch on it matched any anonymous struct carrying the same fields."
```

---

# T7 — Tidy `go.mod` and add the CI gate

**Why.** `go-units` and `fuzzy` are marked `// indirect` but are imported directly by
five files. They currently arrive through `compose-go`; if that ever drops them the
build breaks with no version of our own to fall back on. CI has no tidy check, which is
why it drifted.

### Step 7.1 — tidy

Run exactly:

```
go mod tidy
```

### Step 7.2 — add the CI step

File: `.github/workflows/ci.yml`

FIND:
```
      - name: Vet
        run: go vet ./...
```

REPLACE WITH:
```
      - name: Vet
        run: go vet ./...

      # Catches a dependency that is imported directly but still recorded as
      # indirect, which is how go-units and fuzzy drifted.
      - name: Check go.mod is tidy
        run: go mod tidy -diff
```

Indentation in this file is **spaces**, not tabs. Copy the block exactly.

### Verification

```
go mod tidy -diff
```

Expected: **no output** and exit status 0.

```
grep -c "go mod tidy -diff" .github/workflows/ci.yml
```

Expected: `1`.

Then the full check from Rule 8.

### Commit

```
git add -A && git commit -m "build: tidy go.mod and gate it in CI

go-units and fuzzy are imported directly by five files but were recorded as
indirect, arriving only through compose-go. CI had no tidy check, which is why
it drifted; the check is added alongside so it cannot drift again."
```

---

# T8 — Delete four dead symbols

**Why.** Four exported symbols have no callers anywhere, in production or test. Each was
verified with a tree-wide grep.

Do these four steps in order. **Do not delete anything not listed here.**

### Step 8.1 — `cmds.OpenErrorModal`

Delete the whole file:

```
git rm src/cmds/OpenErrorModal.go
```

Then remove its handler. File: `src/model/Update.go`

FIND:
```
	case cmds.OpenErrorModalMsg:
		finalCmds = append(finalCmds, m.reportForegroundError(msg.Message))

```

REPLACE WITH:
```
```

(That is, replace with nothing — delete those three lines including the blank one.)

### Step 8.2 — `chrome.HandleSpinnerTick`

File: `src/components/chrome/Spinner.go`

FIND:
```
// HandleSpinnerTick updates the spinner and returns the next tick command.
// Returns nil if no spinner is active.
func HandleSpinnerTick(spinnerModel spinner.Model, pendingAction *PendingAction, msg tea.Msg) (spinner.Model, tea.Cmd) {
	if pendingAction == nil {
		return spinnerModel, nil
	}

	var cmd tea.Cmd
	spinnerModel, cmd = spinnerModel.Update(msg)
	return spinnerModel, cmd
}
```

REPLACE WITH:
```
```

If `go build ./...` then reports an unused import in that file, remove exactly the
import it names, and nothing else.

It did: `tea "charm.land/bubbletea/v2"` was used only by this function. Removing it also
left a trailing blank line that `gofmt -l` flagged, so the file was run through
`gofmt -w`.

### Step 8.3 — `apptypes.ServiceListItem.StatusPill`

File: `src/apptypes/ServiceListItem.go`

FIND:
```
// StatusPill returns a styled pill showing the service's current status.
// It uses the same visual language as the group's statusPill in
// GroupDetailsPanel, but for a single service.
func (s ServiceListItem) StatusPill() string {
	var label string
	var bg color.Color

	switch s.Status {
	case "running":
		label, bg = "RUNNING", appstyles.Active.StatusRunning
	default:
		label, bg = "STOPPED", appstyles.Active.StatusStopped
	}
	fg := appstyles.InkOn(bg)

	return lipgloss.NewStyle().
		Background(bg).
		Foreground(fg).
		Bold(true).
		Padding(0, 1).
		Render(label)
}

```

REPLACE WITH:
```
```

This method is the **only** user of `image/color` in that file, so the import must go
too:

FIND:
```
import (
	"image/color"
	"strings"
```

REPLACE WITH:
```
import (
	"strings"
```

Do **not** remove the `appstyles` import — the `Description` method below still uses it.

### Step 8.4 — `envkeymodal.GetKey` and `GetValue`

File: `src/components/envkeymodal/Model.go`

FIND:
```
// GetKey returns the entered key.
func (m Model) GetKey() string {
	return m.keyInput.Value()
}

// GetValue returns the entered value.
func (m Model) GetValue() string {
	return m.valueInput.Value()
}

```

REPLACE WITH:
```
```

### Verification

Each of these must print nothing:

```
grep -rn "OpenErrorModal" src/ main.go
grep -rn "HandleSpinnerTick" src/
grep -rn "StatusPill" src/
grep -rn "GetKey()\|GetValue()" src/
```

Then the full check from Rule 8.

### Commit

```
git add -A && git commit -m "refactor: delete four unused exported symbols

cmds.OpenErrorModal was handled in Update but never constructed anywhere, tests
included. chrome.HandleSpinnerTick has no callers - the two panels that would
use it inline the same four lines. apptypes.ServiceListItem.StatusPill and
envkeymodal's GetKey/GetValue have none either; the env modal reads its inputs
directly instead."
```

---

# T9 — Remove the deselect concept

**Why.** There is no reason to have nothing selected. `configSyncCmds` already re-selects
by name after every reload and falls back to index 0 when the name is gone
(`src/model/Update.go:190-197`), so "something is always selected when the list is not
empty" is already the invariant everywhere — except one `esc` branch that exists purely
to break it.

That branch is a leftover. `docs/DESIGN.md` justifies `esc` as "back: out of the details
panel, to the list", but body-panel focus was removed and there is no details panel you
are "in" any more, so there is nothing to go back from. The Backups page already works
the way this task makes Home and Services work.

Removing it deletes a whole state — non-empty list with nothing selected — and collapses
`keys.Context.Selected` into `!ListEmpty`.

**The empty-list case stays.** An empty compose file still means nothing can be selected.
`detailspanel`'s "Select a service" card, the zero-broadcasts in `configSyncCmds`, and
`TestReloadOfAnEmptyProjectClearsTheSelection` are all still correct. Do not touch them.

### Step 9.1 — delete the `esc` deselect branch

File: `src/model/Update.go`

FIND:
```
			if !m.escKept() && !m.inlineEditing {
				if m.activePage == "Home" && m.selection.groupName != "" {
					m.selection.groupName = ""
					finalCmds = append(finalCmds, cmds.SetSelectedGroup(""))
				} else if m.activePage == "Services" && m.selection.serviceName != "" {
					m.selection.serviceName = ""
					finalCmds = append(finalCmds, cmds.SetSelectedService(types.ServiceConfig{}))
				}
			}
```

REPLACE WITH:
```
```

(Delete those lines entirely.)

If `go build ./...` then reports `types` imported and not used in that file, remove the
`"github.com/compose-spec/compose-go/v2/types"` import line and nothing else. If it does
not report that, leave the imports alone.

### Step 9.2 — retire `keys.Context.Selected`

File: `src/keys/Keys.go`

**Edit A — delete the field.**

FIND:
```
	// Selected reports whether the panel has a subject to act on - a chosen
	// group on Home, a chosen service on Services. Without one, the action
	// keys do nothing and are not offered.
	Selected bool
```

REPLACE WITH:
```
```

**Edit B — derive it from emptiness instead.**

FIND:
```
	case "Home", "Services":
		var bindings []key.Binding
		if ctx.Selected {
```

REPLACE WITH:
```
	case "Home", "Services":
		var bindings []key.Binding
		// A non-empty list always has a row under the cursor, and that row is
		// the selection - there is no way to deselect. So "is there a subject
		// to act on" and "does the list have rows" are the same question, and
		// asking it once is what stops two callers answering it differently.
		if !ctx.ListEmpty {
```

**Edit C — stop advertising `esc` where it does nothing.**

FIND:
```
		// Back is the esc ladder's second rung: deselect. It is only live
		// when something is selected and the filter has not already claimed
		// esc for itself.
		if ctx.Selected && ctx.Filter != list.FilterApplied {
			bindings = append(bindings, Global.Back)
		}

		return bindings
```

REPLACE WITH:
```
		// No Back rung here any more. esc used to deselect; with the selection
		// permanent there is nothing left for it to do on this page that the
		// footer is not already showing - an applied filter takes the slot
		// above as "esc clear filter", and dismissing an error banner works
		// but is unadvertised on every page, Files and Backups included. The
		// bar does not advertise inert keys.

		return bindings
```

### Step 9.3 — drop the footer's `selectedService` mirror

**This step assumes T3 has already been applied**, because the text it removes is the
text T3 wrote. If you skipped T3, STOP.

File: `src/components/keybindingbar/Update.go`

FIND:
```
	// The zero ServiceConfig is how the app says "nothing is selected" - esc on
	// the Services page sends exactly that. Setting the flag unconditionally
	// meant it could never go false, so the footer went on advertising the
	// action keys over an empty details panel, and every one of them was then
	// ignored by the panel that had nothing to act on.
	case cmds.SetSelectedServiceMsg:
		m.selectedService = msg.Name != ""

```

REPLACE WITH:
```
```

File: `src/components/keybindingbar/Model.go`

FIND:
```
	selectedService   bool
```

REPLACE WITH:
```
```

File: `src/components/keybindingbar/Update.go`

FIND:
```
	case cmds.SetServicesListMsg:
		m.servicesListEmpty = len(msg) == 0
		if m.servicesListEmpty {
			m.selectedService = false
		}
```

REPLACE WITH:
```
	case cmds.SetServicesListMsg:
		m.servicesListEmpty = len(msg) == 0
```

**Keep `selectedGroup`.** It is still needed — `bindingsFor` uses it for the
`ReadOnlyGroup` check against the ungrouped row. Only `selectedService` goes.

### Step 9.4 — stop passing `Selected`

File: `src/components/keybindingbar/View.go`

FIND:
```
	listEmpty := m.groupsListEmpty
	selected := m.selectedGroup != ""

	switch m.activePage {
	case "Services":
		listEmpty = m.servicesListEmpty
		selected = m.selectedService
	case "Backups":
```

REPLACE WITH:
```
	listEmpty := m.groupsListEmpty

	switch m.activePage {
	case "Services":
		listEmpty = m.servicesListEmpty
	case "Backups":
```

Then in the same file remove the two remaining references. FIND:
```
		// The version list has no "selected" beyond its cursor, so only
		// emptiness matters here - it is what decides whether / is offered.
		listEmpty = m.backupsListEmpty
		selected = false
	}
```

REPLACE WITH:
```
		// The version list has no "selected" beyond its cursor, so only
		// emptiness matters here - it is what decides whether / is offered.
		listEmpty = m.backupsListEmpty
	}
```

FIND:
```
		ListEmpty:             listEmpty,
		Selected:              selected,
```

REPLACE WITH:
```
		ListEmpty:             listEmpty,
```

File: `src/model/Update.go` — the help overlay's context builder.

FIND:
```
	case "Home":
		ctx.ListEmpty = len(m.listedGroupNames()) == 0
		ctx.Selected = m.selection.groupName != ""
		ctx.ReadOnlyGroup = m.selection.groupName == apptypes.UngroupedGroup
		ctx.UngroupedMaterialized = m.ungroupedMaterialized()
	case "Services":
		ctx.ListEmpty = m.config.configProject == nil || len(m.config.configProject.Services) == 0
		ctx.Selected = m.selection.serviceName != ""
```

REPLACE WITH:
```
	case "Home":
		ctx.ListEmpty = len(m.listedGroupNames()) == 0
		ctx.ReadOnlyGroup = m.selection.groupName == apptypes.UngroupedGroup
		ctx.UngroupedMaterialized = m.ungroupedMaterialized()
	case "Services":
		ctx.ListEmpty = m.config.configProject == nil || len(m.config.configProject.Services) == 0
```

### Step 9.5 — update the tests that encoded the old model

**This whole task was executed once end to end before being written down, so this list
is complete.** Nine test sites break. Fix all nine; do not add or skip any.

**A. `src/keys/Keys_test.go` — the `Selected` field is gone (7 sites).**

Delete the field from every `Context{…}` literal. Mechanically: remove every occurrence
of `Selected: true, ` and every occurrence of `, Selected: true`. Nothing else in the
literals changes. `ListEmpty` defaults to `false`, which now means "populated, so
something is selected" — the same thing `Selected: true` used to say.

**B. `src/keys/Keys_test.go` — two assertions asserted the abolished state.**

FIND:
```
		// Edit/Delete need a selection, which the list does not have here.
		for _, binding := range []key.Binding{List.Edit, List.Delete} {
			if entryIn(t, groups, binding).Available {
				t.Errorf("%q should be dimmed with no selection", binding.Help().Key)
			}
		}

		// The action keys need a selected subject.
		if entryIn(t, groups, Details.Start).Available {
			t.Error("s start should be dimmed with no selection")
		}
```

REPLACE WITH:
```
		// Edit/Delete act on the row under the cursor, and a populated list
		// always has one - there is no way to deselect. The dimmed case is an
		// empty list, covered in its own subtest below.
		for _, binding := range []key.Binding{List.Edit, List.Delete} {
			if !entryIn(t, groups, binding).Available {
				t.Errorf("%q should be available on a populated list", binding.Help().Key)
			}
		}

		// Same for the action keys: the subject is whatever the cursor is on.
		if !entryIn(t, groups, Details.Start).Available {
			t.Error("s start should be available on a populated list")
		}
```

**C. `src/keys/Keys_test.go` — `esc back` is no longer offered on a body page.**

FIND:
```
		global := scopeTitled(t, catalog, "Global")
		if !entryIn(t, global, Global.Back).Available {
			t.Error("esc back should be available with a selection")
		}
	})
```

REPLACE WITH:
```
		// esc is not offered on a body page any more: there is no deselect
		// rung for it, and an applied filter takes the slot as "esc clear
		// filter" when there is one.
		global := scopeTitled(t, catalog, "Global")
		if entryIn(t, global, Global.Back).Available {
			t.Error("esc back should be dimmed on a body page")
		}
	})
```

**D. `src/components/keybindingbar/Model_test.go` — the removed field.**

Replace `Model{activePage: "Services", selectedService: true, editing: true}` with
`Model{activePage: "Services", editing: true}`, and
`Model{activePage: "Services", selectedService: true}` with
`Model{activePage: "Services"}`.

**E. `src/components/keybindingbar/Model_test.go` — expectations that carried `esc back`.**

Remove the trailing `· esc back` from every `want:` string that ends
` · ↑/↓ navigate · esc back"`. Mechanically: replace ` · ↑/↓ navigate · esc back"` with
` · ↑/↓ navigate"`. **Leave the two that are not footer-body cases alone** — the inline
editor case (`ctrl+s save · … · esc back`) and the bare `"esc back"` case still offer it,
because the editor and the pending-action state still return it.

**F. `src/components/keybindingbar/Model_test.go` — three expectations that now show the
verbs.** A populated list with nothing selected was the old model; those literals now
mean "populated, therefore selected", so the verbs appear. Update exactly these three
`want:` strings:

- the *groups list with groups* case: `"n new · / filter · ↑/↓ navigate"` becomes
  `"s start · t stop · r restart · p pull · x remove · L logs · e edit · R rename · n new · d delete · / filter · ↑/↓ navigate"`
- the *groups list with a filter applied* case:
  `"n new · esc clear filter · ↑/↓ navigate"` becomes
  `"s start · t stop · r restart · p pull · x remove · L logs · e edit · R rename · n new · d delete · esc clear filter · ↑/↓ navigate"`
- the *services list with services* case: `"n new · / filter · ↑/↓ navigate"` becomes
  `"s start · t stop · r restart · p pull · x remove · L logs · H healthcheck · B boot · e edit · E open editor · y copy url · n new · d delete · / filter · ↑/↓ navigate"`

Do **not** touch the two `"n new · ↑/↓ navigate"` cases — those are empty lists, and an
empty list still offers only `n new`.

**G. `src/model/esc_test.go` — delete the test for the deleted behaviour.**

Delete the whole `TestEscOnASelectedGroupDeselectsIt` function including its doc comment.
Leave `TestEscClearsAnAppliedFilter` and `TestEscOnAnUnfilteredListDoesNothing` alone.
Run `gofmt -w src/model/esc_test.go` afterwards — the deletion leaves blank lines that
`gofmt -l` will otherwise flag.

**H. `src/model/error_banner_test.go` — the second half of the ladder test.**

FIND:
```
// Esc dismisses the banner before it deselects the current group - the same
// one-key-one-job ladder a filtered list clears on. The first esc clears the
// banner; the second esc deselects the group (there is no panel to return
// focus to anymore).
func TestEscDismissesTheBannerBeforeDeselecting(t *testing.T) {
```

REPLACE WITH:
```
// Esc dismisses the banner and leaves the selection alone. Dismissing is the
// last rung of the ladder now: there is no deselect step under it, because a
// populated list always has a row under the cursor and that row is the
// selection. A second esc therefore does nothing.
func TestEscDismissesTheBannerAndKeepsTheSelection(t *testing.T) {
```

Then, in the same function, FIND:
```
	// Second esc: no banner in the way, so esc deselects the group.
	m = updateForTest(t, m, keyPress(teaKeyEsc()))
	if m.selection.groupName != "" {
		t.Errorf("second esc did not deselect the group: %q", m.selection.groupName)
	}
}
```

REPLACE WITH:
```
	// Second esc: nothing left for it to do, and in particular it must not
	// clear the selection.
	m = updateForTest(t, m, keyPress(teaKeyEsc()))
	if m.selection.groupName != "core" {
		t.Errorf("second esc cleared the selection: %q", m.selection.groupName)
	}
}
```

### Step 9.6 — update the design record

File: `docs/DESIGN.md`

Find the paragraph beginning **`**`esc` is "back", as a ladder of claims.**`** and replace
the sentence describing deselection. The paragraph currently says esc "clears the active
filter first, then — if an error banner is showing — dismisses it, then clears the
current selection (deselecting the group or service so the details panel returns to its
empty state), and does nothing further once both are already clear."

Replace that sentence with:

```
clears the active filter first, then — if an error banner is showing — dismisses it, and
does nothing further. There is no deselect rung: a non-empty list always has a row under
the cursor and that row is the selection, so there is no state to return to. The footer
offers `esc` only as `esc clear filter`; dismissing a banner works but is not
advertised, the same as on Files and Backups.
```

Do not reword anything else in that paragraph.

**Deviation.** The sentence quoted above is preceded by the lead-in "With no panel left
to return focus to, what remains is the selection itself:", and the paragraph's next
sentence is the old footer claim ("The footer offers `esc back` in those contexts
only..."). Leaving either would have made the paragraph contradict itself, since the
replacement says there is no deselect rung and gives the footer's new behaviour. The
lead-in was trimmed to "With no panel left to return focus to," and the old footer
sentence was replaced by the one in the block, with its closing "the bar does not
advertise inert keys" kept.

### Verification

```
grep -rn "ctx.Selected\|Selected:" src/keys/ src/components/keybindingbar/ src/model/ | grep -v _test
```

Expected: **no output.**

```
grep -rn "selectedService" src/ | grep -v _test
```

Expected: **no output.**

Then the full check from Rule 8. The whole task was executed once before being written
down, and it ends green — build, vet, gofmt and the entire suite. If anything fails,
you have deviated from the steps; STOP and report rather than improvising.

Two existing tests must keep passing **unchanged**. They are the guard that the
invariant still holds, so do not edit them:
`TestReloadSelectsTheFirstEntryWithNoPriorSelection` and
`TestReloadFallsBackWhenTheSelectionIsGone` in `src/model/selection_test.go`.

### Manual check

Build and run the app, then confirm all four by eye:

1. On Home, press `esc` with a group highlighted — the group stays highlighted and the
   details panel keeps showing it.
2. The footer no longer shows `esc back` on Home or Services.
3. `/` then a term then `enter` still shows `esc clear filter`, and `esc` still clears it.
4. With an empty compose file, the Services page still shows its "Select a service" card
   and offers only `n new`.

### Commit

```
git add -A && git commit -m "refactor: remove the deselect concept

Nothing selected was a state only esc could produce. configSyncCmds already
re-selects by name after every reload and falls back to the first row when the
name is gone, so a non-empty list always had a selection everywhere except that
one branch.

The branch was a leftover from focusable body panels: esc was documented as
back out of the details panel, but there has been no panel to be in since focus
was removed, so it returned the user to nothing.

With it gone, keys.Context.Selected collapses into !ListEmpty - one fewer field
for the footer and the help overlay to derive separately, and one of the two
they were already disagreeing about. The footer stops offering esc back on Home
and Services, where it would now do nothing; an applied filter still shows esc
clear filter, and banner dismissal stays unadvertised exactly as it already is
on Files and Backups.

The empty-list case is untouched: an empty compose file still selects nothing,
and the panels still render their empty states for it."
```

---

# DO NOT DO THESE

The audit raised these. They look like findings. **They are not. Leave them alone.**

- **`src/components/placeholderpanel`** has no importers, but it is not dead code.
  `docs/DESIGN.md:288` says pages that are not implemented yet get a
  `placeholderpanel.New`. It is there on purpose. **Do not delete it.**

- **`envmodal.Model.termHeight`** is assigned and never read, so it looks dead. It is
  the input the modal needs to window its list, which is deferred work (see D3 below).
  Deleting it now means adding it back later. **Do not delete it.**

- **`utils.SecretHint`** has no production caller. Its own doc comment says that is
  deliberate. **Do not delete it.**

- **`detailspanel.EditorValue`, `detailspanel.EditorCursor`, `appstyles.Contrast`,
  `appstyles.HasBackgroundBleed`** are used only by tests. That is what they are for.
  **Do not delete them.**

- **The `src/cmds` one-file-per-message layout.** It is deliberate. Do not merge files.

- **`Update.go`'s length.** It is long but flat and idiomatic. Do not split it.

---

# Deferred

These are real findings from the same audit. They need either a decision or more
judgement than this plan can encode. **Do not start any of them without being told to.**
They are recorded here so they are not lost.

**D1 — "Is this service running" is implemented four times and two disagree.**
*(Done — see the status table. Decision taken by the owner: **any running container means
running**, one shared helper, all four sites call it. It does not claim cais manages
replicas, only that it does not lie about them.)* `apptypes.ServiceRunning` and
`apptypes.ContainerForService` are that helper.

`ContainerForService` prefers a *running* container rather than the first one found, so a
row's image, uptime and ports describe the live container instead of a stale one that
happens to come first in docker's output. It still falls back to the first: a stopped
service has details worth showing.

`serviceslist.containerStatus` keeps its `containersKnown` guard. "Not asked yet" is a
third state the shared helper deliberately does not model — reporting every service
stopped before docker has answered is a guess dressed up as a fact.

Original text follows.

**D1 — "Is this service running" is implemented four times and two disagree.** Two sites
return true if *any* container for the service is running
(`detailspanel/View.go:125`, `model/AppModel.go:226`); two return on the *first*
container found whatever its state (`groupdetailspanel/View.go:85`,
`serviceslist/Update.go:131`). With replicas, or with a stopped container left beside a
fresh one after a restart, the same screen contradicts itself. **This needs a product
decision** about whether cais supports replicas, and the answer needs a comment. Two
uncontroversial pieces sit next to it and could be shared once that is settled:
`containerForService` and `renderPendingAction` are byte-identical across the two panels.

**D2 — `keys.Context` has two builders.** *(Done — see the status table.)*
`AppModel.helpContext()` derives it from state; `keybindingbar` rebuilds it from mirrored
fields. T9 removes two of them (`Selected` from the context, `selectedService` from the
bar), so this is smaller than the audit found it, but the shape is unchanged. T3 fixes
the one place they currently disagree, but not the reason they *can*. The fix is to
broadcast the resolved `keys.Context` from `AppModel` and delete the mirror — about sixty
lines across two files, plus moving `keybindingbar`'s tests to drive through `Update`
rather than building `Model` literals.

How it went:

- `helpContext()` is now `keyContext()` — it was never only the overlay's.
- The bar's ten mirrored fields became one `keyContext keys.Context`. `composeFile`,
  `composeFileOthers` and `terminalWidth` stay: they are the footer's other job and no
  key depends on them.
- The context is resolved at the **end** of `AppModel.Update`, after the page's panels
  have seen the message, so the footer renders this frame's answer. The bar was
  previously a frame behind on anything that arrived as a follow-up broadcast.
- Net −135 lines.

**Two broadcasts lost their only reader.** `SetUngroupedMaterializedMsg` had one producer
and no consumer at all, so it went with the mirror. `SetListFilterStateMsg` is the open
one: three list components still produce it, nothing in production consumes it any more
(`keyContext()` reads filter state straight off the component through `filterStater`),
and its only remaining readers are three tests — one of which,
`backupslist.TestFilterStateIsBroadcast`, exists purely to assert the broadcast happens
and whose premise ("the bar never sees the list itself") is now false. **Deleting it
needs a decision**, because it is a live design question rather than a typo: is a
component announcing its filter state a seam worth keeping, or is the single
`filterStater` read the whole story? Left in place, not silently removed.

**T3's regression test moved.** `keybindingbar.TestEmptyingTheServicesListClearsTheActionKeys`
tested a bug the bar can no longer have, since it derives nothing. It is now
`model.TestEmptyingTheServicesListClearsTheFooterActionKeys`, asserting on the rendered
footer through the real path. Verified it fails when the hand-down is removed.

**D3 — `envmodal` hand-rolls a list.** It keeps its own cursor index, declares thirteen
key bindings inline instead of using `src/keys`, and renders every row with no
windowing, so a long `.env` overflows the modal. Converting it to a `bubbles/list` with
a custom delegate fixes all three and uses the `termHeight` it already stores. This is
the same conversion the backups list went through; see
`docs/plans/backups-rework.md` for the traps.

**D4 — Fifteen copies of the write-result tail in `Update.go`.** An
`afterWrite(err error) []tea.Cmd` helper collapses about 120 lines. Deferred because two
of the fifteen have drifted — `EditGroupMsg` omits `recomposeFilesCmdIfActive()` and
`CycleRestartPolicyMsg` omits `rebroadcastBodyLayoutIfChanged()` — and **someone must
decide whether those omissions are rules or oversights** before they can be folded in.

**D5 — Duplicated rendering.** The two body-list delegates are the same ~50 lines twice;
the two picker modals are the same modal twice; `renderPendingAction` and the status
pill are each duplicated. All are worth consolidating into `chrome`, all touch
pixel-exact output, and all need a capable model working against the frame-rendering
tests.

**D6 — Six hand-synchronised structures behind the group member table**
(`groupdetailspanel/View.go`), five keyed by bare strings, where a typo compiles into a
silently missing column. *(Done — see the status table.)*

`type column int` with constants, and `tableCols` becomes `[numColumns]int`. The two
20-line `get`/`set` string switches disappear entirely — they existed only to map a
string back to a struct field, which is what an index already is. Net −76 lines from the
file.

`column.String()` was added so failures still name "ports" rather than 6; those strings
were the whole point of the old keys and the tests format them.

One thing the conversion surfaced: `widestShrinkable` returned `""` for "nothing can
shrink". As an index that sentinel collides with `colDot`, which is 0, so it returns
`(column, bool)` now.

**D7 — Two families of compose-file writer.** `ServiceFragment.go` validates by
reloading through compose-go before writing; `GroupTags.go` does not, on five paths.
Routing both through one function costs a reload (~50-200ms) on those paths — a
deliberate trade someone should make explicitly.

**D8 — No seam for `docker`.** *(Done — see the status table.)* `utils` calls
`exec.Command("docker", …)` at nine sites, so about 21 tests fail without Docker
installed and `src/cmds` sits at 4.9% coverage. The fix is a swappable
`var dockerCommand = exec.Command`. The prerequisite should be documented in the
contributor docs immediately regardless, since that costs nothing.

The seam landed as `dockerCommand` / `dockerCommandContext` in `src/utils`. But the seam
alone fixes nothing at the model level, which is where the failures were: it is
unexported, and the 19 failing tests are in `src/model`. Measured them by running the
suite against a `PATH` with no docker on it — 19, not 21, all in `src/model`.

**What fixed them was a `TestMain` stub, not the seam.** It writes a shell script named
`docker` to a temp dir and prepends it to `PATH` for that package. It answers `ps` with
`[]` and everything else with success.

**The stub then failed three tests that had been passing for the wrong reason.**
`TestEditGroupFailureShowsError`, `TestRenameFailureShowsErrorAndKeepsSelection` and
`TestAdoptUngroupedFailureShowsError` all asserted `m.lastError != ""` after sending a
message carrying an error. That was already true before their action: the failed docker
poll had put an error in the banner. They would have passed whatever the handler did.
They now assert an `errormodal.Model` — the same correction T4 made, and it is worth
noting the plan produced this class of mistake twice.

So the suite is now hermetic with respect to docker, and 25% faster in `src/model`
(23.7s vs 31.9s) because the stub returns instantly. Rule 9 above is struck out.

Not done: `src/cmds` coverage. The seam makes it reachable; raising it is separate work.

**D9 — Modals never re-read the terminal size.** *(Done — see the status table.)* Six
capture `termHeight` at construction and ignore the `WindowSizeMsg` they already receive,
so shrinking the terminal with a picker open pushes its bottom border off screen.

Reproduced first: a theme picker built for a 60-row terminal renders 23 rows tall and
stays 23 rows tall when the terminal becomes 20 rows. That is the regression test now.

Five modals fixed — `themepickermodal`, `composefilepickermodal`,
`healthcheckpickermodal` (list height, via a new `chrome.ResizeModalList`) and
`errormodal`, `dockerstatusmodal` (body width, via a new `chrome.ModalBodyWidth` that
replaces the copy of that `min(60, w/2)` clamp each of them carried).

The other two size-carrying modals are deliberately untouched:

- `envmodal` stores `termHeight` and windows nothing with it — that is D3's whole point.
  A resize handler there would have nothing to resize.
- `groupnamemodal` carries `termHeight` for its second step and does not use it itself.

Both would need the resize handler the day their lists get windowed; neither has a
list today.

**D10 — Page identity is a bare string** switched on at nine sites across four packages,
each falling through silently. A `type Page string` with constants makes a typo a
compile error. *(Done — see the status table.)*

The real count was 46 literal sites across six packages, not nine across four.
`apptypes.BackupsFocus` was the precedent: same problem, same place, already solved that
way.

**What the type actually buys, precisely.** A typo'd *constant name*
(`apptypes.PageHom`) is a compile error. A typo'd *string literal*
(`m.activePage == "Hom"`) is **not** — an untyped constant still converts implicitly,
which is what keeps every `Page: "Home"` in the tests compiling unchanged. So the claim
"makes a typo a compile error" is half true, and the half that works is the half that
matters: the constants are now the only spelling anyone writes, a bare literal in
production stands out to a grep (there are none left outside `Pages.go`), and passing a
`string` where a `Page` belongs — the actual mechanism behind the drift — is now
rejected. That last one caught real call sites during the conversion.

`backupslist` and `serviceslist` set `list.Title` to `"Backups"` and `"Services"`. Those
stay strings: a list title is text on a header, not an identity.

**D11 — Documentation drift.** *(Done — see the status table.)* `docs/DESIGN.md` names
several identifiers that moved when `src/components` was split into one package per
model. The contributor docs misstate the theme count (13 vs 14) and the config field
count. `README.md` links a troubleshooting page that does not exist. `docs/ROADMAP.md`
describes a `--no-ff` merge workflow that contradicts `CLAUDE.md`'s one-commit-on-main
rule.

What was actually found, on checking each claim:

- **`DESIGN.md` identifiers — confirmed, six sites.** `components.dockerActionFor` (x2)
  and `components.renderKeyHints` are `chrome.DockerActionFor` and
  `chrome.RenderKeyHints`; the three cited tests are in `groupslist` and `keybindingbar`,
  not `components`; `utils.DockerLogs` is `utils.StreamDockerLogs`.
  `constants.FocusableComponents` and `keys.Live` also fail a grep but are *correct* -
  both sentences say the thing is gone, which is why it is gone. Left alone.
- **Theme count — confirmed, two sites.** `appstyles.Themes` holds 14 (3 cais + 11
  community). `theme-system.md`'s frontmatter and `overview.md`'s link text said 13;
  `theme-system.md`'s own body already said 14, so the page contradicted itself.
- **Config field count — confirmed, two sites.** `config.Config` has `Theme` and
  `URLHost`; `architecture.md` and `project-structure.md` both still said "One field
  today (`theme`)". The user-facing `configuration.md` table was already right.
- **`README.md` troubleshooting link — not a defect.** It points at
  `website/src/content/docs/users/troubleshooting.md`, which exists. Nothing changed.
- **`ROADMAP.md` workflow — confirmed.** Its *Conventions* section described feature
  branches merged `--no-ff`, and its check omitted `gofmt -l src/`. Rewritten to the
  land-on-main rule, noting where the practice changed (`56646b4`, which the same file
  already records as having landed straight on main) and keeping the follow-up `docs:`
  commit that pins the hash, since a commit still cannot contain its own hash.

---

# When you are done

Report, for each task T1-T8: the commit hash, and whether verification passed.

Then run once more and paste the output:

```
go build ./... && go vet ./... && gofmt -l src/ && go test ./... 2>&1 | tail -20
```

List anything you noticed but did not touch, per Rule 5.
