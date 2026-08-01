# Plan: What Docker Is Costing You, in Bars, on One Key

> **Before you start.** Work on a feature branch of small commits, merged
> `--no-ff`; `go build ./... && go vet ./... && go test ./... && gofmt -l .`
> green at **every** commit, not just at the tip — `docs/ROADMAP.md`
> §Conventions is the full contract and `CONTRIBUTING.md` explains how a TUI
> gets tested. Behaviour that only shows on screen gets checked in the real app
> with VHS before it is committed.

Feature request (discovery session, nothing implemented): *"Show the amount of
memory and disc being used by docker somewhere. Preferrably in a easy to see
fashion, like a horizontal bar chart or something."*

## Status — and the decision this has to respect

Nothing shows aggregate usage today. Per-service memory exists (the services
list and the runtime stats table, both from the `docker stats` poll), but
nothing sums it, and nothing anywhere mentions disk.

**`docs/ROADMAP.md` §Decisions already taken with the owner contains this:**

> **No statistics page.** Resource numbers belong as columns in the tables that
> already exist, not on a page of their own. `docker stats` is slow, it needs
> its own polling design, and ctop/lazydocker already own that niche.

This plan does not reopen that. It proposes an **overlay** — the same class of
thing as `?` and `a`, opened deliberately, closed with esc, occupying no
layout, present on no page. The three reasons behind the decision all survive:
no page is added, the slow call is user-initiated rather than polled, and the
data is the half of the story ctop and lazydocker *don't* tell — disk.

Because that half is genuinely uncovered. Measured on the author's machine,
2026-08-01:

```
TYPE            TOTAL   ACTIVE   SIZE      RECLAIMABLE
Images          76      22       60.11GB   42.28GB (70%)
Containers      38      8        384.8MB   342.7MB (89%)
Local Volumes   34      12       1.097GB   1.076GB (98%)
Build Cache     10      0        1.311GB   1.311GB
```

**42 GB of reclaimable images on a working homelab, invisible from every tool
in the stack.** That number is the feature.

## Research — measured on 2026-08-01

### `docker system df --format json` is NDJSON of strings

```json
{"Type":"Images","TotalCount":"76","Active":"22","Size":"60.11GB","Reclaimable":"42.28GB (70%)"}
{"Type":"Containers","TotalCount":"38","Active":"8","Size":"384.8MB","Reclaimable":"342.7MB (89%)"}
{"Type":"Local Volumes","TotalCount":"34","Active":"12","Size":"1.097GB","Reclaimable":"1.076GB (98%)"}
{"Type":"Build Cache","TotalCount":"10","Active":"0","Size":"1.311GB","Reclaimable":"1.311GB"}
```

Three facts to design around:

1. **Every value is a string**, including the counts. Sizes are human-readable
   SI (`60.11GB`), not bytes.
2. **`Reclaimable` sometimes carries a percentage in parentheses and sometimes
   does not** — Build Cache above has none. Parse the leading size field only.
3. **The output is NDJSON**, like the legacy `docker compose ps` format, so the
   existing decoder-loop shape in `utils.ParseContainers` is the model to copy.

### It is slow enough to matter

| Call | Cost |
| --- | --- |
| `docker system df` | **2.3 s** cold, 1.4 s warm |
| `docker system df -v` | 1.4 s, returns one object with `Images[]`, `Containers[]`, `Volumes[]`, `BuildCache[]` |
| `docker info --format '{{.MemTotal}}'` | 162 ms, returns bytes (`33304059904`) |

2.3 seconds is why this cannot go on the five-second poll and why the overlay
fetches on open with a spinner. It is also why it is a good fit for an overlay:
a user who pressed a key is willing to wait a moment; a user who switched pages
is not.

### The size parser already ships

`github.com/docker/go-units` provides `FromHumanSize("60.11GB") (int64, error)`
and `BytesSize(float64) string`. It is **already imported directly** by
`src/components/detailspanel/View.go` for memory limits — it sits in go.mod's
indirect block only because nothing has run `go mod tidy` since. So this plan
adds **no new dependency**; it may promote one line in go.mod.

Note the SI/IEC distinction and do not fight it: docker prints `GB` (powers of
1000) here and `GiB` in `docker stats`. `FromHumanSize` handles the former,
`RAMInBytes` the latter. Using the wrong one is a 7% error that nobody will
notice and everybody will inherit.

### Memory needs no new call

