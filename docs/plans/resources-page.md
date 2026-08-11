# Plan: Networks and Volumes — a Resources Page, Read-Only First

> **Before you start.** Work on a feature branch of small commits, merged
> `--no-ff`; `go build ./... && go vet ./... && go test ./... && gofmt -l .`
> green at **every** commit, not just at the tip — `docs/ROADMAP.md`
> §Conventions is the full contract and `CONTRIBUTING.md` explains how a TUI
> gets tested. Behaviour that only shows on screen gets checked in the real app
> with VHS before it is committed.
>
> **Phase 0 of this plan fixes a latent bug that everything else depends on.**
> Read it first; it is one line and without it every correlation in this plan
> returns nothing.

Feature request (discovery session, nothing implemented): *"Think about a way
to manage docker networks and volumes in the app. This one needs more
investigation. Either deal with it by group / service, or have a dedicated page
to it, or both. Being able to create and link volumes and networks to groups or
services. Or see a network and being able to inspect which services / groups
use it. Please, investigate and come up with a plan on that."*

**Decided with the owner (2026-08-01): read-only inspector first; writes are a
later phase, after the write-safety gate.**

## Status — what the app knows, and the question it cannot answer

The service details panel prints a `Networks` row (names, comma-joined) and a
`Volumes` row (`"2 bind, 1 volume"`). That is the whole surface. Both are
derived from the service, so the app can only ever answer *"what does this
service use?"*

Every interesting question about a network or a volume runs the other way:

- **Who uses this volume?** (before deleting it)
- **Does this volume actually exist**, or will `up` create an empty one?
- **Why is my data gone?** — the question underneath the previous one, and the
  reason this plan exists at all. Rename `navidrome-data` to `navidrome_data`
  in the compose file and `up`: docker creates a new, empty volume, mounts it,
  and the old one keeps the data, unreferenced and invisible. The service comes
  up looking freshly installed. Nothing in the app, or in `docker compose ps`,
  or in lazydocker, says a word.
- **Which of these 42 volumes on my machine are still attached to anything?**
  On the author's box: 42 volumes, 29 with compose labels, 13 anonymous.

DESIGN.md §7's own checklist decides the shape: *"Group or service? If neither,
it probably belongs on a new page."* A volume is neither. It is a project-level
resource with its own identity, its own lifecycle, and — crucially — its own
list of consumers. That is a list-plus-details page, which is a shape this app
already implements twice.

## Research — measured on 2026-08-01

### What the compose file gives, for free

`types.Project` carries `Networks` and `Volumes` as maps keyed by the name used
inside the file, and each service carries its references:

```go
project.Networks   // map[string]NetworkConfig   — top-level networks:
project.Volumes    // map[string]VolumeConfig    — top-level volumes:
svc.Networks       // map[string]*ServiceNetworkConfig
svc.Volumes        // []ServiceVolumeConfig{Type, Source, Target, ReadOnly}
```

`ServiceVolumeConfig.Type` distinguishes `volume` (named) from `bind` (a host
path) — the two things `docker volume ls` and a file browser respectively care
about. Correlating services to resources is a double loop over data already in
memory: **no docker call is needed for the file half of this feature.**

`VolumeConfig.Name` is the *docker-side* name compose-go computed, and
`External` marks a volume the file declares but does not own.

### What docker gives

```
$ docker network ls --format json
{"Name":"arrstack_default","Driver":"bridge","Scope":"local","IPv4":"true",
 "Internal":"false","ID":"0ef4d67b2dae","CreatedAt":"…",
 "Labels":"com.docker.compose.network=default,com.docker.compose.project=arrstack,…"}

$ docker volume ls --format json
{"Name":"appwrite_appwrite-builds","Driver":"local","Scope":"local",
 "Mountpoint":"/var/lib/docker/volumes/…/_data","Size":"N/A","Links":"N/A",
 "Labels":"com.docker.compose.project=appwrite,com.docker.compose.volume=appwrite-builds,…"}
```

Three things to know:

1. **`Size` and `Links` are `"N/A"` from `volume ls`.** The real values come
   only from `docker system df -v --format json`, which returns one object with
   `Images[]`, `Containers[]`, `Volumes[]`, `BuildCache[]`; its `Volumes[]`
   entries carry `Size: "0B"` and `Links: "1"` properly. That call takes 1.4 s
   (measured; same call `docs/plans/docker-disk-usage.md` uses), which is why
   sizes load asynchronously here — D6.
2. **Compose labels every resource it creates**, with
   `com.docker.compose.project` and `com.docker.compose.volume` /
   `com.docker.compose.network` naming the *in-file* key. That is a direct
   join key, better than guessing from the `<project>_<name>` prefix, and it
   survives a project whose name contains an underscore.
