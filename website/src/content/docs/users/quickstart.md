---
title: Quickstart
description: Get cais running against your compose stack in under a minute.
sidebar:
  order: 3
---

# Quickstart

This gets you from installed to managing a stack in under a minute.

## 1. Point cais at your stack

Run it in the directory that holds your compose file:

```bash
cd ~/homelab/media
cais
```

Or point it at a directory or file explicitly:

```bash
cais --dir ~/homelab/media              # resolve a compose file in that directory
cais --file ~/homelab/compose.prod.yml  # open exactly this file
```

With no flags, cais auto-detects `compose.yaml`, `compose.yml`, `docker-compose.yaml`, `docker-compose.yml` — the same order Docker uses. In a directory with no compose file at all, it offers to write one.

## 2. Look around

- **`1` Groups** — your groups (Compose `profiles:` tags) with their member services and state.
- **`2` Services** — every service in the file, with per-service configuration and live runtime stats.
- **`3` Files** — the raw compose file, syntax-highlighted.
- **`4` Backups** — every stored copy of the compose file and `.env`.

`Tab` moves focus between the list and the details panel. `↑`/`↓` (or `k`/`j`) move the cursor; the details panel follows it.

## 3. Do something

With a group or service selected, the essentials:

| Key | Action |
| --- | --- |
| `s` | Start |
| `t` | Stop |
| `r` | Restart |
| `p` | Pull |
| `l` | Stream logs |
| `y` | Copy the service's URL |
| `h` | Add a healthcheck from the template picker |
| `e` | Edit the service's YAML inline |
| `E` | Open the whole compose file in `$EDITOR` |
| `v` | Open the `.env` editor |
| `n` | New group (on Groups) or new service (on Services) |

## 4. Quit

`q` quits. `?` opens the help overlay listing every key, with the ones that do nothing on the current screen dimmed.

That is the whole loop: pick a thing, act on it, move on. For the details, see [Usage](/users/usage/) and the [Keybindings reference](/users/keybindings/).