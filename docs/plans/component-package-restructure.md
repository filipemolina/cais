# Plan: One Folder per Model — Restructure `src/components`

> **Before you start.** Work on a feature branch of small commits, merged
> `--no-ff`; `go build ./... && go vet ./... && go test ./... && gofmt -l .`
> green at **every** commit, not just at the tip — `docs/ROADMAP.md`
> §Conventions is the full contract and `CONTRIBUTING.md` explains how a TUI
> gets tested. Behaviour that only shows on screen gets checked in the real app
> with VHS before it is committed. **Step 1 of the post-alpha order** (`docs/ROADMAP.md` §The order after the alpha) — nothing else should be in flight in `src/components` while this runs.

Ask: *"restructuring the project's sub models so they have the same file
structure as the main model (model name as a folder, then a View, Model and
Update files for each model)."*

Measured against the tree on 2026-07-31.

## Status — what the shape is now, and what it would become

The main model already has the target shape:

```
src/model/AppModel.go   Init.go   Update.go   View.go   (+ 23 test files)
```

The sub-models do not. `src/components` is **one flat Go package**: 25
non-test files, 14 test files, 7,835 lines, holding **20 Bubble Tea models**
and 5 files of shared helpers.

| Kind | Files |
|---|---|
| Models (have `Update` + `View`) | `AboutModal`, `Button`, `ComposeFilePanel`, `ComposeFilePickerModal`, `ConfirmModal`, `ContainersList`, `CreateComposeFileModal`, `DetailsPanel`, `ErrorModal`, `GroupDetailsPanel`, `GroupNameModal`, `GroupsList`, `HelpOverlay`, `KeybindingBar`, `LogsModal`, `MainMenu`, `PlaceholderPanel`, `ServiceChecklistModal`, `ServicesList`, `ThemePickerModal` |
| Shared, no model | `PanelFrame.go`, `styles.go`, `spinner.go`, `BasicInfo.go`, `yamlindent.go` |

Target:

```
src/components/
  chrome/            Model-less shared rendering (was PanelFrame/styles/spinner/…)
  serviceslist/      Model.go  Update.go  View.go
  detailspanel/      Model.go  Update.go  View.go
  …one folder per model…
```

**Verdict up front: worth doing, and worth doing *before* the four feature
plans in `docs/plans/`, not after.** The restructure is mechanical, it is
invisible to users, and its cost is dominated by one thing — how much code
exists when it happens. Every pending plan adds components
(`EnvPanel`, `HealthcheckPickerModal`, `AddServiceModal`, the AI step). Moving
25 files now is cheaper than moving 29 later, and it means those four plans get
written into the new shape instead of being ported into it.

The honest counter-argument is in §Effort / gain, and it is real: this is a
large diff that changes no behaviour, and it will make `git blame` on
`DetailsPanel.go` two hops longer for the next year.

## What actually blocks it (measured, not guessed)

A flat package shares unexported identifiers for free. Splitting it means every
shared identifier must become exported and live somewhere both sides can import.
An analysis of cross-file references in `src/components` (every top-level
declaration, matched against every other file) found **~30 such identifiers**,
in five clusters:

| Source file | Shared identifiers | Used by |
|---|---|---|
| `PanelFrame.go` | `modalSurface`, `modalTitle`, `modalHints`, `modalListHeight`, `modalListChrome`, `renderPanelFrame`, `renderEmptyCard`, `panelBodyWidth`, `panelBodyHeight`, `panelBodyWithActions`, `panelRule`, `renderActionButtons`, `dockerActionFor`, `actionButton`, `actionButtons`, `joinActionButtons`, `lowestDrop`, `buttonLabel` | 10 files |
| `styles.go` | `panelBg`, `fitBox`, `wrapperStyle`, `listWrapperStyle`, `listRowBg`, `barColumn` | 7 files |
| `KeybindingBar.go` | `KeyHint`, `hintAs`, `hintFor`, `renderKeyHints` | 10 files |
| `spinner.go` | `PendingAction`, `newSpinner`, `actionDescription`, `actionLabel`, `kindLabel`, `handleSpinnerTick` | 3 files |
| `GroupDetailsPanel.go` | `healthColor`, `truncate` | `DetailsPanel.go` |

Two more things the same analysis settles:

- **There are no import cycles waiting.** The only model-to-model construction
  is `GroupNameModal` returning a `ServiceChecklistModal` from its `Update`
  (`GroupNameModal.go:72`) — one direction. The apparent `DetailsPanel` ↔
  `GroupDetailsPanel` coupling is comments in both directions plus two real
  helpers (`healthColor`, `truncate`) that belong in `chrome` anyway.