3. **Network membership needs `inspect`**:
   `docker network inspect <name> --format '{{range .Containers}}{{.Name}} {{end}}'`
   returned `radarr sonarr feishin qbittorrent navidrome seerr prowlarr`. One
   call per network, so only for the selected one — D6.

Anonymous volumes (13 of 42 here) carry `com.docker.volume.anonymous=` and a
64-hex-character name. They belong to a container that declared a `VOLUME` in
its Dockerfile without a name in the file. They are real, they hold data, and
they are the single most common cause of "why is /var/lib/docker full".

### Phase 0's bug: the app's project name is a lie

`utils.ReadConfigFile` hardcodes:

```go
cli.WithName("cais")
```

Compose's own name resolution is `-p` flag → `COMPOSE_PROJECT_NAME` → the
file's top-level `name:` → the project directory's basename. `WithName` sits at
the top of that ladder, so it **overrides the file**. Measured against the
repo's own fixture, which declares `name: homelab`:

| | `project.Name` | volume `navidrome-data` resolves to |
| --- | --- | --- |
| today (`WithName`) | `cais` | `cais_navidrome-data` |
| with the option removed | `homelab` | `homelab_navidrome-data` |
| what `docker compose --file …` actually creates | `homelab` | `homelab_navidrome-data` |

And for `mocks/compose.yaml`, which has no `name:` key, removing the option
yields `mocks` — the directory basename, matching docker exactly.

**Today this is harmless**, because nothing in the app reads `project.Name` or
the resolved resource names; every action shells out to docker, which resolves
the name itself. The moment this page correlates the file against the daemon,
it becomes the difference between working and reporting that every volume in
the file is missing and every volume on the machine is an orphan.

## The four states — this is the feature

Every network and every volume is in exactly one of these, and the state is
what the page is *for*:

| State | Means | Why the user cares |
| --- | --- | --- |
| **defined + created** | in the file, exists in docker | normal; show size and consumers |
| **defined + absent** | in the file, not in docker yet | the stack has not been started, or was `down`ed with `-v` |
| **created + undefined** | exists in docker with this project's label, not in the file | **the dangerous one**: a renamed or deleted entry left its data behind |
| **external + missing** | file says `external: true`, docker does not have it | `up` will fail with a clear error — and the app can say so *before* the user presses `s` |

That last row is worth the page on its own. `external: true` means "I promise
this already exists"; when the promise is false, compose refuses to start and
the message arrives at the worst moment.

## Design decisions

### D1. A page, and it is the fourth tab

Per §7's checklist. `apptypes.PageTitles` gains `"Resources"`; `PageLabels`
maps it to `Resources`. Everything downstream derives:

- `pageForNavKey` matches digits by position, so `4` works with no edit, and
  the footer's `1-N page` hint is built from `len(apptypes.PageTitles)`.
- `PageShortcut` derives `alt+r` from the label; verified no collision —
  Groups→g, Services→s, Files→f, Resources→r.
- `AppModel.pages` needs the entry, or the page renders an empty body.

**This plan must rewrite the roadmap line that says the tabs are three.**
`docs/ROADMAP.md` §Decisions already taken with the owner says so explicitly,
and names the rule that outlives the count: *no tab ships empty*. This one does
not — a compose file with no top-level `networks:` still has the implicit
`default` network, which docker creates and which is worth showing.

`docs/plans/env-secrets.md` also adds a tab ("Env"). Whichever lands second
updates the same line and checks the shortcut letters again (`e` is free of
`g`/`s`/`f`/`r`).

### D2. One list, two kinds, a type tag

Networks first, then volumes, each sorted by name, in a single
`bubbles/list` — not two lists side by side, and not a tabbed sub-view.

```
  net  default                 1 service    created
  net  proxy                   3 services   external ✓
  vol  navidrome-data          1 service    2.1 GB
  vol  abs-config              1 service    absent
  vol  sonarr-config           —            orphan
```

Reasons: `/` filtering works across both without a mode; the existing focus
model (list ↔ details, `tab`) needs no new component; and the two kinds share
every column that matters. A three-character dim tag is cheaper than a second
panel.

### D3. The details panel answers "who uses this", both ways

Selecting a resource shows, using `renderPropTable` (the service panel's
existing two-column table, which is already generic):

- **Definition** — driver, `external`, labels, `driver_opts`, and for volumes
  the mountpoint when it exists.
- **State** — one of the four, in the status pill on the title row, coloured
  like every other pill (`StatusRunning` / `StatusStarting` / `StatusError`).
