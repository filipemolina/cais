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

Both panels are always active. `↑`/`↓` (or `k`/`j`) move the cursor — moving it selects the row, and the details panel renders the selection. `tab`/`shift+tab` are inert on body pages.

## 3. Do something

With a group or service selected, the essentials:

| Key | Action |
| --- | --- |
| `s` | Start |
| `t` | Stop |
| `r` | Restart |
| `p` | Pull |
| `L` | Stream logs |
| `y` | Copy the service's URL |
| `H` | Add a healthcheck from the template picker |
| `e` | Edit the service's YAML inline |
| `E` | Open the whole compose file in `$EDITOR` |
| `v` | Open the `.env` editor |
| `n` | New group (on Groups) or new service (on Services) |

## 4. Quit

`q` quits. `?` opens the help overlay listing every key, with the ones that do nothing on the current screen dimmed.

That is the whole loop: pick a thing, act on it, move on. For the details, see [Usage](/users/usage/) and the [Keybindings reference](/users/keybindings/).