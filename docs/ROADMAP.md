# Roadmap

This is the ordered plan the current work follows, and it is **live**.
`TODO.md` is the flat worklist and `docs/plans/` holds the plan for each
individual piece of work; **this file is the order they happen in, and why**,
so picking up mid-sequence does not mean re-deciding what was already settled.

It has two halves. *Where we are* is the road to the first alpha, which is
finished — every phase in it is done, and the design each one produced now
lives in `docs/DESIGN.md`. *The order after the alpha* is the live part: the
sequence through the plans in `docs/plans/`, ending at the one that announces
the thing.

The aim is not a finished product. It is a foundation someone else can extend
without re-litigating decisions, plus enough polish to publish a first alpha.

## Conventions

Each phase is a feature branch of small commits, merged with `--no-ff` so a
phase can be reverted as a unit. `go build ./... && go vet ./... && go test ./...`
green at **every** commit, not just at the merge. The merge commit's hash then
goes into the *Where we are* table as its `done` marker (a tiny "Pin the phase
N merge hash" commit straight to main — the hash cannot exist before the
merge does), and the phase branch is deleted.

Behaviour that only shows up on screen gets checked in the real app with VHS
(`vhs` is installed; write a tape with `Screenshot "name.png"` and run it from a
scratch directory — note that VHS needs its paths quoted, and sometimes drops
the last screenshot, so re-run if the file is missing). For a faster loop, a
throwaway `go run` program that renders one component through
`ansi.Strip(m.View().Content)` catches layout and styling mistakes before they
are committed; phases 4 and 5 used it for the nav and the help overlay.

Docs are part of the phase, not a follow-up: `README.md` for what a user needs to
know, `docs/DESIGN.md` for why a decision went the way it did, `TODO.md` for
what is left. When a phase lands, its section here is removed and the design
moves to `docs/DESIGN.md` — see the *Where we are* table note above the phase
sections.

## Decisions already taken with the owner

Do not re-open these without asking:

- **Tabs for the alpha are Groups, Services, Files.** No dead placeholder tabs.
  *Scoped to the alpha, which has shipped.* `docs/plans/env-secrets.md` adds a
  fourth ("Env"), and the phase that lands it must rewrite this line rather than
  leave the repo arguing with itself. The rule that outlives the count is the
  second sentence: no tab ships empty.
- **Pages switch with digits `1`–`3` and `[`/`]`,** keeping `alt`+letter as an alias.
- **No statistics page.** Resource numbers belong as columns in the tables that
  already exist, not on a page of their own. `docker stats` is slow, it needs its
  own polling design, and ctop/lazydocker already own that niche.
- **No prefix key** (tmux/zellij style). The reasoning is written up in
  `docs/DESIGN.md` — prefixes exist to address a host that is hosting another
  program that owns the keyboard, and this app has no guest.
- **The compose file priority order stays fixed and identical to Docker's.**
  Reasoning in `docs/DESIGN.md` under *Which compose file*.
- **Colors get centralized into a `Theme` value** before the alpha, so a theme
  picker modal can land right after it.

## Where we are

| Phase | Status |
| --- | --- |
| 0 — Rename the module, rename the Dashboard page, drop the Settings tab | done (`231adc7`) |
| 1 — One keymap in `src/keys` | done (`9a68171`) |
| 2 — Footer shows the parsed compose file | done (`62416ef`) |
| 3 — The lists own their keymaps | done (`55173d0`) |
| 4 — The new global keys | done (`a64ec73`) |
| 5 — `?` help overlay | done (`37bf74a`) |
| 6 — Centralize color into a `Theme` | done (`88ce42d`) |
| 7 — Release plumbing | done (`01f75e9`) |
| 8 — Edit group membership, then the Files page | done (`ffe9fed`) |
| 9 — Complete the Files page: syntax highlighting, browse and switch | done (`f450a55`) |

Phases 0–9 are described in `docs/DESIGN.md` (*Where keybindings live*, *Which
compose file* — now including *One resolution, passed down* and the two flags —
*The lists do not get to keep `list.DefaultKeyMap`*, *Navigation and focus*,
*Color lives on a Theme*, *Background tiers, and sealing them*, *Saying which
build this is*, *Editing group membership*, *The Files page*) rather than here,
because they are now how the app works rather than a plan.

That is the whole roadmap to the alpha: every tab is live, and the first
group-membership wall is gone. What follows is the live part.

## The order after the alpha

Everything below is written up in full in `docs/plans/`. **This table is the
order to do them in; each plan is the how.** The destination is
`docs/plans/launch-and-outreach.md`, which is last on purpose: it is the one
that measures whether the rest are done.

