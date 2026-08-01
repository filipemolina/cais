# Plan: Make the Group Member Table's PORTS and IMAGE Columns Readable

> **Before you start.** Work on a feature branch of small commits, merged
> `--no-ff`; `go build ./... && go vet ./... && go test ./... && gofmt -l .`
> green at **every** commit, not just at the tip — `docs/ROADMAP.md`
> §Conventions is the full contract and `CONTRIBUTING.md` explains how a TUI
> gets tested. Behaviour that only shows on screen gets checked in the real app
> with VHS before it is committed.
>
> This is the smallest plan in `docs/plans/` and it should be done **first**,
> before the README assets are re-recorded for the launch: both defects are
> visible in `demo/screenshot-groups.png` as it stands today.

Feature request (discovery session, nothing implemented): *"Adjust the way
ports are shown on the Group Details pane. Maybe make it like the service
details pane does it."* and *"Adjust the way image names are shown on the Group
Details pane. It's too crowded."*

## Status — this is a legibility defect, not a feature

Both columns already render. Neither says anything useful at the width it gets.

**PORTS.** The cell is `container.Ports` — the string `docker compose ps`
emits, passed through untouched (`groupdetailspanel/View.go`, `renderMemberRow`,
the `ports` entry of the `cell` map). Captured from a real homelab on
2026-08-01 (Engine 29.6.0, Compose v5.1.4):

| Service | `Ports` as docker reports it |
| --- | --- |
| navidrome | `0.0.0.0:4533->4533/tcp, [::]:4533->4533/tcp` |
| prowlarr | `0.0.0.0:9696->9696/tcp, [::]:9696->9696/tcp` |
| feishin | `8080/tcp, 0.0.0.0:9180->9180/tcp, [::]:9180->9180/tcp` |
| qbittorrent | `0.0.0.0:6881->6881/tcp, [::]:6881->6881/tcp, 0.0.0.0:8080->8080/tcp, 0.0.0.0:6881->6881/udp, [::]:8080->8080/tcp, [::]:6881->6881/udp` |

`computeCols` gives PORTS 16 columns at its base width, and cells truncate to
one less than the column, so what the user actually sees is:

```
0.0.0.0:4533->…
0.0.0.0:6881->…
```

Every row starts with the same nine characters. The column is 15 columns of
`0.0.0.0:` and an ellipsis — it cannot distinguish two services, which is the
only job a table column has. qbittorrent's six entries are two published ports
(8080/tcp and 6881/tcp+udp) written six times, because docker lists the IPv4
and IPv6 bindings separately and counts protocols separately.

**IMAGE.** The cell is the image reference, also untouched, also 16 columns:

| Image in the file | What the cell shows |
| --- | --- |
| `lscr.io/linuxserver/kavita:latest` | `lscr.io/linuxse…` |
| `lscr.io/linuxserver/radarr:latest` | `lscr.io/linuxse…` |
| `ghcr.io/hotio/sonarr:latest` | `ghcr.io/hotio/s…` |
| `ghcr.io/advplyr/audiobookshelf:latest` | `ghcr.io/advplyr…` |

Three of those four are the registry and nothing else. Two are byte-identical.
The one piece of the reference a reader wants — *which application is this* —
is the part that gets truncated away, because it is last.

Both defects are the same mistake in two places: **the cell renders the widest
possible spelling of a value into the narrowest column that value could get.**
`docs/DESIGN.md` §*Narrow terminals: shed whole things* already names the fix
for this family — drop whole units in a declared order — and the table already
applies it *between* columns (`dropOrder`). This plan applies it *inside* two
of them.

## Solution overview

Two pure functions, one call site each, plus one change of source.

1. **PORTS renders from the compose file, not from the runtime string**, and
   shows the published host ports only: `4533`, `8080, 6881`.
2. **IMAGE sheds parts of the reference in a declared order** until it fits:
   registry, then namespace, then tag — `lscr.io/linuxserver/kavita:latest`
   becomes `linuxserver/kavita`, and `kavita` when the column is narrow.

