# Plan: Guided Healthcheck Insertion

Feature request (discovery session, nothing implemented): *"A lot of my services
don't have health checks. Is there an easy way to add a functional health check
to a service only by editing the compose.yml file?"*

## Status of the feature — and an honest reframing

**The literal ask already works today, with zero code.** A compose-level
`healthcheck:` block is a runtime-only override: no Dockerfile change, no
rebuild. The app already has every piece:

- `e` on a service opens its YAML fragment in the inline editor; `ctrl+s` saves
  through `utils.ApplyServiceFragment`, which validates the whole document as
  compose before writing.
- Start (`s`) maps to `docker compose up -d <service>`, which **recreates** the
  container with the new config. Verified empirically on this machine
  (Engine 29.6.0, Compose v5.1.4, 2026-07-31): adding a `healthcheck:` and
  running `up -d` produced a new container ID, and the container reported
  `healthy` within ~6 s.
- The UI already *shows* health everywhere: HEALTH column in the group member
  table (`healthColor` handles `healthy`/`unhealthy`/`starting`,
  `GroupDetailsPanel.go:421`), the status line on the service header, and the
  Healthcheck row in the config table (`DetailsPanel.go:795`).

So this request is **not** "make it possible" — it is three real gaps around the
possibility:

1. **Discoverability.** Nothing tells the user that `healthcheck:` exists,
   what a working one looks like, or that the tool they need must exist
   *inside the image*.
2. **Authoring is where healthchecks fail in practice.** The single most common
   failure is a check whose probe binary is not in the image (`curl`/`wget`
   absent from scratch/distroless; `bash` absent from alpine — so the
   `/dev/tcp` trick silently fails too). The probe runs *inside the container*
   (`05-services.md`, compose-spec, retrieved 2026-07-31) — there is no
   host-side healthcheck in Docker as of Engine 29.x (checked release notes,
   2026-07-31), and no way to guarantee "functional" from compose.yml alone
   without knowing the image contents.
3. **The apply gap.** After saving a config edit, Restart (`r`) does **not**
   apply it — `restart` reuses the container's existing config. Only `up -d`
   re-reads compose. A user who saves a healthcheck and presses `r` sees
   nothing happen and concludes the feature is broken.

The plan below fixes all three, at a scope that fits the app's existing
patterns. **Verdict up front: worth doing, at the small end — with the
limitation stated plainly: templates can only be *high-confidence*, never
guaranteed, because the probe runs in an image the app cannot inspect.**

## Problem

A self-hoster's compose file is full of services without healthchecks, and the
ones they hand-write often don't work:

