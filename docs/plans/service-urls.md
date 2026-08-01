# Plan: Show (and Hand Over) the URL a Service Is Actually Reachable At

> **Before you start.** Work on a feature branch of small commits, merged
> `--no-ff`; `go build ./... && go vet ./... && go test ./... && gofmt -l .`
> green at **every** commit, not just at the tip — `docs/ROADMAP.md`
> §Conventions is the full contract and `CONTRIBUTING.md` explains how a TUI
> gets tested. Behaviour that only shows on screen gets checked in the real app
> with VHS before it is committed.
>
> Depends on `docs/plans/group-table-legibility.md` having landed: this plan
> reads `chrome.PortLabel` and the `HostIP` handling it introduces. Do that one
> first; it is a day.

Feature request (discovery session, nothing implemented): *"Link the main URL
of the service if that's available. I'm thinking about things like the jellyfin
URL, etc."*

## Status — the app knows the ports and says nothing about the address

The Services details panel prints `14533->4533/tcp` in its config table and
stops there. Everything needed to say `http://192.168.1.10:14533` is already
parsed and sitting in `types.ServiceConfig`; the app just never joins the two
halves.

This is the most-used fact about a self-hosted service and the one the app is
currently silent about. The whole point of Jellyfin, Navidrome, Kavita and
Sonarr is a web UI, and today the workflow is: read the port off the panel,
switch to a browser, retype the host, retype the port.

**Nothing in the competitive set does this.** lazydocker and ctop read the
daemon and show port mappings, not addresses. Portainer and Dockge show links
because they *are* web apps and already know their own hostname. A terminal
tool that hands you a working link is a differentiator, and it is a small one
to build.

## Research — measured on 2026-08-01

### Terminal hyperlinks work, are zero-width, and survive the layout engine

`github.com/charmbracelet/x/ansi` v0.11.7 is already a direct dependency and
ships `SetHyperlink`/`ResetHyperlink` (OSC 8). Verified in a scratch program
against the pinned versions:

```
link := ansi.SetHyperlink("http://x:80") + "open" + ansi.ResetHyperlink()

ansi.StringWidth(link)  -> 4      (same as the bare text)
lipgloss.Width(link)    -> 4
lipgloss.NewStyle().Width(10).Render(link) -> width 10, sequence intact
ansi.Truncate(link, 6, "…") -> sequence intact
```

So a hyperlink can go in a table cell without disturbing a single column of the
layout. **One trap, and it is the only sharp edge in this plan:**
`chrome.Truncate` is `runewidth.Truncate`, which is *not* ANSI-aware and will
happily cut a string in the middle of an escape sequence. The rule that follows
is D5: **truncate and pad first, wrap in the hyperlink last.**

Terminal support is good and degrades to plain text where it is absent (the
sequence is simply not rendered): iTerm2, WezTerm, Kitty, foot, Alacritty
(0.11+), GNOME Terminal/VTE, Windows Terminal. `Terminal.app` ignores it and
shows the text. Nothing breaks anywhere.

### The clipboard works over SSH

`charm.land/bubbletea/v2` v2.0.7 exposes `tea.SetClipboard(string) tea.Cmd`,
which emits OSC 52. OSC 52 is answered by the *terminal emulator*, which is on
the user's laptop — so copying works from an SSH session into the local
clipboard, which is exactly the situation this app is designed for. (tmux needs
`set -g set-clipboard on`; that is the user's config, and a note in the README
is the right amount of help.)

### `app_protocol` is in the compose spec

`types.ServicePortConfig` carries `AppProtocol` (`app_protocol:` in the file,
alongside `mode`, `host_ip`, `target`, `published`, `protocol`). The spec's own
examples are `http` and `https`. **This is the sanctioned way for a user to
tell the app the scheme, and it means the plan does not have to invent a label
for it.** Nothing else in the ecosystem reads it, which is a mild argument that
nobody sets it — but it costs one field to honour and it is the correct answer
when someone does.

### Which host is "the host" — `SSH_CONNECTION`

`localhost` is wrong for the primary use case. The app is a terminal UI you run
over SSH on the box that runs the stack; the browser is on a different machine.

The clean answer is already in the environment. `SSH_CONNECTION` is set by
`sshd` for every session and holds four space-separated fields:

```
<client-ip> <client-port> <server-ip> <server-port>
```

