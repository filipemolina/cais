# Edit Existing Services — Design

> **Status — in progress.** This is a live design, not a historical record.
> Unlike the other documents in this directory, the "today" statements below
> describe current behavior at the time of writing (2026-07-25). See
> [TODO](../../../TODO.md) for where it sits in the backlog.

## Context

Editing existing services is the last open item from the original roadmap.
`DESIGN.md` §6.3 already commits to where it lands: **the Dashboard, not
Home** — Home is for groups, the Dashboard is for per-service work.

Everything needed to write to the compose file exists. The
create/delete-groups work built a comment-preserving `yaml.Node` edit path
(`src/utils/GroupTags.go`), and writes became crash-safe when
`writeComposeNode` moved to `utils.ReplaceFileAtomically`. What's missing is
a way to change anything other than a `profiles:` tag.

Today the Dashboard's details panel is read-only: it renders name,
PUID/PGID, image, groups and ports (`src/components/BasicInfo.go`), and the
only keys it accepts are the five docker actions plus `l` for logs.

## The shape of the feature: YAML, not a form

The obvious design is a form — a text input per field, starting with image,
then ports, then env vars. **That is not what this feature is.**

A form has to choose a representation for every field, and for two of the
three fields the choice is destructive. `ports:` has two valid spellings in
compose: the short string form (`"8080:80"`) and the long mapping form
(`target:`/`published:`/`protocol:`). `environment:` likewise: a list of
`KEY=value` strings or a mapping of `KEY: value`, with values that may be
quoted, empty, or interpolated (`${VAR}`). Round-tripping a user's file
through form inputs silently rewrites whichever spelling they chose, and
throws away their comments and key ordering along the way.

A form is also a permanent tax: every compose field that isn't modelled
is a field the user cannot edit, and compose has a lot of fields.

So the user edits **the actual YAML for the service**, and the app's job is
to put that text in front of them, splice their edit back into the right
place in the file, and refuse to write anything that doesn't parse. Every
compose field is editable on day one, and comments, quoting and key order
are preserved. Blank lines are not - see the note on `yaml.v3` in
`DESIGN.md`.

## Three ways to put YAML in front of the user

In increasing order of difficulty, and increasing order of how good they
feel:

1. **Open the whole compose file in `$EDITOR`.** No YAML manipulation at
   all — suspend the TUI, run the editor on the real file, reload on exit.
2. **Open just that service's YAML in `$EDITOR`.** Extract the service's
   subtree to a temp file, edit it, splice the result back into the original
   document, validate, write atomically.
3. **Edit the YAML inline in the details panel.** A `textarea` in the panel
   itself, with validation as you type. No suspend, no temp file, and the
   app never leaves the screen.

(3) is the preferred end state. (1) and (2) are not throwaway steps toward
it: (1) stays useful forever, because editing the whole file is how you add
a service or touch top-level `volumes:`/`networks:`, which per-service
editing structurally cannot do. And (2) survives inside (3) as an escape
hatch — a `textarea` in a half-width panel is a poor place to restructure a
large service, so the inline editor offers "open this in `$EDITOR`" as a
key.

They also share their hard part. (2) and (3) both need the same three pure
functions — extract a fragment, splice a fragment back, validate the
candidate — and those are the substance of the feature. (1) and (2) share
the editor-suspension machinery. So the phasing is genuinely incremental:
nothing built early is discarded later.

## Phase 1 — the whole file in `$EDITOR`

`E` on the Dashboard details panel opens the compose file in the user's
editor. `tea.ExecProcess` suspends the TUI, hands over the terminal, and
resumes on exit; the callback queues `cmds.GetConfig`, so the app reloads
whatever the user saved.

**Editor resolution** follows the usual convention: `$VISUAL`, then
`$EDITOR`, then `vi`. The value is split on whitespace rather than executed
through a shell, so `EDITOR="code --wait"` works. A shell would let a
crafted `$EDITOR` do arbitrary things with the terminal handed to it, and
buys nothing here.

There is no validation step and nothing to roll back: the user edited the
real file directly, exactly as they would have outside the app. If they save
something broken, `GetConfig` fails and the existing error banner says so —
which is already what happens if the file is broken at startup.

