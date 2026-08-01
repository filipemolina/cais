# Plan: Announcing Stack Stitcher to the Self-Hosting Community

Ask: *"how to advertise the TUI to the self-hosting community, what angle to
pitch it at given the tools that already exist (lazydocker, HTML GUIs like
Portainer), and what the best channels to announce it are. Not interested in
gaining a lot of stars — just having the tool out in the open once it's at a
decent usable stage: covers the entire lifecycle for a self-hosting user, from
scratch — creating the compose.yml, adding services, health checks, creating
groups, monitoring logs, starting/stopping groups and services, editing the
compose file."*

Researched 2026-07-31. Every number and rule below carries its source and the
date it was checked; the two claims that could not be verified directly (Reddit
blocks automated fetches, selfh.st returns 403 to them) are marked as such and
have a manual check attached rather than being asserted.

## Status of the ask — an honest reframing

The ask contains its own gate: *"when it's on a decent usable stage."* The
stage is defined in the ask itself, and it is a **lifecycle**, not a feature
count. So the first half of this plan is not marketing at all — it is the
readiness checklist, because announcing before the lifecycle closes is the one
mistake that cannot be undone. A stranger tries a tool once.

The second half is angle and channels, and there the research says something
useful and slightly counter-intuitive: **the self-hosting community and the
terminal-tool community are two audiences with two different front doors**, and
this project is more competitive at the second one than the first.

### Where the lifecycle actually stands

Measured against the tree on 2026-07-31.

| Lifecycle stage (from the ask) | Today | What closes it |
| --- | --- | --- |
| Create a `compose.yml` from scratch | **Works.** No compose file in the directory → the app offers to write one, seeded with one service (`utils.WriteNewComposeFile`, `src/utils/GroupTags.go:249`) | — |
| Add services | **Missing.** `ApplyServiceFragment` refuses a name not already in the file (`src/utils/ServiceFragment.go:92`); `E` and `$EDITOR` are the only way in — `docs/DESIGN.md:393` says so outright | `image-search.md` Phase 1 (`n` on Services → `utils.AddServiceFragment`), which `ai-service-authoring.md` then builds on |
| Health checks | **Manual only.** A `healthcheck:` block can be typed into the inline editor and `s` recreates the container with it — verified empirically, Engine 29.6.0 / Compose v5.1.4. There is no guided path | `healthcheck-insertion.md` |
| Create groups | **Works.** Create, rename, delete, edit membership, all written back as `profiles:` tags | — |
| Monitor logs | **Works.** `l` streams a service or a whole group, follow mode and scrollback | — |
| Start / stop groups and services | **Works.** Five actions at both scopes, confirm-guarded remove, five-second status re-poll | — |
| Edit the compose file | **Works.** Per-service YAML inline with live validation; `E` for the whole file; `b` to switch files | — |
| *(implied by the rest)* Secrets in `.env` | **Missing.** Interpolation works; the file is invisible in the app and nothing is masked | `env-secrets.md` |

Four of eight stages are done. **The gap between here and "announceable" is
three plans: adding a service, healthchecks, and `.env`.** That is the honest
answer to "when", and it is why this plan sits last in the execution order.

### The gate that is not a feature

One more thing has to be true before a stranger's compose file is at risk, and
no existing plan covers it:

- **A trust story for writes.** The app rewrites the user's compose file in
  place. Today the only safety net is that a write which fails Compose
  validation is refused, plus the documented fact that blank lines between
  services do not survive (`TODO.md`). For someone whose 40-service file *is*
  their homelab, "it reformats a bit" is a deal-breaker discovered after the
  fact. Before announcing: either write a `.bak` alongside the first write of a
  session, or detect that the file is untracked by git and say so once, and put
  the blank-line caveat in the README's own words rather than a link. **This is
  a new work item, not a documentation task** — see the execution order.

Everything else on the pre-flight list is cheap:

- **A tagged release.** There is none (`git tag -l` is empty, 2026-07-31), so
  today the only way in is `go install` or a clone — which quietly filters the
  audience down to people with a Go toolchain. The pipeline is already built
  and drafts a release on a `v*` tag; `release-distribution.md` extends the
  matrix. Tagging also starts a clock that matters — see *Directories*, below.
