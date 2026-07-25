# Edit Existing Services — Design

> **Status — in progress.** This is a live design, not a historical record.
> Unlike the other documents in this directory, the "today" statements below
> describe current behavior at the time of writing (2026-07-25). See
> [TODO](../../../TODO.md) for where it sits in the backlog.

## Context

Editing existing services is the last open item from the original roadmap.
The README has said "Editing existing services is still on the roadmap"
since the project started, and `DESIGN.md` §6.3 already commits to where it
lands: **the Dashboard, not Home** — Home is for groups, the Dashboard is
for per-service work.

Everything needed to write to the compose file already exists. The
create/delete-groups work built a comment-preserving `yaml.Node` edit path
(`src/utils/GroupTags.go`), and writes became crash-safe when
`writeComposeNode` moved to `utils.ReplaceFileAtomically`. What's missing is
a way to change anything other than a `profiles:` tag.

Today the Dashboard's details panel is read-only: it renders name, PUID/PGID,
image, groups and ports (`src/components/BasicInfo.go`), and the only keys it
accepts are the five docker actions plus `l` for logs.

## Goals

- Change a service's **image** from inside the TUI, writing through the same
  comment-preserving, atomic path the group tags use.
- Keep the compose file the single source of truth: after a write, reload it
  from disk via the existing `GetConfig` → `configSyncCmds` path rather than
  mutating the in-memory project.
- Establish the pattern — keybinding → modal → command → `yaml.Node` edit →
  reload — so ports and env vars can follow without re-litigating the design.

## Non-goals

- **Ports and env vars in this phase.** They are the natural next fields
  (TODO lists all three), but see "Why image first" below.
- Editing anything that isn't a service field: top-level `volumes:`,
  `networks:`, `x-` extensions, or the compose version.
- Adding or deleting whole services. Creating a service exists only in the
  bootstrap flow today; a general "add service" belongs with the Compose
  Files page.
- Applying the change to a running container as a side effect of saving. See
  "The file is not the container".

## Why image first, not all three fields at once

The three fields the TODO names are not equally hard, and bundling them
hides that:

- **`image:`** is a scalar. Editing it is `node.Value = newImage` — one
  unambiguous representation, and a text input is the honest UI for it.
- **`ports:`** is a sequence with two valid spellings: the short string form
  (`"8080:80"`) and the long mapping form (`target:`/`published:`/
  `protocol:`). A user who wrote the long form has done so deliberately, and
  round-tripping their list through a comma-separated text input would
  silently rewrite it to short form.
- **`environment:`** has the same problem twice over: a list of `KEY=value`
  strings or a mapping of `KEY: value`, and values that may be quoted,
  empty, or interpolated (`${VAR}`) — which `compose-go` resolves in the
  parsed project but which must be preserved verbatim in the file.

Image alone exercises the entire path end-to-end while the yaml work stays
trivially correct. The list-shaped fields need their own editor and their own
decision about syntax preservation; that decision is better made against
working code than in advance. **This phase does not build a generic form
that ports and env vars can slot into**, because a row of text inputs is the
wrong shape for both of them.

## UX

**Where.** Dashboard → services list → select a service (`space`) → tab to
the details panel → `e`.

`e` is free on both details panels. It is added to the Dashboard's service
details panel only: `GroupDetailsPanel` operates on a whole group, and
"edit the group's image" is meaningless.

**The modal.** A single-field form titled with the service name, prefilled
with the current image and the cursor at the end — the common case is
bumping a tag (`nginx:1.27` → `nginx:1.28`), so the existing value must be
editable rather than retyped.

```
Edit web

Image:
> nginx:alpine

enter save · esc cancel
```

Validation is inline in the modal, in the established convention: form
errors never reach `m.lastError`, IO errors always do. The rules are
deliberately thin — non-empty, and no whitespace. Image references have a
genuinely complicated grammar (registry host, port, namespace, tag, digest)
and a validator that is stricter than docker's is worse than no validator,
because it blocks references docker would have accepted.

**Esc cancels and writes nothing.** Consistent with every other modal in the
app; the compose file is never half-edited.