- **The dependency direction outside the package is already clean.** Only
  `src/model` imports `src/components`; `src/components` imports `cmds`,
  `appstyles`, `keys`, `apptypes`, `constants`, and once each `utils` and
  `highlight`. Nothing downstream imports components, so a `chrome` package
  cannot create a cycle no matter who imports it.

So the whole restructure is: **extract `chrome`, then move models into folders
one at a time.** No redesign, no behaviour change, no new abstraction.

## Design decisions

### D1. Package name = the model's name, lowercase, no underscores

Go package names are lowercase with no separators, so `ServicesList` →
`serviceslist`, `GroupDetailsPanel` → `groupdetailspanel`. Ugly at eight
syllables, and it is the convention (`net/http`, `httptest`, `strconv`); an
underscore or camelCase package name would be the first non-idiomatic thing in
the tree.

The directory name matches the package name exactly. No `_test`-only packages:
tests stay in the same package as their code (they test unexported behaviour
today, and that is fine).

### D2. Files inside a package: `Model.go`, `Update.go`, `View.go`

Mirroring `src/model`, one concern per file:

- **`Model.go`** — the struct, its constructor, and the small accessors
  (`OwnsKeyboard`, `KeepsEsc`, `FilterState`, `path`, …).
- **`Update.go`** — `Update` and the message handlers it delegates to
  (`updateFilename`, `updateServiceFields`, …).
- **`View.go`** — `View` and its render helpers.
- **`Init.go`** — only where `Init` does something. Most return `nil`; those
  stay in `Model.go`. Fourteen files whose entire content is
  `func (m X) Init() tea.Cmd { return nil }` is worse than the inconsistency.

Where a model is small (`ConfirmModal`, `ErrorModal`, `PlaceholderPanel`,
`Button` — all under 120 lines), splitting into three files is worse than
leaving one `Model.go`. **Rule: split when the file is over ~150 lines or when
`View` has its own helpers; otherwise one `Model.go` per package.** The folder
per model is the ask; three files in a 60-line package is cargo cult.

### D3. The constructor becomes `New`

`components.ServicesList(...)` reads well when the package is `components`.
`serviceslist.ServicesList(...)` stutters. Go's convention for a package
producing one type is `New`:

```go
// before
components.ServicesList(services, width, height)
components.CreateComposeFileModal(dir)

// after
serviceslist.New(services, width, height)
createcomposefilemodal.New(dir)
```

Long package names make the call sites long. That is acceptable and it is where
import aliases earn their keep in `src/model` (`ccfmodal "…/createcomposefilemodal"`)
— but alias sparingly and consistently, because an alias per call site is how a
codebase becomes unsearchable.

The exported model type keeps its name (`serviceslist.ServicesListModel`) or
shortens to `serviceslist.Model` — **recommend `Model`**, matching
`appstyles.Active` / `config.Config` house style and the file it lives in.

### D4. `chrome` is the shared package, and it is exported

`src/components/chrome`, package `chrome`, holding the five clusters above with
exported names:

| Was | Becomes |
|---|---|
| `modalSurface`, `modalTitle`, `modalHints`, `modalListHeight` | `chrome.ModalSurface`, `ModalTitle`, `ModalHints`, `ModalListHeight` |
| `renderPanelFrame`, `renderEmptyCard`, `panelBodyWidth/Height`, `panelRule`, `panelBodyWithActions` | `chrome.PanelFrame`, `EmptyCard`, `PanelBodyWidth/Height`, `PanelRule`, `PanelBodyWithActions` |
| `renderActionButtons`, `dockerActionFor` | `chrome.ActionButtons`, `DockerActionFor` |
| `panelBg`, `fitBox`, `wrapperStyle`, `listWrapperStyle`, `listRowBg`, `barColumn` | `chrome.PanelBg`, `FitBox`, `WrapperStyle`, `ListWrapperStyle`, `ListRowBg`, `BarColumn` |
| `KeyHint`, `hintAs`, `hintFor`, `renderKeyHints` | `chrome.KeyHint`, `HintAs`, `HintFor`, `RenderKeyHints` |
| `PendingAction`, `newSpinner`, `actionDescription`, `handleSpinnerTick` | `chrome.PendingAction`, `NewSpinner`, `ActionDescription`, `HandleSpinnerTick` |
| `healthColor`, `truncate` (from `GroupDetailsPanel.go`) | `chrome.HealthColor`, `Truncate` |
| `yamlIndent`, `indentAfter` | `chrome.YAMLIndent`, `IndentAfter` (or a `yamledit` package — see below) |

Two judgement calls in that table:

- **`Button` is a model, not chrome.** It has `Update`/`View`. It is only used
  by `PanelFrame`, so it can live in `chrome` as an internal type — simpler
  than a `button` package imported by exactly one caller. Put it in
  `chrome/Button.go` and do not export a constructor beyond what `chrome` needs.
