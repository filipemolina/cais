# Plan: Search Docker Images From Inside the TUI

> **Before you start.** Work on a feature branch of small commits, merged
> `--no-ff`; `go build ./... && go vet ./... && go test ./... && gofmt -l .`
> green at **every** commit, not just at the tip — `docs/ROADMAP.md`
> §Conventions is the full contract and `CONTRIBUTING.md` explains how a TUI
> gets tested. Behaviour that only shows on screen gets checked in the real
> app with VHS before it is committed. **Step 1 of the post-alpha order**
> (moved there 2026-08-01 — see *Why this jumped the queue* below).
>
> **Read the whole plan before writing code, and read
> *Instructions for the implementing model* at the bottom twice.** This plan
> is written to be picked up by a free/weaker model, possibly a different
> one per session. Every design decision below states its *why*, not just
> its *what*, specifically so a model with no memory of this conversation can
> still make the right call on the edge case this plan didn't think of.

Feature request (original, 2026-07-31): *"a feature where the user could
search for docker images from inside the TUI, so it would be even easier to
set up the entire server without having to go out of the TUI."*

Feature request (refinement, 2026-08-01, superseding the original's UX):
*"Typing `n` to create a new service should open a modal with the image
search text input already in place and a table of images (empty to start)
below it — like Neovim's fuzzy finder, or Spotlight. After the user selects
one, he's moved to the service editor with the basic fields filled — the
image's name is assumed as the service name, with the possibility to change
it — and he can add more fields or paste a configuration from the web."*

## Status of the feature — what already shipped, and what this plan changes

**Phase 1 of the original version of this plan is done.** `n` on the
Services page opens `addservicemodal`, which asks for a **name, then an
image** (two plain text fields, Tab between them), writes a minimal
two-line service via `utils.AddServiceFragment`, and opens the inline YAML
editor on it. `utils.AddServiceFragment`, the `cmds.AddService` /
`cmds.AddServiceMsg` round trip, and the `AddServiceMsg` handling in
`src/model/Update.go:688-721` (select the new service, focus the details
panel, open the inline editor atomically) are all live, tested, and — this
is the important part for what follows — **entirely unaffected by this
redesign.** They stay exactly as they are.

**What changes is everything upstream of that write: the two-field modal
becomes a search-first one.** The refinement request describes a materially
different flow from the original plan's Phases 2-3 (which bolted `/`-to-
search onto the *image* field of the existing two-field form, then a
separate tag-picker screen after that). The new shape is:

```
n ──> search stage: one text input, a live results table under it
        (Spotlight/Telescope-style: type, results update, Enter to pick)
      ──> confirm stage: name + image, prefilled and editable
      ──> Enter ──> the existing write path (unchanged) ──> inline editor
```

Two stages, not three, and the tag-picker screen is gone — folded into a
best-effort background enhancement (Phase 2B below) rather than a blocking
step, because the refinement request never asks the user to pick a tag
explicitly and adding a screen it didn't ask for is exactly the kind of
scope creep this codebase's docs argue against elsewhere
(`docs/DESIGN.md` §Editing services, on why a form is a standing tax).

**Verdict up front: worth doing, and worth doing first** — see *Why this
matters for a self-hoster* below for the value case, and *Why this jumped
the queue* for why it now sits ahead of `docker-disk-usage.md` and
`env-secrets.md` in `docs/ROADMAP.md`.

## Research — what actually exists, measured

This section is carried over from the original plan (researched and
verified 2026-07-31 against Docker 29.6.0 / Compose v5.1.4) with one
correction, marked below. Re-verify anything time-sensitive (rate limits,
undocumented-endpoint shape) before relying on it if this plan sits unbuilt
for long.

### `docker search` works, is authenticated for free, and is the right transport

```console
$ docker search --limit 3 --format json linuxserver
{"Description":"A Sonarr container, brought to you by LinuxS…","IsOfficial":"false","Name":"linuxserver/sonarr","StarCount":"2127"}
{"Description":"A Radarr container, brought to you by LinuxS…","IsOfficial":"false","Name":"linuxserver/radarr","StarCount":"1796"}
{"Description":"A Jackett container, brought to you by Linux…","IsOfficial":"false","Name":"linuxserver/jackett","StarCount":"1205"}
```

Facts that matter, in order of how much they shape the design:

- **`--format json` emits one JSON object per line** (not an array), with
  exactly four fields — `Name`, `Description`, `StarCount`, `IsOfficial` —
  and **all four are strings**, including the number and the boolean.
  Decode into `struct{ Name, Description, StarCount, IsOfficial string }`
  and convert; decoding `StarCount` as `int` fails.
- **Descriptions are truncated with a Unicode ellipsis (`…`)** unless
  `--no-trunc` is passed. Pass `--no-trunc`; the results table does its own
  truncation (`chrome.Truncate`) and knows its own column width.
- **It goes through the daemon.** The CLI's `runSearch` calls the moby
  client's `ImageSearch` and attaches the user's Docker Hub credentials from
  `~/.docker/config.json` when they exist (`docker/cli`
  `cli/command/registry/search.go`, read 2026-07-31). A user who has run
  `docker login` gets their authenticated quota for free and the app never
  touches a credential; the app also inherits whatever daemon the user is
  pointed at (`DOCKER_HOST`, contexts) with no extra work.
- **It is Docker Hub only.** No GHCR, no lscr.io, no Quay. Real limitation
  for this audience — see *Why this matters for a self-hoster*.
- **`is-automated` is gone** (deprecated in CLI v25.0, removed by v28.2), so
  the JSON has no `IsAutomated`. Do not filter on it.
