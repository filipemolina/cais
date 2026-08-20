---
title: Keybindings
description: The complete keybinding reference, organized by scope. Every key is declared once in src/keys and rendered by the footer and ? overlay.
sidebar:
  order: 5
---

# Keybindings

Every key in cais is declared exactly once, in `src/keys/Keys.go`. The footer bar and the `?` help overlay render from that same declaration, so what they advertise is what the handlers do — they cannot drift apart.

`?` lists every key in context, with the ones that do nothing on the current screen dimmed. The tables below are the full reference.

## Global

Work anywhere that no overlay owns the keyboard.

| Key | Action |
| --- | --- |
| `1`–`4` | Switch page (Groups / Services / Files / Backups) |
| `[` `]` | Previous / next page |
| `alt+g` `alt+s` `alt+f` `alt+b` | Page aliases (letter derived from the tab label) |
| `tab` / `shift+tab` | Move focus between the list and the details panel |
| `esc` | Back — a ladder of claims: closes a modal, abandons a filter being typed, clears an applied filter, then returns focus from the details panel to the list |
| `?` | Help overlay |
| `a` | About overlay |
| `T` | Theme picker (live preview; `Enter` applies and persists, `Esc` restores) |
| `u` | Usage overlay (Docker disk and memory usage) |
| `v` | Open the `.env` editor modal |
| `q` | Quit (yields to modals and filters) |
| `ctrl+c` | Force quit (yields to nothing) |

## List

Act on the body's left panel — the groups list and the services list.

| Key | Action |
| --- | --- |
| `↑` `↓` `k` `j` | Move the cursor |
| `space` / `enter` | Select (start the selected item) |
| `t` | Stop the highlighted item — the quick-action pair to `space`/`enter`, no `Tab` to the details panel required |
| `n` | New — a group on the Groups page, a service on the Services page |
| `e` | Edit — a group's membership on the Groups page |
| `d` | Delete (confirm-guarded) — a group on the Groups page, a service's whole compose entry on the Services page |
| `R` | Rename group — groups list only |
| `/` | Filter the list by name |
| `esc` | Clear an applied filter |
| `enter` / `esc` | Apply / cancel a filter being typed |
| `g` / `G` | First / last row |

Deleting a service is refused if another service still names it in `depends_on:` — the confirm explains why instead of leaving the file broken.

## Details

Act on whatever the body's right panel is showing. The first six are shared verbatim between the group panel and the service panel — same key, same meaning, one scope wider or narrower.

| Key | Action |
| --- | --- |
| `s` | Start (the whole group, or the one service) |
| `t` | Stop |
| `r` | Restart |
| `p` | Pull |
| `x` | Remove (confirm-guarded) |
| `l` | Stream logs — the service, or every service in the group |
| `y` | Copy the service's URL (when it publishes one) — service panel only |
| `h` | Add a healthcheck from the template picker — service panel only |
| `e` | Edit the service's YAML inline — service panel only |
| `E` | Open the whole compose file in `$EDITOR` — service panel only |
| `ctrl+s` | Save the inline editor |
| `ctrl+o` | Open the inline editor's fragment in `$EDITOR` |

## Editor

Act inside the inline YAML editor, and only there — the editor owns the whole keyboard while it is open.

| Key | Action |
| --- | --- |
| `enter` | New line (auto-indented) |
| `tab` / `shift+tab` | Indent / outdent |

## Files

Act on the Files page's read-only file viewer.

| Key | Action |
| --- | --- |
| `↑` `↓` | Scroll |
| `E` | Open the file in `$EDITOR` |
| `b` | Browse the other compose files in the same directory |

## Backups

Act on the Backups page's version list.

| Key | Action |
| --- | --- |
| `↑` `↓` | Navigate the list |
| `enter` / `r` | Restore the selected copy (confirm-guarded) |

## Overlays

The keys every modal answers to.

| Key | Action |
| --- | --- |
| `enter` | Confirm / submit |
| `esc` | Cancel |
| `tab` | Next field |
| `space` | Toggle (and reveal a masked `.env` value in the env modal) |
| `y` / `Y` | Yes |
| `n` / `N` | No |
| `f` | Follow (logs overlay) |
| `c` | Copy (env modal) |
| `o` | Raw edit (env modal) |

## Design notes

- **One verb is one binding.** Start is the same key on the group panel and the service panel because both read the same binding — not because two switch statements happen to agree.
- **There is no prefix key.** Prefixes exist to address a host program without stealing keys from a guest; cais has no guest, so a prefix would add a mode to teach, render, and exit, and resolve no conflict.
- **`q` yields, `ctrl+c` does not.** `q` is the one global key that yields to a modal or a filtering list that needs the letter for typing; `ctrl+c` is checked before the modal handoff and quits whatever owns the keyboard.
- **The footer sheds, never wraps.** On a narrow terminal the keybinding bar drops whole hints in a declared priority order rather than wrapping to a second line. `? help` and `q quit` are never dropped.