| # | Plan | Why it sits here |
| --- | --- | --- |
| 1 | `component-package-restructure.md` | First, because it is pure churn and gets more expensive every week. Four of the plans below add components; doing the move first means they land in the new shape instead of being rewritten into it. The plan reaches the same conclusion on its own ("recommended, before the feature plans"), and names the clinching argument: any in-flight branch touching `src/components` conflicts with it. |
| 2 | `image-search.md` — **Phase 1 only** (`n` adds a service) | The biggest hole in the lifecycle, and the one every later plan is blocked on. `ApplyServiceFragment` can edit a service but not create one, so "add a service" means leaving the app. Phases 2–4 (Docker Hub search, tag picker) are the *feature*; Phase 1 is the *foundation*, and splitting it out is what unblocks 3, 4 and 8. |
| 3 | `service-aware-empty-state.md` | Small, and it belongs directly after step 2 rather than before it: once `n` can add a service, the first-run empty state can say something useful to a user with an empty file instead of explaining a concept they cannot act on yet. Doing it earlier means writing that copy twice. |
| 4 | `healthcheck-insertion.md` | ~2 days, and it completes a lifecycle stage the ask names explicitly. It depends on nothing from steps 2–3 in code, but it reads as a natural pair with them: adding a service and giving it a working probe are the same sitting for the user. |
| 5 | `env-secrets.md` | The largest of the feature plans and the last lifecycle gap. Placed after the smaller ones so the pattern for "a new page with its own modals" is already established twice over — and so a long branch is not the thing blocking everything else. Has open owner decisions listed in its *Who decides* section; settle those before starting, not during. |
| 6 | `cross-platform-testing.md` | Before the release work, not after: it costs $0 on GitHub's runners and it answers *what actually works on macOS and Windows*. Publishing binaries first and finding out afterwards is the wrong order. |
| 7 | `release-distribution.md` | Now the tool is worth installing, make it installable by people without a Go toolchain — which today is most of the audience. |
| 8 | `launch-and-outreach.md` | The announcement, and the gate: it opens with the lifecycle checklist that steps 2–5 close. |
| — | `ai-service-authoring.md` | **Deliberately out of the sequence.** It is the only plan that adds a dependency on something outside the repo, its own Phase 1 (an offline catalog) delivers most of the value with none of that, and it needs step 2's insert path to exist first. Pick it up after the launch, or take its Phase 1 alone at any point. |

Three items in that sequence are not plans and would otherwise fall through the
cracks:

- **Cut a `v0.1.0` tag early — before step 1, not at step 7.** The pipeline
  already drafts a release on a `v*` tag, so this is an afternoon, and several
  of the directories worth being listed in require a first release older than
  four months (`launch-and-outreach.md` §Directories). The clock starts at the
  tag, so starting it now costs nothing and saves a wait later.
- **A write-safety story, before step 8.** The app rewrites the user's compose
  file and there is no backup, no undo, and no prominent statement of what a
  write does not preserve. This is the risk that does not survive contact with
  a stranger's forty-service homelab. Sized in `launch-and-outreach.md`.
- **The footer's narrow-terminal wrap** (`TODO.md`, still open). It is visible
  in the demo recording, which makes it the one open bug a first-time visitor
  is guaranteed to see.

### Done, and kept for the record

The plans in `docs/plans/` that have already landed: `theme-picker-modal.md`
and `theme-overhaul.md` (14 themes, live preview, persisted), `group-rename.md`
(`R`), and the five editor steps — `editor-paste.md`,
`editor-indent-policy.md`, `editor-enter-autoindent.md`,
`editor-indent-keys.md`, `editor-key-advertising.md`.

Also done and not carrying a plan file: the **UX improvements** (auto-select on
navigation, `n` on both panels, spinner feedback, error modals — see
`docs/PLAN-UX-IMPROVEMENTS.md`) and the **service details panel** redesign
(`docs/DESIGN.md` §Services layout).

Still unplanned, and intentionally so: live CPU/MEM columns from `docker
stats`, an `x`-style action menu, and the remaining config fields (default
file, keybinding overrides) that the config struct was shaped to absorb.

## Loose ends worth knowing about

- ~~**`CPU: 0.0` in the services list is not telemetry.**~~ **Fixed** — and no
  longer possible: `CPUPercent` does not exist anywhere in `src/` (verified
  2026-07-31). `ServiceListItem` renders memory only, from real `docker stats`
  values stored raw and formatted once at render time.
- **The footer wraps far earlier than this file used to claim.** The old figure
  here was "below roughly 60 columns"; it is closer to **130**. Measured while
  recording the README screenshots on 2026-07-31: at 1280px / 16pt (~133
  columns) the Groups footer wraps `q quit` onto a second line, and at 1440px
  (~150 columns) it fits. The logs-overlay context, which advertises more keys,
  wraps wider still. That makes it the one open bug a first-time visitor is
  guaranteed to see, since it is visible in `demo/demo.gif`. Tracked in
  `TODO.md`; the bar needs to shed hints in priority order the way the compose
  file name already does. The details table headers and the panels' action
  button row had the same problem. The button row was fixed by shedding whole
  buttons in a declared order — that fix is the worked example for this one,
  though the row itself has since been removed (see *The panel footer* in
  `docs/DESIGN.md`), so it is prior art in the git history rather than code to
  read.
- **Panel keypresses through the e2e rig — fixed.** `TODO.md` has the details:
  panel keys need `Text` set on the `tea.KeyPressMsg` because `key.Matches`
  compares `msg.String()` against the binding strings. `TestRigGroupListEditKey`
  now verifies a panel key ('e') reaches the focused groups list through the
  full program. The rig gained a `letterKey` helper for printable keys; special
  modal keys (esc/enter/tab/backspace) still go through the existing `keyPress`
  helper.
- **The two flaky bootstrap tests were an app bug, not a rig bug** (fixed in
  Phase 7). Worth remembering before blaming the rig again: a failed reload was
  replacing the open modal with a fresh one. If a rig test starts failing
  intermittently, ask what else the app is doing to itself first.
- **`--version` is unstamped in a `go run` / plain `go build`.** It reports the
  commit instead, which is intended. Only `make build` and the release build
  stamp a version, so a screenshot of a dev build shows a hash in the nav bar.