## The file is not the container

Writing `image: nginx:1.28` does not change a running container. This is the
one place where the feature can mislead, so the modal says so directly, and
the wording names the key that resolves it:

> Applies on next start (`s`) — restart won't recreate the container.

That is accurate: `start` maps to `docker compose up -d`
(`src/utils/DockerCompose.go`), which recreates a container whose config has
changed. `restart` maps to `docker compose restart`, which does not.

Offering to recreate the container from the save prompt is deliberately not
part of this phase — it silently couples a file edit to a destructive
container operation, and `s` is one keypress away.

## A prerequisite: the selection resets on reload

`configSyncCmds` (`src/model/Update.go`) re-broadcasts the services list
after every config reload and always selects `orderedServices[0]` — the
alphabetically first service, not the one the user was on.

This is pre-existing and mostly invisible today: creating or deleting a
group reloads the config, and the selection jumping back to the top of the
list is a minor annoyance on a page the user is about to leave anyway.

For editing it is disqualifying. The user edits `web`, the write succeeds,
the config reloads, and the details panel jumps to `api` — so the one thing
they want to see, their change reflected in the panel, is the one thing they
don't get. It reads as "the edit didn't work".

So the reload must preserve the current selection when the name still
exists, falling back to the first entry when it doesn't (the service was
renamed or removed outside the app). This is a small change to
`configSyncCmds`, it fixes create/delete-group too, and it lands **before**
the edit feature rather than as a follow-up.

## Data flow

Unchanged from the create/delete-groups shape:

1. `DetailsPanel` sees `e`, emits `cmds.OpenEditServiceModal(service)`.
2. `AppModel.Update` sets `activeModal` — the same handling as
   `OpenConfirmModalMsg`.
3. The modal validates, then emits
   `cmds.CloseModal(cmds.SetServiceImage(name, image))`.
4. `cmds.SetServiceImage` calls `utils.SetServiceImage`, which reads the
   compose node, edits it, and writes it back atomically.
5. `SetServiceImageMsg` is handled like `CreateGroupMsg`: an error goes to
   the banner, a success queues `cmds.GetConfig`, which reloads from disk
   and re-broadcasts — now preserving the selection.

## Behavior of the yaml edit

`utils.SetServiceImage(fileName, serviceName, image string) error`, living
alongside the group-tag functions and reusing `readComposeNode`,
`servicesMappingNode`, `findMappingValue` and `writeComposeNode`.

- **Existing `image:` key** — assign to the scalar's `Value` and leave the
  node otherwise untouched, so the user's quoting style survives.
- **Missing `image:` key** — append it, mirroring how `AddGroupTag` appends
  a missing `profiles:` key. A compose service may legitimately have `build:`
  and no `image:`; adding one names the built image, which is well-defined,
  and refusing would be a confusing dead end for a service whose Image field
  the panel renders as blank.
- **Unknown service** — return an error naming it, as `AddGroupTag` does.

**Anchors and merge keys.** A service that inherits `image:` from a YAML
anchor (`<<: *common`) has no local `image:` key, so the edit appends one.
That override is correct YAML and correct compose semantics, and it is the
only reasonable local edit — rewriting the shared anchor would silently
change every service that uses it. Worth knowing, not worth blocking.

## Testing

- `src/utils/` — unit tests for `SetServiceImage`: replaces an existing
  image, preserves surrounding comments and formatting, appends when the key
  is missing, errors on an unknown service. The existing
  `GroupTags_test.go` fixtures already carry inline comments to assert
  against.
- `src/model/` — a selection-preservation test on `configSyncCmds`, and an
  end-to-end test through the existing in-process rig (`rig_test.go`):
  select a service, press `e`, type, save, and assert both the file on disk
  and the refreshed panel.

## Related

- [Design](../../DESIGN.md) §3 (compose file as source of truth), §6.3
  (editing belongs on the Dashboard).
- [Create/delete profiles design](2026-07-22-create-delete-profiles-design.md)
  — established the modal → command → `yaml.Node` → reload path.
