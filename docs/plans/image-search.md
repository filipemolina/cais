# Plan: Search Docker Images From Inside the TUI

> **Before you start.** Work on a feature branch of small commits, merged
> `--no-ff`; `go build ./... && go vet ./... && go test ./... && gofmt -l .`
> green at **every** commit, not just at the tip — `docs/ROADMAP.md`
> §Conventions is the full contract and `CONTRIBUTING.md` explains how a TUI
> gets tested. Behaviour that only shows on screen gets checked in the real app
> with VHS before it is committed. **Step 2 of the post-alpha order** — but only **Phase 1** is step 2; Phases 2-4 come after the rest of the lifecycle. Do it after the package restructure (step 1), so new files land in the new layout.

Feature request: *"a feature where the user could search for docker images from
inside the TUI, so it would be even easier to set up the entire server without
having to go out of the TUI."*

This plan was researched in a session on 2026-07-31 that died before it wrote
anything (`pi-session-…d5c5.html` in the repo root). The research is preserved
and re-verified below; every empirical claim was re-run against Docker
29.6.0 / Compose v5.1.4 on 2026-07-31 unless marked otherwise.

## Status of the feature — an honest reframing

**Searching is the easy half, and it is not the half that is missing.**
`docker search nginx` already works in the terminal the user is one keystroke
away from. What the user actually asked for — *set up the entire server without
leaving the TUI* — needs the other half: turning a search result into a service
in their compose file.

**The app cannot add a service at all today.** Verified:

- `utils.ApplyServiceFragment` (`src/utils/ServiceFragment.go:91`) returns
  `service %q not found in compose file` when the name is not already there. It
  edits; it cannot create.
- `utils.WriteNewComposeFile` (`src/utils/GroupTags.go:249`) seeds a service,
  but only into a **brand-new file**, and it refuses to touch an existing one.
- `docs/DESIGN.md` §Editing services says it outright: *"`E` still opens the
  whole compose file in `$EDITOR`, which is the only way to add a service or
  touch top-level keys."*
- `n` (`List.New`) is bound only on Home, gated by `m.activePage == "Home"`
  (`src/model/Update.go:441-444`). On the Services page it does nothing.

So the shape of this feature is: **`n` on the Services page adds a service, and
image search is how you fill in the `image:` field.** Search alone would be a
lookup tool that duplicates a command the user already has. Search plus insert
is the thing they asked for.

**Verdict up front: worth doing, in that order.** Phase 1 (add a service) is
useful with the user typing the image name by hand and is a prerequisite for
two other plans (`docs/plans/ai-service-authoring.md` needs exactly the same
insert path). Phase 2 (search) makes it good. Phase 3 (tags) is what stops it
producing `image: postgres` with no tag, which is the wrong answer for a
self-hosted stack.

## Research — what actually exists, measured

### `docker search` works, is authenticated for free, and is the right transport

```console
$ docker search --limit 3 --format json linuxserver
{"Description":"A Sonarr container, brought to you by LinuxS…","IsOfficial":"false","Name":"linuxserver/sonarr","StarCount":"2127"}
{"Description":"A Radarr container, brought to you by LinuxS…","IsOfficial":"false","Name":"linuxserver/radarr","StarCount":"1796"}
{"Description":"A Jackett container, brought to you by Linux…","IsOfficial":"false","Name":"linuxserver/jackett","StarCount":"1205"}
```

Facts that matter, in order of how much they shape the design:

- **`--format json` emits one JSON object per line** (not an array), with
  exactly four fields — `Name`, `Description`, `StarCount`, `IsOfficial` — and
  **all four are strings**, including the number and the boolean. Decode into
  `struct{ Name, Description, StarCount, IsOfficial string }` and convert;
  decoding `StarCount` as `int` fails.
- **Descriptions are truncated with a Unicode ellipsis (`…`)** unless
  `--no-trunc` is passed. Pass `--no-trunc`; the panel does its own truncation
  and knows its own width.
- **It goes through the daemon.** The CLI's `runSearch` calls the moby client's
  `ImageSearch` and attaches the user's Docker Hub credentials from
  `~/.docker/config.json` when they exist (`docker/cli`
  `cli/command/registry/search.go`, read 2026-07-31). Two consequences: a user
  who has run `docker login` gets their authenticated quota for free and the
  app never touches a credential, and the app inherits whatever daemon the user
  is pointed at (`DOCKER_HOST`, contexts) with no extra work.