- `--filter is-official=true` and `--filter stars=N` work; `--limit` caps
  results (default 25). Use `--limit 20` — enough to fill a modal-sized
  table without the picker needing its own pagination.

Shelling out to `docker` is also simply what this app does:
`utils.DockerCompose`, `DockerComposePs`, `DockerStats` and `DockerLogs` are
all `exec.Command("docker", …)`. A new `utils/DockerSearch.go` is the fifth
of the same shape, not a new pattern.

**Correction to the original plan's testing claim.** The original plan said
`SearchImages` should take "the command runner as a value the test can
substitute, the same way the docker action tests already work." **That
precedent does not exist.** `DockerStats.go` and `DockerCompose.go` call
`exec.Command` directly and are **not unit-tested at that layer at all** —
verified by grep, 2026-08-01: there is no `DockerStats_test.go` or
`DockerCompose_test.go`, and no runner-injection interface anywhere in
`src/utils`. Do not invent one for this feature either. The actual, correct
pattern (and what makes `DockerSearch.go` testable without Docker) is to
split it in two: a thin `SearchImages` that shells out and is not unit
tested, and a pure `parseSearchOutput(output []byte) ([]ImageResult, error)`
that decodes the line-delimited JSON and *is* unit tested against fixture
bytes captured from the real command (§Phase 2A below). This mirrors how
`utils.HealthcheckTemplate.go`'s pure catalog logic is tested without
Docker, not a pattern from the docker-action tests.

### The Hub HTTP API works but should not be the primary search path

`https://hub.docker.com/v2/search/repositories/?query=nginx&page_size=2`
returns 200 unauthenticated (verified 2026-07-31) and is richer than
`docker search` (it has `pull_count`, a better popularity signal), but **it
is undocumented** — absent from the current Hub OpenAPI description. An
undocumented endpoint can change shape or vanish without warning, and when
it does the failure lands on the user as an empty results table. Use
`docker search` for search. The Hub HTTP API is still the only option for
**tags** (next section), where no CLI equivalent exists at all.

### Tags: not part of the core flow, but not abandoned either