- **`yamlindent.go` is not chrome** — it is editor logic used only by
  `DetailsPanel`. Move it into `detailspanel/` and leave it unexported. Do not
  put things in `chrome` because they were in the same package once; put them
  there because two packages need them.

`BasicInfo.go` is a renderer with one caller; it moves to whichever package
renders it and stays unexported.

### D5. `src/model` keeps its filenames

`AppModel.go` holds five structs (`navigationModel`, `configModel`,
`containersModel`, `selectionModel`, `Components`, `AppModel`), so renaming it
`Model.go` is defensible symmetry and a pointless churn of the file most likely
to conflict with in-flight work. **Recommend leaving it.** If symmetry wins,
it is a one-commit `git mv` at the very end, after everything else is green.

## Phases

Every phase is a feature branch of small commits merged `--no-ff`
(`docs/ROADMAP.md` §Conventions), and **every commit builds**: `go build ./...
&& go vet ./... && go test ./... && gofmt -l .` (the last must print nothing).
A commit that does not build is a commit nobody can bisect through
(`CONTRIBUTING.md`).

### The move procedure (follow this exactly, per component)

For a component `X` in `src/components/X.go` (+ `X_test.go`):

```bash
cd /home/filipe/Documents/projects/tui
mkdir -p src/components/x
git mv src/components/X.go      src/components/x/Model.go
git mv src/components/X_test.go src/components/x/Model_test.go   # if it exists
```

1. Change the package clause in both files to `package x`.
2. Split `Model.go` into `Update.go` / `View.go` per D2 (a second commit, so
   the move and the split are separately reviewable and separately revertable).
3. Rename the constructor to `New` (D3) and the model type to `Model`.
4. `go build ./...` — the compiler now lists every unresolved identifier. Each
   one is either (a) a `chrome` helper, so prefix it `chrome.`; or (b) another
   component, so import that package.
5. Update `src/model` call sites the compiler points at.
6. `gofmt -w` the moved files; `go vet ./...`; `go test ./...`.
7. Commit: `Move X into its own package`.

Rules while doing it:

- **No behaviour changes in a move commit.** No renamed variables, no
  "while I'm here" fixes, no comment rewrites. If something is wrong, note it
  and fix it in a separate commit before or after — a review of a 400-line move
  cannot also be a review of a logic change.
- **Never move two components in one commit.**
- Prefer `git mv` over delete+create so `git log --follow` and `git blame -C`
  can track the file.

### Phase 1 — Extract `chrome`

The only phase with any thinking in it. `PanelFrame.go`, `styles.go`,
`spinner.go`, the four `KeybindingBar` hint helpers, and `healthColor` /
`truncate` from `GroupDetailsPanel.go` move to `src/components/chrome`, are
exported per D4, and every caller in `src/components` gains a `chrome.` prefix.
`src/components` stays one package otherwise.

The two test files that exercise these helpers directly —
`action_buttons_test.go` and `modal_chrome_test.go` — move to `chrome/` with
them. `KeybindingBar_test.go` stays with `KeybindingBar` (it tests the model);
move only what tests the helpers.

Acceptance: `src/components/chrome` exists; `src/components` compiles against
it; the full test suite passes unchanged; **no test assertion text changed** —
if a rendering test needed editing, something moved that should not have.

Suggested commits: one per cluster (`chrome: panel frame`, `chrome: styles`,
`chrome: key hints`, `chrome: spinner`, `chrome: health and truncate`), each
green.

### Phase 2 — The leaves (7 packages)

Components with no other component dependency and small surfaces, easiest
first, to prove the procedure before it meets `DetailsPanel`:

`placeholderpanel`, `errormodal`, `confirmmodal`, `aboutmodal`, `mainmenu`,
`helpoverlay`, `containerslist`.

### Phase 3 — The lists and panels (5 packages)

`groupslist`, `serviceslist`, `composefilepanel`, `keybindingbar`,
`logsmodal`.

`keybindingbar` is the one to watch: it is the footer, it renders from
`keys.Active`, and its test file is 256 lines of layout assertions. Move it
alone, in its own commit, and read the diff.

### Phase 4 — The modals with handoffs (4 packages)

`groupnamemodal`, `servicechecklistmodal`, `composefilepickermodal`,
`createcomposefilemodal`, `themepickermodal`.

Order matters here: move `servicechecklistmodal` **before** `groupnamemodal`,
because `GroupNameModal.Update` constructs a `ServiceChecklistModal`
(`GroupNameModal.go:72`) and the import has to point at something that already
exists.

### Phase 5 — The two big panels (2 packages)