- **It is Docker Hub only.** No GHCR, no lscr.io, no Quay. This is a real
  limitation for the self-hosting audience — `lscr.io/linuxserver/sonarr` is
  the canonical LinuxServer address — but the same images are all mirrored on
  Hub under `linuxserver/*`, which is what the search returns.
- **`is-automated` is gone** (deprecated in CLI v25.0, removed by v28.2), which
  is why the JSON has no `IsAutomated`. Do not filter on it.
- `--filter is-official=true` and `--filter stars=N` work; `--limit` caps
  results (default 25).

Shelling out to `docker` is also simply what this app does: `utils.DockerCompose`,
`DockerComposePs`, `DockerStats` and `DockerLogs` are all `exec.Command("docker", …)`.
A new `utils/DockerSearch.go` is the fifth of the same shape, not a new pattern.

### The Hub HTTP API works but should not be the primary path

`https://hub.docker.com/v2/search/repositories/?query=nginx&page_size=2`
returns 200 unauthenticated (verified 2026-07-31) with
`count`/`next`/`previous`/`results[]`, each result carrying `repo_name`,
`short_description`, `star_count`, **`pull_count`**, `repo_owner`,
`is_official`. It is richer than `docker search` — pull counts are a better
popularity signal than stars for infrastructure images.

**It is also undocumented.** The current Hub OpenAPI description
(`https://docs.docker.com/reference/api/hub/latest.yaml`) does not contain this
endpoint; it is a legacy path the web UI and older clients use. An undocumented
endpoint can change shape or vanish without a deprecation window, and when it
does the failure lands on the user as an empty list. Use `docker search`.

### Tags need the HTTP API, and tags are not optional

`docker search` returns **repository names only — no tags**. A compose service
needs `image: repo:tag`, and `image: postgres` means `postgres:latest`, which
for a self-hosted database is the single worst pin available.

Tag listing is a different, *documented-by-use* endpoint, verified 2026-07-31:

```console
$ curl -s "https://hub.docker.com/v2/repositories/library/nginx/tags?page_size=3&ordering=last_updated"
{"count":1283,"next":"…","results":[{"name":…,"images":[{"architecture":"amd64",…}],…}]}

$ curl -s -o /dev/null -w "%{http_code}" "https://hub.docker.com/v2/repositories/linuxserver/sonarr/tags?page_size=3"
200
```

Notes: official images live under the `library/` namespace (`library/nginx`),
unofficial ones under their own (`linuxserver/sonarr`); `ordering=last_updated`
is what makes the list useful; each tag carries an `images[]` array with
`architecture`, which is how the picker can say "this tag has no arm64 build" —
the single most common surprise for a self-hoster on a Raspberry Pi.

### Rate limits

- **Hub abuse limit:** applies to *all* requests to Hub properties — web, API
  and pulls — counted per IPv4 address or per IPv6 /64, "in the order of
  thousands of requests per minute", and it answers with a bare
  `429 Too Many Requests` (docs.docker.com/docker-hub/usage, retrieved
  2026-07-31). A human typing into a search box cannot reach this; a
  keystroke-triggered search with no debounce, on a shared IP, can.
- **Pull limits:** 100 pulls per 6 hours unauthenticated, 200 for a free
  authenticated Personal account (same source). This does not bind on search,
  but it binds on the `p` (pull) the user presses immediately afterwards, and
  it is worth one line in the docs.

### Prior art

**lazydocker does not do this.** It manages images that are already on the host
— list, remove, inspect — and has no Hub search (README and docs, 2026-07-31).
k9s has no analogue either; its object is a cluster, not a registry. The
nearest precedents are Docker Desktop's own image search pane and the
`docker search` CLI itself. There is no TUI convention to follow here, which
means the design should follow *this app's* conventions and nothing else.

## Scope

**In:** `n` on the Services page adds a service to the loaded compose file;
inside that flow, `/`-style search of Docker Hub by name; a tag picker for the
chosen repository; the new service lands in the existing inline editor for
review before anything is written.

**Out, stated:**

- **Registries other than Docker Hub.** `docker search` cannot do it. A user
  who wants `ghcr.io/...` types it — the image field is free text, and it stays
  free text for exactly this reason.
- **Pulling as part of the flow.** `p` already pulls a service. Adding a
  service does not pull it; starting it does.
- **Image inspection** (layers, size, Dockerfile, vulnerabilities). Different
  feature, different panel, no compose relevance.
