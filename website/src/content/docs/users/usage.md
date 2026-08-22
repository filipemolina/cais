---
title: Usage
description: The four pages of cais — Groups, Services, Files, and Backups — and how to work with each.
sidebar:
  order: 4
---

# Usage

cais has four pages, switched with the digit keys `1`–`4` (or `[`/`]` to step through them). Each page is a two-pane layout — a list on the left, details on the right — except Files and Backups, which are single panels.

`Tab`/`Shift+Tab` moves focus between the list and the details panel. `↑`/`↓` (or `k`/`j`) move the cursor; the details panel follows it. `/` filters the focused list by name.

## 1 · Groups

The home page operates on **groups of services** — a group being a Compose `profiles:` tag. This is a navigation rule, not just a feature: you start, stop, and otherwise act on groups from here, never on individual services.

![The Groups page: a group selected, its member services and their state](/screenshot-groups.png)

- The **groups list** shows every derived group with a status header.
- The **group details** panel shows a header card with a status pill, a running/stopped/services summary, and the member-services table (status dot, name, image, state, health, uptime, ports).
- With a group selected: `s` starts every service in the group, `t` stops, `r` restarts, `p` pulls, `x` removes (confirm-guarded), `l` tails all their logs. `space`/`enter` (start) and `t` (stop) also work straight from the list, no `Tab` to the details panel required.
- `n` creates a new group (pick a name, then check which services belong to it). `e` edits a group's membership. `R` renames, `d` deletes (confirm-guarded).
- Services with no `profiles:` tag appear in a reserved **`ungrouped`** row at the bottom of the list — every untagged service, selectable and actionable like any other group. The name is reserved on both counts: you cannot create or rename a group to `ungrouped`, and the row itself is read-only (no `e`, `R` or `d`), because its membership is derived from the file rather than chosen. Nothing is written to your compose file to make it appear.
- `A` on the `ungrouped` row **adopts** it: cais writes `profiles: [ungrouped]` onto every service that has no profile, so the grouping survives outside cais. `A` again **releases** it and takes the tag back off. Both are confirm-guarded, and the file is backed up first — the Backups page can restore it.

**A service with no `profiles:` tag is not "in no group" — it is simply left alone.** Compose itself starts an untagged service for *any* `--profile` you request, but cais no longer requests a profile: a group action names the group's member services, so an untagged service is never touched by a cais group action — a broken one can no longer block a group's start. Once at least one group exists, the groups list's stats line at the bottom counts these for you (`N ungrouped`) — worth checking there if a service never seems to start.

**Adopting changes what a bare `docker compose up -d` does.** Once *every* service in the file carries a profile, `docker compose up -d` with no arguments outside cais starts **nothing** — it prints `no service selected` until you name a service or a profile. That is Compose's own rule, not a cais one, and it is why cais never writes the `ungrouped` tag on its own: if a cron job, a systemd unit or your own habit runs a bare `docker compose up -d`, adopting will stop it working. The confirmation says so before anything is written, and `A` again releases the tag and puts things back.

## 2 · Services

The Services page is the counterpart to Home for single-service operations.

![The Services page: one service's configuration side by side with its live runtime stats](/screenshot-service.png)

- The **services list** shows every compose service with status and memory summary per row. `space`/`enter` starts the highlighted service and `t` stops it, straight from the list; `d` opens a confirm to delete its whole entry from the compose file (refused if another service still names it in `depends_on:`).
- The **service details** panel shows a header card (name, image, status line with coloured dot, state, health, uptime) and a compact PROPERTY | VALUE table of the service's compose configuration: ports, a live **web** hyperlink to the service's resolved URL, container name, restart policy, networks, volumes, healthcheck, `depends_on`, pull policy, PUID/PGID, memory limits, and label count.
- When the service has a running container, a live runtime stats table joins it: memory usage + percentage, CPU, network I/O, disk I/O, PIDs, uptime.
- With a service selected: `s`/`t`/`r`/`p`/`x` act on that one service, `l` streams its logs, `y` copies its URL, `h` opens the healthcheck template picker.
- `e` opens the service's own YAML fragment in an inline editor (real YAML, not a form — every Compose field is reachable). It validates as you type, auto-indents on Enter, indents with `tab`/`shift+tab`, and refuses to write a fragment that would not parse as Compose. `ctrl+s` saves, `ctrl+o` opens the same fragment in `$EDITOR`, `esc` cancels.

![The inline YAML editor open on a service, with live validation](/screenshot-editor.png)

- `n` adds a new service: a small modal asks for a name and an image reference (with live validation on both), then writes a minimal `image:` fragment into the compose file and opens the inline editor on it — so ports, volumes and everything else land in the same YAML you would have hand-written.
- `E` opens the whole compose file in `$EDITOR` — the only way to touch top-level keys (`name:`, `volumes:`, …).
- `l` streams logs — live output in a scrollable overlay with follow mode.

![Streaming logs for a service](/screenshot-logs.png)

## 3 · Files

The Files page answers "which file am I acting on, and what is in it?" in full.

![The Files page: the loaded compose file with syntax highlighting](/screenshot-files.png)

- A single panel showing the active compose file's path on the title row and its raw contents — comments and blank lines included — in a read-only, scrollable viewport, syntax-highlighted.
- `E` opens the file in `$EDITOR`.
- `b` browses the other compose files in the same directory and switches which one the app is driving — a way to *choose*, like `--file`, not a resolution order.
- The view re-syncs from disk after every write through the app, so it never goes stale.

## 4 · Backups

The Backups page answers "what did this file used to be, and can I have it back?" in full.

![The Backups page: every stored copy of the compose file and the .env, with a live preview](/screenshot-backups.png)

- A list on the left of every stored copy of the compose file and the `.env` (when one sits next to the loaded compose file), newest first, with a live preview on the right — the exact bytes a restore would put back.
- `enter` or `r` restores the chosen copy over the live file, through a confirm modal. Because the write is atomic, the live file is snapshotted first — so a restore is itself undoable.
- A `.env` restore brings the secrets back too, which the confirm makes clear.

See [Backups](/users/backups/) for how the store works.

## Overlays and modals

- `?` opens the help overlay: every key grouped by scope, with the ones that do nothing on the current screen dimmed. Closes with `?`, `esc`, or `q`.
- `a` opens the About overlay: brand mark, version, license, repo link.
- `T` opens the theme picker with live preview; `Enter` applies and persists, `Esc` restores the theme you started with.
- `u` opens the usage overlay: Docker disk and memory usage bars.
- `v` opens the `.env` editor modal: the variable table, a key editor, and the raw file editor in one surface. Variables are masked by default — `space` reveals the selected value, `c` copies it, `n`/`e`/`d` add/edit/delete, `o` opens the whole file in an inline editor. Writes are line-preserving, so comments and the variables you did not touch survive.
- Destructive actions (`x` remove, `d` delete group or service) always go through a confirm modal. `y` confirms, `n` or `esc` cancels.