The five-second poll already runs `docker stats --no-stream` and stores
`MemUsage` (`"21.71MiB / 31.02GiB"`) on every container. Summing the used side
across running containers gives docker's total footprint, and the limit side —
or `docker info`'s `MemTotal` — gives the denominator. **The memory half of
this feature is arithmetic on data the app already has.**

## Solution overview

`u` opens a **Usage** overlay:

```
┌ Usage ─────────────────────────────────────────── docker 29.6.0 ┐
│                                                                 │
│  DISK                                             63.0 GB total │
│                                                                 │
│  Images        ████████████████░░░░░░░░░░░░░░░░░   60.1 GB      │
│    22 of 76 active                                 42.3 GB idle │
│  Containers    █░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  384.8 MB      │
│    8 of 38 active                                 342.7 MB idle │
│  Volumes       █░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░    1.1 GB      │
│    12 of 34 active                                  1.1 GB idle │
│  Build cache   ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░    1.3 GB idle │
│                                                                 │
│  45.0 GB is reclaimable — `docker system prune -a --volumes`    │
│                                                                 │
│  MEMORY                                                         │
│                                                                 │
│  Containers    ███████░░░░░░░░░░░░░░░░░░░░░░░░░░   6.2 / 31.0 GB│
│    18 running                                                   │
│                                                                 │
│  r refresh · esc close                                          │
└─────────────────────────────────────────────────────────────────┘
```

Filled = in use, shaded = reclaimable. Bars are scaled to the largest category
so the row that matters is the row that is long.

## Design decisions

### D1. An overlay, not a page, not a footer element

Per the roadmap decision above. Concretely, it follows `aboutmodal` and
`helpoverlay`: a `tea.Model` in `AppModel.activeModal`, drawn over the frozen
panels, closing on `esc` / `q` / `u`.

Not the footer: the footer is width-constrained enough to have needed its own
shedding mechanism (`docs/DESIGN.md` §*Narrow terminals*), and a permanently
visible number would need a permanent poll to keep it honest, which is the
thing the roadmap decision rules out.

### D2. `u` for usage

Free, verified: the only occurrence of `"u"` in the codebase is
`groupslist/Model_test.go:179`, a test asserting that the bubbles list's paging
keys (`l h f b u`) are *not* bound to the list. A global `u` handled in
`AppModel` does not page the list, so that test stays green — but read it
before you start, because it is the exact same check
`docs/plans/healthcheck-insertion.md` had to make for `h`.

Declared in `keys.Global` beside `About` and `Theme`, advertised in the `?`
overlay's Global scope, **not** in the footer — the same treatment `a` and `T`
get, for the same reason (the bar is full).

### D3. Fetch on open, spinner while it runs, cache for the session

- Opening dispatches `cmds.GetDockerUsage()`; the overlay renders a centred
  spinner (`chrome.Spinner`, already used by the pending-action indicator)
  until the result arrives.
- The result is cached on `AppModel` and reused if the overlay is reopened —
  disk usage does not change in a way that matters over a few minutes.
- `r` inside the overlay refetches, spinner and all. (`r` is `Details.Restart`
  elsewhere; a modal owns the keyboard, so there is no collision — the same
  argument `y`/`n` already rely on.)
- **Never on the ticker.** The five-second poll must not learn about this.

### D4. Two bars' worth of arithmetic, one rendering primitive

```go
// bar renders a proportional two-part bar: filled for used, shaded for the
// remainder, in width columns.
func bar(used, total int64, width int) string
```

It stays **unexported inside the overlay's package**. `docs/DESIGN.md` §6:
a helper earns its way into `chrome` by having a second caller, and this has
one. Move it later if the Resources page wants it.

Rules that keep it honest:

- **Non-zero rounds to at least one cell.** A 1.3 GB build cache beside a 60 GB
  image store is 0.7 cells; rendering it as empty says "nothing here" about
  something that is 1.3 GB.
- **The bar always occupies exactly `width` cells**, so rows line up; the
  filled count is rounded and the shaded count is `width - filled`.
- **`total == 0` renders an empty bar**, not a divide-by-zero.

### D5. Colour comes from tokens that already exist

Two colours, not five:

| Meaning | Token |
| --- | --- |
| in use | `appstyles.Active.Accent` |
| reclaimable / free | `appstyles.Active.TextDim` |

One row per category means the categories never need to be distinguished by
hue — they are distinguished by being on different lines with labels. That
avoids inventing a four-colour categorical palette, which would mean four new
fields on `Theme` and edits to all fourteen registered themes.

Read `appstyles.Active.X` fresh at render time, never into a package-level
`var` (`docs/DESIGN.md` §*Color lives on a Theme*).

### D6. No prune. Print the command.