Neither function belongs to the group panel alone, so both go in
`src/components/chrome` — the second caller for each is the service details
panel, which gets the long forms (see D2 and D4). That satisfies the rule in
`docs/DESIGN.md` §6: a helper earns its way into `chrome` by having a second
caller, not by convenience.

## Design decisions

### D1. The PORTS column reads the compose file, not `docker compose ps`

Today the row prefers `container.Ports` and falls back to `-`. After this
change it reads `svc.Ports` — the `[]types.ServicePortConfig` the member
already carries — and the runtime string is not consulted for this column at
all.

Why:

- **It is what the ask says.** The service details panel builds its Ports rows
  from `svc.Ports` (`detailspanel/View.go`, `configRows`). "Make it like the
  service details pane does it" is, most literally, *read the same field*.
- **The file is the app's source of truth.** That is the first sentence of the
  README and §3 of `docs/DESIGN.md`. A column that reads the daemon disagrees
  with the panel beside it the moment the file is edited and the service has
  not been recreated — and the app cannot show both in 15 columns.
- **The runtime string is unparseable at this width anyway.** Folding
  `0.0.0.0:` / `[::]:` duplicates and dropping container-only ports is exactly
  the work D3 does for the config form, except from a string that has to be
  re-parsed, with per-version formatting risk, to arrive at the same numbers
  the file already states structurally.
- **A stopped service gets a useful cell.** Today every stopped row reads `-`,
  because there is no container to ask. The file knows what the service *will*
  publish, and that is worth more than a dash.

