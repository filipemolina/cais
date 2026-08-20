---
title: Configuration
description: The config file, environment variables, CLI flags, and how cais finds your compose file.
sidebar:
  order: 6
---

# Configuration

cais is deliberately low-configuration. There is one persisted preference file, a couple of environment variables, and a small set of CLI flags.

## The config file

cais reads `~/.config/cais/config.yaml` (or `$XDG_CONFIG_HOME/cais/config.yaml` when `XDG_CONFIG_HOME` is set). A missing or malformed config silently yields the defaults.

```yaml
# ~/.config/cais/config.yaml
theme: cais-dusk
```

| Field | Meaning | Default |
| --- | --- | --- |
| `theme` | The theme to start with. Any of the [14 themes](/users/themes/). | `cais-dusk` |

The theme is written here when you confirm a choice in the theme picker (`T`), and loaded before the program starts.

## Environment variables

| Variable | Effect |
| --- | --- |
| `XDG_CONFIG_HOME` | Where cais looks for its config file (instead of `~/.config`). |
| `SSH_CONNECTION` | Used to resolve the host part of service URLs when no `url_host` is configured — the address this SSH client measurably used. |
| `$EDITOR` / `$VISUAL` | The editor `E` (whole compose file) and `ctrl+o` (service fragment) open files in. |

## CLI flags

| Flag | Meaning |
| --- | --- |
| `-f`, `--file <path>` | Open exactly this compose file. Skips resolution. |
| `-d`, `--dir <path>` | Resolve a compose file in that directory. |
| `-v`, `--version` | Print the version (or commit hash on an unstamped build) and exit. |
| `-h`, `--help` | Print usage and exit. |

`--file` and `--dir` are refused together: `--dir` says where to look and `--file` says what to open, so honouring both would mean deciding which of two answers you meant.

Bad paths fail before the alternate screen is entered — a typo belongs in the shell you typed it into, not in an error banner behind a full-screen app.

## Which compose file

With no `--file`, cais resolves the compose file in this order — the same order Docker uses:

1. `compose.yaml`
2. `compose.yml`
3. `docker-compose.yaml`
4. `docker-compose.yml`

The file that wins is named in the footer, and it is passed as `--file` to every `docker compose` call, so the commands always act on the file the panels describe. When several candidates exist, the footer marks the winner with `+N` and the help overlay lists the losers by name.

The resolution order is fixed and not a setting — making it configurable would only mean answering "which file?" differently from the tool cais is a front end for.