The overlay names the reclaimable total and shows the command that would
reclaim it, to be copied. It does not run it.

`docker system prune -a --volumes` **deletes data**: `--volumes` removes
volumes no container currently references, which includes every volume
belonging to a stack that is merely stopped. Running that from a TUI, behind a
`y`/`n` confirm, on a homelab, is how someone loses their Sonarr database.

If pruning is ever added it needs its own plan, its own preview of exactly what
would be deleted, and a confirmation stronger than one keystroke — the same
bar `docs/plans/resources-page.md` sets for volume deletion. Out of scope here,
deliberately, and the plan says so in the overlay's own text: the command shown
is the *safe* one first (`docker system prune`, no `-a`, no `--volumes`) with
the aggressive one mentioned in the note.

### D7. Docker being unavailable defers to the preflight

If the usage fetch fails, the overlay shows the same diagnosis
`docs/plans/docker-preflight.md` produces, not a raw error. If that plan has
not landed yet, show the error text plainly and leave a `TODO` naming it —
do not build a second diagnosis path.

## Detailed changes

1. **`src/utils/DockerSystemDf.go` (new)**

   ```go
   type DiskUsage struct {
       Type        string // "Images", "Containers", "Local Volumes", "Build Cache"
       TotalCount  int
       Active      int
       Size        int64 // bytes
       Reclaimable int64 // bytes
   }

   // ParseSystemDf turns `docker system df --format json` output into usage
   // rows. NDJSON, like the legacy compose ps format.
   func ParseSystemDf(output string) ([]DiskUsage, error)

   func DockerSystemDf() (string, error) // the exec half, mirroring DockerComposePs
   ```

   Split exactly the way `DockerComposePs` / `ParseContainers` are split: the
   exec function returns a string, the parser is pure and takes one. That is
   what makes the parser testable against a captured fixture.

2. **`src/utils/DockerMemTotal.go` (new)** — `docker info --format
   '{{.MemTotal}}'` → `int64`. Twelve lines. A failure is not fatal: the memory
   bar falls back to the limit side of any container's `MemUsage` string, and
   if that is absent too, the memory section is omitted.

3. **`src/cmds/GetDockerUsage.go` (new)** — `GetDockerUsage() tea.Cmd` running
   both calls and returning `DockerUsageMsg{Disk []utils.DiskUsage, MemTotal
   int64, Err error}`.

4. **`src/components/usageoverlay/` (new package)** — `Model.go`, `Update.go`,
   `View.go`. The model holds the disk rows, the mem total, the container
   memory sum (passed in from `AppModel`, which already has the containers), a
   spinner, and a loading flag.

5. **`src/keys/Keys.go`** — `Global.Usage` = `u` / "usage"; add to the Global
   scope in `Catalog` and to `pressableNow`'s always-live list beside
   `Global.About` and `Global.Theme`.

6. **`src/model/Update.go`** — `u` opens it (inside the `keyboardOwned()`
   guard, like every other global key, so `u` stays a letter while a filter is
   being typed); handle `DockerUsageMsg`; cache it.

7. **`src/apptypes/` or the overlay** — the memory sum helper:

   ```go
   // SumContainerMemory adds the used side of every container's MemUsage.
   // Returns the total and how many containers reported one.
   func SumContainerMemory(containers []DockerContainer) (int64, int)
   ```

   Note the existing lesson in `TODO.md`: `MemUsage` is stored raw and
   formatted at render time because `FormatMemUsage` is not idempotent. This
   function parses the raw form (`RAMInBytes` on the part before `/`), which is
   only possible *because* it is stored raw. Do not change that.

8. **`README.md`** — one row in the key table (`u` — usage) and a bullet under
   *More screens*, with a screenshot. This is good screenshot material: the
   42 GB number does the talking.

## Tests

### `src/utils/DockerSystemDf_test.go`

**Capture the fixture; do not write it.** `TODO.md`'s HEALTH-column entry is
the standing lesson: fixtures invented from the Go struct test nothing but
themselves, and that bug survived for months because the test agreed with the
code and both were wrong about docker. Run `docker system df --format json`,
paste the four lines verbatim into the test file, and assert on parsed values.

Cases:

| input | want |
| --- | --- |
| the captured four lines | 4 rows; Images `Size` = 60110000000-ish, `Reclaimable` = 42280000000-ish |
| a row whose `Reclaimable` has no percentage (`"1.311GB"`) | parses, no error |
| `Reclaimable: "42.28GB (70%)"` | parses to the size, percentage ignored |
| `Size: "0B"` | zero, no error |
| empty output | empty slice, no error (the `ParseContainers` precedent) |
| a JSON array instead of NDJSON | parses too — cheap insurance, same as `ParseContainers` |
| garbage | error |

