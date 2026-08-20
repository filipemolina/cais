---
title: Testing
description: The testing philosophy — unit over e2e, rendering as a string, the in-process rig, and VHS.
---

# Testing

Most behaviour is testable without a terminal. The philosophy is **unit over e2e**: the closer a test is to the code it exercises, the faster and more reliable it is.

## The three levels

### 1. Components take messages and hand back a model

Components are nested Bubble Tea models: they take a message and return an updated model. Drive one directly and assert on the result — no terminal, no timing.

```go
// src/components/serviceslist/Model_test.go
model := serviceslist.New(...)
updated, _ := model.Update(msg)
// assert on updated
```

### 2. Rendering is a string

`ansi.Strip(m.View().Content)` gives you the plain text of a component, which is enough to catch layout and styling mistakes.

```go
// src/components/mainmenu/Model_test.go
plain := ansi.Strip(model.View().Content)
// assert on the plain text
```

### 3. Whole flows go through the e2e rig

`src/model/rig_test.go` runs a real `tea.Program` against an in-memory buffer. This is the only way to test a full flow end to end, but it is timing-based, so its assertions have to wait for output (`r.WaitFor`) rather than sleep and hope.

**Prefer the first two.** The rig is for flows that only make sense end to end.

## What the tests pin down

The test suite is the standing guard for the design decisions in `docs/DESIGN.md`:

- `components.TestFooterHints` pins every footer context — the bar cannot drift from `keys.Active`.
- `components.TestDeleteKeyDoesNotAlsoPageTheList` and `TestPanelLettersDoNotPageTheList` fail against the default bubbles list keymap.
- `TestDetailsPanelsPinPendingActionToBottom` pins both details panels to the same footer line.
- `TestNarrowPanelsStayInsideTheirBox`, `TestFooterNeverWraps`, and `TestMemberTableHeadingsNeverCollide` guard the narrow-terminal shedding rules.
- `src/model/background_test.go` applies `appstyles.HasBackgroundBleed` to fully rendered frames across both pages and their empty, populated, narrow, and error-banner states — once per registered theme, via a `forEachTheme` helper.
- `src/appstyles/Contrast_test.go` verifies `InkOn(fill)` clears 4.2:1 on every status pill, the accent chip, and the error banner for every registered theme.
- `src/appstyles/Theme_test.go` checks theme fields one level down, on the fields themselves rather than a rendered frame.

## Running the tests

```bash
go test ./...
go test -race ./...   # CI runs this; run it locally too
```

Run the race detector locally if you touch anything that shells out or streams: the docker calls and the log stream each run on their own goroutine.

## VHS for what only shows on screen

Anything that shows up only on screen is worth checking in the real app with [VHS](https://github.com/charmbracelet/vhs) before it is committed: write a tape with `Screenshot "name.png"` and run it from a scratch directory. The `demo/` directory holds the tapes, the recorded gif and screenshots, and their fixture stack.

VHS wants its paths quoted, and sometimes drops the last screenshot — re-run if the file is missing.