**The trade-off, stated plainly:** a container started before an edit publishes
what it published then, and this column will show what the file says now. The
row's STATE, HEALTH and UPTIME stay runtime, so the row mixes sources — that is
already true (IMAGE prefers the container's image, see D5). The mitigation is
the one the app already relies on for every other config field: `s` runs
`up -d`, which recreates the container from the file. If a future phase wants
to surface config drift, it should do it as its own signal (a `~` marker beside
the state, say), not by making one column silently mean something else.

### D2. `chrome.PortLabel` is the long form; the details panel adopts it

The service details panel builds its port strings inline:

```go
portStr := fmt.Sprintf("%d/%s", port.Target, protocol)
if port.Published != "" {
    portStr = port.Published + "->" + portStr
}
```

Move that to `chrome`, unchanged in behaviour except for one addition — the
host IP, when it is set and is not a wildcard:

```go
// PortLabel is one published port in full: the form the service details
// panel shows, where there is room for it.
//
//	14533->4533/tcp
//	127.0.0.1:14533->4533/tcp   (bound to loopback: not reachable off-host)
//	4533/tcp                    (exposed, not published)
func PortLabel(p types.ServicePortConfig) string
```

`HostIP` is worth the columns it costs: a service published on `127.0.0.1` is
unreachable from the LAN, which is the difference between "my dashboard is
broken" and "I bound it to loopback eight months ago". It is also the fact
`docs/plans/service-urls.md` needs, and adding it here means that plan finds it
already rendered.

### D3. `chrome.PublishedPorts` is the short form: host ports, deduplicated

```go
// PublishedPorts returns the host-side ports a service publishes, in file
// order, deduplicated, for a column too narrow for the full mapping. A
// service that publishes nothing returns an empty slice.
//
//	[]{14533:4533/tcp}                        -> ["14533"]
//	[]{8080:8080/tcp, 6881:6881/tcp+udp}      -> ["8080", "6881"]
//	[]{4533/tcp}                              -> []          (not published)
//	[]{53:53/udp}                             -> ["53/udp"]
func PublishedPorts(ports []types.ServicePortConfig) []string
```

Rules, in order:

1. **Skip unpublished entries.** `Published == ""` means `expose:`-style — the
   port exists inside the network and there is nothing to type into a browser.
   The details panel still lists it (D2 renders it as `4533/tcp`).
2. **Key on the published value.** `Published` is a string, not an int, because
   compose accepts ranges (`"8000-8010"`). Pass it through as written; do not
   parse it into numbers.
3. **Deduplicate by published value, preserving file order.** Two entries with
   the same published port and different protocols collapse to one.
4. **Protocol is shown only when every entry for that published value is
   non-TCP.** `6881/tcp` + `6881/udp` renders `6881`; a lone `53/udp` renders
   `53/udp`. TCP is the default in the compose spec and printing it on every
   cell spends four columns saying nothing.
5. **Host IP is not shown.** It does not fit, and it is in the details panel.

The cell joins with `", "`; an empty result renders `—` (an em dash, the same
"nothing here" mark `renderPropRow` already uses), not `-`.

### D4. `chrome.ShortImage` sheds parts of the reference in a declared order

```go
// ShortImage renders an image reference to fit width columns, giving up
// whole parts of the reference in a fixed order rather than truncating the
// tail — the name is the part a reader needs and it is the part a plain
// truncation destroys first.
func ShortImage(ref string, width int) string
```

The ladder, widest first. Each rung is tried in full; the first that fits wins:

| Rung | `lscr.io/linuxserver/kavita:latest` | `postgres:16-alpine` |
| --- | --- | --- |
| 0. as written, minus `:latest` | `lscr.io/linuxserver/kavita` | `postgres:16-alpine` |
| 1. drop the registry host | `linuxserver/kavita` | *(no registry)* |
| 2. drop the namespace | `kavita` | *(no namespace)* |
| 3. drop the tag | `kavita` | `postgres` |
| 4. `chrome.Truncate` | `kavi…` | `postg…` |

Specifics the implementation must get right:

- **`:latest` is dropped unconditionally, at rung 0.** It is the tag docker
  assumes when none is given, so `x` and `x:latest` are the same reference; it
  costs seven columns and carries no information. Every other tag survives to
  rung 3, because `16-alpine` and `16` are the difference between two databases.
- **"Registry host" is docker's own rule, not "the first segment".** The first
  slash-separated segment is a registry only if it contains a `.` or a `:`, or
  is exactly `localhost`. `linuxserver/kavita` has a namespace and no registry;
  `lscr.io/linuxserver/kavita` has both. Getting this wrong turns
  `linuxserver/kavita` into `kavita` one rung too early.
- **`docker.io/library/postgres` loses both parts by the normal rules** and
  needs no special case: `docker.io` contains a dot, `library` is a namespace.
- **Digest references keep a recognisable stub.** `postgres@sha256:9f86d081…`
  renders `postgres@9f86d08` (the first seven hex characters, the same length
  git uses for a short hash) at rung 0, and drops the digest at rung 3 as if it
  were a tag. A cell reading `postgres@…` says less than nothing.
- **Rung 4 exists because rung 3 can still overflow.** A name longer than the
  column has nothing left to shed, and `chrome.Truncate` is the documented
  backstop (`docs/DESIGN.md`: *`MaxHeight` is the backstop, not the fix* — same
  principle, one dimension over).

**The ambiguity this accepts:** `linuxserver/sonarr` and `hotio/sonarr` both
render `sonarr` at rung 2. That is fine and is the same trade every other
dropped column makes — the row is identified by its NAME, and the full
reference is one keypress away in the service details panel, which renders it
in the header subtitle untouched. Document it in the function comment so nobody
"fixes" it later by re-adding the namespace at a narrow width.

### D5. IMAGE keeps preferring the container's image; only the rendering changes

`renderMemberRow` already prefers `container.Image` over `svc.Image` when a
container exists. Leave that alone. It is the one place where runtime and file
genuinely disagree in a way worth seeing: after `p` (pull) and before `s`
(start), the container is still running the old image, and the row saying so is
correct. `ShortImage` renders whichever string the existing logic picked.

## Detailed changes

### 1. `src/components/chrome/Image.go` (new)

`ShortImage`, plus the two unexported helpers it needs: `splitRef` (registry /
namespace / name / tag-or-digest) and the rung builder. No dependency beyond
`strings` and `chrome`'s existing `runewidth` use through `Truncate`.

Parsing note: split the tag *after* splitting the path, and only on the last
path segment — `registry:5000/app` has a colon in the registry, and a naive
`strings.LastIndex(ref, ":")` finds the port, not a tag. The rule that gets
this right: find the last `/`; look for `:` and `@` only after it.

### 2. `src/components/chrome/Ports.go` (new)

`PortLabel` and `PublishedPorts`. Imports `types` from compose-go, which
`chrome` does not import today — that is fine and is not a layering violation:
`chrome` already imports `apptypes`, and the compose types are the app's domain
model everywhere else.

### 3. `src/components/groupdetailspanel/View.go`

In `renderMemberRow`:

- Delete the `ports` variable's runtime branch (`if container.Ports != ""`).
  Build the cell from `chrome.PublishedPorts(svc.Ports)` joined with `", "`,
  falling back to `"—"`.
- Change the `image` cell to `chrome.ShortImage(image, max(1, cols.image-1))`.
  Note the `-1`: the row already truncates every cell to one less than its
  column so two values keep a gap, and `ShortImage` must be given the width it
  is actually allowed to fill. **Do not let the generic `chrome.Truncate` in
  the cell loop run over the result** — it is a no-op when the string already
  fits, so leaving it in place is harmless and one less special case. Prefer
  leaving it.

### 4. `src/components/detailspanel/View.go`

Replace the inline port formatting in `configRows` with `chrome.PortLabel`.
This is the second caller that earns both files their place in `chrome`, and
it picks up the `HostIP` prefix from D2 — the one user-visible change on this
panel.

### 5. `docs/DESIGN.md`

The table in §*Narrow terminals: shed whole things* gains a row:

| Surface | Order lives in | Never dropped |
| --- | --- | --- |
| Image reference parts | `ShortImage`'s ladder (`chrome/Image.go`) | the image name |

and one paragraph noting that this is the first surface where the shedding
happens *inside* a unit rather than between units — the unit being the
reference, the parts being registry / namespace / tag. The rule survives the
move: a part is whole or absent, never a fragment, which is why rung 4 is a
last resort and not the mechanism.

### 6. `README.md`

No prose change needed. The screenshots are stale the moment this lands —
`demo/screenshot-groups.png` shows both defects — so re-record with
`demo/screenshots.tape` as part of this branch, per the note in `f5413bd`
(*the README-accuracy gate reopens with every chrome change*).

## Tests

All four new functions are pure, which is the point: this whole plan is
testable without a terminal, a docker daemon or a rig.

### `src/components/chrome/Image_test.go`

Table-driven, one case per row, asserting exact strings:

| ref | width | want |
| --- | --- | --- |
| `lscr.io/linuxserver/kavita:latest` | 40 | `lscr.io/linuxserver/kavita` |
| `lscr.io/linuxserver/kavita:latest` | 20 | `linuxserver/kavita` |
| `lscr.io/linuxserver/kavita:latest` | 15 | `kavita` |
| `ghcr.io/hotio/sonarr:latest` | 15 | `sonarr` |
| `postgres:16-alpine` | 40 | `postgres:16-alpine` |
| `postgres:16-alpine` | 10 | `postgres` |
| `postgres` | 40 | `postgres` |
| `docker.io/library/postgres:latest` | 20 | `postgres` |
| `linuxserver/kavita` | 30 | `linuxserver/kavita` |
| `linuxserver/kavita` | 12 | `kavita` |
| `registry:5000/app:v2` | 40 | `registry:5000/app:v2` |
| `registry:5000/app:v2` | 12 | `app:v2` |
| `postgres@sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08` | 40 | `postgres@9f86d08` |
| `postgres@sha256:9f86d0…` (as above) | 10 | `postgres` |
| `` (empty) | 10 | `` |
| `kavita` | 3 | `ka…` |
| `kavita` | 0 | `` |

Plus two invariants, both walking every width from 1 to 40:

- **`TestShortImageNeverRendersAFragmentOfARegistry`** — no output ever
  contains a `/` preceded by a string containing `.`; i.e. the registry is
  whole or absent, never `lscr.io/linuxse…`. This is the defect, pinned.
- **`TestShortImageNeverExceedsItsWidth`** — `runewidth.StringWidth(got) <=
  width` for every case. `Truncate` is the only rung allowed to produce an
  ellipsis, and it must still fit.

### `src/components/chrome/Ports_test.go`

`PublishedPorts` cases, built from the real fixtures above:

| input | want |
| --- | --- |
| `14533:4533/tcp` | `["14533"]` |
| `8080:8080/tcp`, `6881:6881/tcp`, `6881:6881/udp` | `["8080", "6881"]` |
| `4533/tcp` (no published) | `[]` |
| `53:53/udp` | `["53/udp"]` |
| `8000-8010:8000-8010/tcp` | `["8000-8010"]` |
| `127.0.0.1:14533->4533/tcp` | `["14533"]` (host IP not shown) |
| none | `[]` |

`PortLabel` cases: with and without `Published`, with and without `HostIP`,
with an empty `Protocol` (must default to `tcp`), and with `HostIP` set to
`0.0.0.0` (must **not** be printed — it is the default and means "everywhere").

### `src/components/groupdetailspanel/`

The existing suites must stay green untouched:
`TestMemberTableHeadingsNeverCollide`, `TestNarrowPanelsStayInsideTheirBox`.

Add one: **`TestMemberRowPortsComeFromTheFile`** — a member with `svc.Ports`
set and a container whose `Ports` string says something different renders the
file's ports. That pins D1 against a future revert-by-accident.

### Fixtures

`demo/fixtures/compose.yaml` already publishes ports in the 14000+ range and
uses four different registries — no fixture change needed for the visual check.
For the unit tests, build `types.ServicePortConfig` values in Go rather than
parsing YAML; they are three fields.

## Verification in the real app

The unit tests prove the strings. VHS proves the columns. Run
`demo/screenshots.tape` from a scratch directory and look at the group panel at
two widths (the tape's own, and one narrow enough to drop columns) for:

- no cell starting `0.0.0.0:`,
- no cell that is only a registry,
- the PORTS and IMAGE headings still aligned over their own columns,
- nothing wrapped to a second line.

## Effort / gain

**Half a day to a day**, and it is the highest ratio in `docs/plans/`. Four
pure functions, two call-site edits, two test files. No new keys, no new
messages, no new component, no docker calls, no config, no migration.

The gain is out of proportion to that: these two columns are 40% of the group
member table's width and currently carry approximately zero bits. They are also
in the first screenshot on the README, which makes this the cheapest available
improvement to what a stranger sees in the first ten seconds.

## Blast radius

- `chrome` gains two files. Nothing existing changes shape.
- `groupdetailspanel/View.go`: two lines inside `renderMemberRow`.
- `detailspanel/View.go`: one block inside `configRows`, plus the visible
  addition of `HostIP` to published ports that have one.
- No behaviour outside the two panels. No docker command changes. No file
  writes.

## Do not

- **Do not parse `container.Ports`.** If a future need arises for the runtime
  view, it is a separate function with its own name and its own tests, and the
  reason for wanting it goes in `docs/DESIGN.md` first.
- **Do not make either function width-aware beyond what is specified.**
  `PublishedPorts` returns a slice and lets the caller join and truncate;
  `ShortImage` takes a width because its whole job is the ladder. Adding a
  width parameter to `PublishedPorts` would put two shedding policies in one
  cell.
- **Do not add a PORTS column to the services list.** That list has one line
  per service and its own layout; this plan is about one table.
- **Do not drop the tag before the namespace.** The order in D4 is argued: a
  reader who has two `sonarr` images cares which publisher; a reader with one
  cares not at all about `:latest`. Reordering the rungs needs a reason written
  down beside them.
- **Do not "fix" the D4 ambiguity by falling back to the namespace at narrow
  widths.** That reintroduces a two-policy cell and breaks the fragment
  invariant the tests pin.

## Acceptance criteria

1. On the fixture stack, the group member table's PORTS column reads `14533`,
   `14378`, `14230`, `14989` … — one short number per row, different per row.
2. The IMAGE column reads `navidrome`, `audiobookshelf`, `kavita`, `sonarr` …
   at the default width, and widens to `deluan/navidrome` and
   `linuxserver/kavita` on a wide terminal.
3. A stopped service shows its ports, not `-`.
4. A service that publishes nothing shows `—`.
5. The service details panel shows `127.0.0.1:14533->4533/tcp` for a
   loopback-bound port and `14533->4533/tcp` otherwise.
6. Every existing test passes, plus the two new invariants, at every commit.
7. `demo/screenshot-groups.png` is re-recorded and shows the new columns.