The **third field is the address the client used to reach this machine** — the
literal, correct, no-guessing-required answer to "what should I put in the
browser". Not the hostname, not a guess at the LAN address, not the first
non-loopback interface: the address that demonstrably worked, thirty seconds
ago, for this user.

### What the author's own stack uses

Real labels from the author's homelab (`docker ps --format '{{.Labels}}'`,
2026-08-01):

```
tsdproxy.enabled=true, tsdproxy.name=Shelfmark
tsdproxy.enabled=true, tsdproxy.name=CalibreWeb
tsdproxy.enabled=true, tsdproxy.name=Navidrome
```

No traefik, no `homepage.*`. tsdproxy publishes each service on a Tailscale
hostname — `https://<tsdproxy.name>.<tailnet>.ts.net` — and **the tailnet name
is not in the labels**, so the app cannot construct that URL from the file
alone. That is the argument for the `stitcher.url` escape hatch (D2) and for
keeping reverse-proxy label support in a later phase rather than the first
(D8): the single most common proxy in this particular stack still needs one
piece of information only the user has.

## Solution overview

One pure resolver, one row in the details panel, one key.

```
Web          http://192.168.1.10:14533          ← a real hyperlink; y copies it
```

- **Phase 1** — the resolver (ports + host + scheme + the `stitcher.url`
  override), the panel row, the hyperlink, `y` to copy, one config field.
- **Phase 2** — reverse-proxy labels: traefik `Host()` rules, `homepage.href`,
  tsdproxy with a configured tailnet.

Phase 1 is the whole feature for a stack that publishes ports, which is most of
them. Phase 2 is for a stack behind a proxy, where the published port is not
the address anyone actually uses.

## Design decisions

### D1. The resolver is one pure function, and it returns a reason

```go
// ServiceURL is the address a service is reachable at, plus how the app
// worked it out — the "how" is shown dimmed beside the URL so a wrong guess
// is diagnosable instead of merely wrong.
type ServiceURL struct {
    URL    string
    Source URLSource // Label, AppProtocol, KnownPort, PublishedPort
    Note   string    // "bound to 127.0.0.1", "container port (host network)"
}

// ResolveURL works out the service's main URL. ok is false when the service
// publishes nothing and declares nothing — the row is then omitted entirely
// rather than rendered empty (the config table's existing rule).
func ResolveURL(svc types.ServiceConfig, host string) (ServiceURL, bool)
```

`host` is passed in, never read from the environment inside the function. That
is what makes the whole thing a table test.

### D2. The resolution ladder, first match wins

| # | Source | Where it comes from |
| --- | --- | --- |
| 1 | `stitcher.url` label | the user said so outright; used verbatim, no parsing |
| 2 | reverse-proxy labels | **Phase 2** — traefik, homepage, tsdproxy |
| 3 | published port + `app_protocol` | the spec's own scheme field |
| 4 | published port + known-port scheme | `443`/`8443`/`9443` → https, else http |

The `stitcher.url` label is the escape hatch, and it is deliberately first and
deliberately unparsed. Every heuristic below it will be wrong for somebody —
a Tailscale name, a reverse proxy on another host, a service behind Cloudflare
— and the answer to "the app guessed wrong" must be one line in the compose
file, documented, that the app then never second-guesses:

```yaml
labels:
  stitcher.url: "https://navidrome.tailnet-name.ts.net"
```

Namespaced under `stitcher.` because that is this app's name and because
squatting on an unprefixed key in the user's compose file is rude.

### D3. Which published port is "the main one"

Given several, in order:

1. **Discard loopback-bound ports for the purpose of choosing** — a port on
   `127.0.0.1` is not reachable from another machine. If *every* published port
   is loopback-bound, take the first anyway and set
   `Note: "bound to 127.0.0.1 — reachable only on this host"`. Saying so is the
   whole value; silently offering an unreachable URL is worse than no URL.
2. **Discard non-TCP.** A UDP port is not a web address.
3. **Prefer a container port that is a known web port**, matched on the
   **target** (container-internal) port, not the published one. The container
   port is the stable fact — Jellyfin listens on 8096 inside no matter what the
   host maps it to — and the published port is whatever was free that day.
4. Otherwise **the first published TCP port in file order**. File order is the
   user's own ordering and is more meaningful than "lowest number".

The known-port table, keyed on container port. Keep it short and obvious;
this is a tiebreaker, not a service catalog:

| Port | | Port | |
| --- | --- | --- | --- |
| 80, 443, 8080, 8443 | generic web | 8096 | Jellyfin |
| 3000, 5000, 8000, 9000 | generic app | 8920 | Jellyfin https |
| 8989 | Sonarr | 32400 | Plex |
| 7878 | Radarr | 4533 | Navidrome |
| 9696 | Prowlarr | 5055 | Overseerr/Jellyseerr |
| 8686 | Lidarr | 13378 | Audiobookshelf |
| 8787 | Readarr | 5000 | Kavita |
| 6767 | Bazarr | 9091 | Transmission |
| 8112 | Deluge | 8081 | qBittorrent (alt) |

**Do not grow this into a catalog of every self-hosted app.** It exists to
break ties on multi-port services, and `stitcher.url` covers everything it
misses. If it ever needs a hundred entries, the design was wrong.

### D4. Host resolution: `SSH_CONNECTION`, then config, then localhost

```go
// URLHost is the host part of every service URL the app builds.
//
//  1. config `url_host:`   — the user's explicit answer, wins always
//  2. SSH_CONNECTION[2]    — the address this SSH client used to get here
//  3. "localhost"          — running locally, which is then correct
func URLHost(cfg config.Config, env func(string) string) string
```

Order matters and the middle rung is the interesting one: over SSH,
`SSH_CONNECTION`'s server address is *measured*, not guessed. It is the address
that just worked.

Two details:

- **IPv6 must be bracketed.** `SSH_CONNECTION` may hold `fe80::1`, and
  `http://fe80::1:8096` is not a URL. Wrap any host containing `:` in `[]`.
- The config field goes on the existing `config.Config` struct, which was
  explicitly designed to absorb fields without changing callers
  (`src/config/config.go`). No new file, no migration; a missing field is the
  zero value and rung 2 takes over.

### D5. Rendering: truncate first, hyperlink last

The row is built like every other row in the config table (`propRow{"Web",
[]string{url}}`), so it inherits the label column, the value column and the
truncation. Then — and only then — the value is wrapped:

```go
ansi.SetHyperlink(u.URL) + truncatedAndPadded + ansi.ResetHyperlink()
```

Doing it the other way round hands `runewidth.Truncate` a string with an escape
sequence in it and it will cut through the middle of one. This is the one way
to get this feature wrong in a way that corrupts the whole screen, so it gets a
comment at the call site as well as a test.

Two consequences worth accepting:

- The *displayed* text may be truncated while the *link target* is the full
  URL. That is correct and is what OSC 8 is for.
- `renderPropRow` is shared with every other row. Do not teach it about
  hyperlinks — build the decorated string in `configRows` and hand it over as
  an ordinary value. The row renderer stays dumb.

### D6. `y` copies. Nothing opens a browser.

`y` (yank) on the Services details panel copies the resolved URL via
`tea.SetClipboard`, and the panel's status line confirms it: `copied
http://…`.

**The app must not spawn a browser**, and this is not a limitation to be fixed
later:

- Over SSH — the primary use case — `xdg-open` on the server either fails
  (headless) or opens a browser on the *wrong machine*, possibly on a monitor
  in another room.
- The hyperlink already solves opening, and solves it *correctly*: the terminal
  emulator is on the user's laptop, so ctrl-click opens the local browser.

So the division is: **the terminal opens, the app copies.** One key, one job.

`y` is free on the details panel. It is bound as `Overlay.Yes` inside confirm
modals, which is not a collision — a modal owns the keyboard while it is open,
and `n` already lives this exact double life (`List.New` and `Overlay.No`).
Declare it in `keys.Details` as `CopyURL`, and the footer and `?` overlay pick
it up with no further work.

### D7. `network_mode: host` is a real case, not an edge case

A service with `network_mode: host` publishes nothing — `ports:` is ignored —
and is reachable on **the container's own port, on the host**. `*arr` stacks
and Plex are routinely run this way.

So: when `svc.NetworkMode == "host"`, use `expose:`d / `ports:`-declared
**target** ports as host ports directly, with
`Note: "host network"`. Getting this wrong means the app shows no URL for
exactly the services most likely to have one.

### D8. Reverse-proxy labels are Phase 2, and each one is a documented parse

- **traefik** — `traefik.http.routers.<name>.rule` containing
  ``Host(`music.example.com`)``. Extract the first backtick-quoted host from a
  `Host(...)` clause. Scheme from
  `traefik.http.routers.<name>.tls` being present/true → https, else http.
  Rules can be compound (`Host(...) && PathPrefix(...)`); take the host and
  append the path prefix only if it is a literal.
