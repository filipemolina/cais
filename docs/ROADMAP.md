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

- **Tabs are Groups, Services, Files, Env.** No dead placeholder tabs.
  *Groups/Services/Files scoped to the alpha; Env added in post-alpha.* Another
  plan (`docs/plans/resources-page.md`, "Resources") will add one more, and will
  rewrite this line. The rule: no tab ships empty. The digit range and `alt`+letter
  shortcuts (`1`–`4`, `g`/`s`/`f`/`e`) derive from `apptypes.PageTitles`.
- **Pages switch with digits and `[`/`]`,** keeping `alt`+letter as an alias.
  The digit range derives from `apptypes.PageTitles`, so a new tab extends it
  without an edit.
- **No statistics page.** Resource numbers belong as columns in the tables that
  already exist, not on a page of their own. `docker stats` is slow, it needs its
  own polling design, and ctop/lazydocker already own that niche.
  `docs/plans/docker-disk-usage.md` respects this and does not reopen it: it is
  an overlay on a key, fetched on open rather than polled, and its subject is
  disk — the half neither ctop nor lazydocker covers.
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
| 1 | `image-search.md` — **Phases 2A–2B** | Moved to the front 2026-08-01, redesigned around a search-first `n` modal (Spotlight/Telescope-style: type, a live results table, Enter to pick) rather than the original two-field-then-search flow — see the plan's *Why this jumped the queue*. Phase 1 (the two-field version) is already done; this is the redesign of everything upstream of the existing, unchanged write path. Phase 3 (the bootstrap flow adopting search too) stays deliberately deferred. |
| 2 | `docker-disk-usage.md` | A day and a half for an overlay that answers a question nothing else in the stack does — 70% of this machine's 60 GB of images is reclaimable and no tool says so. Not a page, so the *no statistics page* decision above stands. |
| 3 | `env-secrets.md` | The largest of the feature plans and the last lifecycle gap. Placed after the smaller ones so the pattern for "a new page with its own modals" is already established twice over — and so a long branch is not the thing blocking everything else. Has open owner decisions listed in its *Who decides* section; settle those before starting, not during. |
| 4 | `resources-page.md` — **Phases 0–1** | The read-only networks/volumes inspector. Last of the feature work because it is the least urgent and because its Phase 2 (writes) must wait for the write-safety story below. **Its Phase 0 is a one-line correctness fix and can be pulled forward at any time** — see the note under this table. |
| 5 | `cross-platform-testing.md` | Before the release work, not after: it costs $0 on GitHub's runners and it answers *what actually works on macOS and Windows*. Publishing binaries first and finding out afterwards is the wrong order. |
| 6 | `release-distribution.md` | Now the tool is worth installing, make it installable by people without a Go toolchain — which today is most of the audience. |
| 7 | `launch-and-outreach.md` | The announcement, and the gate: it opens with the lifecycle checklist that steps 1–3 close. |
| — | `adopt-unmanaged-containers.md` | **After the launch.** Phase 2 is blocked on nothing at the code level any more (`utils.AddServiceFragment` landed with `image-search.md` Phase 1) but stays post-launch anyway: the feature finds nothing at all on a tidy machine, and its audience - the homelab that grew by accretion - is a good post-launch story rather than a gate. Phase 1 (see them, switch to their file, remove them) is standalone and can be pulled forward if it is wanted sooner. |
| — | `resources-page.md` — **Phase 2** (writes) | **After the write-safety story.** Attaching a volume is a two-place edit to the user's file, and adding a new write surface before there is a backup or an undo is the wrong order. |
| — | `ai-service-authoring.md` | **Deliberately out of the sequence.** It is the only plan that adds a dependency on something outside the repo, and its own Phase 1 (an offline catalog) delivers most of the value with none of that. It needed `image-search.md`'s insert path to exist first, which is now done. Pick it up after the launch, or take its Phase 1 alone at any point. |

`group-table-legibility.md`, `docker-preflight.md`, `service-urls.md` and
`docker-disk-usage.md` are the round of feature ideas discussed on 2026-08-01
and were put **before** the launch deliberately, with the owner: two of them
are defects in the first screenshot, one is what a stranger sees when their
docker is broken, and one is the announcement's best screenshot. They add
roughly six days to the road to step 7, and that trade was made knowingly.
`group-table-legibility.md`, `docker-preflight.md` and `service-urls.md`
Phase 1 are the first three of the four to land — see *Done, and kept for
the record* below, which is also where `image-search.md` Phase 1 and
`service-aware-empty-state.md` landed (both predate this round, but share
the same table).

Three items in that sequence are not plans and would otherwise fall through the
cracks:

- **Cut a `v0.1.0` tag early — before step 1, not at step 6.** Done, on
  2026-08-01, straight after this six-plan round was sequenced and before
  `group-table-legibility.md` landed. The pipeline drafts a release on a `v*`
  tag, so the clock several launch directories require (a first release older
  than four months, `launch-and-outreach.md` §Directories) has already started.
- **A write-safety story, before step 7.** The app rewrites the user's compose
  file and there is no backup, no undo, and no prominent statement of what a
  write does not preserve. This is the risk that does not survive contact with
  a stranger's forty-service homelab. Sized in `launch-and-outreach.md`. It also
  gates `resources-page.md` Phase 2, which would otherwise add a second write
  surface before the first one is safe.
