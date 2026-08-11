# Plan: Containers Your File Doesn't Know About

> **Before you start.** Work on a feature branch of small commits, merged
> `--no-ff`; `go build ./... && go vet ./... && go test ./... && gofmt -l .`
> green at **every** commit, not just at the tip — `docs/ROADMAP.md`
> §Conventions is the full contract and `CONTRIBUTING.md` explains how a TUI
> gets tested. Behaviour that only shows on screen gets checked in the real app
> with VHS before it is committed.
>
> **Phase 2 of this plan is blocked on `docs/plans/image-search.md` Phase 1**,
> which builds `utils.AddServiceFragment` — the primitive that inserts a new
> service into the compose file. Phase 1 here needs nothing and can land at any
> time.

Feature request (discovery session, nothing implemented): *"In the services
page, identify the services that are running but are not defined in the compose
file then show options on how to deal with them. Either move them to compose
file or something else."*

## Status — the app is scoped to the file, and cannot see past it

`utils.DockerComposePs` runs `docker compose --file <the file> ps`, which
filters by the project label. That is exactly right for the Services page and
it means **the app is structurally incapable of noticing anything else on the
machine.** A container started with `docker run` eight months ago, or one left
behind when a service was deleted from the file, is invisible.

Compose itself half-notices the second case: `up` prints
`Found orphan containers ([x]) for this project` and offers
`--remove-orphans`. That warning scrolls past in the logs overlay, names no
remedy the app implements, and says nothing at all about the `docker run` case.

## Research — measured on 2026-08-01

### Three categories, and they want three different offers

Every container on the machine is one of:

| Category | How it is identified | What the user wants |
| --- | --- | --- |
| **This project's orphan** | compose label `project` == this project, `service` not in the file | adopt it into the file, or remove it |
| **Another project's** | compose label `project` != this project | *switch the app to that file* — it is a compose stack, just not this one |
| **Unmanaged** | no compose labels at all | adopt it into the file, or leave it alone |

The middle row is the pleasant surprise: the label
`com.docker.compose.project.config_files` carries the **absolute path of the
compose file that owns it**, and the app already knows how to switch which file
it is driving (`b` on the Files page, `cmds.SwitchComposeFile`). So "this
container belongs to `/home/…/arrStack/compose.yml`" comes with a working
action attached, for free.

Measured on the author's machine:

```
$ docker ps -a --format '{{.ID}}\t{{.Names}}\t{{.Label "com.docker.compose.project"}}\t{{.Label "com.docker.compose.project.config_files"}}'
c30cf03d7734  shelfmark             arrstack  /home/filipe/Documents/Docker/arrStack/compose.yml
9343f4b6bf52  calibre-web-automated arrstack  /home/filipe/Documents/Docker/arrStack/compose.yml
…
```

38 containers, 38 compose-managed, 0 unmanaged. Worth stating plainly: **on a
tidy machine this feature finds nothing**, and that is the correct outcome —
the surface must cost nothing when the count is zero (D4).

### Read labels with `{{.Label "…"}}`, never by splitting the blob

`docker ps --format json` renders `Labels` as one comma-joined `k=v` string,
and label *values* legitimately contain commas —
`org.opencontainers.image.description` is a sentence. Splitting on `,` is a
bug waiting for the right image.

Docker's own template function does the lookup properly:

```
--format '{{.ID}}\t{{.Names}}\t{{.Label "com.docker.compose.project"}}'
```

Verified above. Use it; do not parse `Labels`.

### The environment diff makes adoption viable

The hard part of turning a running container into a compose service is that
`docker inspect` reports the **merged** environment — the image's `ENV`
directives plus whatever the user set — and dumping all of it produces a
service block full of `PATH=/usr/local/sbin:…` and `LANG=C.UTF-8`.

The image's own environment is one call away, so the difference is the user's:

```
$ docker inspect --format '{{len .Config.Env}}' <container>   → 10
$ docker image inspect --format '{{len .Config.Env}}' <image> → 6
$ comm -23 <(container env sorted) <(image env sorted)
ND_LASTFM_APIKEY=…
ND_LASTFM_SECRET=…
ND_PLUGINS_AUTORELOAD=true
ND_PLUGINS_ENABLED=true
```

Four real variables out of ten. **The same trick applies to `Cmd`,
`Entrypoint`, `User`, `WorkingDir`, `Healthcheck` and `Labels`** — every one of
them is inherited-unless-overridden, and every one of them is noise in a
compose file when it merely repeats the image.

That single technique is the difference between a useful draft and forty lines
of garbage, and it is the reason this plan is worth building rather than
pointing users at `docker-autocompose`.

## Solution overview

**Phase 1 — see them, and act on what needs no writing.**
The Services page shows a dim line when the count is non-zero; `o` opens a
review overlay listing the three categories with per-container actions: switch
to that file, stop, remove (confirmed), ignore.