**The background poll is suspended while the editor runs.** It shells out to
`docker compose ps`, which reads the compose file; more importantly, a
resumed TUI should not process a backlog of ticks. `shouldPollContainers()`
gains an "external editor is open" condition alongside its existing modal
and project checks.

## Phase 2 — one service in `$EDITOR`

`e` on the Dashboard details panel opens **just the selected service**.

**The fragment** is the service as a single-key mapping, exactly as it
appears in the file:

```yaml
web:
  image: nginx:alpine
  # the app the whole thing exists for
  ports:
    - "8080:80"
```

Keeping the service name as the top-level key does two things. It gives the
user the context they'd have in the real file, and it gives us a place to
put an explanatory header comment that cannot leak into the compose file:
comments above `web:` attach to the *key* node, and the splice only ever
takes the *value* node. Comments the user writes inside the body attach to
nodes within the value, and are preserved.

**Splicing back**, in `utils`:

- Parse the edited fragment. It must be a mapping with exactly one key.
- The key must still be the original service name. **A renamed key is an
  error, not a rename feature** — other services may reference the old name
  in `depends_on:`, and a rename that leaves those dangling is worse than a
  refusal. Service rename can be its own item later.
- Replace the value node in the parent mapping, keeping the original key
  node so its own comments survive.
- Encode the whole document and validate it (below) before writing.
- Write through `writeComposeNode` → `ReplaceFileAtomically`.

**Validation** is the real advantage over Phase 1, and it has two levels.
YAML syntax comes free from parsing the fragment. Compose validity needs the
loader: write the candidate document to a temp file in the compose file's own
directory and run `utils.ReadConfigFile` over it. The directory matters
because compose resolves relative paths (build contexts, `env_file:`)
against it; the file name does not, because `ReadConfigFile` pins the
project name to `stack-stitcher`. This validation is exactly as strict as
what the app already requires to display anything at all, so it cannot
reject a file the app would otherwise have been happy with.

**Nothing is written unless it validates**, which raises the question of
what happens when it doesn't.

The `visudo` answer — reopen the editor on the failed text until it passes —
is the wrong one here. It decides on the user's behalf that they want to
keep going, and traps them in an editor at the exact moment they may have
concluded the edit was a mistake. Getting out should never require fixing
something first.

So a failed validation **reports the error and returns to the TUI, with the
compose file untouched**. The error goes to the existing banner, naming what
the loader objected to. From there the user presses `e` to try again, or
does something else entirely — the app is in a completely normal state
either way, and there is no mode to escape.

The cost is that their text is gone, which is fine for a typo and annoying
for a substantial edit. That is worth fixing, but as its own later step
rather than by holding the user hostage: **the rejected fragment is kept as
a draft**, and a subsequent `e` on that service resumes from the draft
instead of from the file. Discarding is then an explicit choice rather than
the price of leaving.

Drafts live outside the project — `$XDG_CACHE_HOME/stack-stitcher/drafts/`,
keyed by the compose file's absolute path and the service name — because
writing them next to the compose file would put junk in the user's repo. A
draft is removed on a successful save or an explicit discard.

**A draft can go stale.** The compose file may have changed between the
failed edit and the retry — the user tagged a group with `n`, or edited the
file elsewhere. Saving a stale draft would silently revert those changes, so
the fragment the draft was based on is stored with it. If the service's
current YAML no longer matches, the draft is not resumed silently: the app
says the file changed underneath and starts from the current file, keeping
the draft available rather than destroying it. Letting the user choose
between the two is a refinement, not the first cut.

**An unchanged or emptied file cancels**, writing nothing — the compare is
on bytes, so quitting the editor without saving is a reliable cancel.

## Phase 3 — inline in the details panel

`e` becomes an inline editor: the details panel swaps its rendered card for
a `textarea` holding the same fragment Phase 2 produces, and the Phase 2
`$EDITOR` path moves to a key *inside* the editor (`ctrl+o`), for when the
panel is too small for the job.

Three things this must get right:

**The panel's action keys must be dead while editing.** `s`, `t`, `r`, `p`,
`x` and `l` are single-letter docker actions on that panel today. Typing
`ports:` into an editor that reads `p` as "pull" and `t` as "stop" would be
a disaster, and `x` opens a container-destroying confirmation. The key
handling gates on edit mode before anything else.

**Save is `ctrl+s`, not `enter`.** `enter` inserts a newline in a multi-line
editor. `esc` cancels, and confirms first if the text has changed — this is
the one modal in the app where discarding on `esc` could throw away real
work. Once drafts exist, `esc` on changed text should offer to keep one
rather than only offering to discard.

Inline editing needs the draft mechanism far less than Phase 2 does: a save
that fails validation simply keeps the editor open with the text still in
it, because the app never left the screen. The work is only at risk on an
explicit `esc`.

**Validation is live**, on a status line under the editor: YAML syntax on
every change (cheap, it's a small document), and the full compose load on
save. A save that fails validation keeps the editor open with the error
shown, which is the inline equivalent of Phase 2's reopen loop and rather
more pleasant.

**Open question, to settle against working code:** the details panel is the
right-hand half of the body, which is narrow for YAML. If it proves too
cramped, the editor can expand to the full body width while active. That is
a layout change and this design does not commit to it — the inline editor
should be built first and looked at.

## A prerequisite: the selection resets on reload

`configSyncCmds` (`src/model/Update.go:118`) re-broadcasts the services list
after every config reload and always selects `orderedServices[0]` — the
alphabetically first service, not the one the user was on.

This is pre-existing and mostly invisible today: creating or deleting a
group reloads the config, and the selection jumping to the top of the list
is a minor annoyance on a page the user is about to leave anyway.

For editing it is disqualifying. The user edits `web`, the write succeeds,
the config reloads, and the details panel jumps to `api` — so the one thing
they want to see, their change reflected in the panel, is the one thing they
don't get. It reads as "the edit didn't work".

So the reload must preserve the current selection when the name still
exists, falling back to the first entry when it doesn't. This is a small
change to `configSyncCmds`, it fixes create/delete group too, and it lands
before any of the three phases.

## The file is not the container

Writing to the compose file does not change a running container. The app
says so wherever an edit is saved:

> Applies on next start (`s`) — restart won't recreate the container.

That is accurate: `start` maps to `docker compose up -d`
(`src/utils/DockerCompose.go`), which recreates a container whose config has
changed. `restart` maps to `docker compose restart`, which does not.

Recreating the container as a side effect of saving is deliberately not part
of any phase — it couples a file edit to a destructive container operation,
and `s` is one keypress away.

## Non-goals

- **Adding or deleting whole services**, via the per-service editor. Phase 1
  covers it by editing the file directly; a structured version belongs with
  the Compose Files page.
- **Renaming a service.** See Phase 2 — it needs `depends_on:` rewriting to
  be safe.
- **Reacting to the compose file changing on disk** while the app runs. The
  periodic poll refreshes container state, not config. Out of scope
  throughout, and worth its own item.
- **A schema-aware YAML editor** — completion, field docs, structural
  folding. The validation described here is parse-and-load, nothing richer.

## Testing

- `src/utils/` — the three pure functions carry the feature and take the
  bulk of the tests: fragment extraction preserves comments and formatting;
  splicing replaces only the target service and leaves neighbours, key
  order and comments untouched; a renamed key, a multi-key document, a
  non-mapping body and unparseable YAML each error; validation rejects a
  structurally-valid-but-invalid-compose document.
- `src/model/` — selection preservation across a reload, and edit mode
  swallowing the docker action keys (Phase 3).
- The editor phases shell out to `$EDITOR`, which the tests set to a script
  that rewrites the temp file — the same trick the TODO already proposes for
  faking `docker` on `PATH`.

## Related

- [Design](../../DESIGN.md) §3 (compose file as source of truth), §6.3
  (editing belongs on the Dashboard).
- [Create/delete profiles design](2026-07-22-create-delete-profiles-design.md)
  — established the modal → command → `yaml.Node` → reload path.