- **Local image browsing** (`docker images`). That is lazydocker's job and it
  answers a different question ("what is on this box") than this feature ("what
  should I run").
- **Editing an existing service's image through search.** Tempting one-liner;
  it is a different verb on a different object, and `e` already edits.

## Design decisions

### D1. `n` on the Services page = "new service"

`List.New` is `n` and its help text is "new" (`src/keys/Keys.go:172`). It is
handled centrally in `src/model/Update.go:441` and gated to Home. Widen that
gate to the Services page rather than adding a key: one verb, one binding,
which is the rule the `keys` package exists to enforce.

`n` is free on Services today — the services list installs `ListKeyMap`
(`Keys.go:240+`), which does not claim it, and nothing in `ServicesList.Update`
matches it. Verified by reading both.

### D2. The flow is a modal that ends in the inline editor

```
n ──> AddServiceModal
        step 1: service name      (validated: compose service-name rules)
        step 2: image             (free text, with "/ to search Hub")
          └─ / ──> search results list ──> Enter ──> tag list ──> Enter
        step 3: Enter ──> write minimal service, open the inline editor on it
```

The modal collects the two things a service cannot exist without (a name and an
image) and then **gets out of the way**: the service is written as a minimal
two-line fragment and the existing inline editor (`e`'s editor, the `textarea`
in `DetailsPanel`) opens on it, focused, so the user adds ports and volumes in
the same YAML they would have hand-written.

This is the design the repo already argued for. `docs/DESIGN.md` §Editing
services rejects forms outright: *"A form has to pick a representation for
every field … A form is also a standing tax: any field nobody modelled is a
field nobody can edit."* A "new service" wizard with fields for ports, volumes,
environment and restart policy would re-introduce exactly that. Two fields,
then YAML.

**Extract the step from `CreateComposeFileModal`; do not copy it.** The
bootstrap flow already asks these two questions. `CreateComposeFileModal`
(`src/components/CreateComposeFileModal.go`) has a `stepServiceFields` step
holding a name input and an image input, tab between them
(`keys.Overlay.NextField`), esc to cancel, and all three validations this modal
needs — empty name, empty image, invalid name — with their messages already
written (`updateServiceFields`, `CreateComposeFileModal.go:127`;
`isValidServiceName`, `:166`). The only thing that differs at the end is which
command it emits: `cmds.CreateComposeFile(path, name, image)` there,
`cmds.AddService(name, image)` here.

The argument for extracting rather than copying is Phases 2 and 3. If
`AddServiceModal` is a clone, `/`-to-search and the tag picker get built into
the clone — and the **bootstrap flow becomes the one place in the app where
image search does not work.** That is the first service a new user ever
creates, in an empty directory, at the exact moment they are least likely to
know the image name. Every later phase would widen the gap.

So: one shared component — call it `ServiceFieldsStep` — owning the two inputs,
their validation and their key handling, parameterised by its title and by what
it emits on submit. `CreateComposeFileModal` renders it as its third step;
`AddServiceModal` renders it as its whole body. Phases 2 and 3 then add search
and tags in one place and both flows get them.

**The shared component takes exactly two parameters: a title string, and a
function that turns `(name, image)` into the `tea.Cmd` to run on submit.**
That is the whole contract — do not add a third. If a third looks necessary,
that is the signal to stop and copy the step into `AddServiceModal` instead;
sixty duplicated lines beat a shared component with five knobs. Nothing else
about the flow changes in that case, so it is a local decision, not a redesign.

Do this **after** the `component-package-restructure` (step 1 in
`docs/ROADMAP.md`), so the extracted file is placed in the new layout rather
than moved twice.

There is a second-order cleanup available once `AddServiceFragment` exists:
`WriteNewComposeFile` currently seeds the first service itself, and could
instead write an empty `services:` map and hand off to `AddServiceFragment` —
collapsing two write paths into one. Phase 1's test list already covers
inserting into a file with no `services:` key, which is the case that makes it
work. Optional, and only worth doing if it comes out smaller.

### D3. Search is a step *inside* the image field, not a page

`/` inside the image field swaps the modal's body for a results list. Not a new
page (nothing to come back to), not a second modal (`GroupNameModal` →
`ServiceChecklistModal` is the app's precedent for handing off, but that is two
*questions*; this is one question with a lookup attached).

- Search runs on **Enter, not on keystroke.** No debounce timer to test, no
  accidental burst against the Hub abuse limit, and a `tea.Cmd` per deliberate
  search is trivially testable. The status line shows "searching…" using the
  existing spinner (`components/spinner.go`).
- Results render name / stars / official-badge / description, sorted as Hub
  returns them (relevance), `--limit 25`.
- Enter on a result moves to the tag step. Esc returns to the typed image field
  with what was typed intact — a failed search must never cost the user their
  typing.

### D4. The tag step defaults to something defensible

The tag list comes from the Hub HTTP API (§Research), newest first, showing tag
name, age, and the architectures it was built for. Above the list sit two fixed
entries, in this order:

1. **the tag the user is most likely to want** — the newest tag that is not
   `latest` and parses as a version (`v?\d+(\.\d+)*`), because pinning is what
   makes a self-hosted stack reproducible;
2. **`latest`**, labelled "moves without warning".

If the API call fails — offline, 429, undocumented endpoint changed — the step
is **skipped with a message**, not blocked: the image field keeps the repo name
and the user types the tag. Every network dependency in this feature degrades
to "type it yourself", because typing it yourself is what they do today.

### D5. The insert primitive: `utils.AddServiceFragment`

New function in `src/utils/ServiceFragment.go`, next to the one it mirrors:

```go
// AddServiceFragment inserts a new service into the compose file at fileName.
//
// It is ApplyServiceFragment's opposite number: same fragment shape, same
// validation, same atomic write, but it refuses when the name is already
// taken instead of when it is absent. Insertion is at the end of the
// services: mapping, which is where a reader expects a new entry and the
// only position that never reorders the user's file.
func AddServiceFragment(fileName string, serviceName string, fragment []byte) error
```

Implementation is `ApplyServiceFragment` (`ServiceFragment.go:64-105`) with the
replace loop inverted:

1. `parseServiceFragment(serviceName, fragment)` — unchanged, gives the value
   node and enforces the one-service-single-mapping shape;
2. `readComposeNode` → `servicesMappingNode`;
3. scan for `serviceName`; **if found, return an error** naming the collision;
4. append key node + value node to `servicesNode.Content`;
5. `encodeNode` → `ValidateComposeCandidate(filepath.Dir(fileName), candidate)`
   → `ReplaceFileAtomically`.

Nothing new is invented: steps 1, 2, 5 are the existing calls in the existing
order, which is what makes this safe to write and cheap to review.

Two shapes to handle that `ApplyServiceFragment` never meets:

- **`services:` is absent or null.** A compose file with only `name:` and
  `volumes:` is legal. `servicesMappingNode` must create the mapping in that
  case, or `AddServiceFragment` returns a clear error and the user is told to
  use `E`. Decide by reading `servicesMappingNode`; whichever it does today,
  the test for this case is not optional.
- **A duplicate name** returns an error the modal shows inline on the name
  field, the same way `CreateComposeFileModal` shows "already exists".

### D6. Errors are the app's existing errors

Search failures are foreground errors (the user pressed a key and is waiting):
they go down the `reportForegroundError` path already used by docker actions,
which means an error modal for a hard failure and the status line for a soft
one. No new error mechanism. Specifically:

| Failure | What the user sees |
|---|---|
| `docker` not on PATH | the app's existing "docker not found" surface — it already fails this way for every action |
| daemon down | docker's own stderr, verbatim, in an error modal |
| `--format json` unsupported (old CLI) | "image search needs a newer docker CLI; type the image name instead" — the field still works |
| 429 from the tag API | "Docker Hub is rate-limiting; type the tag instead", tag step skipped |
| no results | "no images matched" in the list body, not an error |

## Phases

Each phase is a feature branch of small commits, merged `--no-ff`, per
`docs/ROADMAP.md` §Conventions. `go build ./... && go vet ./... && go test ./...`
and `gofmt -l .` green at **every** commit.

### Phase 1 — Add a service by hand

The whole feature minus the network. Ships alone.

| File | Change |
|---|---|
| `src/utils/ServiceFragment.go` | `+AddServiceFragment` (D5) |
| `src/utils/ServiceFragment_test.go` | insert into a normal file; into a file with no `services:`; duplicate name refused; comments and key order elsewhere in the file unchanged; the file is untouched when validation fails |
| `src/components/ServiceFieldsStep.go` | new — the name+image step **extracted** from `CreateComposeFileModal.stepServiceFields` (D2): two inputs, tab between them, the three existing validations, parameterised by title and submit-message factory |
| `src/components/CreateComposeFileModal.go` | its third step becomes the extracted component. Behaviour identical — the bootstrap flow must look and behave exactly as it does today, which its existing tests already pin |
| `src/components/AddServiceModal.go` | new — thin: the extracted step plus the `n`-on-Services entry point |
| `src/components/AddServiceModal_test.go` | validation messages, step transitions, Esc at each step |
| `src/cmds/OpenAddServiceModal.go`, `src/cmds/AddService.go` | new — the open command and the request/response pair |
| `src/model/Update.go` | widen the `keys.List.New` gate (`:441`) to the Services page; handle `AddServiceMsg` → reload config → select the new service → open the inline editor on it |
| `src/model/rig_test.go`-style test | `n` on Services opens the modal; a completed flow leaves the new service selected with the editor open |
| `README.md`, `docs/DESIGN.md`, `TODO.md` | `n` on Services; why two fields and then YAML (D2) |

Acceptance: with a compose file loaded, `n` on the Services page, a name, an
image, Enter — the service exists in the file, the file's other services are
byte-identical apart from the addition, and the inline editor is open on the
new service. `docker compose config` on the resulting file exits 0.

### Phase 2 — `/` searches Docker Hub

| File | Change |
|---|---|
| `src/utils/DockerSearch.go` | new — `SearchImages(term string, limit int) ([]ImageResult, error)`; `exec.Command("docker", "search", "--format", "json", "--no-trunc", "--limit", …, term)`; decode **line-delimited** JSON with all-string fields (§Research) |
| `src/utils/DockerSearch_test.go` | decoding fixtures captured from the real command (including the string `"true"`/`"false"` for `IsOfficial` and a `StarCount` of `"0"`); a stderr-only failure; empty output |
| `src/cmds/SearchImages.go` | new — the search command and its result message |
| `src/components/ServiceFieldsStep.go` | `/` swaps the body to the results list; Enter fills the image field; Esc restores the typed text. In the shared step, not in `AddServiceModal` — which is what makes the bootstrap flow inherit search for free (D2) |
| Tests | results render name/stars/official; Enter fills the field; a failed search leaves typing intact; **and the same `/` works from the bootstrap flow's service step** |

Acceptance: `/` then `sonarr` then Enter lists `linuxserver/sonarr` among the
results; Enter on it puts `linuxserver/sonarr` in the image field; with `docker`
removed from PATH the field still accepts typing and the error is legible.

**No test in this phase may hit the network.** `SearchImages` takes the
command runner as a value the test can substitute, the same way the docker
action tests already work (`src/components/DockerAction_test.go`).

### Phase 3 — the tag picker

| File | Change |
|---|---|
| `src/utils/HubTags.go` | new — `ListTags(repo string, limit int) ([]Tag, error)` against `https://hub.docker.com/v2/repositories/{ns}/{repo}/tags?page_size=N&ordering=last_updated`; `library/` prefix for un-namespaced repos; `net/http` with an explicit **5 s timeout** and a `stitch/<version>` User-Agent |
| `src/utils/HubTags_test.go` | `httptest.Server` fixtures: normal page, 429, malformed JSON, a repo with no arm64 build |
| `src/components/AddServiceModal.go` | tag step (D4): pinned suggestion, `latest`, then the list; skip-with-message on error |
| Tests | the version-tag heuristic picks the right default; a 429 skips the step and keeps the flow alive |

Acceptance: choosing `linuxserver/sonarr` offers a version tag above `latest`;
choosing it yields `image: linuxserver/sonarr:<tag>`; with the network
unplugged the flow still completes with a hand-typed tag.

### Phase 4 — docs and demo

README (the `n` flow and the Hub-only limitation), `docs/DESIGN.md` (why search
is a step inside a field and not a page, why the app never talks to a registry
except for tags), `TODO.md`, and a VHS tape of the flow per house convention.

## Edge cases and unknowns

1. **No compose file loaded.** `n` must do nothing (or say so) — every page is
   already gated on a loaded file; follow the existing gate rather than adding
   a second rule.
2. **Read-only compose file / directory.** `ValidateComposeCandidate` writes a
   temp file *into the compose file's directory* (`ServiceFragment.go:148-158`),
   so a read-only directory fails at validation with a confusing message. Worth
   one explicit error path: "cannot write in <dir>".
3. **The name the user picks collides with a service in the file** — refused by
   `AddServiceFragment`, shown inline (D5).
4. **`services:` missing or null** — D5; test it.
5. **An image name with a registry prefix** (`ghcr.io/…`, `lscr.io/…`) typed by
   hand: accepted verbatim, never searched. Do not "helpfully" strip it.
6. **Digest pins** (`repo@sha256:…`): accepted verbatim, no tag step.
7. **A tag with no build for the host architecture.** Shown in the list; not
   blocked. The app cannot know the host arch of the *docker daemon* reliably
   (remote `DOCKER_HOST`), and guessing wrong to block a valid choice is worse
   than a label.
8. **Hub search is Docker Hub only** — stated in the UI (the list header says
   "Docker Hub"), not hidden.
9. **`docker search` on an old CLI.** `--format json` exists in v24 and v25's
   formatter sources (checked 2026-07-31); older than that is untested. Detect
   by failure, degrade to typing (D6), do not version-sniff — parsing
   `docker --version` to gate a feature is a maintenance liability.
10. **Rate limiting.** Search-on-Enter only (D3); no per-keystroke requests;
    the tag call is one request per chosen repository.
11. **Typosquatting.** Search results are attacker-influenced content: a
    repository named `ngjnx` with a plausible description can appear. The
    official badge is shown, star counts are shown, and **the user reviews the
    YAML in the editor before anything runs**. Do not add an auto-start.
12. **Very long descriptions / non-ASCII.** `--no-trunc` plus the panel's own
    width-aware truncation (`truncate` in `GroupDetailsPanel.go`); the repo
    already uses `go-runewidth` for this.
13. **The new service and groups.** A new service belongs to no group. That is
    correct (§3 of `docs/DESIGN.md`: a group is a `profiles:` tag); the user
    adds it to one on the Groups page. Do not offer group membership in this
    modal.

**Unknowns worth ten minutes before Phase 1:** whether `servicesMappingNode`
creates a missing `services:` mapping or errors; and whether the inline editor
can be opened programmatically on a named service or whether that needs a small
addition to the details panel's message handling.

## Effort / gain

| Option | Effort | Gain | Verdict |
|---|---|---|---|
| 0 — do nothing | 0 | the user leaves the TUI to run `docker search`, and leaves it again to `E` the file to add a service | fails the ask twice |
| **1 — Phase 1 only** | **~1 day** | the app can add a service; the biggest hole in "manage compose from the TUI" is closed; unblocks two other plans | **do this regardless** |
| **2 — Phases 1–3** | **~3 days** | the literal ask, degrading to typing at every network step | **recommended** |
| 3 — 2 + registry abstraction (GHCR, Quay) | +3 days | serves the `lscr.io` habit properly | defer; Hub mirrors the same images and free text already works |

## Blast radius

| Area | Effect |
|---|---|
| `src/keys` | none — `n` is already declared; only its gate widens |
| Footer / `?` overlay | the Services scope gains "n new" automatically, from the key catalog |
| `src/utils` | +2 files (`DockerSearch.go`, `HubTags.go`), +1 function in `ServiceFragment.go` |
| Dependencies | **none** — `os/exec`, `net/http`, `encoding/json` are stdlib |
| Network | first time the app talks to anything other than the local docker daemon. Say so in the README, and keep it to one documented host (`hub.docker.com`) reached only on an explicit keypress |
| Other plans | `docs/plans/ai-service-authoring.md` depends on Phase 1's `AddServiceFragment` and on `n`; land Phase 1 first |

## Do not

- Do not write to the compose file without going through
  `ValidateComposeCandidate` → `ReplaceFileAtomically`. Every writer in this
  repo does; a new one that does not is how a user loses their file.
- Do not build a form for ports/volumes/environment (D2, and
  `docs/DESIGN.md` §Editing services).
- Do not search on every keystroke (D3, and the Hub abuse limit).
- Do not use `hub.docker.com/v2/search/repositories` as the search transport —
  it is undocumented (§Research). It is acceptable *only* for tags, where no
  CLI equivalent exists, and only with a timeout and a skip-on-failure path.
- Do not add a Docker SDK dependency. The app shells out to `docker`; that is a
  decision, not an accident, and it is why `DOCKER_HOST` and contexts work.
- Do not auto-`pull` or auto-start a newly added service.
- Do not let a network failure block the flow. Every network step degrades to
  typing (D4, D6).
