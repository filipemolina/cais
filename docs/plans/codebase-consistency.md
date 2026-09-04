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
| T1 | Move the ANSI decoder out of the shipped binary | | todo |
| T2 | Stop the list tests blocking on the cursor-blink timer | | todo |
| T3 | Fix the footer advertising dead keys | | todo |
| T4 | Two error banners that forget the layout row | | todo |
| T5 | Sort Networks and Depends-on before rendering | | todo |
| T6 | Make `GetConfigMsg` a defined type | | todo |
| T7 | Tidy `go.mod` and add the CI gate | | todo |
| T8 | Delete four dead symbols | | todo |

Tasks T9 and beyond are in *Deferred* and **must not be started** without being told to.

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

9. **Docker must be installed and on `PATH`** or roughly 21 tests in `src/model` fail
   for reasons unrelated to your change. Check with `docker --version` before you start.
   If it is missing, STOP and report that.

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
unconditionally, so it never becomes false when a service is deselected. Deselection is
sent as a *zero* `ServiceConfig`, not as an absent message. Press `esc` on the Services
page and the footer keeps offering ten action keys over an empty panel; every one of
them is then ignored by the details panel.

The group case one line above is already correct — it stores the string, so `""` clears
it. This makes the service case behave the same way.

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

// Deselecting a service has to take its action keys off the footer. The app
// says "nothing is selected" by sending a zero ServiceConfig rather than by
// sending nothing, so a handler that treats any SetSelectedServiceMsg as a
// selection leaves the bar advertising keys the details panel will ignore.
func TestDeselectingAServiceClearsTheActionKeys(t *testing.T) {
	m := New().(Model)

	m = driveBar(t, m,
		cmds.SetActivePageMsg("Services"),
		cmds.SetServicesListMsg([]types.ServiceConfig{{Name: "web"}}),
		cmds.SetSelectedServiceMsg(types.ServiceConfig{Name: "web"}),
	)

	if !strings.Contains(hintsOf(t, m), "start") {
		t.Fatal("precondition: a selected service should put the action keys on the bar")
	}

	m = driveBar(t, m, cmds.SetSelectedServiceMsg(types.ServiceConfig{}))

	if hints := hintsOf(t, m); strings.Contains(hints, "start") {
		t.Errorf("the footer still offers the action keys after deselection: %q", hints)
	}
}
```

This file has been compiled and run against both the broken and the fixed code. With
the bug present it fails with a 16-hint list; with T3 Step 3.1 applied it passes.

### Verification

```
go test ./src/components/keybindingbar/ -run TestDeselectingAServiceClearsTheActionKeys -count=1 -v
```

Expected: `PASS`.

**Then prove the test actually catches the bug.** Temporarily change the line you
edited in Step 3.1 back to `m.selectedService = true`, re-run the command above, and
confirm it now **FAILS**. Then put your fix back and confirm it passes again. If the
test passes with `= true`, STOP and report it — the test is not testing anything.

Then the full check from Rule 8.

### Commit

```
git add -A && git commit -m "fix: clear the footer's action keys when a service is deselected

The bar set selectedService unconditionally on SetSelectedServiceMsg, so the
flag could never go false. Deselection is broadcast as a zero ServiceConfig -
esc on the Services page sends one - so after esc the footer went on offering
s t r p x L H B e E y over a details panel showing its empty state, and every
one of those keys was then dropped by the panel.

The group case beside it was already right: it stores the string, so an empty
one clears. This makes the service case match.

The existing tests missed it because they build Model literals with
selectedService already set, rather than driving the message in. The new test
drives it."
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
fifteen sites do. No existing test asserts on this message, so nothing should break. If
a test does fail, STOP and report it.

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

**D1 — "Is this service running" is implemented four times and two disagree.** Two sites
return true if *any* container for the service is running
(`detailspanel/View.go:125`, `model/AppModel.go:226`); two return on the *first*
container found whatever its state (`groupdetailspanel/View.go:85`,
`serviceslist/Update.go:131`). With replicas, or with a stopped container left beside a
fresh one after a restart, the same screen contradicts itself. **This needs a product
decision** about whether cais supports replicas, and the answer needs a comment. Two
uncontroversial pieces sit next to it and could be shared once that is settled:
`containerForService` and `renderPendingAction` are byte-identical across the two panels.

**D2 — `keys.Context` has two builders.** `AppModel.helpContext()` derives it from
state; `keybindingbar` rebuilds it from thirteen mirrored fields. T3 fixes the one place
they currently disagree, but not the reason they *can*. The fix is to broadcast the
resolved `keys.Context` from `AppModel` and delete the mirror — about sixty lines across
two files, plus moving `keybindingbar`'s tests to drive through `Update` rather than
building `Model` literals. Needs a capable model.

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
silently missing column.

**D7 — Two families of compose-file writer.** `ServiceFragment.go` validates by
reloading through compose-go before writing; `GroupTags.go` does not, on five paths.
Routing both through one function costs a reload (~50-200ms) on those paths — a
deliberate trade someone should make explicitly.

**D8 — No seam for `docker`.** `utils` calls `exec.Command("docker", …)` at nine sites,
so about 21 tests fail without Docker installed and `src/cmds` sits at 4.9% coverage.
The fix is a swappable `var dockerCommand = exec.Command`. The prerequisite should be
documented in the contributor docs immediately regardless, since that costs nothing.

**D9 — Modals never re-read the terminal size.** Six capture `termHeight` at
construction and ignore the `WindowSizeMsg` they already receive, so shrinking the
terminal with a picker open pushes its bottom border off screen.

**D10 — Page identity is a bare string** switched on at nine sites across four packages,
each falling through silently. A `type Page string` with constants makes a typo a
compile error.

**D11 — Documentation drift.** `docs/DESIGN.md` names several identifiers that moved
when `src/components` was split into one package per model. The contributor docs
misstate the theme count (13 vs 14) and the config field count. `README.md` links a
troubleshooting page that does not exist. `docs/ROADMAP.md` describes a `--no-ff` merge
workflow that contradicts `CLAUDE.md`'s one-commit-on-main rule.

---

# When you are done

Report, for each task T1-T8: the commit hash, and whether verification passed.

Then run once more and paste the output:

```
go build ./... && go vet ./... && gofmt -l src/ && go test ./... 2>&1 | tail -20
```

List anything you noticed but did not touch, per Rule 5.
