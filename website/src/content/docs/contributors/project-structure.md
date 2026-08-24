---
title: Project Structure
description: The src/ tree with per-package responsibilities.
---

# Project Structure

```
main.go              # entry point — flags, config load, starts the Bubble Tea program
src/
├── model/           # the top-level Bubble Tea model (AppModel, Init, Update, View)
├── components/      # one nested model per panel, list, modal and overlay
├── cmds/            # message types, and the tea.Cmds that produce them
├── apptypes/        # shared data types (list items, docker container, pages)
├── keys/            # every keybinding, declared once — panels, footer and ? all read it
├── utils/           # the non-Bubble Tea half: compose loading, YAML writing, docker exec
├── appstyles/       # the Theme type and the 13 registered themes
├── config/          # persisted preferences (~/.config/cais/config.yaml)
├── highlight/       # read-only YAML highlighting for the Files page
├── banner/          # the "cais" figlet ASCII banner, gradient-tinted from the active theme
└── constants/       # layout widths, branding, version reader
demo/                # VHS tapes, the recorded gif and screenshots, and their fixture stack
docs/                # DESIGN.md (why), ROADMAP.md (order), plans/ (what's next, in sequence)
website/             # this documentation site (Astro / Starlight)
```

## Package responsibilities

### `src/model` — the app

`AppModel`, its `Init`/`Update`/`View`, and the message-routing that owns every screen. This is the only package that imports the leaf component packages — nothing downstream imports `src/components`, so `chrome` cannot become part of an import cycle no matter who ends up depending on it.

### `src/components` — the leaf models

One folder per model (`serviceslist`, `detailspanel`, `groupnamemodal`, …), each holding `Model.go`, `Update.go`, and `View.go` split out once the model earns it (roughly 150 lines, or `View` growing its own render helpers — a 60-line model stays one file). The constructor is always `New`; the exported type is always `Model`.

`src/components/chrome` is the one shared package: rendering and layout that more than one model needs (`PanelFrame`, panel body layout, key-hint rendering, the spinner, `HealthColor`/`Truncate`). A helper earns its way into `chrome` by having a second caller, not by convenience.

No model package imports another's internals; the only inter-model reference in the tree is `groupnamemodal` constructing `servicechecklistmodal.New` at the handoff point in the create-group flow, both through exported API.

### `src/cmds` — messages and commands

Message types, and the `tea.Cmd`s that produce them. The request/response split lives here: panels emit intents (`cmds.RunDockerActionMsg`, `cmds.CreateGroupRequestMsg`, `cmds.OpenEditorMsg`), and `AppModel` turns them into commands that carry the resolved file.

### `src/apptypes` — shared data types

List items (`ServiceListItem`, `GroupListItem`), `DockerContainer`, `ThemeItem`, and the page definitions (`PageTitles`, `PageLabels`, `PageShortcut`). The page list drives the nav bar, the footer's `1-N` hint, and the alt+letter aliases — a new tab extends all of them instead of drifting from them.

### `src/keys` — the keybinding source of truth

Every keybinding, declared once. Components match against these bindings; the footer and the `?` overlay render from them. See [Keybinding system](/contributors/keybinding-system/).

### `src/utils` — the non-Bubble Tea half

Compose loading (`GetComposeFileName`, `ComposeFileArgs`), YAML writing (`ApplyServiceFragment`, `AddServiceFragment`, `SetGroupMembers`, `ApplyHealthcheck`), docker exec (`DockerCompose`, `DockerComposePs`, `DockerLogs`), atomic writes (`ReplaceFileAtomically`, `SnapshotFile`), the backup store (`ListBackups`, `RestoreBackup`), service URL resolution (`ResolveURL`), healthcheck templates (`TemplatesFor`, `HealthcheckCatalog`), and the Docker preflight classifier.

### `src/appstyles` — themes

The `Theme` type and the 13 registered themes, built by `newTheme` from a handful of base colors. `appstyles.Active` is the one `Theme` in effect; every call site reads it fresh. See [Theme system](/contributors/theme-system/).

### `src/config` — persisted preferences

`~/.config/cais/config.yaml`. One field today (`theme`), designed to absorb more without changing existing callers.

### `src/highlight` — YAML highlighting

A hand-rolled, line-oriented YAML highlighter for the Files page. It colors keys, quoted strings, and comments from the active theme as a *display layer over the raw bytes* — it changes no byte, so the view still matches the file `E` opens. It tracks block scalars so a `command: |` body is treated as literal text, and it is best-effort by design, degrading to plain text on anything it does not recognize.

### `src/banner` — the About modal's brand mark

The "Cais" figlet ASCII art rendered by the About overlay (`a`). The letters are baked in as string literals; only the horizontal color gradient (`Accent` → `StatusStarting`) is theme-derived at render time.

### `src/constants` — layout and branding

Layout widths, branding, and the version reader (`constants.Version()`).

## Two seams worth knowing before you touch anything

1. **`src/keys` is the single declaration of every binding.** A key added anywhere else will not be advertised and may collide.
2. **`appstyles.Active` is the single source of color.** Read it fresh at each call site (`appstyles.Active.TextPrimary`, never a cached package-level `var`) or a theme switch will not repaint what you wrote.