### `src/components/usageoverlay/bar_test.go`

| used | total | width | want |
| --- | --- | --- | --- |
| 50 | 100 | 10 | 5 filled |
| 1 | 1000 | 10 | **1** filled (the non-zero rule) |
| 0 | 1000 | 10 | 0 filled |
| 100 | 100 | 10 | 10 filled |
| 50 | 0 | 10 | 0 filled, no panic |
| 150 | 100 | 10 | 10 filled, no overflow |

Plus: every case renders exactly `width` visible cells
(`ansi.Strip` then `runewidth.StringWidth`).

### `src/model/usage_test.go`

Mirroring `about_test.go`: `u` opens the overlay; `esc` closes it; `u` while a
modal is open does nothing; a `DockerUsageMsg` arriving after the overlay was
closed does not reopen it.

### `src/apptypes/`

`SumContainerMemory` against real `MemUsage` strings, including one container
with an empty string (skipped, not counted) and one with `MiB`/`GiB` mixed.

## Edge cases and unknowns

- **A machine with no images** returns rows with `0B` and zero counts. Every
  bar is empty and the total line reads `0 B`. Fine; do not special-case it
  into an empty state.
- **`docker system df` on some storage drivers** can be much slower than 2.3 s
  (btrfs and zfs walk more). The spinner covers it; do not add a timeout that
  would turn a slow answer into an error.
- **Build cache is absent** on installs without BuildKit; the row is simply not
  in the output. Render whatever rows came back rather than a fixed four.
- **`docker info`'s `MemTotal` inside a VM** (Docker Desktop, colima) reports
  the VM's memory, not the Mac's. That is the correct denominator for
  containers, and worth one dimmed word — the overlay shows `docker 29.6.0` in
  its title row; on Desktop it is honest to leave the number as docker reports
  it.
- **`docker system df -v`** is not needed for this plan. It is what
  `docs/plans/resources-page.md` will want for per-volume sizes; note the
  overlap there rather than fetching it here.
- **Rounding.** `BytesSize` gives `60.11GB`; the mock-up above shows `60.1 GB`.
  Pick one and use `units.BytesSize` — matching docker's own output means a
  user can compare the two screens without arithmetic.

## Effort / gain

**One to one and a half days.** The parser and its tests are two hours; the
overlay follows `aboutmodal`'s shape and is half a day with its bar helper; the
wiring is an hour; docs and a screenshot, an hour.

The gain is one genuinely surprising number per user. Nobody knows what docker
is costing them until something asks, and 70% reclaimable is not unusual — it
is what a homelab that has pulled `:latest` for two years looks like. It is
also, unlike most of this list, immediately actionable in one command.

## Blast radius

- One new key, one new overlay, two new util files, one new command.
- Nothing on the poll path changes. Nothing on any page changes.
- The overlay is the only consumer of the new data; if the fetch fails, only
  the overlay is affected.
- No writes, no new dependency (go-units is already imported directly).

## Do not

- **Do not put a usage number in the footer or the nav.** D1, and the footer
  has no room by construction.
- **Do not poll `docker system df`.** It is 2.3 seconds; a five-second poll
  would spend half its life in it.
- **Do not run any prune command.** D6.
- **Do not invent theme colours for the categories.** D5 — one row per
  category and two tokens.
- **Do not reformat `MemUsage` on the way in.** `TODO.md` has the story;
  `FormatMemUsage` is not idempotent and a test asserts it.
- **Do not use `RAMInBytes` for `docker system df` sizes or `FromHumanSize` for
  `docker stats` sizes.** GB and GiB, respectively.
- **Do not build a second docker-error diagnosis.** D7.

## Acceptance criteria

1. `u` opens the overlay from any page; `esc`, `q` and `u` close it.
2. The first open shows a spinner and then the four disk rows with bars.
3. The reclaimable total matches `docker system df`'s own arithmetic to the
   nearest displayed unit.
4. The memory row's used value is within rounding of
   `docker stats --no-stream` summed by hand.
5. Reopening the overlay is instant (cached); `r` refetches with the spinner.
6. With a stopped daemon the overlay says what is wrong, not `exit status 1`.
7. The five-second poll's timing is unchanged — verify by watching the app for
   a minute with the overlay closed and confirming no `docker system df`
   process appears (`pgrep -af 'docker system df'`).
8. Every existing test passes at every commit.