`docker search` returns repository names only — no tags. The original plan
made tag selection a blocking screen between search and the editor. The
refinement request doesn't mention tags at all, and forcing one more
decision at the exact moment a self-hoster is most likely to bail is the
wrong trade — but shipping `image: postgres` (⇒ `postgres:latest`, "the
single worst pin available" for a self-hosted database, per the original
plan's own framing) silently is also wrong. This plan resolves the tension
by making tag resolution a **non-blocking background enhancement** — see D4
and Phase 2B.

```console
$ curl -s "https://hub.docker.com/v2/repositories/library/nginx/tags?page_size=3&ordering=last_updated"
{"count":1283,"next":"…","results":[{"name":…,"images":[{"architecture":"amd64",…}],…}]}
```

Official images live under `library/` (`library/nginx`); unofficial ones
use their own namespace (`linuxserver/sonarr`); `ordering=last_updated` is
what makes the list useful; each tag's `images[]` carries `architecture`.

### Rate limits

- **Hub abuse limit** applies to all Hub requests — web, API, pulls —
  counted per IP, "in the order of thousands of requests per minute"
  (docs.docker.com/docker-hub/usage, 2026-07-31), answered with a bare
  `429`. This plan's live-search design (D3) fires on a debounce, not per
  keystroke, specifically to stay far under this.
- **Pull limits** (100/6h unauthenticated, 200/6h free-authenticated) do not
  bind on search, but bind on the `p` the user presses right after adding a
  service. Worth one line in the docs (Phase 4), unchanged from the
  original plan.

### Prior art

**lazydocker does not do this** — it manages images already on the host, no
Hub search. k9s has no analogue. The nearest precedents for the *interaction
style* (not the domain) are Neovim's `telescope.nvim`/`fzf-lua` pickers and
macOS Spotlight — both **live-filtering a query against a source as you
type, with the first result always highlighted and Enter always meaning
"go with what's shown."** That interaction contract (§D3) is what the
refinement request is asking this app to match, and it is a real UI
pattern, not a made-up one — the fact that this app hits a network endpoint
per query where those two hit an in-memory index is the one place the
metaphor has to bend, and D3 says how.

## Scope

**In:** `n` on the Services page opens a search-first modal (D2/D3); Enter
on a highlighted result or on typed free text (D3's unification) advances to
a confirm stage with name and image prefilled and editable (D2); the new
service lands in the existing inline editor, unchanged, for everything else
— ports, volumes, environment, or a config snippet pasted from the web
(already just works — see D6b).

**Out, stated:**

- **Registries other than Docker Hub, as a *search* target.** `docker
  search` cannot reach them. Typing `ghcr.io/foo/bar:v2` still works —
  D3's unification is specifically what keeps this possible, correcting a
  gap the naive "search is now mandatory" reading of the refinement request
  would otherwise create.
- **A blocking tag-picker screen.** Folded into a background enhancement
  (D4, Phase 2B) instead of a step. If it never lands, the feature still
  works exactly as the refinement request describes.
- **Pulling as part of the flow.** `p` already pulls a service; adding one
  does not.
- **Image inspection, local image browsing.** Different features, no
  compose relevance (lazydocker's job, not this app's).
- **Editing an existing service's image through search.** `e` already
  edits; this feature only ever creates.
- **The bootstrap flow (`CreateComposeFileModal`'s optional first
  service).** Deliberately deferred — see D7.

## Design decisions

### D1. `n` on the Services page = "new service" (unchanged)

`List.New` is `n`, gated to Home and Services in `src/model/Update.go:470-
476`. Nothing here changes; the modal `cmds.OpenAddServiceModal()` opens is
what's being redesigned.

### D2. Two stages: search, then confirm — not three, and no wizard

```
n ──> AddServiceModal, stage = search
        one text input (query = image), focused
        a results table under it, empty until 2+ characters are typed
        Enter, with a row highlighted ──> that image
        Enter, with nothing highlighted ──> the typed text, verbatim (D3)
      ──> stage = confirm
        Service name: [prefilled from the image, editable]
        Image:        [prefilled, editable]
        Enter ──> the existing, unchanged path:
                  collision check ──> cmds.AddService ──> inline editor
```

This is a `type addStep int` with `stepSearch`/`stepConfirm` constants,
the exact idiom `createcomposefilemodal.Model`'s `createStep` already uses
(`src/components/createcomposefilemodal/Model.go:12-17`) — copy that shape,
do not invent a new one.

**Esc at either stage closes the whole modal.** No partial back-navigation.
Every existing modal in this app works this way (`groupnamemodal`,
`healthcheckpickermodal`, both steps of `createcomposefilemodal`) — grep
`cmds.CloseModal(nil)` in any of them before assuming a "go back one stage"
gesture is wanted. It isn't; it would be the only modal in the app that
works that way.

**The confirm stage is a small, deliberately un-shared copy of
`servicefieldsstep`'s two-field shape** — see D7 for why it is not the same
component, and what *is* shared with it (one pure function, not the whole
step).

### D3. The search box **is** the image field — live, debounced, and it never traps you

This is the one place the refinement request's "like Spotlight" framing
changes the original plan's design, and it needs to be right, because it is
also where the original plan's free-text escape hatch (D-original-Scope:
"the image field is free text, and it stays free text for exactly this
reason") has to survive the redesign or the feature regresses.

**The fix: there is one text input, and it is simultaneously the search
query and the image field.** Not two things swapped between. This resolves
three problems at once:

1. **Non-Hub registries keep working with no special-casing.** `docker
   search` returns nothing sensible for a query shaped like
   `ghcr.io/foo/bar:v2` or `repo@sha256:…` (verify empirically before
   shipping — untested as of this plan) — so the results table stays empty,
   and D3's Enter rule (below) falls through to "use what you typed,"
   which is exactly the right answer. No detection logic needed for
   "this looks like a non-Hub reference"; an empty results table already
   means the same thing whether the cause is a non-Hub path, zero matches,
   or a search failure (`docker` not on PATH, daemon down, rate-limited).
   One fallback, three causes — do not build three code paths for this.
2. **A search failure degrades exactly like every other network failure in
   this app (D6):** the results area shows a short message instead of a
   table, typing keeps working, and Enter still submits the typed text.
3. **It matches what Spotlight and Telescope actually do**, not just what
   they look like: both let you press Enter on your own typed text when
   nothing in the results is what you want, rather than trapping you until
   something matches.

**Enter's rule, precisely:** if the results table has a highlighted row,
that row's image wins; if the table is empty (short query, no matches, or a
search error), the literal text in the input is used as the image,
verbatim, no validation beyond "not empty." Either way, advance to the
confirm stage.

**Live search is debounced, not per-keystroke and not Enter-gated** —
the original plan's Phase 2 was Enter-only specifically to dodge the Hub
abuse limit and avoid a debounce timer; the refinement request explicitly
asks for the live-typing feel, so the debounce timer has to be built, but
it still has to respect the same limit:

- **Minimum 2 characters** before any request fires. A single character
  against Hub's real catalog is noise, not a result.
- **350ms of no keystrokes** before a search fires — `tea.Tick`, the
  existing pattern for delayed dispatch (`src/cmds/RefreshContainers.go`,
  the 5s poll). Every keystroke resets the timer.
- **Stale results are dropped by a generation counter, not cancelled.**
  Tag every outgoing search with a monotonically increasing `int` the
  component owns; when a result message arrives, compare its generation to
  the component's current one and discard it silently if they don't match.
  **Do not reach for `context.CancelFunc` or `exec.CommandContext` for
  this** — it is real, tested-nowhere-else-in-this-repo machinery for a
  problem the generation counter already solves correctly (a stray
  `docker search` subprocess that finishes late and gets ignored costs
  nothing but a few wasted milliseconds of CPU; it cannot corrupt state,
  because its result is thrown away). If a later contributor wants to
  cancel the in-flight process too, that is a pure optimization on top —
  not a correctness requirement, and not this plan's job to build.
- **`docker search` is asked for at most `--limit 20` results** — the table
  does not need its own pagination.

**No row is auto-highlighted while the table is empty**, and the first row
is highlighted the instant results land (index 0) — same convention
`healthcheckpickermodal` already uses, so Enter-after-a-fast-search reliably
does something sane without the user having to press Down first.

**Cursor movement bypasses `list.Update` entirely** — the exact mechanism
`healthcheckpickermodal.Update` already uses
(`src/components/healthcheckpickermodal/Update.go:23-35`):

```go
case keyMsg.Code == tea.KeyUp:
    m.results.CursorUp()
    return m, nil
case keyMsg.Code == tea.KeyDown:
    m.results.CursorDown()
    return m, nil
```

...with **every other key** — not just the letters bubbles' list keymap
claims, *every* printable key — forwarded to the query input. This is a
stricter version of the same problem the healthcheck picker's port field
solved: there, only the generic template needed unclaimed letters; here,
the query field needs *all* of them, always, because it is never not
focused. Match on `keyMsg.Code == tea.KeyUp` / `tea.KeyDown` directly, the
same as that file — do not use `key.Matches` against `list.DefaultKeyMap`
or any binding that includes a letter.

### D4. Tags are a best-effort background upgrade, not a step

When the search stage hands off to confirm (D2/D3), fire a background,
best-effort tag lookup for the chosen repository (bare `Name`, no tag) at
the same moment the confirm stage renders, using the smart-default
algorithm the original plan's D4 specified: the newest tag that is not
`latest` and parses as a version (`v?\d+(\.\d+)*`), falling back to
`latest` if nothing parses. A 2-second timeout — tighter than the general-
purpose 5s `HubTags` timeout, because this one sits directly in the user's
way and must never make the confirm stage feel stuck.

**The guard that matters:** if the lookup resolves *after* the user has
already started typing in the Image field, discard the result instead of
overwriting what they typed. The only safe rule that doesn't need to track
"did the user touch this field" as separate state: **only apply the
upgrade if the Image field's current value is still exactly what it was
pre-filled with.** A value equality check, nothing more — this is the kind
of race an async background update racing live keystrokes gets wrong by
default, and it is worth stating explicitly rather than trusting it to be
obvious.

If the lookup never resolves (timeout, 429, offline), the confirm stage
already shows the bare repo name — same "type it yourself" degradation as
everywhere else in this app (D6). Do not block the confirm stage on this
call, and do not show a spinner for it — the confirm stage's own fields are
interactive the whole time; a spinner over an editable field is a UI lie
about what is and isn't ready.

### D5. The insert primitive: `utils.AddServiceFragment` (unchanged)

Nothing here changes — it already exists and is tested. See
`src/utils/ServiceFragment.go:112` onward. This plan's redesign is entirely
upstream of this call.

**One fact this redesign leans on that is worth stating plainly:**
`ExtractServiceFragment` keeps the service name as the fragment's top-level
key (`src/utils/ServiceFragment.go:19-31`), and `parseServiceFragment`
explicitly refuses a fragment whose top key doesn't match
(`ErrServiceRenamed`, `src/utils/ServiceFragment.go:13-17, 208-210`).
**Renaming a service is not supported anywhere in this app, including
through the inline editor.** This is *why* the confirm stage's Service Name
field has to exist and has to run before the write: the "possibility to
change it" the refinement request asks for has exactly one place it can
happen — before `cmds.AddService` is called, not after. Do not try to make
the name editable inside the inline editor instead; it will hit
`ErrServiceRenamed` and confuse whoever is testing it into thinking
something is broken.

### D6. Errors degrade to typing; paste already works

**D6a — errors (from the original plan, unchanged in substance, extended
by D3's unification):**

| Failure | What the user sees |
|---|---|
| `docker` not on PATH / daemon down | results area shows the message instead of a table; typing and Enter-with-typed-text still work |
| `--format json` unsupported (old CLI) | same as above — detect by failure, do not version-sniff `docker --version` |
| no results | "no images matched" in the results area, not an error — and Enter still submits the typed text |
| tag lookup fails/times out (D4) | silent — the confirm stage already shows the bare repo name, nothing to report |

**D6b — pasting a config from the web already works, and needs no new
code.** The refinement request asks for the ability to "paste a
configuration from the web" once inside the editor. The inline editor is a
real terminal `textarea` (`src/components/detailspanel/View.go:525`
onward); paste already arrives as `tea.PasteMsg` and is forwarded to it
(`src/components/detailspanel/Update.go:132-135`, pre-existing, unrelated
to this feature). **Confirm this behaves reasonably with a large multi-line
paste (a whole `services:` block copied from a README) as part of Phase
2A's manual/VHS check — not because the code path is new, but because it
has likely never been exercised with input that large, and "the textarea
lags or the validator chokes on a big paste" would be a bad first
impression to discover after ship.** If it already handles this fine (most
likely outcome — Bubble Tea delivers a paste as one message, not one
`KeyPressMsg` per character), Phase 4's docs get one sentence about it and
nothing else changes.

### D7. The confirm stage copies `servicefieldsstep`'s shape; it does not extend the shared component

`servicefieldsstep.Model` is used by exactly one other caller after this
plan lands: `createcomposefilemodal`'s bootstrap "add a first service"
step (D-original still applies there — see D-out-of-scope above). Its own
doc comment is explicit about why it takes exactly two constructor
parameters: *"a third parameter is the signal to stop sharing this
component and copy the step instead"* (`src/components/servicefieldsstep/
Model.go:38-40`). Prefilled initial values for name/image are exactly a
third knob. **Do not add them to `servicefieldsstep.New`.** Build a small,
separate type local to the search modal's package that has the same two
fields, the same Tab-between behavior, and the same validation — call it
sixty-ish duplicated lines, which the same doc comment pre-authorizes
("sixty duplicated lines beat a shared component with five knobs").

**What genuinely is worth sharing is `isValidServiceName`
(`src/components/servicefieldsstep/Model.go:154-166`) — a pure function,
no UI, no state.** Extract it to a small shared home (`src/utils`, next to
the other pure compose-name helpers) so the confirm stage and
`servicefieldsstep` both call the one implementation instead of the confirm
stage growing a second copy of a validation rule that must stay identical
in both places. This is a different kind of sharing than the one the doc
comment above is warning against — it has no knobs, nothing to disagree
about, and nothing to grow — so it does not conflict with D7's first
paragraph.

**Deriving the assumed service name from the image:** strip any `:tag` or
`@sha256:…` suffix, then take the substring after the last `/` (`nginx`
stays `nginx`; `linuxserver/sonarr` becomes `sonarr`). Run the result
through the shared `isValidServiceName`; if it fails (rare — Hub repo names
allow a couple of characters compose service names don't), **do not try to
auto-sanitize it.** Leave the confirm stage's Service Name field showing
whatever was derived, let the existing validation message explain what's
wrong, and let the user fix it — same "the error is legible, the user
fixes it" contract every other validation in this app already uses. Do not
invent an auto-suffixing scheme for name collisions either (e.g.
`sonarr-2`); show the same inline "already exists" message
`addservicemodal` already produces (`src/components/addservicemodal/
Model.go:38-51`), just run the check as soon as the confirm stage renders
instead of waiting for submit — a small, genuine UX improvement this
redesign gets for free because the confirm stage is now a distinct moment
instead of a single combined step.

### D8. The results table reuses `list.Model`, not `bubbles/table`

No file in this repo imports `charm.land/bubbles/v2/table` — verified by
grep, 2026-08-01. Every existing selectable list in this app, including the
one this plan's closest sibling (`docs/plans/healthcheck-insertion.md`)
just shipped, is `list.Model` with a custom item + delegate
(`src/components/healthcheckpickermodal/Model.go:17-25, 55-77` for the
list construction; `View.go:15-19` for the one-line delegate). **Do the
same here** — a `searchResultItem` wrapping one `utils.ImageResult`
(name/description/stars/official), a delegate that renders one row with
those columns and `chrome.Truncate`-safe truncation on the description.
Do not add `bubbles/table` as a new import just because "table" is the word
in the refinement request; the word describes the shape on screen, not a
specific package, and this repo already has a working, tested pattern for
that shape.

Filtering stays off on this list (`SetFilteringEnabled(false)`, same
`healthcheckpickermodal` gotcha about calling it *before*
`SetShowPagination` — `src/components/healthcheckpickermodal/Model.go:65-
76` explains why in detail; read it, do not rediscover the bug) — this
list's "filtering" is the network search itself, not `list.Model`'s local
substring filter over an already-fetched slice.

## Why this matters for a self-hoster

The audience is an amateur self-hoster, not a platform engineer: someone
who found this app because they're tired of `docker compose` and half-
remembered YAML, and who adds a new service every few weeks as they
discover something new to run — a *Arr app, a dashboard, a monitoring tool
someone mentioned in a Discord. For that person, adding a service today
means leaving the terminal: a browser tab to confirm "is it
`linuxserver/sonarr` or `lscr.io/linuxserver/sonarr` or just `sonarr`,"
another to check whether `:latest` is safe for this one, and a mental
context-switch back before they can even start typing YAML. This is
precisely the tax the README's own positioning names directly (*"Where it
fits"*: *"`docker compose` + `vim` — what most of us actually do. Stack
Stitcher is that, minus the remembering"*) — and today, image names are
the one thing this app still makes you remember.

Three concrete, specific gains for this audience, in order of how often
they'd actually hit them:

1. **Fewer typos that fail silently.** A misspelled image reference doesn't
   error until `docker compose up`, and the message ("pull access denied,"
   "manifest unknown") doesn't obviously point back at a typo to someone
   who doesn't already know Docker's error vocabulary. Picking from a
   results table instead of hand-typing removes the whole failure class.
2. **The official badge and star count are a trust signal a newcomer
   doesn't otherwise have.** An experienced Docker user already knows
   `linuxserver/*` is trustworthy; a first-timer has no way to tell it
   apart from a random fork with a similar name, and Docker Hub itself is
   full of abandoned or typosquatted repos. Surfacing the signal Hub
   already computes is cheap and closes a real gap for exactly this
   audience.
3. **It's where the value concentrates.** A self-hoster's active decision-
   making time skews heavily toward *adding new things* (discovering and
   trying software) rather than maintaining a fixed, known stack the way an
   operator running the same ten services for years would. That is also
   why this plan is being moved ahead of `docker-disk-usage.md` (a
   maintenance overlay for a stack you already have) and `env-secrets.md`
   (a surface for values you've already set) in the sequence — see below.

**The honest limit, stated as plainly as the original plan stated it:**
Docker Hub only. A self-hoster who wants a GHCR-only project (increasingly
common — many newer OSS tools publish there and nowhere else) gets nothing
from the search table and has to already know the reference. D3's
unification means typing it still works with zero friction beyond "the
table stays empty," but it is worth one honest line in the docs (Phase 4)
rather than a silent gap.

## Why this jumped the queue

`docs/ROADMAP.md`'s post-alpha table had this plan's Phase 1 already done
and its remaining phases unscheduled, with `docker-disk-usage.md` next.
This plan now moves to the front of that table. Reasoning, plainly: the
value case above is squarely about the app's stated audience and its
stated pitch (*"set up the entire server without leaving the TUI"* is the
literal original ask this plan answers), it's the deepest gap the README's
own "Where it fits" section admits to, and — now that the flow is search-
first rather than a three-step wizard — the redesigned scope (Phase 2A) is
still roughly the size the original plan estimated for Phases 1-3
combined, not larger. `docker-disk-usage.md` and `env-secrets.md` are both
real and both still queued; nothing about them was wrong, they're simply
maintenance-of-what-you-already-run rather than growth-of-what-you're-
trying, and growth is where this audience spends more of its attention.

## Phases

Each phase is a feature branch of small commits, merged `--no-ff`, per
`docs/ROADMAP.md` §Conventions. `go build ./... && go vet ./... && go test
./...` and `gofmt -l .` green at **every** commit — see *Instructions for
the implementing model* below for why this matters more than usual here.

### Phase 2A — the search-first modal (the whole redesign, minus tags)

The refinement request's whole ask, without D4's background tag upgrade.
Ships alone and is already the complete, correct feature — D4 is an
enhancement on top, not a missing piece.

| File | Change |
|---|---|
| `src/utils/DockerSearch.go` | new — `SearchImages(term string, limit int) ([]ImageResult, error)`, a thin `exec.Command` wrapper (not unit tested, matches `DockerStats.go`/`DockerCompose.go` — see the Research correction above); `parseSearchOutput(output []byte) ([]ImageResult, error)`, pure, decodes line-delimited JSON with all-string fields |
| `src/utils/DockerSearch_test.go` | `parseSearchOutput` only, against fixture bytes captured from a real `docker search --format json --no-trunc` run (include the string `"true"`/`"false"` for `IsOfficial`, a `StarCount` of `"0"`, an empty-output case) |
| `src/utils/ComposeName.go` (or similar) | `isValidServiceName` extracted here from `servicefieldsstep` (D7); `servicefieldsstep` calls the shared one |
| `src/cmds/SearchImages.go` | new — the search command and its result message, carrying the generation number (D3) |
| `src/components/addservicemodal/*` | restructured: `addStep` (`stepSearch`/`stepConfirm`, D2); the search stage owns a `list.Model` results table (D8) and the unified query/image `textinput.Model` (D3, including the debounce `tea.Tick` and generation counter); the confirm stage is the new small copied-shape type (D7) |
| `src/components/addservicemodal/*_test.go` | debounce fires after 350ms of no keystrokes, not per-keystroke; a stale (lower-generation) result is dropped; Enter on a highlighted row vs. Enter with an empty table both reach the confirm stage with the right image; the confirm stage prefills the derived name and lets it be edited; a duplicate name is flagged inline as soon as the confirm stage renders (D7); Esc at either stage closes the whole modal |
| `README.md`, `docs/DESIGN.md`, `TODO.md` | the new `n` flow description, replacing the old two-field description; the Hub-only limitation (§Why this matters); why search lives in one unified field (D3) |

Acceptance: with a compose file loaded, `n` on the Services page, typing
`sonarr` shows `linuxserver/sonarr` in the results within ~400ms of the
last keystroke; Enter on it opens the confirm stage with `sonarr`/
`linuxserver/sonarr` prefilled; Enter again writes the service and opens
the inline editor, byte-identical to today's `AddServiceMsg` behavior.
Separately: typing `ghcr.io/foo/bar:v2` (no Hub matches expected) and
pressing Enter reaches the confirm stage with that text as the image,
unchanged. `docker compose config` on the resulting file exits 0 in both
cases.

**No test in this phase may hit the network or shell out to the real
`docker` binary.** `parseSearchOutput` is tested against fixture bytes;
everything upstream of it (the debounce, the generation counter, the two
stages) is tested by driving the component's `Update` directly with
synthetic `cmds.SearchImagesMsg` values, the same way `healthcheckpicker
modal`'s tests drive it without a real catalog lookup.

### Phase 2B — the tag upgrade (D4, optional, additive)

| File | Change |
|---|---|
| `src/utils/HubTags.go` | new — `ListTags(repo string, limit int) ([]Tag, error)` against `https://hub.docker.com/v2/repositories/{ns}/{repo}/tags?page_size=N&ordering=last_updated`; `library/` prefix for un-namespaced repos; `net/http` with an explicit **2s timeout** (D4 — tighter than a general-purpose default because it sits in the user's way) |
| `src/utils/HubTags_test.go` | `httptest.Server` fixtures: normal page, 429, malformed JSON, a repo with no arm64 build. **This is the app's first `net/http` code — verified by grep, 2026-08-01, nothing to copy from elsewhere in this repo.** Use a package-level `var hubBaseURL = "https://hub.docker.com"` the test overrides to the `httptest.Server`'s URL; do not add a client-interface abstraction beyond that one variable |
| `src/components/addservicemodal/*` | fires `ListTags` the instant the confirm stage renders; applies the result to the Image field only if it still equals its pre-fill value (D4's race guard) |
| Tests | the version-tag heuristic picks the right default; a slow/failed lookup leaves the bare repo name in place; **an upgrade that arrives after the user edited the field is discarded — this is the one test that must exist, since it is the one bug a model implementing this from a text description is most likely to get wrong** |

Acceptance: selecting `linuxserver/sonarr` and waiting ~1s shows the image
field update from `linuxserver/sonarr` to `linuxserver/sonarr:<version
tag>` without the user doing anything; typing into the field within that
window leaves whatever was typed untouched when the lookup later resolves.

### Phase 3 — deferred: the bootstrap flow

`CreateComposeFileModal`'s optional first-service step keeps using
`servicefieldsstep` (two plain fields) for now, not this redesign.
**Deliberate, not an oversight:** the bootstrap flow is the very first
thing a brand-new user does, often before they've confirmed Docker even
works on their machine — adding a live network dependency (Hub search) to
that exact moment is a worse trade than it looks, and the original plan's
argument for sharing (*"the bootstrap flow becomes the one place image
search doesn't work"*) is weaker now that the shared component is a whole
search modal rather than two text fields. Revisit after Phase 2A has run
in the wild for a while; if it's adopted, D7's "copy, don't extend" rule
applies there too — do not retroactively add a third parameter to anything
to make the sharing easier.

### Phase 4 — docs and demo

README (the new `n` flow, the Hub-only limitation, D6b's paste note),
`docs/DESIGN.md` (why the query and image field are one thing — D3 — and
why tag resolution is a background upgrade, not a step — D4), `TODO.md`,
and a VHS tape of the flow per house convention (`CONTRIBUTING.md`
§Testing) — this feature is exactly the kind of "shows up only on screen"
behavior that section calls out: the debounce timing and the live table
updating are not really verified until someone watches them happen.

## Edge cases and unknowns

1. **No compose file loaded.** `n` must do nothing — every page is already
   gated on a loaded file (unchanged from the original plan).
2. **Read-only compose file/directory.** `ValidateComposeCandidate` writes a
   temp file into the compose file's directory; a read-only directory fails
   at validation with a confusing message. One explicit error path: "cannot
   write in `<dir>`" (unchanged from the original plan).
3. **The derived or typed name collides with an existing service** —
   flagged inline as soon as the confirm stage renders (D7), not only on
   submit.
4. **`services:` missing or null** — `AddServiceFragment` already handles
   this (D5, pre-existing, tested).
5. **An image with a registry prefix or a digest pin** — accepted verbatim
   via D3's unification; never stripped, never "helpfully" parsed.
6. **A tag with no build for the host architecture** — out of scope for
   Phase 2B's tag upgrade; the app cannot reliably know the *daemon's* host
   architecture (`DOCKER_HOST` may be remote), and D4 already only offers
   one silent default, not a picker with per-tag arch labels the way the
   original plan's D4 sketched. If Phase 2B feels incomplete without it,
   that is a good reason to make the tag list a visible, opt-in step later
   — not a reason to block Phase 2B on it now.
7. **`docker search` on an old CLI.** `--format json` exists in v24/v25;
   older is untested. Detect by failure, degrade (D6a); do not parse
   `docker --version` to gate the feature.
8. **Typosquatting.** Results are attacker-influenced content. The official
   badge and star count are shown, and the user reviews the YAML in the
   editor before anything runs (unchanged from the original plan — no
   auto-start, ever).
9. **A query containing characters `docker search` treats specially**
   (e.g. a colon, as in typing `nginx:1.25` directly into the unified
   field before any result is chosen). **Unverified as of this plan** —
   run `docker search "nginx:1.25"` for real and record what comes back
   before Phase 2A ships, the same evidentiary bar the rest of this plan
   holds itself to. If it returns nothing useful, that's fine (D3's
   fallback already covers it); if it errors, make sure the error still
   degrades cleanly (D6a) rather than surfacing a raw stderr string in the
   results area.
10. **Very long descriptions / non-ASCII.** `--no-trunc` plus
    `chrome.Truncate`, same as every other truncated row in this app.
11. **The new service and groups.** Still out of scope — a new service
    belongs to no group; the user adds it on the Groups page (unchanged).
12. **A background tag upgrade racing the user's own typing (D4).** The one
    new race this plan introduces. Guarded by the value-equality check in
    D4 — flagged here again because it is the single most likely place a
    model implementing this introduces a subtle bug, and subtle bugs in
    "a field I'm looking at changed on its own" are exactly the kind of
    thing that erodes trust in a TUI fast.
13. **A large multi-line paste into the inline editor (D6b).** Confirm it
    behaves reasonably; almost certainly already fine, worth five minutes
    in the manual/VHS check rather than assumed.

## Effort / gain

| Option | Effort | Gain | Verdict |
|---|---|---|---|
| 0 — leave the two-field modal as-is | 0 | the literal original ask ("set up the server without leaving the TUI") stays half-answered; the refinement request goes unbuilt | rejected — the refinement request is the point of this revision |
| **1 — Phase 2A only** | **~2 days** | the whole refinement request: search-first, live, unified field, confirm stage, unchanged write path | **do this regardless — it is already the complete feature** |
| **2 — 2A + 2B** | **~2.5 days** | closes the `:latest` pin trap silently, no extra screen | **recommended** |
| 3 — 2A + 2B + 3 (bootstrap flow adopts search) | +1 day | consistency between the two "make a service" entry points | defer — see D7/Phase 3's reasoning |

## Blast radius

| Area | Effect |
|---|---|
| `src/model/Update.go` | **none.** `AddServiceMsg` handling (`:688-721`) and the `n`-on-Services gate (`:470-476`) are already correct and untouched by this plan — confirmed by reading both before writing this plan. Do not touch either file's logic; if a phase seems to need to, stop and re-read D2/D5 first |
| `src/cmds/AddService.go`, `src/utils/ServiceFragment.go` | **none.** The write path is unchanged (D5) |
| `src/components/servicefieldsstep` | one function (`isValidServiceName`) leaves it for a shared home (D7); the component itself is otherwise untouched and keeps its one remaining caller (`createcomposefilemodal`) |
| `src/components/addservicemodal` | rewritten internals; same package, same entry point (`New(fileName, existingServiceNames)`), same external contract as far as `model/Update.go` is concerned |
| `src/utils` | +2 files (`DockerSearch.go`, `HubTags.go` if Phase 2B lands), +1 extracted file for `isValidServiceName` |
| Dependencies | **none** — `os/exec`, `net/http`, `encoding/json`, `time` are all stdlib |
| Network | first time the app talks to anything other than the local docker daemon (unchanged fact from the original plan) — document it, keep it to two hosts reached only from this one flow: the local daemon (via `docker search`) and, if Phase 2B lands, `hub.docker.com` |
| Other plans | `docs/plans/ai-service-authoring.md` still only depends on `AddServiceFragment` and `n` (D5, unchanged) — unaffected by this redesign |

## Do not

- Do not write to the compose file without going through
  `ValidateComposeCandidate` → `ReplaceFileAtomically`. Unchanged — the
  write path this plan hands off to already does this (D5).
- Do not build a form for ports/volumes/environment. Two fields (name,
  image), then the existing YAML editor — unchanged from the original
  plan's D2, and the whole reason `docs/DESIGN.md` §Editing services gives
  for rejecting forms still applies.
- Do not search per keystroke. Debounce, minimum length, generation counter
  (D3) — not optional, this is what keeps the feature under the Hub abuse
  limit.
- Do not reach for `context.CancelFunc`/`exec.CommandContext` to cancel
  stale searches. The generation counter (D3) already makes stale results
  harmless; adding real cancellation is an optional later optimization, not
  a correctness requirement, and it is new machinery this repo has exactly
  one precedent for (`DockerLogs.go`, a genuinely different problem —
  streaming, not a one-shot query).
- Do not invent a command-runner interface for testing `SearchImages`. That
  precedent does not exist in this codebase (§Research correction); split
  exec-wrapper from pure decoder instead.
- Do not add a third parameter to `servicefieldsstep.New`. Copy the step
  instead (D7) — the component's own doc comment asks for exactly this.
- Do not make the tag picker a blocking screen (D4). If it needs to become
  one later, that is a deliberate future decision with its own reasoning,
  not a default to fall back to because the background version felt
  incomplete.
- Do not lose the free-text/non-Hub escape hatch. If you find yourself
  writing code that requires a result to be selected before Enter does
  anything, stop — re-read D3, the whole point of the unified field is that
  Enter always does something reasonable.
- Do not add `bubbles/table` (D8) — `list.Model` with a custom delegate is
  this repo's one pattern for a selectable list-shaped-like-a-table, and
  it's already proven working in `healthcheckpickermodal`.
- Do not use `hub.docker.com/v2/search/repositories` as the search
  transport — undocumented, acceptable only for tags (§Research).
- Do not add a Docker SDK dependency. Shelling out to `docker` is a
  decision, not an accident (unchanged from the original plan).
- Do not auto-`pull` or auto-start a newly added service (unchanged).
- Do not touch `CreateComposeFileModal` / the bootstrap flow in this round
  (Phase 3 is explicitly deferred — D7).

## Instructions for the implementing model

This plan is written assuming it may be picked up by a free or otherwise
weaker model, possibly in a different session than the one that reads this
sentence, with no memory of the reasoning above beyond what's written down.
These are not stylistic suggestions; treat them as hard constraints.

1. **Read this whole file before writing any code**, including the sections
   above that look like background reading. The "why" in each design
   decision exists specifically so you can resolve a case this plan didn't
   anticipate the same way its author would have, instead of guessing.
2. **Every file:line citation in this plan was verified against the actual
   source on 2026-08-01.** If one doesn't match what you find (the repo has
   moved on since), trust the current code and flag the mismatch in your
   commit message — do not silently patch around it, and do not assume the
   plan's *reasoning* is also stale just because a line number moved.
3. **Do not invent a pattern this plan didn't name.** If you think a
   problem needs an interface, a new dependency, a config flag, or a helper
   package this plan doesn't mention, stop and search the codebase first
   (`grep -rn` for the concept, not just the exact name) before writing it.
   Three specific traps this plan already found and fixed once — do not
   reintroduce them: a command-runner interface for testing exec-wrapped
   functions (does not exist anywhere in this repo); `bubbles/table` (never
   imported); `context`-based cancellation for the search debounce (the
   generation counter is the correct, sufficient mechanism).
4. **Never write to the compose file outside
   `ValidateComposeCandidate` → `ReplaceFileAtomically`.** This plan's
   entire write path already goes through it and is unchanged; if new code
   you're writing touches the file directly, you have gone outside this
   plan's scope and should stop.
5. **No test may hit the real network or shell out to the real `docker`
   binary.** `parseSearchOutput`/`ListTags` are pure functions tested
   against fixture bytes or `httptest.Server`; the component logic above
   them is tested by constructing synthetic result messages
   (`cmds.SearchImagesMsg{...}`) and driving `Update` directly, the same
   way every other component test in `src/components/*/Model_test.go`
   already works. If a test you're writing needs Docker installed or
   internet access to pass, it is wrong.
6. **Do not add a new dependency to `go.mod`.** Everything this plan needs
   — `os/exec`, `net/http`, `encoding/json`, `time` — is already available
   without one. If you find yourself about to run `go get`, stop and ask
   instead.
7. **Match existing file and symbol naming exactly**: one exported type or
   function's name per file, filename matching it
   (`src/utils/DockerSearch.go` exports `SearchImages`, the same pattern as
   `src/utils/HealthcheckTemplate.go` exporting `HealthcheckTemplate` and
   `ApplyHealthcheck`). Do not group unrelated helpers into one file to
   save a file.
8. **Commit granularly** — one logical step per commit (a util function, a
   cmd/message pair, a component, the wiring, the docs), matching this
   repo's existing history (`git log --oneline` on `main` shows the
   pattern). `go build ./... && go vet ./... && go test ./... && gofmt -l
   .` must be green **at every commit**, not just the last one on the
   branch — a session that stops partway through must leave the tree in a
   state the *next* session (possibly a different model) can build on
   without first fixing a broken intermediate commit.
9. **Verify empirically before asserting a fact about Docker or Hub
   behavior**, the same standard every claim in this plan's Research
   section was held to. If you need to know what `docker search` actually
   returns for some input, run it and paste the real output into your
   commit message or a code comment — do not guess at JSON shape or
   describe behavior you have not actually observed.
10. **If something in this plan turns out to be wrong once you're in the
    code** — a cited line number that moved, a claim about the codebase
    that doesn't hold, a design decision that doesn't fit what you find —
    **say so explicitly** (commit message, or a note back to whoever is
    directing the work) rather than silently doing something different.
    A quiet deviation from a written plan is much harder for the next
    session to notice and recover from than an explicit "the plan says X,
    I found Y, here's what I did instead and why."
11. **When a decision in this plan seems to conflict with a rule stated
    elsewhere in this repo's docs** (`docs/DESIGN.md`, `CONTRIBUTING.md`,
    another file's doc comment — e.g. D7's careful navigation of
    `servicefieldsstep`'s own "do not add a third parameter" comment),
    **the more specific, more recently-written rule wins**, and this plan
    has already tried to resolve every conflict it found at the time it
    was written. If you find one it missed, resolve it the same way this
    plan resolved D7's: explain the tension in writing, then make a call,
    rather than picking one silently.