- **An issue template and a "what this is not" line,** so the first three
  issues are not "add Kubernetes support".
- **A screenshot-accurate README.** Done (2026-08-01). Re-recorded after the
  action row was removed and the keybinding bar stopped wrapping — worth
  knowing that this item reopens every time the chrome changes, so it belongs
  after the UI work rather than beside it.

## The angle

### One sentence

> **Stack Stitcher makes the compose file you already have operable from the
> terminal — it edits the file, not a database.**

That sentence is doing three jobs. *Compose file you already have* says there is
no import, no migration, no adoption cost. *Operable from the terminal* says SSH
and no browser. *Edits the file, not a database* is the whole differentiator,
and it is the one thing no competitor in the table below can say.

### The competitive map, measured

Star counts and last-push dates read from the GitHub API on 2026-07-31.

| Project | Stars | Shape | What it does not do |
| --- | --- | --- | --- |
| [lazydocker](https://github.com/jesseduffield/lazydocker) | 52,262 | TUI over the Docker daemon | Cannot change your compose file. It operates containers, not configuration |
| [ctop](https://github.com/bcicen/ctop) | 17,807 (last push 2024-07-08) | TUI, container metrics | Metrics only; effectively dormant |
| [dcv](https://github.com/tokuhirom/dcv) | 250 | TUI **viewer** for docker-compose — the closest neighbour | Its own README says *viewer*: lists, logs, exec, inspect. No compose-file editing |
| [oxker](https://github.com/mrjackwills/oxker) | 1,777 | TUI, view & control containers | Same class as lazydocker, smaller |
| [Portainer](https://github.com/portainer/portainer) | 38,097 | Web GUI + agent | A service you host, secure, update, expose. Stacks live in its own store |
| [Dockge](https://github.com/louislam/dockge) | 23,936 | Web GUI, compose-stack oriented, edits `compose.yaml` | Closest in *philosophy* — and it is a web app you must run, reach and protect |
| [Komodo](https://github.com/moghtech/komodo) | 11,764 | Web, multi-server build & deploy | Different scale of problem; heavy for one box |
| [easiarr](https://github.com/muhammedaksam/easiarr) | 73 | TUI that **generates** compose files for the \*arr stack | One-shot generator, not an operating surface |

The map has an obvious hole. Terminal tools in this space **read** the daemon
(lazydocker, ctop, dcv, oxker). File-editing tools are **web apps** (Dockge,
Portainer, Komodo). One project generates a file and stops (easiarr). Nothing
occupies *terminal + owns the file + full lifecycle*, which is exactly where
this lands.

### How to say it, and what not to say

**Do:**

- Lead with the file. "It writes your compose file, preserving comments and key
  order" is the sentence that makes a self-hoster's eyebrows go up, because
  every GUI they have tried mangled something.
- Name the groups idea concretely. "A group is a Compose `profiles:` tag" is
  instantly legible to this audience and costs one line.
- Say what it is not, early and in your own voice. It converts the sceptical
  reply into your own bullet point.

**Don't:**

- **Don't position it against lazydocker as a replacement.** It is 52k stars of
  goodwill and it does a different job; "lazydocker killer" earns the one
  comment thread you cannot win. The honest framing — *lazydocker for
  containers, this for the file* — is also the more persuasive one.
- **Don't lead with the theme picker.** Fourteen themes is a nice detail in
  paragraph six and a "so it's a toy" signal in paragraph one.
- **Don't claim metrics parity.** `docs/ROADMAP.md` deliberately has no
  statistics page, because ctop and lazydocker own that. Repeating that decision
  out loud is a strength; being vague about it invites the comparison.
- **Don't hide the age.** "Early, I use it on my own box, here is what does not
  work yet" outperforms polish claims with this audience specifically, and it
  pre-empts the bug reports that would otherwise read as disappointment.

## Channels

Ordered by expected return *for this project*, which is not the same as ordered
by size.

### Tier 1 — worth real preparation

**1. Show HN.** The single best-evidenced channel here. Terminal tools of this
exact shape do well on Hacker News, repeatedly, over years (points/comments via
the Algolia API, 2026-07-31):

| Story | Points | Comments |
| --- | --- | --- |
| Lazydocker: a terminal GUI for Docker (2019) | 340 | 47 |
| Lazydocker: a lazier way to manage everything Docker (2023) | 481 | 73 |
| Lazygit: A simple terminal UI for Git commands (2021) | 395 | 141 |
| Show HN: Portainer — a lightweight management UI for Docker (2016) | 146 | 50 |
| Dockge — self-hosted compose.yaml stack manager (2024) | 13 | 4 |

Read that last row against the one above it: the *web* compose manager with 24k
stars got 13 points on HN, while terminal tools clear 300 routinely. HN is a
terminal-tool audience. Use it.

The [Show HN rules](https://news.ycombinator.com/showhn.html) (retrieved
2026-07-31) are compatible: "something you've made that other people can play
with", non-trivial, "ideally without barriers such as signups", early-stage work
explicitly welcome — but *"if your work isn't ready for users to try out, please
don't do a Show HN."* Which is the gate again. Title: `Show HN: Stack Stitcher –
a terminal UI that edits your Docker Compose file`. Post Tuesday–Thursday
morning US Eastern, then be at the keyboard for six hours; the comments are the
actual value of this channel, and a Show HN whose author vanishes reads as
abandoned.

**2. r/selfhosted.** The largest concentration of exactly this user. Two
cautions:

- Reddit blocks automated fetching, so **the current rules could not be read for
  this plan — read the sidebar the day you post.** What is consistent across
  secondary sources (2026-07-31) is the usual shape: self-promotion tolerated
  when it leads with substance rather than a link, participation expected
  outside your own thread, and a required flair for release/project posts.
- The format that works there is not an announcement, it is a **write-up**:
  screenshots inline, the problem stated in the first two sentences, the repo
  link at the bottom rather than the top. Title the post around the problem
  ("I got tired of `docker compose logs -f` and wrote a TUI that edits the file
  too"), not the product name — nobody is searching for the name yet.

**3. selfh.st / Self-Host Weekly.** A weekly newsletter read by precisely this
audience, with a public submission form and a software-spotlight section; it
reportedly takes 25+ submissions a day, so it is competitive but genuinely
open (selfh.st returned 403 to automated fetches — find the form on
<https://selfh.st/> manually). Submit **the week before** the Reddit/HN posts:
newsletters have lead time, and a mention landing the same week compounds.

**4. Terminal Trove.** A directory of terminal tools with a "Post a Tool"
submission form (<https://terminaltrove.com/>, retrieved 2026-07-31) and a
"tool of the week" slot. Small, high-intent traffic — people browsing it are
looking for exactly a new TUI. Cheap to submit, no lead time.

### Tier 2 — cheap, do them the same week

- **r/docker** and **r/homelab**: same write-up, adjusted opener. r/homelab
  skews hardware, so lead with the SSH-only angle there.
- **Lemmy `selfhosted`** communities: small but unusually receptive to
  terminal-first tooling, and the crossposting culture means one good post
  travels.
- **Mastodon / Fediverse with `#selfhosted` and `#docker`,** with the demo gif
  attached. The gif is the post; this audience reshares moving pictures.
- **`awesome-docker`** ([veggiemonk/awesome-docker](https://github.com/veggiemonk/awesome-docker),
  36,558 stars) and the **`awesome-tuis`** list — a PR each, and they are the
  lists people actually search when they want this category.
- **The Charm community** (Bubble Tea Discord / discussions). Built with their
  stack, and they amplify good-looking terminal apps. Lead with the UI, not the
  problem — different audience, different hook.

### Explicitly not worth the effort

**awesome-selfhosted.** It looks like the obvious target and it is not eligible.
Its
[CONTRIBUTING.md](https://github.com/awesome-selfhosted/awesome-selfhosted-data/blob/master/CONTRIBUTING.md)
(retrieved 2026-07-31) excludes "desktop, mobile, or command-line applications"
and "generic deployment/virtualization/container tools" — this project is both.
Confirmed empirically: a code search of `awesome-selfhosted-data` on 2026-07-31
returns **zero** matches for `lazydocker`, `dockge`, or `portainer`. The list is
for the services you host, not the tools you host them with. Do not spend a PR
on it.

Its sibling constraint is still worth knowing for the lists that *do* fit: many
awesome-lists require a first release older than **four months**. Nothing is
tagged yet, so **tag `v0.1.0` as early as it is defensible** — the clock starts
at the release, not at the announcement, and a tag costs nothing.

**Paid promotion, star campaigns, "please star this" appeals.** The ask rules
them out and they would also work against the credibility the honest-status
framing buys.

## The launch week

Assumes the three feature plans and the write-safety item have landed.

| When | Do |
| --- | --- |
| −14 days | Tag `v0.1.0`. Verify the GoReleaser draft actually produces working binaries on a machine that is not yours |
| −7 days | Submit to selfh.st and Terminal Trove. Open PRs to awesome-docker and awesome-tuis |
| −3 days | Write the Show HN first comment and the Reddit write-up. Re-read the r/selfhosted sidebar. Add an issue template |
| Day 0, Tue–Thu AM ET | Show HN. Stay in the thread |
| Day 0, +4h | r/selfhosted, once the HN thread has produced the first round of questions — the answers make the Reddit post better |
| Day 1–2 | r/docker, r/homelab, Lemmy, Mastodon |
| Day 3–14 | Triage. Ship one visible fix from feedback and say so in the threads that are still alive |

**Have these four answers written before you post,** because they are certain to
be asked:

1. *Why not lazydocker?* — It reads the daemon; this writes the file.
2. *Will it mangle my compose file?* — Comments, quoting and key order are
   preserved; blank lines between services are not, and here is why (the
   `yaml.v3` block-scalar trap in `TODO.md`). Answering this one *before* it is
   asked is worth more than any feature.
3. *Windows?* — No, and the reason: it shells out to `docker compose` and hands
   the terminal to `$EDITOR`, neither tested there. `cross-platform-testing.md`
   is the path if demand shows up.
4. *Does it need root / the Docker socket?* — It runs `docker compose` as you,
   with whatever access you already have. Nothing new is exposed.

## What success looks like

The ask says not stars, so measure what was actually wanted — *the tool being
out in the open and usable*:

- **Issues filed by people you do not know.** One thoughtful bug report from a
  stranger's 40-service homelab is worth more than 500 stars; it is proof the
  install worked and the tool survived contact with a real file.
- **Release download counts** (the GitHub API reports per-asset counts), which
  measure installs rather than bookmarks — the number stars cannot give you.
- **A second contributor**, or even a pull request. That is the point of
  `CONTRIBUTING.md` and `docs/DESIGN.md` existing before launch rather than
  after.
- **Being findable.** Six months out, does searching "docker compose TUI" reach
  it? Directory listings do that job; social posts do not.

## Risks worth naming

- **Announcing before the lifecycle closes.** A visitor who hits "I can't add a
  service without leaving the app" concludes it is lazydocker-but-worse and does
  not return. This is the risk the whole ordering exists to avoid.
- **The write-trust problem** (above). One "it ate my compose file" thread is
  unrecoverable, and the fix is cheap compared to the damage.
- **Support load arriving before the maintainer is ready for it.** A Show HN
  that lands well produces a week of issues. Decide in advance what you will not
  do — a `docs/SUPPORT.md` sentence about scope and pace costs nothing and
  prevents the guilt spiral that kills side projects.
- **Go-only distribution.** Without binaries, most of the audience cannot try
  it at all, and the ones who can are not the ones you are aiming at.

## If this is also a portfolio piece

It is, and the launch serves that too — but through different artifacts than
the ones that serve users. A hiring reader looks at: the pinned repo, the
README's first screen, the commit messages, and whether the project shows
*judgment* rather than volume. Those are already the project's strengths — the
plans in `docs/plans/`, `docs/DESIGN.md` recording rejected alternatives, and a
git history where each commit explains why. Two additions carry
disproportionate weight for that audience and nearly none for users:

- **A short write-up of one decision that went wrong and got reversed** — the
  blank-line preservation that was built and deliberately removed is the ready
  example, and it reads as senior in a way no feature list does.
- **A tagged release with a real changelog.** "Ships things" is the signal.

Do them, but do not let them delay the launch: they are worth an afternoon, not
a sprint.