**Phase 2 — adopt.**
`a` on an adoptable row generates a compose service block from `docker inspect`
and **opens it in the inline YAML editor**, pre-filled, unsaved. The user reads
it, fixes it, and saves through the existing validated write path. The
container is never touched.

## Design decisions

### D1. The draft lands in the editor. Always. Never a silent write.

This is the whole safety story and it is not negotiable:

- A generated service block is a **guess about a container's intent**, and some
  of it will be wrong — see D6's list of what cannot be captured.
- The app already has the right surface for "here is some YAML, check it": the
  inline editor, with live validation, `ctrl+s` to save through
  `utils.ApplyServiceFragment`, `ctrl+o` to hand it to `$EDITOR`, and `esc` to
  throw it away. Reusing it means adoption inherits every guarantee the editor
  already provides, including that an invalid fragment is refused and the file
  is left untouched.
- It also means the feature degrades honestly: a container the generator
  handles badly costs the user thirty seconds of editing, not a broken file.

The container is **not stopped, removed, relabelled or recreated** by adoption.
After saving, the user presses `s` and compose recreates it under its own
management — that is `up -d`'s job and it already works.

### D2. Detection is three commands and one pure function

```go
type ForeignContainer struct {
    ID          string
    Name        string
    Image       string
    State       string
    Category    ForeignKind // ProjectOrphan | OtherProject | Unmanaged
    OtherProject string     // for OtherProject
    OtherFile    string     // config_files, for the switch action
}

// ClassifyForeign takes every container on the host, the project name the
// loaded file resolves to, and the service names it declares, and returns
// only the containers that are not accounted for. Pure.
func ClassifyForeign(all []HostContainer, project string, services []string) []ForeignContainer
```

`HostContainer` comes from one `docker ps -a` with the template above.
Everything else is set arithmetic.

**`project` must be the name docker actually uses.** Today
`utils.ReadConfigFile` hardcodes `cli.WithName("cais")`, which
overrides the file's own `name:` key — see
`docs/plans/resources-page.md` §Phase 0, which fixes it. **This plan depends on
that fix**; without it every container of the loaded project classifies as
another project's.

Skip containers whose `com.docker.compose.oneoff` label is `True`: those are
`docker compose run` leftovers, not services.

### D3. Where it lives: the Services page, behind one key

A dim line under the services list when the count is non-zero:

```
3 containers here aren't in this file — o to review
```

and `o` opens the review overlay. Not a permanent panel, not a fifth tab, and
**not extra rows in the services list** — that list is "the services this file
declares", and putting things in it that are not in the file breaks the one
invariant that makes the page comprehensible.

`o` is free (verified: no binding, no test asserting it, not in the bubbles
list keymap). Declared in `keys` as `List.Review` or `Global.Foreign` — decide
at implementation time based on where the handler ends up, and declare it once.

### D4. Zero cost when there is nothing to find

The detection call is one `docker ps -a`, which is fast, but it is still a
subprocess. Rules:

- It runs **on entry to the Services page and on `o`**, not on the five-second
  poll.
- When the result is empty, **nothing is rendered** — no line, no count, no
  "0 containers". The author's own machine finds zero; a feature that adds a
  permanent line saying so on every tidy machine is a feature that costs
  everybody something to benefit nobody.

### D5. Removing is confirmed, and it is `docker rm`, not compose

