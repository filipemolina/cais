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

## A group won't start, and the error doesn't mention it

A group action names the group's member services, so a service with no `profiles:` tag is never touched by it — a broken untagged service can no longer block a group's start. If a group still won't start, the error modal names the failing service.

- Once at least one group exists, the groups list's stats line at the bottom of the panel counts the untagged services: `N ungrouped`.
- Switch to the Services page (`2`) to see every service, tagged or not, and find the one causing the failure.
- If a service never starts, it may simply be untagged: tag it into a group (or a dedicated one of its own) so a group action reaches it. If it's not something you want at all, `d` on the Services page deletes it (confirm-guarded, refused if another service still `depends_on:` it).

## I deleted a service and something else broke

`d` on the Services page removes the service's whole entry from the compose file — not just its container, the way `x` (Remove) does. If another service still names it in `depends_on:`, the delete is refused and the file is left untouched; the confirm's error names which service depends on it. Remove that `depends_on:` entry (or delete the dependent too) first.

## The footer is missing hints

On a terminal narrower than roughly 130 columns, the keybinding bar starts dropping hints in a declared priority order rather than wrapping to a second line. `? help` and `q quit` are never dropped, and `?` lists every key — so a shed hint is hidden, not lost. A narrow details panel also drops columns from the member table, widest and least important first, and never `q quit`, `? help`, or the service's own name.

## Something looks stale

cais re-polls container status every five seconds, so panels reflect changes made outside the app. The Files page re-reads the compose file from disk after every write through the app. If a panel still looks wrong, check the footer for which file is loaded — the panels describe the file the footer names.

## `cais: command not found` after SSHing in

`go install` (and `make build`) put the binary in `$(go env GOPATH)/bin`, which
defaults to `~/go/bin`. That directory only ends up on `PATH` because your
shell's startup files put it there — and which startup files run depends on
*how* the SSH session was started:

| How you connected | Shell type | Startup files read |
| --- | --- | --- |
| `ssh host` (opens a shell) | login, interactive | `/etc/profile`, then the first of `~/.bash_profile`, `~/.bash_login`, `~/.profile` (bash) or `~/.zprofile`, `~/.zshrc` (zsh) |
| `ssh host 'some command'` | **not** login, **not** interactive | none of the above — only the raw system `PATH` |

If you're running `cais` as an inline command over SSH, no per-user startup
file runs at all, so anything they add to `PATH` is missing. Either open a
plain interactive session first (`ssh host`, then run `cais`), or force a
login shell for the inline command: `ssh -t host 'bash -lc cais'` (swap in
your shell).

If an actual interactive session still can't find it, check what `PATH`
looks like inside that session and compare it to a local shell:

```bash
echo $PATH
which cais
```

A common cause when `~/go/bin` is missing from an otherwise-normal-looking
`PATH`: a startup file that builds the Go bin path by *calling* `go`, e.g.

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

This only works if `go` itself is already resolvable at the point this line
runs. If some other file adds Go's own install directory (`/usr/local/go/bin`
or similar) to `PATH` *later* in the startup sequence — which is easy to get
backwards across `.bashrc`/`.profile` or `.zshrc`/`.zprofile` — `go env
GOPATH` fails silently and nothing gets appended. It can work by accident in
sessions that inherited a `PATH` from somewhere else (a long-lived terminal,
a multiplexer) and only break on a genuinely fresh login, which is exactly
what SSH gives you.

The fix is to stop depending on `go` being resolvable yet — the default
`GOPATH/bin` is always `$HOME/go/bin` (or `%USERPROFILE%\go\bin` on
Windows), so hardcode it instead of shelling out:

```bash
export PATH="$PATH:$HOME/go/bin"
```

Open a new session and confirm with `echo $PATH` / `which cais`.

## Colors look wrong after SSHing in

TUI color rendering (cais included, via Lip Gloss) picks a color profile —
true color (24-bit), 256-color, or basic 16-color — based on environment
variables, mainly `COLORTERM` and `TERM`. Over SSH those two variables don't
travel the same way:

- **`TERM`** is sent as part of the SSH protocol's pty request whenever a
  session allocates a terminal, so it survives regardless of configuration.
- **`COLORTERM`** is an ordinary environment variable. Like most env vars, it
  only crosses an SSH connection if the *client* is configured to send it
  (`SendEnv`) **and** the *server* is configured to accept it (`AcceptEnv`).
  Most default `sshd_config`s only allow `LANG` and `LC_*` through, so
  `COLORTERM` is silently dropped — even though `TERM` looks fine and the
  session otherwise behaves normally.

The result: a terminal that renders true color locally falls back to an
approximated 256-color palette once you're SSHed in, and colors that were
exact hex values on the local run look subtly (or very) wrong remotely.

Confirm this is what's happening:

```bash
echo $COLORTERM   # should be "truecolor" or "24bit" locally
```
then compare against the same command run inside the SSH session.

**Fix — no sshd config or restart needed.** If the terminal you're
connecting *from* supports true color (most modern ones do — iTerm2, Kitty,
WezTerm, Alacritty, Windows Terminal, Ghostty, and others), export it
yourself on the machine you're connecting *to*, gated on the session
actually being SSH so you don't force it in cases where it might not apply:

```bash
# ~/.bashrc, ~/.zshrc, or equivalent, on the machine being SSHed into
[ -n "$SSH_TTY" ] && export COLORTERM=truecolor
```

`SSH_TTY` is set by `sshd` itself for any session with an allocated
terminal, so this doesn't depend on `SendEnv`/`AcceptEnv` at all — it works
whatever the client is.

**Alternative — configure the SSH env forwarding properly.** If you'd
rather have the real client-side value forwarded instead of assuming
true color support, add `COLORTERM` to both sides and restart `sshd`:

```
# client: /etc/ssh/ssh_config or ~/.ssh/config
SendEnv COLORTERM

# server: /etc/ssh/sshd_config
AcceptEnv COLORTERM
```
```bash
sudo systemctl restart sshd
```

This is more invasive (it's shared system configuration, and needs a
service restart), so the `SSH_TTY` shell snippet above is the simpler
default unless you specifically need the forwarded value to differ from
`truecolor`.

## Still stuck?

Open an issue at [github.com/filipemolina/cais](https://github.com/filipemolina/cais). Include the output of `cais --version` (or the commit hash it reports on an unstamped build) and what you were doing when it failed.