- wrong probe tool for the image (the #1 failure),
- probe on the published host port instead of the container-internal port,
- no `start_period`, so a slow-starting service (DB init) trips to `unhealthy`
  and — with autoheal/restart-on-unhealthy setups — churns,
- saved but never applied (`restart` doesn't re-read compose),
- or an `unhealthy`-forever check silently left in place, which now *blocks*
  any dependent that uses `depends_on: condition: service_healthy`.

## Solution overview

Three pieces, in dependency order. The first two are the feature; the third is
a one-line-ish UX fix that the feature would be incomplete without.

### 1. A template catalog (`h` inserts a healthcheck)

New key `h` on the Services details panel (a service must be selected): opens a
small picker of healthcheck templates. Choosing one inserts the block into the
service's YAML through the existing validated write path, reloads config, and
the Healthcheck row appears in the config table immediately.

Templates are the whole product here, and the honest constraint is: **a
template is only as good as its assumption about what the image contains.**
Two confidence tiers, and the low tier is *not* auto-inserted:

| Tier | Template | Probe | Why it is safe |
| --- | --- | --- | --- |
| high | postgres / postgresql | `["CMD-SHELL", "pg_isready -h 127.0.0.1 -p 5432"]` | `pg_isready` ships in the official image; needs no auth |
| high | mariadb | `test: healthcheck.sh --connect --innodb_initialized` | official image ships the script (gist, supermarsx, active 2026-06-03) |
| high | redis | `["CMD-SHELL", "redis-cli -h 127.0.0.1 ping \| grep PONG"]` | `redis-cli` ships in the official image |
| high | nginx | `["CMD-SHELL", "wget -qO- http://127.0.0.1/ >/dev/null 2>&1"]` | busybox `wget` in `nginx:alpine`; GNU wget in the debian variant |
| medium | generic HTTP | `wget -qO- http://127.0.0.1:PORT/ >/dev/null 2>&1` | port is typed **inline in the same modal** (prefilled from the first `ports: target`, else 80); only offered for images with no known template |
| — | scratch / distroless | — | **not offered** — nothing to probe with; the plan says so instead of pretending |

Every template ships explicit `interval: 30s`, `timeout: 5s`, `retries: 3`,
`start_period: 10s`, and **deliberately omits `start_interval`**: it needs
Compose ≥ 2.20.2 (compose-spec `05-services.md`, retrieved 2026-07-31), and
while the app's own parser (compose-go v2.12.1) accepts it, the user's
`docker compose` CLI may be older and would reject the file at `up` time. The
app's validator would say "fine" and the CLI would say "no" — the worst kind of
validation gap. Omitting it costs nothing.

**Insertion semantics:** the picker replaces an existing `healthcheck:` key
(the modal labels it "replace"), never duplicates it — two `healthcheck:` keys
in one service is a YAML error. Removal stays the editor's job (`e`).

**Why `h`:** unbound everywhere — verified by grep; the only occurrence in the
codebase is a test asserting the bubbles list's paging keys (`l h f b u`) are
*not* bound. `l` (logs) and `h` (health) sit naturally together in the Details
scope. The `?` overlay and footer pick it up from the key catalog with no extra
work (they render from `keys.Active`). Trade-off, same as `R` in
`docs/plans/group-rename.md`: the Details footer grows one hint, and the
narrow-terminal footer wrapping is already an open TODO item.

### 2. The apply hint

When a saved fragment contains a `healthcheck:` (or a service with one was
started), and the service is running, the editor status line / banner says:
**"running: press `s` to apply — restart won't re-read the compose file."**

This is the difference between "I added a healthcheck and nothing happened"
and the feature working. It is a hint, not an action: auto-running `up -d` on
save would surprise (the editor edits *any* config, not just healthchecks) and
recreating a running container unprompted is destructive.

### 3. (Deferrable) "test this check once"

`docker compose exec <service> <test>` run once, output shown, when the
container is running. This is the only way to verify "functional" before
trusting a check, and it surfaces the missing-binary failure ("executable not
found") in seconds instead of a minute of `unhealthy`. Deferred: the health
column already gives feedback within `interval × retries`, the exec plumbing is
new surface, and the catalog's tiering already avoids the worst guesses.

## UX: the modal — one modal; the port is a parameter, not a step

Reuse the modal chrome (`ServiceChecklistModal` is the closest shape: a
centered surface, a title, a list, a hint line). One new component,
`HealthcheckPickerModal`, listing the catalog entries relevant to the selected
service (image-matched first, generic last).

**While the generic HTTP entry is highlighted, a text input row appears under
the list** — prefilled from the service's first `ports:` target (else 80),
labelled "port inside the container". Selecting any other template makes the
field disappear. The field's visibility is *derived from the selection*, so
there is no focus state to manage, and one rule covers every key: while the
field is visible, printable keys type into it; arrows still move the list
selection; Enter always submits for the highlighted template; Esc cancels. No
tab, no focus switching, no second modal.

Why this is the right shape, not just a shorter one:

- The app's core metaphor is list-left / details-right; a field appearing
  under the selected row is the same gesture the user already reads.
- The prefill turns the field into confirmation, not a chore. The common case
  (an unknown image → generic) is `h` → Enter: one modal, one Enter.
- The two-step alternative has a real handoff cost, not just an extra
  keypress: following the create-group precedent (`GroupNameModal.go:78`
  returns `ServiceChecklistModal` directly from Update), the picker would
  return a second modal model — two constructors, two Views, two test
  surfaces — for a step that in the common case was pure ceremony (accepting
  a prefill). Merging removes the handoff and the second modal's tests; the
  risky logic (the yaml.Node insertion) is untouched by the merge.
- A wrong-port insert is equally possible in either design (the two-step user
  also presses Enter on a prefilled port modal); here the value is visible
  next to the cursor at submit time, which is the more honest of the two.

No confirm step beyond the modal itself: insertion is a validated, reversible
write (the editor can undo it), same reasoning as group-rename's no-confirm.

## Message flow

```
DetailsPanel:  h ──> cmds.OpenHealthcheckPickerMsg{ServiceName}
AppModel:      OpenHealthcheckPickerMsg ──> activeModal = HealthcheckPickerModal(...)
Modal:         Enter ──> cmds.RequestAddHealthcheck{ServiceName, Template, Port}
               (Port is empty for templates without a parameter)
AppModel:      RequestAddHealthcheck ──> utils.ApplyHealthcheck(configFileName, name, template, port)
util:          extract service value node, insert/replace healthcheck mapping,
               ValidateComposeCandidate, ReplaceFileAtomically
AppModel:      HealthcheckAddedMsg{Err} ──> error → reportForegroundError
               success → close modal, reload config, re-sync lists,
               if service running: set "press s to apply" hint
```

## Data-layer change: `utils.ApplyHealthcheck`

Build on the existing `readComposeNode` / `findMappingPair` machinery that
`ApplyServiceFragment` uses (a fragment of the *whole service* is unnecessary —
the healthcheck is one mapping under the service's value node):

```go
// ApplyHealthcheck inserts or replaces the healthcheck mapping under
// serviceName in the compose file at fileName. The block is built as a
// yaml.Node from the template, so it round-trips through the same encoder
// as every other write (comments preserved, blank lines still closed up —
// the existing, documented yaml.v3 limitation).
func ApplyHealthcheck(fileName string, serviceName string, t HealthcheckTemplate) error
```

The catalog is a data table (`src/utils/HealthcheckTemplate.go`):

```go
type HealthcheckTemplate struct {
    Name        string // shown in the picker
    Matches     []string // image substrings, e.g. {"postgres", "postgresql"}
    Test        []string // first element CMD / CMD-SHELL
    Interval    string
    Timeout     string
    Retries     uint64
    StartPeriod string
    Generic     bool // true → ask for the port first
}
```

Pure and table-driven means the whole feature is testable without Docker:
fixture compose files in, validated `healthcheck:` block out, existing
`ApplyServiceFragment`-style tests as the model.

## Edge cases and unknowns

1. **Probe binary not in image → `unhealthy` forever.** Mitigated by the
   tiering (known-image templates only use tools that ship in those images) and
   by visible feedback in the health column within ~a minute. Not fully
   solvable: compose healthchecks run in-container, and the app cannot inspect
   image contents without more machinery. State this in the feature docs.
2. **Wrong port for generic HTTP.** The inline port field prefills from the first
   `ports:` target — but the probe must hit the *container-internal* port, not
   the published host port. The two are different numbers; the prompt label
   must say "port inside the container".
3. **Service not running when inserted.** No feedback until start; the apply
   hint covers the running case, and a stopped service simply shows the
   Healthcheck row (config) with no status yet.
4. **Apply gap.** `restart` reuses container config; only `up -d` applies.
   Empirically verified recreating on Engine 29.6.0 / Compose 5.1.4, but older
   compose versions have a history of recreation misses (docker/compose#13045 —
   different trigger, inline `configs:` content, closed as
   kind/question, 2026-07-31). The hint should not promise recreation, just
   "press `s` to apply".
5. **`depends_on: condition: service_healthy` dependents.** Adding a
   healthcheck changes dependent startup ordering — they now wait for
   healthy. That is the point of healthchecks, but a check that never turns
   healthy now *blocks* dependents. Worth one line in the feature docs.
6. **`start_interval` version skew.** The app's compose-go v2.12.1 accepts it;
   a user's `docker compose` CLI < 2.20.2 rejects it. Templates omit it
   entirely (see above).
7. **`extends:` merging.** Inserting a literal `healthcheck:` on a service
   that `extends:` a base with one simply overrides it, per compose merge
   semantics. The spec's one restriction (can't `disable: true` an inherited
   check) never applies — the catalog never emits `disable: true`.
8. **YAML round-trip.** Insertion goes through the same encoder as every other
   write: comments preserved, blank lines between services still closed up.
   Existing documented limitation, unchanged.
9. **LSIO images** (dominant in `mocks/compose.yaml`): curl/wget presence is
   **UNVERIFIED (2026-07-31)**. The catalog prefers busybox `wget` precisely
   because it is the common denominator of alpine-based images; verify before
   finalizing the catalog with a one-liner against a real image
   (`docker run --rm <img> which wget curl`).
10. **Non-HTTP services with the generic template.** A TCP-only service would
    fail a wget probe; the generic tier only claims HTTP. A pure-TCP check
    needs `bash`-only `/dev/tcp` or `nc`, neither portable — explicitly out of
    scope for the generic template, noted in the picker's help text.
11. **`network_mode: host`** — probing `127.0.0.1` still works; no special
    case.
12. **Groups page.** The member table shows HEALTH but the insert affordance is
    per-service; `h` lives on the Services page only (same shape as `e` for
    editing). A group-level "healthcheck everything" is a different feature,
    not planned.
13. **Existing `test: NONE` / `disable: true` display quirk** — the config
    table hides the Healthcheck row when the test trims to empty
    (`DetailsPanel.go:795`). Pre-existing; out of scope; noted.

**Unknowns to resolve before implementation** (cheap, all doable in one
session): LSIO image contents (9); which images dominate real user compose
files; whether the maintainer wants the apply-hint healthcheck-scoped or
generic to any config edit.

## Blast radius

| File | Change |
| --- | --- |
| `src/keys/Keys.go` | +1 binding (`Details.Healthcheck`, `h`) — footer + `?` follow automatically |
| `src/components/DetailsPanel.go` | +1 branch in the non-editing key handler (`DetailsPanel.go:194` region) |
| `src/components/HealthcheckPickerModal.go` | new — picker (list) + inline port field, visibility derived from selection |
| `src/utils/HealthcheckTemplate.go` | new — catalog + node-building + `ApplyHealthcheck` |
| `src/cmds/` | 2 small files: open-picker, request-add-healthcheck |
| `src/model/Update.go` | 2–3 cases in the modal/open/result switch |
| `src/utils/HealthcheckTemplate_test.go`, modal tests, model tests | catalog → YAML fixtures, replace-vs-insert, field visibility per selection |
| Footer | Details scope grows one hint (narrow-terminal wrapping is an open TODO; slightly worsened) |
| `README.md`, `docs/DESIGN.md` | user docs + the "why" (per house convention, docs are part of the phase) |
| Visual check | VHS tape or throwaway `go run` render of the picker (house convention) |

Every commit stays `go build ./... && go vet ./... && go test ./...` green;
the work lands as a small feature branch merged `--no-ff`, per ROADMAP.md
conventions.

## Effort / gain — the honest call

| Option | Effort | Gain | Verdict |
| --- | --- | --- | --- |
| 0 — Do nothing; add a README paragraph ("healthchecks are compose YAML, `e` to edit, `s` to apply") | ~0.5 day | discoverability only; the two real failure modes (bad probe tool, apply gap) untouched | defensible floor |
| **1 — `h` + catalog + validated insert + apply hint** | **~2 days** (data layer is a table + one yaml.Node function; modal and key reuse existing patterns) | the actual ask: one keypress to a working, reviewed-by-Docker healthcheck, with feedback that it took effect | **recommended** |
| 2 — 1 + `docker compose exec` test-once | +1 day | closes the only remaining "is it functional?" doubt before trusting a check | defer; revisit if users report wrong guesses |

**Verdict:** worth it, at the small end. Reasons: the app already surfaces
health state in three places, so authoring is the missing link; the catalog
fixes the single most common real-world failure (probe binary absent from the
image); and the apply hint prevents the "I added it and nothing happened"
dead-end that would otherwise make the feature feel broken. Against it: the
feature's ceiling is inherently capped — compose-only healthchecks can never be
*guaranteed* functional (in-container probe, unknown image contents), so the
deliverable is guidance plus validation, not automation; and for a user happy
to write YAML by hand, the inline editor already covers the literal request.

**Decision owners:** the key (`h` vs another unbound key — maintainer call,
`h` recommended); direct validated write vs editor-prefill (this plan
recommends direct write for ease, with the generic tier's port typed inline
in the same modal so nothing low-confidence lands unexamined); catalog
scope (the 4+1 rows above vs a longer community catalog — small is better:
each row is a correctness claim about an image, and wrong rows make
`unhealthy`).
