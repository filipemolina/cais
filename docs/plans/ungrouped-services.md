# Ungrouped Services — Implementation Plan

> **Before you start.** Work on a feature branch of small commits, merged
> `--no-ff`; `go build ./... && go vet ./... && go test ./... && gofmt -l .`
> green at **every** commit, not just at the tip — `docs/ROADMAP.md`
> §Conventions is the full contract and `CONTRIBUTING.md` explains how a TUI
> gets tested. Behaviour that only shows on screen gets checked in the real app
> with VHS before it is committed. **P1 is a bug fix and landed ahead of the
> post-alpha sequence (2026-08-22); P2 and P3 remain in the order after the
> alpha** (`docs/ROADMAP.md` §The order after the alpha).

## Problem

Docker Compose starts every service that carries no `profiles:` key alongside
ANY profile you request. Cais ran group actions as
`docker compose --profile <group> <verb>`, so every group action reached
services the user never selected.

Verified against real Docker (Compose v5.5.0) on 2026-08-21 with a
three-service file — `core_a` in profile `core`, `untagged_b` and
`untagged_c` with no `profiles:` key:

    $ docker compose --profile core up -d
      -> started core_a, untagged_b AND untagged_c
    $ docker compose --profile core stop
      -> stopped all three

It is not only `start`. Every verb cais routes through `--profile`
over-reaches: start, stop, restart, pull, remove (`rm -fs`) and logs.
Stopping a service the user did not select is the worst of them.

## Solution

Compose scopes to exactly the services you name, and naming a service
auto-enables its own profile. Verified in the same session:

    $ docker compose up -d core_a              -> started ONLY core_a
    $ docker compose stop core_a               -> stopped ONLY core_a
    $ docker compose rm -fs core_a untagged_b  -> removed only those two
    $ docker compose logs --tail 5 core_a      -> only that service

So: stop passing `--profile <group>`, pass the group's member service names
instead. No compose file is written. Nothing about the user's file changes.
`--profile` is not needed alongside the names and should be dropped.

AppModel already knows the members: `AppModel.groupMembers(groupName)` in
`src/model/AppModel.go` returns every service carrying that profile, sorted.

### What does NOT change

| Thing | Why not |
| --- | --- |
| The user's compose file | The fix names services on the command line; nothing is written. |
| The groups list rows | No "Ungrouped" row is added by P1 — that is P2. |
| `ComposeFileArgs` / `--file` handling | Unaffected; the file pinning stays as it is. |
| Single-service actions | They already named their one service; the change is only in what a group action passes. |
| The pending-action spinner, error modals, keybindings | The plumbing that feeds the docker call sites changes; the UI around them does not. |

### The one trap: an empty member list

`docker compose up -d` with NO service names starts every profile-less
service in the file. Verified:

    $ docker compose up -d      (no names)
      -> started untagged_b and untagged_c

So a group action with zero members MUST NOT run the command at all. It has
to fail fast with a clear message. Getting this wrong turns the bug into a
worse version of itself. A group can legitimately have zero members: the user
deleted the last service in it, or the compose file was edited on disk.

## Phases

### P1 — group actions name their members (done, 2026-08-22)

The bug fix, scoped to the docker call sites, the plumbing that feeds them,
and the copy that described the old behaviour:

- `utils.RunDockerCompose` takes the member list and names the services
  instead of `--profile`; an empty member list is refused with a backstop
  error (`src/utils/DockerCompose.go`).
- `cmds.RunDockerAction` threads the members through; AppModel resolves them
  from `groupMembers` and refuses an empty group with a foreground error
  about the group, before any spinner is raised (`src/cmds/RunDockerAction.go`,
  `src/model/Update.go`).
- The logs stream takes the same fix: `StreamDockerLogs` names the members,
  `logsmodal.New` threads them through (`src/utils/DockerLogs.go`,
  `src/components/logsmodal/Model.go`).
- Tests pin the arg building (`composeActionArgs`, `dockerLogsArgs`), the
  empty-group refusal at both layers, and the model-level guard
  (`TestEmptyGroupActionIsRefusedRatherThanRun`).
- The six places that described the old behaviour are updated: the stats
  footer's `N ungrouped, always run` becomes `N ungrouped` ("in no group, so
  no group action reaches them"), plus `docs/DESIGN.md` §3, the website usage
  and troubleshooting pages, and `TODO.md`.

### P2 — a derived, read-only "ungrouped" group (medium)

Show the ungrouped set as a group row, still with zero file writes:

- Add the reserved `apptypes.UngroupedGroup` constant.
- AppModel: `ungroupedServices`, `listedGroupNames`, `membersOf` — the
  derived group is computed, never stored.
- `groupdetailspanel`: derive the row, delete the service overview.
- Make the row read-only: no `d`, `e` or `R`, and no footer hints for it.
- Reserve the name in the group name modal.
- Tests for the derived group and the read-only guards.
- Docs, then drive the real app on a profile-less compose file.

### P3 — opt-in write of the ungrouped profile (low)

The real write, opt-in and reversible:

- Bind `A` on the groups list, offered only on the ungrouped row.
- A confirmation modal that spells out the `docker compose up` consequence.
- Wire adopt and release through the existing `GroupTags` writers.
- Exit rule both ways: leave ungrouped on join, rejoin when orphaned.
- Tests, docs, and drive the whole flow against a real file.

## Testing

- Component/unit level first (CONTRIBUTING.md "Testing"): the arg builders
  are pure helpers asserted without shelling out; the model-level guard is
  driven through `AppModel.Update` directly.
- `go test -race ./...` — this touches code that shells out and streams.
- The real-Docker verification is the whole argument: a scratch directory
  outside the repo with a three-service file (`core_a` in profile `core`,
  two untagged), then start/stop/logs/remove the group and confirm ONLY
  `core_a` is touched; then empty the group and confirm the refusal starts
  nothing.

## Risks

**The reason P3 is opt-in rather than automatic.** Verified on Compose
v5.5.0: once every service in a file carries a profile, a plain
`docker compose up -d` outside cais prints `no service selected` and starts
nothing. Anyone with a cron job, a systemd unit, or the habit of running
`docker compose up -d` by hand would find their stack silently stops coming
up. That is why cais never writes the tag on its own.

The original proposal was to tag every untagged service with
`profiles: [ungrouped]` as soon as cais opens the file. That design has
exactly that consequence, which is why the work is split into the three
phases above: P1 fixes the actual bug with zero file writes, P2 gives the
first-run screen the user asked for, still with zero writes, and P3 does the
real write, opt-in and reversible.