These containers are outside the project, so `docker compose rm` cannot reach
them. Use `docker rm -f <id>` — and route it through
`cmds.OpenConfirmModal` like every other destructive action
(`docs/DESIGN.md` §*State refresh and destructive actions*: "Do not dispatch a
remove action directly from a panel").

The confirm text must name the container **and** say what is not being deleted:
its volumes survive, which is the reassurance that makes the action pressable.

### D6. Fragment generation: what is captured, and what is admitted

Two inspects (`docker inspect <container>` and
`docker image inspect <image>`), then field by field:

| Compose key | From | Rule |
| --- | --- | --- |
| `image` | `Config.Image` | as written; if it has no registry and no tag, it may be a locally built image — flag it in the header comment |
| `container_name` | `Name` | strip the leading `/`; omit when it equals the service name |
| `restart` | `HostConfig.RestartPolicy` | `no` → omit; `on-failure` keeps `MaximumRetryCount` |
| `ports` | `HostConfig.PortBindings` | `"<hostIP:>hostPort:containerPort<:proto>"`, IPv6 wildcard entries deduplicated against IPv4 |
| `volumes` | `Mounts[]` | `bind` → `Source:Destination[:ro]`; `volume` → `Name:Destination[:ro]`; `tmpfs` → the `tmpfs:` key instead |
| `environment` | `Config.Env` **minus the image's** | the diff from the research section |
| `networks` | `NetworkSettings.Networks` | omit `bridge` (the default); a named network needs a top-level entry too — say so in the header comment rather than writing one |
| `network_mode` | `HostConfig.NetworkMode` | only for `host`, `none`, `container:…` |
| `labels` | `Config.Labels` **minus the image's**, minus `com.docker.compose.*` | keeps traefik/tsdproxy/homepage labels, which are the ones worth keeping |
| `command` / `entrypoint` | `Config.Cmd` / `.Entrypoint` **if different from the image's** | otherwise omit entirely |
| `user`, `working_dir` | same rule | |
| `healthcheck` | `Config.Healthcheck` **if different from the image's** | `Test`, `Interval`, `Timeout`, `Retries`, `StartPeriod`, converting nanoseconds to `30s` form |
| `cap_add`, `devices`, `privileged`, `sysctls`, `extra_hosts` | `HostConfig.*` | straight copy when non-empty |
| `profiles` | the group being viewed | see D7 |

**What it cannot capture, and must say so in a comment at the top of the
draft:**

```yaml
# Adopted from the running container "navidrome" on 2026-08-01.
# Check before saving:
#   - depends_on: not recorded by docker; add it yourself
#   - named networks/volumes need top-level entries in this file
#   - build: if this image was built locally, add a build: section
#   - secrets/configs, logging options and ulimits are not captured
```

The comment survives the write — `yaml.v3` round-trips comments, which is the
whole reason the editor path is safe (`docs/DESIGN.md` §*Editing services*).

### D7. Adopting into a group is one extra field, and it is the app's whole idea

The Services page always knows which group context the user came from, and this
app is groups-first (`docs/DESIGN.md` §1). So the adoption modal offers a group
before generating: the chosen group becomes `profiles: ["<group>"]` in the
draft, and the adopted service appears in that group immediately on reload.

Offer the existing groups plus "none". Do not create a group here — `n` on the
Groups page does that, and one verb, one place.

### D8. Ignore is persistent, and it is per-container-name

A container someone deliberately keeps outside compose (a database they manage
by hand, a one-off) should stop being mentioned. `i` adds its **name** to
`ignored_containers` in `~/.config/cais/config.yaml`; ignored
containers are excluded from the count and shown in the overlay only under a
collapsed "ignored (3)" line.

By name, not by ID: IDs change every time the container is recreated, which
would make the ignore last exactly until the next `docker run`.

## Phases

### Phase 1 — see them (~1.5 days, no dependencies)

1. `utils/HostContainers.go` — the `docker ps -a` call with the template, and
   its parser.
2. `utils/ClassifyForeign.go` — the pure classifier and its table test.
3. `components/foreignmodal/` — the review overlay: three sections, per-row
   actions (switch file / stop / remove / ignore), esc to close.
4. `cmds/GetForeignContainers.go`, the Services-page hint line, the `o` key.
5. Config: `ignored_containers []string`.
6. Docs.

### Phase 2 — adopt (~2 days, **after** `image-search.md` Phase 1)

7. `utils/ContainerToService.go` — the generator from D6, plus the image-diff
   helper. This is the bulk of the work and it is pure once the two inspect
   payloads are in hand.
8. `utils/DockerInspect.go` — the two inspect calls, returning decoded structs.
9. The group picker (D7) and the handoff into the inline editor.
10. Wiring: `a` on an adoptable row → generate → open the editor pre-filled.

## Tests

### `src/utils/ClassifyForeign_test.go`

| host containers | project | file services | want |
| --- | --- | --- | --- |
| one labelled `homelab`/`navidrome` | `homelab` | `[navidrome]` | nothing (accounted for) |
| one labelled `homelab`/`old` | `homelab` | `[navidrome]` | ProjectOrphan |
| one labelled `arrstack`/`sonarr` + config_files | `homelab` | `[navidrome]` | OtherProject, `OtherFile` set |
| one with no labels | `homelab` | `[navidrome]` | Unmanaged |
| one labelled oneoff=True | `homelab` | `[navidrome]` | nothing |
| one in `ignored_containers` | `homelab` | `[navidrome]` | excluded from the count |
| none | `homelab` | `[navidrome]` | empty, and the hint line does not render |

### `src/utils/ContainerToService_test.go` (Phase 2)

**Capture the inspect payloads.** Run `docker inspect` and `docker image
inspect` on a real container, trim them to the fields the generator reads, and
commit them as testdata JSON. Inventing them from the Go structs is the failure
mode `TODO.md` documents for the HEALTH column, and this generator reads twenty
fields — the odds of inventing one of them wrong are high.

Cases: env diff removes exactly the image's variables; a `Cmd` equal to the
image's is omitted; a `Cmd` that differs is emitted; ports deduplicate IPv4/IPv6;
a bind mount becomes `src:dst`; a named volume becomes `name:dst`; `bridge` is
omitted from networks; `com.docker.compose.*` labels are stripped and
`tsdproxy.*` labels survive; the header comment is present.

And one round-trip test that matters more than the rest: **the generated
fragment parses as valid compose** through the same validation
`ApplyServiceFragment` uses. If it does not parse, the editor would open on
something the user cannot save.

### `src/model/`

A rig test: with a fake `docker` on `PATH` reporting one unmanaged container,
the Services page renders the hint line and `o` opens the overlay. (`TODO.md`
already lists "docker actions against a fake `docker` on `PATH`" as wanted rig
coverage; this is a good first use of it.)

## Edge cases and unknowns

- **A container whose image no longer exists locally.** `docker image inspect`
  fails, so there is no baseline to diff against. Fall back to emitting the
  full environment with a comment saying the image was unavailable and the list
  may contain image defaults. Do not fail the adoption.
- **Containers created by another tool that uses compose labels** (Portainer
  stacks, Dockge) classify as OtherProject with a `config_files` path that may
  not exist on this machine. Offer the switch action only when the file exists.
- **A stopped unmanaged container** is still worth listing — it is often
  exactly the forgotten one. `docker ps -a` includes it; the row shows its
  state.
- **Swarm services / Kubernetes containers** carry their own labels and no
  compose ones, so they classify as Unmanaged and would be offered for
  adoption. Adopting one into a compose file is wrong but harmless (the draft
  is never written silently). A future refinement could skip anything labelled
  `com.docker.swarm.*`; note it, do not build it.
- **The count changes while the overlay is open.** Do not refresh under the
  user's cursor; the overlay holds the snapshot it opened with, like the help
  overlay holds its context snapshot.
- **`docker rm -f` on a container in another project** is offered and is
  legitimate (it is the user's machine), but the confirm must name the project
  it belongs to, or someone will remove half of a stack they forgot they had.

## Effort / gain

**Phase 1: ~1.5 days. Phase 2: ~2 days**, most of it the generator and its
captured fixtures.

The gain is real but narrower than it first appears, and worth stating
honestly: on a *disciplined* machine — the author's, for instance — this finds
nothing. Its audience is the homelab that grew by accretion, where three or
four `docker run` invocations from a tutorial are still running and nobody
remembers the flags. For that user it is the difference between a compose file
that describes their stack and one that describes most of it.

It is also the one feature here that no adjacent tool offers: lazydocker shows
you the container, Portainer lets you manage it, and neither helps you *fold it
into the file you keep in git*. That framing belongs in
`docs/plans/launch-and-outreach.md` if this lands before the announcement —
though the recommended order puts it after.

## Blast radius

- Phase 1 adds one docker call on Services-page entry, one overlay, one key,
  one config field. Nothing renders when the count is zero.
- Phase 2 adds two inspect calls, on demand, and a generator whose output goes
  into the editor rather than the file.
- The only destructive path is `docker rm -f`, behind the existing confirm
  modal.
- Depends on `resources-page.md` Phase 0 (the project name) and, for Phase 2,
  on `image-search.md` Phase 1 (`AddServiceFragment`).

## Do not

- **Do not write the generated fragment to the file without the editor.** D1.
- **Do not touch the container during adoption** — no stop, no rename, no
  relabel. Compose adopts it on the next `up -d` because the file now describes
  it; that is the mechanism, and it is docker's, not ours.
- **Do not parse the `Labels` blob by splitting on commas.** Use
  `{{.Label "…"}}`.
- **Do not add foreign containers to the services list.** D3.
- **Do not poll `docker ps -a`.** D4.
- **Do not render anything when the count is zero.** D4.
- **Do not dump the whole environment.** The image diff is the feature.
- **Do not offer `docker compose rm` for these containers** — they are not in
  the project and it will not reach them.
- **Do not build a general container manager.** The scope is "what is running
  here that this file does not describe, and what can I do about it". Browsing
  every container on the host is lazydocker's job and it is better at it.

## Acceptance criteria

1. On a machine with no foreign containers, the Services page is unchanged —
   no line, no key hint, no extra docker call on the poll.
2. `docker run -d --name testbox nginx:alpine` makes the hint line appear on
   the next Services-page entry, and `o` lists it under **Unmanaged**.
3. A container from another compose project lists under that project's name
   with its file path, and the switch action loads that file into the app.
4. A service deleted from the file while its container runs lists as **this
   project's orphan**.
5. `i` on a row makes it stay hidden across restarts of the app.
6. Remove asks first, names the container, and says volumes are kept.
7. (Phase 2) `a` on `testbox` opens the inline editor containing a
   `testbox:` block with `image: nginx:alpine`, no `PATH=` in its environment,
   and a header comment listing what to check; `ctrl+s` writes it and the
   service appears in the list.
8. (Phase 2) The generated fragment for every container on the author's machine
   parses as valid compose.