`detailspanel` (1,048 lines + `yamlindent.go` + `BasicInfo.go` + 2 test files)
and `groupdetailspanel` (548 lines + its stats tests). Last because they are
the largest, they use the most `chrome`, and by now the procedure is boring.

`detailspanel` is the one component where D2's split genuinely pays: `View`
and its table renderers are most of the file.

### Phase 6 — Sweep and document

- `grep -rn "src/components\"" --include=*.go .` returns nothing (the flat
  package is gone).
- `docs/DESIGN.md` gains a short *Package layout* section: one folder per
  model, `Model`/`Update`/`View`, `chrome` for what two packages need, and the
  rule that a helper earns its way into `chrome` by having a second caller.
- `CONTRIBUTING.md` §Testing points at the new paths (it names
  `src/components/ServicesList_test.go` and `src/components/MainMenu_test.go`
  today — both move).
- `README.md` if it names any path.
- Optional: `src/model/AppModel.go` → `Model.go` (D5).

## Verification, beyond the test suite

The suite is good here — component tests drive models directly and assert on
`ansi.Strip(m.View().Content)`, so a rendering regression fails loudly. Two
extra checks, because a restructure is exactly where a silent change hides:

1. **Binary-level sanity:** `make build` and run the app against
   `mocks/compose.yaml` before Phase 1 and after Phase 6; the screens should be
   indistinguishable. A VHS screenshot before and after is the cheap version
   (house convention already has VHS in the loop).
2. **Diff hygiene per commit:** `git show --stat` on a move commit should show
   renames (`R100`) and import lines, nothing else. Any commit with a large
   `+`/`-` on a file that was supposed to move is a commit to re-read.

## Effort / gain — decided: option 2

**Do the full restructure, Phases 1–6.** The table is the reasoning, not a
menu; option 3 in it is explicitly a different plan and must not be smuggled
into this one.

| Option | Effort | Gain | Verdict |
|---|---|---|---|
| 0 — leave it flat | 0 | zero risk; the package keeps growing and every helper stays implicitly global to 25 files | defensible, and it gets worse per feature |
| 1 — `chrome` only (Phase 1) | ~0.5 day | fixes the actual problem: shared rendering is now a named thing with an API, not ambient state | **the minimum worth doing** |
| **2 — full restructure (Phases 1–6)** | **~2–3 days** | the asked-for shape; each model gets a namespace; new components have an obvious home; the compiler starts enforcing what is shared | **← build this**, and before the feature plans |
| 3 — 2 + `src/model` split by page | +2 days | `Update.go` is 972 lines and growing | separate plan; do not smuggle it in here |

**What this does not buy:** no user-visible change, no performance change, no
new capability. It buys navigability and a real boundary around shared
rendering, and it removes the "where do I put this?" question from four
pending feature plans.

**What it costs beyond the time:** `git blame` gets one indirection deeper
(mitigated by `git mv` and `-C`), and any in-flight branch touching
`src/components` will conflict — which is the strongest argument for doing it
now, while nothing is in flight, rather than after `env-secrets`,
`healthcheck-insertion`, `image-search` and `ai-service-authoring` have each
added a component.

## Risks

1. **A move commit that also changes behaviour.** The mitigation is the
   procedure and the diff-stat check, not care.
2. **Export creep.** Every helper that moves to `chrome` becomes public API for
   the whole tree. Move only what has a second caller (D4); leave the rest
   unexported inside its model's package. Re-run the coupling analysis after
   Phase 5 and demote anything in `chrome` that ended up with one caller.
3. **Import alias soup in `src/model`.** Twenty imports with long names.
   Decide the aliasing convention once, at Phase 2, and apply it identically —
   or accept the long names, which is the simpler choice.
4. **Test files that test two things.** `modal_chrome_test.go` and
   `action_buttons_test.go` are chrome tests; `DetailsPanel_test.go` may assert
   on chrome output through the panel. That is fine — leave it — but do not
   split a test file mid-restructure.
5. **Stale docs.** `CONTRIBUTING.md` and `docs/DESIGN.md` both name specific
   files. Phase 6 exists for this; do not skip it because the code is green.

## Do not

- Do not change behaviour in a move commit.
- Do not move more than one component per commit.
- Do not create `Init.go`, `Update.go` and `View.go` for a 60-line model (D2).
- Do not put something in `chrome` because it used to be in the same package —
  it goes there when a second package needs it (D4).
- Do not use underscores or camelCase in package names (D1).
- Do not do this at the same time as a feature. It is a whole phase, alone, on
  its own branch, merged `--no-ff` — that is what makes it revertible if it
  turns out to be a mistake.
- Do not start Phase 2 before Phase 1 is merged: moving models while their
  shared helpers are still ambient means every move commit also touches
  `PanelFrame.go`, and the conflicts compound.