- **Used by** — the services from the *file* that reference it, **grouped by
  the group (profile) they belong to**, because that is how this app thinks:

  ```
  Used by      media: navidrome, audiobookshelf
               downloads: sonarr
  ```

- **Attached now** — the containers docker reports as attached, but **only when
  it differs from the file's list**. Two identical lists stacked on top of each
  other is noise; a difference is the whole story (a container still attached to
  a network the file no longer mentions).
- **Size** for volumes, when the async fetch has arrived (D6).

### D4. Reverse lookup for the group and service panels — one marker, not a section

The ask mentions "by group / service" as well. Resist duplicating the page into
the panels. One addition each:

- The service details panel's existing `Volumes` row gains a marker when any
  named volume it uses is **absent** or **external+missing**: `2 bind, 1 volume
  (1 missing)`. That is the fact worth interrupting for, and it is where the
  user already looks.
- The group details panel gets nothing. The member table has no room (see
  `docs/plans/group-table-legibility.md`) and the group's resources are the
  union of its services', which is a page's worth of information, not a cell's.

### D5. Read-only. No writes in Phase 1, and never a silent delete.

Phase 1 adds no key that changes anything. This is not timidity:

- The app rewrites the user's compose file and **there is still no backup and
  no undo** — `TODO.md` lists write safety as an open launch gate. Adding a new
  write surface before that lands is the wrong order.
- Attaching a volume to a service is a two-place edit (the service's `volumes:`
  list *and* the top-level `volumes:` map), which is a genuinely new kind of
  write. `utils.ApplyServiceFragment` handles one service's block;
  top-level-map editing is new machinery and deserves its own plan section
  (Phase 2), not a corner of this one.