- **`resources-page.md` Phase 0, whenever it is convenient.**
  `utils.ReadConfigFile` hardcodes `cli.WithName("stack-stitcher")`, which
  overrides the file's own `name:` key — so the app's idea of the project name
  is not the one `docker compose` uses (measured: `stack-stitcher` where docker
  says `homelab`). Harmless today because nothing reads it; a silent
  correctness bug the moment anything correlates the file against the daemon.
  One line, one test.

### Done, and kept for the record

The plans in `docs/plans/` that have already landed:
`component-package-restructure.md` (one folder per model, `chrome` extracted —
finished at `56646b4`, which landed straight on main rather than as a merge),
`theme-picker-modal.md` and `theme-overhaul.md` (14 themes, live preview,
persisted), `group-rename.md` (`R`), the five editor steps —
`editor-paste.md`, `editor-indent-policy.md`, `editor-enter-autoindent.md`,
`editor-indent-keys.md`, `editor-key-advertising.md` — and
`group-table-legibility.md` (`chrome.PublishedPorts` and `chrome.ShortImage`;
the service details panel's Ports rows now share the same `chrome.PortLabel`
the group table adopted).

`docker-preflight.md` is also done: `utils.DockerPreflight` classifies which
of the five states a broken docker is in from *which probe failed*, never
from parsing what it printed (D1), and `dockerstatusmodal` shows the
diagnosis with the exact command that fixes it — copyable, never run (D2).
The probe runs at startup and re-runs on every docker error
(`reportDockerError`, alongside the existing `reportForegroundError`), so a
diagnosable failure replaces the raw `exec.ExitError` rather than leaving it
as the last word.

`image-search.md` **Phase 1** is done: `n` on the Services page adds a
service. `servicefieldsstep` - the name+image step - is shared with
`createcomposefilemodal`'s optional first service rather than copied, so
Phases 2-3 (Docker Hub search, a tag picker) reach both flows at once when
they land. `utils.AddServiceFragment` is `ApplyServiceFragment`'s insertion
counterpart, and is the primitive `adopt-unmanaged-containers.md` Phase 2
and `ai-service-authoring.md` were both waiting on. Phases 2-4 are not
scheduled in the table above; Phase 1 alone was step 1's scope.

**Phases 2A and 2B are done too** (merged 2026-08-01): the two-field step is
replaced by a search-first modal — `n` opens a live, debounced `docker
search` results table (official images marked, star counts shown, the typed
text as a free-text escape hatch when nothing is highlighted), Enter moves
to a confirm stage that pre-fills a service name derived from the image,
flags a collision on render, and fires a silent background best-tag lookup
from hub.docker.com's tags API whose result only applies if the user has
not typed over the field by the time it arrives. The write path is
unchanged (`cmds.AddService` → `utils.AddServiceFragment` → the inline
editor). Phase 3 (the tag picker) and the bootstrap flow adopting search
stay deferred.

`service-aware-empty-state.md` is done too: when a compose file has services
but no groups reference any of them, the Home details panel shows a live
overview (count header + the same member table a selected group uses)
instead of the static "Getting started" card. `knownGroups() == 0` already
means every loaded service is the ungrouped set, so no new filtering was
needed - `renderServiceOverview` reuses `renderMemberTable` verbatim. Falls
back to onboarding only when the file has no services at all.

`service-urls.md` **Phase 1** is done: the service details panel's `Web` row
is a real OSC 8 hyperlink to `utils.ResolveURL`'s guess at the service's
address (`stitcher.url` label > `app_protocol` > a fixed https-port set >
plain http, ties among published ports broken by a short known-web-port
table then file order), with the host resolved once at startup from
`config.URLHost` > `SSH_CONNECTION`'s measured server address > `localhost`.
`y` copies it via `tea.SetClipboard`. `chrome.Truncate` gained an
`ansi.StringWidth` fast path so a pre-sized hyperlink is not re-truncated
by the generic row renderer and corrupted - the one sharp edge in the plan,
pinned by a width-sweep test. Phase 2 (reverse-proxy labels - traefik,
gethomepage, tsdproxy) is not scheduled.

`healthcheck-insertion.md` is done: `h` on the Services details panel opens
`HealthcheckPickerModal`, listing `utils.TemplatesFor(image)` -
image-matched templates (Postgres, MariaDB, Redis, nginx - each using a probe
tool that ships in the real image) before the generic HTTP fallback, whose
container-internal port field appears inline, prefilled, only while that row
is highlighted - one modal, no second step. `utils.ApplyHealthcheck` inserts
or replaces the `healthcheck:` mapping through the same validated
read-modify-write path every other config write uses, and the catalog
deliberately omits `start_interval`, which this app's own parser accepts but
a user's `docker compose` CLI older than 2.20.2 may reject. The apply gap -
`restart` doesn't re-read compose, only `up -d` does - gets a footer hint
(`running: press s to apply`) rather than an automatic recreate, set by
`detailspanel` itself when `AddHealthcheckMsg` succeeds against a running
service. The catalog stays at the 4+1 rows the plan specifies; it is a
correctness claim per row, not a growing directory.

See *Docker's absence is a diagnosis, not an error*, *Home layout*,
*A service's URL is a guess, shown with its reasoning*, and *A healthcheck
template is a correctness claim, so the catalog stays small* in
`docs/DESIGN.md`.

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
  `TODO.md`. **Fixed**, along with the two other overflows the same terminals
  showed — the group table's colliding column headers and its cells running
  flush into each other. The bar sheds whole hints in `keys.Priority` order and
  the table drops whole columns in `dropOrder`; see *Narrow terminals: shed
  whole things* in `docs/DESIGN.md`, which generalises the rule the details
  panels' action row established before it was removed.
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