- **gethomepage** — `homepage.href` is a complete URL. Verbatim, like
  `stitcher.url`, but ranked below it.
- **tsdproxy** — `tsdproxy.name` plus a **configured** `tailnet:` in
  `config.yaml` gives `https://<name>.<tailnet>.ts.net`. Without the config
  field, tsdproxy labels are ignored (they cannot produce a URL), and the
  panel's note says why in four words: `tsdproxy: set tailnet in config`.

Phase 2 exists as a phase because each of these is a parser with its own edge
cases, and none of them is needed for a stack that publishes ports.

## Detailed changes

### Phase 1

1. **`src/utils/ServiceURL.go` (new)** — `ResolveURL`, `URLHost`, the known-port
   map, and the loopback/host-network rules. Pure; imports `types` and
   `config`, nothing else.
2. **`src/config/config.go`** — add `URLHost string \`yaml:"url_host,omitempty"\``.
   One field, one line of doc comment, no other change.
3. **`src/keys/Keys.go`** — `Details.CopyURL` = `y` / "copy url". Add it to
   the Services details branch of `Active` (after `Details.Logs`, before
   `EditService`) and to the `Details` scope in `Catalog`. It is a verb, so
   `Priority` needs no entry — `priorityVerb` is the default.
4. **`src/components/detailspanel/View.go`** — a `Web` row in `configRows`,
   placed directly under `Ports` (they are the same fact at two altitudes).
   Omitted entirely when `ResolveURL` returns false, per the table's existing
   "a service states only what it defines" rule.
5. **`src/components/detailspanel/Update.go`** — handle `y`: resolve, emit
   `tea.SetClipboard`, set a status-line message.
6. **`src/model/AppModel.go`** — the panel needs the host string. Resolve it
   once at startup (it cannot change during a run) and pass it to the panel the
   same way the compose file path is passed.
7. **README** — one row in the key table, and one paragraph in *What it does*.
   This is announcement material; write it as such.

### Phase 2

8. **`src/utils/ProxyLabels.go` (new)** — the three parsers, each with its own
   test table. `ResolveURL` gains rung 2, which calls it.
9. **`src/config/config.go`** — add `Tailnet string`.

## Tests

### `src/utils/ServiceURL_test.go`

Table-driven, and the table is the specification:

| service | host | want |
| --- | --- | --- |
| `ports: ["14533:4533"]` | `10.0.0.5` | `http://10.0.0.5:14533` |
| `ports: ["18096:8096"]` (Jellyfin) + `ports: ["18920:8920"]` | `10.0.0.5` | `http://10.0.0.5:18096` (known port wins over file order? **no** — see below) |
| `ports: ["19443:9443", "18000:8000"]` | `10.0.0.5` | `https://10.0.0.5:19443` |
| `ports: ["16881:6881/udp", "18080:8080"]` | `10.0.0.5` | `http://10.0.0.5:18080` (udp skipped) |
| `ports: ["127.0.0.1:14533:4533"]` | `10.0.0.5` | `http://10.0.0.5:14533`, Note set |
| `ports: ["14533:4533"]`, `app_protocol: https` | `10.0.0.5` | `https://10.0.0.5:14533` |
| `network_mode: host`, `ports: ["8096:8096"]` | `10.0.0.5` | `http://10.0.0.5:8096`, Note "host network" |
| `labels: {stitcher.url: "https://x.ts.net"}` + ports | `10.0.0.5` | `https://x.ts.net` |
| no ports, no labels | `10.0.0.5` | `ok == false` |
| `ports: ["14533:4533"]` | `fe80::1` | `http://[fe80::1]:14533` |
| `ports: ["8000-8010:8000-8010"]` | `10.0.0.5` | `http://10.0.0.5:8000` (range → first) |

The second row deserves a decision written into the test's comment: when two
ports are both "known", **file order breaks the tie**, because the user wrote
the one they meant first. The known-port table only promotes a port above ports
that are *not* in the table.

`URLHost` cases: config set (wins over everything); `SSH_CONNECTION="192.168.1.9
54321 192.168.1.10 22"` → `192.168.1.10`; `SSH_CONNECTION` malformed (fewer
than four fields) → `localhost`; unset → `localhost`.

### `src/components/detailspanel/`