- **Deleting a volume destroys data that nothing else on the machine has a copy
  of.** `x` on this page in Phase 1 does nothing. If deletion ever ships, it
  needs a typed confirmation (the volume's name, retyped), not `y`/`n` — a
  single keystroke is the right weight for `docker compose rm`, which can be
  undone by `up`, and the wrong weight for something that cannot.

### D6. Fetch discipline: fast now, slow later, one at a time

| Data | Call | When |
| --- | --- | --- |
| networks | `docker network ls --format json` | on page entry |
| volumes | `docker volume ls --format json` | on page entry |
| sizes | `docker system df -v --format json` (1.4 s) | dispatched with the above, applied when it lands |
| attached containers | `docker network inspect <name>` | on selection, for that one network |

Nothing here joins the five-second poll. The page refreshes on entry and on
`r`. (`r` is `Details.Restart` on the other pages; here the page has no docker
actions at all, so binding it to refresh is available — but **declare it as its
own binding in `keys`**, do not reuse `Details.Restart` for a different verb.
One verb, one binding, is the rule the whole `keys` package exists to enforce.)

If `docs/plans/docker-disk-usage.md` has landed, its `docker system df -v`
plumbing is shared; if not, this plan adds it and that one reuses it. Either
order works; whoever is second deletes their duplicate.

### D7. Correlation is one pure function, and it is the testable core

```go
// Resource is one network or volume, correlated across the file and the daemon.
type Resource struct {
    Kind        ResourceKind // Network | Volume
    Key         string       // the name used in the compose file
    DockerName  string       // what docker calls it, when it exists
    State       ResourceState
    External    bool
    Driver      string
    Mountpoint  string
    SizeBytes   int64  // -1 when unknown
    UsedBy      []ServiceRef // {Service, Groups}
    AttachedNow []string
}

// CorrelateResources joins what the file declares against what docker has.
// Pure: every input is data, so the whole state machine is a table test.
func CorrelateResources(
    project *types.Project,
    networks []DockerNetwork,
    volumes []DockerVolume,
) []Resource
```

Join rules, in order:

1. **By compose label.** A docker resource whose `com.docker.compose.project`
   equals `project.Name` and whose `com.docker.compose.volume` /
   `.network` equals the file's key is that resource. Authoritative.
2. **By computed name.** `VolumeConfig.Name` / `NetworkConfig.Name` — what
   compose-go says docker will call it. Catches resources created by an older
   compose that labelled differently.
3. **By literal name**, for `external: true` entries, which carry no project
   prefix and no label because the project did not create them.

Anything left over from the daemon side that carries *this project's* label is
**created + undefined**. Anything left over that carries *another* project's
label, or none, is not this project's business and **is not shown** — that is
`docs/plans/adopt-unmanaged-containers.md`'s territory, and mixing them turns
this page into a machine-wide docker browser, which is not what it is for.

### D8. Bind mounts are listed, not managed

A `bind` volume is a host path, not a docker object: no size, no driver, no
lifecycle. Showing them in the same list as named volumes would put two
different kinds of thing under one heading.

They appear in the *service* panel (they already do, in the count) and not on
this page — with one exception worth the code: a bind mount whose **host path
does not exist** is a real, silent, common failure (docker happily creates an
empty directory, and the service comes up with no library). If a later phase
adds one line to the service panel for that, it is welcome; it is not this
page's job.

## Phases

### Phase 0 — the project name (do this first, alone, one commit)

Remove `cli.WithName("cais")` from `utils.ReadConfigFile`. Verify:

- the fixture (`name: homelab`) resolves to `homelab`,
- `mocks/compose.yaml` (no `name:`) resolves to `mocks`,
- a directory whose basename is not a legal project name (spaces, uppercase —
  try `My Stack`) still loads. **This is the reason the option was probably
  there.** If it errors, keep `WithName` but pass a *derived* name (normalized
  basename) rather than a constant, and only when the file declares none.

Ship it as its own commit with a test, whatever the outcome. It is a
correctness fix independent of this page.

### Phase 1 — the page, read-only (~2–3 days)

1. `apptypes.Pages` entry, `AppModel.pages` wiring, nav renders `4 Resources`.
2. `utils/DockerNetworks.go`, `utils/DockerVolumes.go` — exec + pure parser
   split, exactly like `DockerComposePs`/`ParseContainers`.
3. `utils/Resources.go` — `CorrelateResources` and its types.
4. `components/resourceslist/` — the list, typed items, type tag, state column.
5. `components/resourcedetailspanel/` — the details, reusing
   `chrome.PanelFrame`, `chrome.PanelRule`, and a local copy of the prop-table
   renderer **only if** `renderPropTable` cannot be shared; prefer promoting it
   to `chrome` at that point, since this is its second caller and that is the
   rule.
6. `cmds/GetDockerResources.go`, and the page-entry refresh.
7. The service panel's "(1 missing)" marker (D4).
8. Docs: README bullet + screenshot, DESIGN.md section, ROADMAP tab-count line.

### Phase 2 — writes (~3–4 days, **after** the write-safety gate)

Not specified in detail here on purpose; it needs its own plan once Phase 1 has
been used for a while and the actual friction is known rather than guessed.
What is known now:

- The primitive is `utils.ApplyTopLevelEntry(doc, "volumes", name, body)` —
  read-modify-write on the whole document, comment- and order-preserving, the
  same discipline `GroupTags.go` follows.
- Attaching is two edits in one pass, or the file is left inconsistent.
- The two obvious verbs are `n` (new volume/network) and `a` (attach to the
  selected service — but `a` is About; pick at the time, from `keys`).

### Phase 3 — deletion

Only with the typed-confirmation design from D5, and only after someone asks
for it. `docker volume rm` on a volume this app cannot restore is the single
most destructive thing in the whole feature list.

## Tests

### `src/utils/Resources_test.go` — the whole state machine, as a table

This is where the value is. Every case is data in, data out:

| file declares | docker has | want state |
| --- | --- | --- |
| `volumes: {data: {}}` | `homelab_data` labelled project=homelab, volume=data | defined+created |
| `volumes: {data: {}}` | nothing | defined+absent |
| nothing | `homelab_old` labelled project=homelab, volume=old | created+undefined |
| `volumes: {shared: {external: true}}` | `shared` (no labels) | defined+created |
| `volumes: {shared: {external: true}}` | nothing | **external+missing** |
| `volumes: {data: {}}` | `otherproject_data` labelled project=otherproject | defined+absent (and the other project's volume is **not** listed) |
| `networks: {default: {}}` implicit | `homelab_default` | defined+created |

Plus consumers: a volume used by two services in different profiles produces
`UsedBy` grouped by group, in file order, deduplicated.

### `src/utils/DockerNetworks_test.go`, `DockerVolumes_test.go`

Parsers, against **captured** output — paste real lines from
`docker network ls --format json` and `docker volume ls --format json`. The
HEALTH-column lesson in `TODO.md` is the standing reason: a fixture invented
from the struct tests nothing but itself. Include one anonymous volume (the
64-hex name with `com.docker.volume.anonymous=`) and one network with empty
`Labels`.

### `src/utils/ReadConfigFile_test.go` (Phase 0)

The three name-resolution cases from the table in the research section, asserted
as exact strings.

### Component tests

Follow `groupdetailspanel`'s: render at descending widths, assert nothing wraps
and nothing renders as a fragment (`TestNarrowPanelsStayInsideTheirBox` is the
model). The state column must never truncate to a prefix that reads as another
state — `"external"` and `"orphan"` truncated to four characters are `"exte"`
and `"orph"`, which is fine; `"created"` and `"created+"` would not be. Keep
the state words short and distinct at four characters.

## Edge cases and unknowns

- **The implicit `default` network.** compose-go materialises it into
  `project.Networks` even when the file says nothing. It should appear, and its
  state is real. Verify it carries `Name: "<project>_default"`.
- **`network_mode: host` / `container:`** services reference no network object;
  they contribute no consumers. Do not let them produce an empty-string entry.
- **A service using a volume the file never declares** is a compose error and
  the project would not have loaded — compose-go validates it. No handling
  needed, but do not assume the reverse: a *declared* volume that no service
  uses is legal and common, and shows `UsedBy: —`.
- **Volume size is `"N/A"`** on drivers other than `local`, and on `local` when
  `system df -v` has not run. `-1` means unknown; render `—`, never `0 B`.
- **A very large `docker system df -v`** on a machine with 76 images returns a
  large JSON object. Only `Volumes[]` is needed; decode into a struct with just
  that field so the rest is skipped by `encoding/json` for free.
- **Permissions**: `docker network inspect` needs the same socket access as
  everything else. If it fails, show the resource without the "attached now"
  row rather than failing the page.
- **A project whose name changed** (the file gained a `name:` key after the
  stack was started) makes every existing resource look like another project's,
  so they vanish rather than showing as orphans. Correct, and worth a sentence
  in the docs: the page is scoped to the project the file names.

## Effort / gain

**Phase 0: an hour.** Phase 1: **2–3 days**, of which the correlation function
and its table are half a day and the two components are the rest. Phase 2:
3–4 days, later, with its own plan.

The gain, honestly stated: this is the only plan in the list that answers a
question the user cannot answer today by any other means short of `docker
volume ls | grep` and reading their own compose file side by side. The "created
+ undefined" state in particular is a data-loss detector, and the "external +
missing" state turns a confusing `up` failure into a red pill on a list.

It is also the least urgent of the four pre-launch items, which is why the
recommended order puts it last among them.

## Blast radius

- **Phase 0 touches every consumer of `ReadConfigFile`** by changing
  `project.Name`. Nothing reads it today (verified: no `project.Name` or
  `.Name` on a project outside compose-go), which is exactly why it is safe to
  fix now and dangerous to fix later.
- A new page adds a tab, which changes the nav, the digit range, and the
  footer's `1-N` hint — all of which derive from `PageTitles` and need no edit,
  but all of which are visible in every screenshot.
- Three new docker commands, none of them on the poll.
- No writes in Phase 1.

## Do not

- **Do not skip Phase 0.** Every correlation depends on the project name being
  the one docker uses.
- **Do not show other projects' resources.** D7. This page is scoped to the
  loaded file's project; machine-wide browsing is a different tool.
- **Do not add delete in Phase 1**, and do not add it later behind a `y`/`n`
  confirm. D5.
- **Do not put bind mounts in the resource list.** D8.
- **Do not poll any of these commands.** D6.
- **Do not build a second prop-table renderer.** If the service panel's is not
  reachable, promote it to `chrome` — that is what the second-caller rule is
  for.
- **Do not reuse `Details.Restart` for the refresh key.** Declare a new
  binding; one verb, one binding.
- **Do not let the page become a docker dashboard.** It answers questions about
  *this compose file's* resources. Images, build cache and containers belong to
  `docs/plans/docker-disk-usage.md` and
  `docs/plans/adopt-unmanaged-containers.md` respectively.

## Acceptance criteria

1. `4` (and `alt+r`) opens a Resources page listing the fixture stack's nine
   volumes and its default network.
2. With the stack down, every volume reads **absent**; after `s` on a group,
   the ones that group uses read **created**.
3. Renaming a volume in the file and reloading shows the old one as
   **orphan** and the new one as **absent** — the data-loss detector, working.
4. A volume with `external: true` that does not exist reads **external
   missing** in `StatusError` colour.
5. Selecting a volume lists the services that use it, grouped by group; the
   list matches what `grep` on the compose file says.
6. Selecting a network lists the containers docker reports attached.
7. Volume sizes appear a second or two after the page opens, and `—` where
   docker reports none.
8. The service details panel's `Volumes` row says `(1 missing)` when one of its
   named volumes is absent.
9. `project.Name` equals what `docker compose --file … config --format json`
   reports for the same file, for both a file with `name:` and one without.
10. No docker command runs on the five-second poll that did not run before.
