---
title: Troubleshooting
description: Common problems and their fixes — Docker issues, compose file detection, and configuration.
sidebar:
  order: 9
---

# Troubleshooting

## Docker is missing, the daemon is down, or permissions are wrong

cais classifies which of five states it is in — Docker missing, the Compose plugin missing, the daemon stopped, the socket unreadable, or a healthy machine — and says so, with the exact command that fixes it. The command is copyable, never run: cais never installs, starts, or configures anything on your machine, even behind a confirmation prompt.

The probe runs once at startup and again on every docker error, so a daemon that stops mid-session gets the same diagnosis, not a raw `exit status 1`.

If the diagnosis says the daemon is up but wedged, cais has nothing to offer beyond the raw error — that is a state it cannot reproduce.

## "No compose file found"

With no `--file`, cais looks for `compose.yaml`, `compose.yml`, `docker-compose.yaml`, `docker-compose.yml` in that order — the same order Docker uses. If none exist in the directory, cais offers to write one.

- Run `cais --dir <path>` to resolve in a specific directory.
- Run `cais --file <path>` to open exactly one file, skipping resolution.
- The file that wins is named in the footer. When several candidates exist, the footer marks the winner with `+N` and the `?` help overlay lists the losers by name.

## The theme I set is not applied

The theme is read from `~/.config/cais/config.yaml` (or `$XDG_CONFIG_HOME/cais/config.yaml`) at startup. A missing or malformed config silently yields the default (`cais-dusk`).

- Check the file exists and the `theme:` value matches a registered theme name exactly (see [Themes](/users/themes/)).
- Set it from inside the app instead: `T`, pick a theme, `Enter` — that writes the config for you.

## A service edit was rejected

The inline editor refuses to write a fragment that would not parse as Compose, and the whole resulting document must still load as compose. A rejected save keeps the editor open with the error on the status line; the file is untouched. A rejected `$EDITOR` edit reports the error and returns to a normal TUI with the file untouched — pressing the key again is the retry.

## I made a bad edit — how do I undo it?

Every write is snapshotted into `.cais/backups/` before it lands. Tab `4` (Backups), pick the copy you want, `enter`/`r` to restore. A restore is itself undoable. See [Backups](/users/backups/).

## The footer is missing hints

On a terminal narrower than roughly 130 columns, the keybinding bar starts dropping hints in a declared priority order rather than wrapping to a second line. `? help` and `q quit` are never dropped, and `?` lists every key — so a shed hint is hidden, not lost. A narrow details panel also drops columns from the member table, widest and least important first, and never `q quit`, `? help`, or the service's own name.

## Something looks stale

cais re-polls container status every five seconds, so panels reflect changes made outside the app. The Files page re-reads the compose file from disk after every write through the app. If a panel still looks wrong, check the footer for which file is loaded — the panels describe the file the footer names.

## Still stuck?

Open an issue at [github.com/filipemolina/cais](https://github.com/filipemolina/cais). Include the output of `cais --version` (or the commit hash it reports on an unstamped build) and what you were doing when it failed.