- **`TestWebRowIsAHyperlink`** — the rendered panel contains
  `\x1b]8;;http://…` and the *stripped* output still has the right width. Use
  `ansi.Strip` for the second half.
- **`TestWebRowSurvivesNarrowPanel`** — render at descending widths; assert no
  output ever contains a truncated escape sequence (search for `\x1b]8` not
  followed by a terminator). This is the D5 trap, pinned.
- **`TestNoWebRowWithoutPorts`** — a service with no ports renders no `Web`
  label at all.

### Phase 2: `src/utils/ProxyLabels_test.go`

traefik rules to cover: simple ``Host(`x.com`)``; with TLS; compound
``Host(`x.com`) && PathPrefix(`/api`)``; two routers on one service (first in
sorted label order wins, deterministically); a rule with no `Host()` (no URL).

## Edge cases and unknowns

- **The URL can be wrong.** It is a guess from a heuristic, and the `Source`
  field exists so the panel can show *how* it was guessed, dimmed. Wrong URL +
  visible reason + a documented one-line override is an acceptable place to
  land; a wrong URL with no explanation is not.
- **Services with no web UI** (databases, redis, qbittorrent's torrent port)
  get a URL if they publish a TCP port. `http://host:5432` is meaningless.
  Accepted for Phase 1 — the alternative is a service-type catalog, which is
  the thing D3 says not to build. If it grates, the honest fix is a
  `stitcher.url: ""` (empty = suppress), which the resolver should honour as
  "explicitly none" from the start. **Implement that in Phase 1**; it is three
  lines and it is the only way for a user to say "no, there isn't one".
- **tmux and OSC 52.** Copying silently does nothing if the user's tmux has
  `set-clipboard off`. The status line says "copied", which would then be a
  lie. There is no way to detect it. One README sentence, and move on.
- **Multi-host stacks** (a compose file whose services run on another machine
  via a docker context) will get this machine's host. Out of scope; the
  `stitcher.url` label is the answer.
- **`published` may be empty with `expose:`** — already handled: those ports
  are not published and never chosen (except under `network_mode: host`, D7).

## Effort / gain

**One and a half to two days for Phase 1**, half of it the resolver and its
table. Phase 2 is another day and can wait indefinitely.

The gain is the best demo material in `docs/plans/`: a terminal panel showing
`http://192.168.1.10:14533` as a clickable link is a screenshot that explains
the whole app's premise in one frame — *your compose file, made operable*. It
is also the feature most likely to make someone who already uses lazydocker
install this one.

## Blast radius

- One new pure file plus its test, one new config field, one new key, one new
  row in an existing table.
- The escape sequence is the only genuinely new *kind* of thing on screen.
  Contained by D5 and its test.
- No docker calls at all — this feature is entirely file-derived, which is why
  it works for stopped services too.
- No writes.

## Do not

- **Do not open a browser**, on any platform, behind any flag. D6.
- **Do not build the hyperlink before truncating.** D5. If a helper is
  tempting, make it `chrome.Hyperlink(text, url string) string` that takes
  *already-sized* text, and say so in its doc comment.
- **Do not grow the known-port table into a catalog.** D3.
- **Do not put the URL in the group member table.** There is no room; the
  legibility plan exists because there was already no room.
- **Do not read `SSH_CONNECTION` inside the resolver.** It takes `host` as a
  parameter so the tests are a table. The environment is read once, in
  `AppModel`.
- **Do not invent a `stitcher.scheme` label.** `app_protocol` is in the compose
  spec and already does that job.
- **Do not make the row conditional on the service running.** It is derived
  from the file, and knowing the address of a stopped service is how you check
  whether starting it worked.

## Acceptance criteria

1. On the fixture stack, the Navidrome details panel shows
   `Web   http://<host>:14533`, and the host is this machine's SSH address when
   run over SSH and `localhost` when run locally.
2. Ctrl-clicking (or the terminal's equivalent) that text opens the service in
   the local browser.
3. `y` copies the full URL and the status line says so.
4. `url_host: homelab.lan` in `~/.config/stack-stitcher/config.yaml` overrides
   the detected host everywhere.
5. `stitcher.url` on a service overrides everything; `stitcher.url: ""`
   suppresses the row.
6. A service with no published ports shows no `Web` row.
7. A loopback-bound service shows its URL *and* the reason it may not work.
8. The panel renders identically to today at every width when no service has a
   URL, and never emits a truncated escape sequence at any width.
