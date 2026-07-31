# Plan: .env Secrets Support — Management Surface + Masking Discipline

Feature request: *"add support to secrets on a .env file."* Researched the field
(dotenv-tui, dockge, Portainer, k9s, compose's own semantics, the encrypted-.env
ecosystem), read the actual code the app runs (`compose-go v2.12.1` in the module
cache, not docs), and asked the owner for scope. Owner chose: **both** — a .env
management surface in the TUI *and* masking discipline across the app.

## Status of the feature — an honest reframing

**The .env is already read.** `src/utils/ReadConfigFile.go` passes
`cli.WithOsEnv` and `cli.WithDotEnv`, so every compose load already merges the
project `.env` for `${VAR}` interpolation, and every `docker compose` call the
app makes agrees on the same file: the app resolves the compose path and passes
it as `--file` (`src/utils/ComposeArgs.go`), and compose-go's `GetWorkingDir`
derives the `.env` location from that same path's directory. Verified in
`compose-go@v2.12.1/cli/options.go:410` — `.env` is `filepath.Dir(ConfigPaths[0])/.env`,
which is what `docker compose -f <path>` uses too.

**What is missing is the whole user-visible story:**

1. **No surface.** The user can never see, edit, or manage the file that
   configures everything the TUI operates on. The Files page shows the compose
   file; `.env` — where the secrets live — is invisible. The inline service
   editor even shows `environment: KEY=value` lines, but there is no way to
   reach the file that should hold those values.
2. **No masking.** Nothing in the app classifies or masks secret values. Today
   this leaks almost nothing (the details panels show PUID/PGID only), but the
   moment anyone adds an Environment row to a details panel, or the .env page
   exists, plaintext values are on screen by default.
3. **No hygiene on write.** `.env` files are commonly created world-readable
   (`0644`) by tutorials, and a naive writer would keep it that way or worse —
   replace a symlinked `.env` (dotfiles pattern) with a regular file, or rewrite
   the file and destroy quoting/comments.

**The plan below is the full feature: a new page that manages the project `.env`
with secrets masked by default, plus a stated masking boundary for the rest of
the app.** Verdict up front: worth doing; it is the natural completion of the
Files-page work, and it reuses established patterns (`cmds`, `ReplaceFileAtomically`,
the pages mechanism) — but it must be built in the order in §Implementation
order, because the write path has sharp edges (symlinks, mode, quoting, reload).

## Research: what the field actually does

Sources retrieved 2026-07-31 unless noted.

**Compose's own semantics — the ground truth.**

- *The two flavors of `.env`.* There are two unrelated things both called
  ".env": the **project `.env`** next to `compose.yml`, used only for `$VAR`
  interpolation inside the compose file, and **`env_file:`** at service level,
  injected as runtime environment into the container (Guillaume Lours, "Docker
  Compose Tip #56", https://lours.me/posts/compose-tip-056-env-file-advanced/,
  published 2026-04-29). Mixing them up is the most common `.env` bug in the
  wild (echoed by env.dev's guide). **This feature targets the project `.env`.**
- *Precedence.* Shell environment > `.env` > compose-file defaults, with
  `$VAR`, `${VAR:-default}`, and `${VAR:?error}` forms (Docker docs,
  "Environment variables precedence in Docker Compose",
  https://docs.docker.com/compose/how-tos/environment-variables/envvars-precedence/,
  page dated 2026-06-22). Verified against the actual library this app uses:
  compose-go's `Mapping.Merge` (types/mapping.go:183) only fills keys *not
  already defined*, so the shell beats `.env` — consistent with the CLI.
- *The real compose "secrets" mechanism is not `.env`.* Compose has a top-level
  `secrets:` with file/environment-based secrets (Docker docs,
  https://docs.docker.com/compose/how-tos/use-secrets/). `.env` is a plaintext
  config file that happens to hold secrets — the plan treats it that way: mask,
  manage, protect the file; do not pretend it is a vault.

**The direct precedent: dotenv-tui** (jellydn, MIT, Bubble Tea,
https://github.com/jellydn/dotenv-tui). The closest tool to this ask. Its ideas
worth borrowing: **secret detection by key-name patterns and value shape**;
**format-hint placeholders** (`sk_***`, `ghp_***`, `eyJ***`) so an `.env.example`
shows the *shape* of a secret without the value; **preserving comments, blank
lines and key order**; and `.env.*` variant support. Its masking is not
reusable as-is (it classifies to *generate examples*, not to protect a screen),
but the key-name pattern list is a good starting vocabulary. Its `.env.*`
multi-file scanning is a deliberate **non-goal** here (see Scope).

**The TUI masking precedent: k9s.** k9s's data-masking approach is
*presentation-only*: configured patterns replace sensitive values in rendered
resource views while the real data is untouched (hoop.dev blog, "K9S Data
Masking", https://hoop.dev/blog/k9s-data-masking-shield-your-kubernetes-secrets,
retrieved 2026-07-31 — vendor blog, cited for the technique, not as an authority
on k9s internals). Two properties matter for this plan: masking is a **render
decision**, never a data mutation; and masking applies to *resource views* —
k9s does not (and cannot) mask container logs, because logs are arbitrary
stdout. This plan adopts both properties.

**The web-UI precedents: dockge and Portainer.** Both let users edit a stack's
compose file and its environment from a web UI; Portainer's `stack.env` is
container-environment-only and does *not* do full compose interpolation
(https://docs.portainer.io/faqs/troubleshooting/stacks-deployments-and-updates/environment-variable-management-in-docker-.env-vs.-stack.env).
Nothing there is TUI-specific enough to copy; they confirm the product shape
(compose stack managers expose the env file), not the technique.

**The encrypted-.env ecosystem: dotenvx.** `dotenvx encrypt` turns `.env`
into AES-256-encrypted values with keys in `.env.keys` (secp256k1 ECIES),
specifically so plaintext secrets are not committed to git
(https://dotenvx.com/docs/quickstart/encryption; npm package description,
retrieved 2026-07-31). This is the future-proofing constraint: **the app must
never assume `.env` is parseable** — an encrypted or otherwise malformed file
must surface as a parse error, not crash, not silently rewrite. Decryption is
out of scope (see Alternatives).

**CLI secret hygiene — the constraints masking exists for.** Secrets on the
command line leak via `ps`; scrollback and clipboard managers are standing
leak vectors (GitGuardian, "How to Handle Secrets at the Command Line",
https://blog.gitguardian.com/secrets-at-the-command-line/, retrieved 2026-07-31;
Smallstep, "How to Handle Secrets on the Command Line",
https://smallstep.com/blog/command-line-secrets/). Consequences for this plan:
never pass a secret value on a docker argv; mask on the alternate screen;
treat clipboard as a deliberate, confirmed act.

## Scope — the boundary, explicitly

**In (v1):** the project `.env` in the compose file's directory — the one
compose-go already loads. View (masked), reveal, copy, add, edit, delete,
raw inline edit, `$EDITOR` handover, mode hygiene, parse-error reporting.

**Out (stated, not accidental):**

- **`env_file:` service files** — a different feature (container injection, not
  interpolation). Listing them from compose and editing them is a follow-up.
- **Docker `secrets:`** — a different mechanism; nothing to add here.
- **`.env.*` variants** (`.env.production`, `.env.local`) — dotenv-tui's
  differentiator, but compose only auto-loads the one `.env`; the variants are
  only reachable via `--env-file`/`COMPOSE_ENV_FILES`, which the app does not
  pass. Follow-up, and only if users actually hit it.
- **`.env.example` generation** — dotenv-tui's core feature; a nice follow-up,
  and the *format-hint* idea belongs in it, not in v1 masking.
- **Encrypted `.env` (dotenvx) support** — detect and report, never decrypt.
- **Secret generation/rotation** (openssl/uuid) — one-key filler later; not
  needed for the surface to be useful.

## Design decisions

### D1. Where it lives: a new page, label "Env"

The decision checklist in `docs/DESIGN.md` §6.1: a feature that is neither group
nor service belongs on a new page. `.env` is project-level config — but the
Files page is deliberately a *raw-bytes viewport* for the compose file
(`docs/DESIGN.md` §The Files page), and the env surface needs a *value table*
with per-row operations (reveal/copy/edit/delete). Folding that into Files
would blur "which file am I acting on" — the Files page's whole identity — with
a different file that has a different action set.

So: a fourth page. `src/apptypes/Pages.go` makes this a well-trodden path —
append `"Env"` to `PageTitles`, add a label, the digit `4` and the `alt+e`
chord come along automatically (pages are an ordered list; digits and chords
are derived, per `PageShortcut`). Keybindings for the page join the single
keymap in `src/keys` (established in Phase 1). The page is only reachable when
a compose file is loaded, like every other page; it is a single panel, always
focused, mirroring the Files page's "always focused" rule.

Label: **"Env"**, not "Secrets" — the file holds non-secrets too (`TZ`,
`PUID`), and calling it Secrets invites users to treat it as a vault, which it
is not (see Research, compose `secrets:`).

### D2. The view: key/value table, mask-by-default, no classification here

The page body is a table: KEY | VALUE | (mode of the file, one line in the
header). Key facts:

- **Every value renders as a fixed-width mask** (`••••••••`), regardless of
  key name or value shape. Fixed width, not length-matched: mask length must
  not leak `sk_live_…` vs `sk_test_…` (the k9s lesson — masks hide content,
  and length is content).
- **`Enter` / `r` reveals the selected row.** Reveal is per-row, transient:
  any navigation (arrow keys), page switch, modal open, or focus change
  re-masks. No auto-timeout timer in v1: a timer is untestable in the rig and
  the navigation rule already closes the leak window (the value is only on
  screen while the row is selected). A config knob (`mask: false` for users
  who want plain view) is a follow-up.
- **`c` copies the revealed value** to the system clipboard
  (`github.com/atotto/clipboard`, already an indirect dep) with a transient
  confirmation on the status line. Copying is the deliberate act — clipboard
  managers make auto-clear unreliable; say so in the docs rather than fake it.
- **No heuristics on this page.** Mask-all has zero false negatives in the one
  place that exists to show secrets; classification (dotenv-tui's key-name +
  value-shape lists) moves to the details panels (D3), where showing *some*
  values is the point.

The table is fed from the **raw file lines**, not from compose-go's parsed
`Environment` map: the map loses comments, quoting, ordering, duplicates and
unparsable lines, all of which this page must show. Parse it with the same
parser compose-go uses (`github.com/compose-spec/compose-go/v2/dotenv`) so the
page and the interpolation agree.

### D3. Masking elsewhere: a `secretHint` helper and a stated boundary

The only env values rendered outside the new page today are PUID/PGID
(`DetailsPanel.go:824`, `BasicInfo.go:49`) — not secrets. The rule going
forward, in one `src/utils/SecretHint.go` helper:

- Key-name patterns (suffix/prefix/contains on the uppercased key):
  `KEY`, `TOKEN`, `SECRET`, `PASSWORD`, `PASS`, `CREDENTIAL`, `APIKEY`,
  `APISECRET`, `AUTH` — the dotenv-tui vocabulary, trimmed to false-positive-free
  terms. `URL`/`HOST`/`PORT`/`TZ`/`PUID`/`PGID` explicitly **not** secret
  (`DATABASE_URL` contains credentials but is not a credential itself; showing
  it is what the details panel is for).
- Used by any component that renders arbitrary env values. Today that is
  nothing; the helper exists so the *next* panel (an Environment row in the
  service details config table, the obvious follow-up) masks by default.
- Value-shape heuristics (JWT `eyJ`, `sk_`, `ghp_`, long base64) **rejected**:
  noisy, and the page that shows secrets already masks everything. dotenv-tui's
  use of them (format hints in `.env.example`) is a different purpose.

**Deliberately unmasked — this is a documented boundary, not an omission:**

- The inline service editor and the Files raw viewport: an editor that masks
  the thing you are editing is broken. Editing *is* reveal.
- The logs modal: arbitrary container stdout; k9s itself cannot mask logs.
- `$EDITOR` handover: the whole file is shown to a real editor by definition.

### D4. Operations

| Key | Action |
|---|---|
| `Enter` / `r` | reveal selected value (re-mask on any navigation) |
| `c` | copy revealed value to clipboard + confirm |
| `a` | add variable (modal: key field + masked value field) |
| `e` | edit value of selected key (inline masked input) |
| `d` | delete key (confirm modal; deletes *all* occurrences, count shown) |
| `o` | raw inline edit of the whole file (textarea, dotenv validation on save) |
| `E` | open the file in `$EDITOR` (reuses the existing handover) |
| `m` | chmod 600 (only shown/offered when the file is world-readable) |

`a`/`e`'s value input uses a masked text field (renders `••`, stores real
bytes) — the same reveal discipline applied to input. `o` is the Files-page
model (raw bytes, line-oriented); save validates by parsing with the dotenv
parser and reports errors on the status line without touching the file,
mirroring the YAML editor's save-validation.

### D5. The write path — where the sharp edges are

All writes go through `utils.ReplaceFileAtomically` (already exists,
`src/utils/AtomicWrite.go`) with these additions:

1. **Resolve symlinks before writing.** `ReplaceFileAtomically` renames a temp
   file onto the target — if `.env` is a symlink (the dotfiles-repo pattern:
   `~/.config/foo/.env -> ~/secrets/foo.env`), the rename replaces the *symlink*
   with a regular file and the original target keeps the old secret. Resolve
   `filepath.EvalSymlinks` first and atomic-write the resolved target. Detect
   and warn on the status line that the file is a symlink.
2. **Line-preserving edits.** Only the lines that changed are rewritten.
   Comments, blank lines, key order, quoting, `export ` prefixes, CRLF
   endings and BOM survive — the same house value the compose writer already
   honors ("your comments, quoting and key order are kept").
   A small quoting/escaping util round-trips values per the godotenv rules
   compose-go parses (quote when the value contains `#`, a space, or is empty;
   escape `\`, `"` inside quotes) — with tests that a value survives
   parse → write → parse unchanged.
3. **File mode.** New `.env` created `0600` (a change from
   `ReplaceFileAtomically`'s `0644` default — the env writer passes an explicit
   mode). Existing mode preserved. When the file is world-readable (`0644`),
   the status line warns and `m` offers the chmod.
4. **Duplicates.** The parser is last-wins (verified: `out[key] = value` in
   `dotenv/parser.go`), so the effective value is the *last* occurrence. The
   table shows one row per key; a duplicate-key warning goes on the status
   line; edit replaces the last occurrence (the effective one); delete removes
   all occurrences after a confirm that states the count.
5. **Unparsable lines** are shown in the table as a parse-error row (line
   number + reason) and are never touched by writes.

### D6. A write changes interpolation — reload the project

Editing `.env` changes what `${VAR}` resolves to in the compose file, which
changes services, images, ports, and the details panels. So every successful
write triggers the same background reload the Files page uses after a write
(`cmds.GetConfig`), re-reading compose + `.env` together. If the edit breaks
load — `${VAR:?err}` now missing, or a parse error — the existing error path
surfaces it and the app **keeps the last good project**; it does not revert the
`.env` file. The Files page's pattern ("re-sync from disk instead of going
stale") applies unchanged; the stale-view risk is the same class.

### D7. Truth on the status line

The page's footer shows: resolved `.env` path, **loaded vs not-loaded**,
variable count, parse-error count, duplicate count, mode (with warning), and a
symlink marker. "Loaded" requires one small change to the load path:
`ReadConfigFile` doesn't currently expose which env files compose-go actually
consumed (`ProjectOptions.EnvFiles` is populated inside `LoadProject`). The
reload message should carry back the resolved `.env` path and whether it was
loaded, so the page never claims authority it doesn't have —
`COMPOSE_DISABLE_ENV_FILE=true` in the app's environment makes compose-go skip
the file entirely (verified in `cli/options.go:262`), and the shell-shadowing
rule (shell > `.env`) means the *effective* value can differ from the file's
value. The page shows file bytes; the status line reports whether they are
in force.

## Edge cases

- **No `.env`** — empty state with an "add first variable" affordance; adding
  one creates the file at `0600`.
- **`COMPOSE_DISABLE_ENV_FILE=true`** — page still works on the file, status
  says "present, not loaded".
- **Shell shadows `.env`** — file value differs from effective value; status
  says "shell overrides N keys" (compute by diffing the file's parsed map
  against the project's Environment map — cheap, both already exist).
- **`.env` is a directory** — compose-go errors at load (verified
  `dotenv/env.go:47`); page shows the error, no write path.
- **Symlinked `.env`** — see D5.1; warn, write through to the target.
- **World-readable mode** — warn + `m` to chmod (D5.3).
- **Duplicate keys** — effective-value row + count warning (D5.4).
- **Unparsable/encrypted file (dotenvx)** — parse-error rows, untouched on
  write; never crash, never silently rewrite an encrypted file into garbage.
- **`${OTHER_VAR}` inside `.env` values** — the parser expands against the
  shell env (verified `dotenv/env.go` lookup). The table shows *file* bytes,
  not expanded values; editing `OTHER_VAR` changes the effective value of a
  dependent key — that is compose semantics, note it in the docs, don't
  paper over it.
- **Quoted values, `export ` prefixes, escapes, values containing `=` or `#`** —
  the round-trip util (D5.2) with tests; unchanged lines untouched.
- **CRLF / BOM / unicode** — preserved byte-for-byte on unchanged lines; new
  lines use the file's dominant line ending.
- **Concurrent external edits** — the existing 5 s background refresh picks the
  file up; the page re-syncs from disk on activation like the Files page.
- **Large files** — the table scrolls; no new limit. (`.env` files are
  practically < 200 lines; no virtualization needed.)
- **The demo/VHS** — demo fixtures use a fake `.env`; masked renders make
  good screenshots. Verify the tape's fixture has no real secrets.

## Security drawbacks — what masking does not fix (say it in the docs)

1. **Alternate screen is not a vault.** The app renders on the alternate
   screen buffer, but tmux copy-mode, terminal selection, and screen
   recorders still see revealed values. Reveal-on-demand + re-mask-on-navigation
   shrinks the window; it does not close it.
2. **Clipboard managers** retain copied values. `c` is explicit and confirmed;
   auto-clear is unreliable across platforms/managers — document, don't fake.
3. **argv discipline.** No code path may pass a secret value to a `docker`
   argument (it would show in `ps`). The app's docker calls take no env values
   today; the new code adds none. A code-review note, not a feature.
4. **Logs.** The logs modal can and will show secrets the services print.
   Unmaskable; out of the masking boundary.
5. **Mask length.** Fixed-width masks only (D2); a length-matched mask is a
   fingerprint.
6. **The file itself is plaintext.** Masking protects the *screen*, not the
   disk. The docs should point at `chmod 600`, at dotenvx for committed
   encryption, and at docker `secrets:` for runtime secrets — three sentences,
   not a feature.

## Alternatives considered

1. **Do nothing.** The `.env` stays invisible; the app keeps silently
   depending on a file its users cannot inspect or fix from inside it. The
   feature ask is explicit; do-nothing fails it.
2. **Files-page toggle** (view `.env` raw in the Files viewport, `E` to edit).
   Cheapest possible surface; reuses everything. Rejected as the *primary*: it
   cannot host per-value operations (reveal/copy/add/edit/delete), and it
   muddies the Files page's "which file am I acting on" identity. Kept as the
   fallback if the owner rejects the new tab.
3. **A modal opened from the Files page.** Even cheaper; same objections as 2,
   plus a modal can't hold a table with a persistent footer. Rejected.
4. **Heuristic masking everywhere (dotenv-tui's classifier as the masker).**
   False positives and negatives in the one place that exists to show secrets;
   mask-all (D2) is simpler and strictly safer. Heuristics survive only as
   `secretHint` for future details-panel rows (D3).
5. **Encrypted `.env` support (dotenvx) / integration with pass/sops/Doppler.**
   Real capability, but it is a different product axis (external key
   management, new deps, new failure modes). The plan's encrypted-file
   handling (detect, report, never destroy) is the correct v1 posture.
6. **Auto-timeout re-mask timer.** Rejected: untestable in the rig, and the
   navigation-based re-mask covers the leak window. Config knob later if
   users ask.

## Who decides / blockers

- **No external sign-off.** Everything is repo-local; no money, no accounts,
  no new dependencies beyond what already exists.
- **Owner decisions (recommendation in parens):** new "Env" tab vs Files-page
  toggle (new tab, D1); mask-all vs heuristic (mask-all, D2); `m` chmod action
  vs warn-only (action, D5.3 — ~15 lines and directly serves the "secrets"
  ask); reload-on-write vs manual reload (reload, D6 — manual reload would be
  a stale-view bug the moment a user edits `.env` and wonders why nothing
  changed); keymap bindings for `a`/`c`/`d`/`e`/`o`/`m` (the table in D4 is
  the draft; final letters live in `src/keys`).

## Blast radius per step

| Step | Files touched | Other effects |
|---|---|---|
| 1 — `SecretHint` helper + tests | `src/utils/SecretHint.go` (+test) | None (nothing calls it yet; grep after adding) |
| 2 — load-path truth | `src/utils/ReadConfigFile.go`, `src/cmds/GetConfig.go`, reload messages | The compose-load message carries `.env` path + loaded flag; `src/model` message structs grow fields — all existing consumers compile-checked by the compiler |
| 3 — Env page, masked table, reveal/copy | `src/apptypes/Pages.go`, `src/keys/Keys.go`, new `src/components/EnvPanel.go`, `src/cmds/GetEnvFileContents.go`(+`SaveEnvFile.go`), `src/model/Update.go`, `src/model/View.go` | Fourth tab: digits `1`–`4`, `[`/`]`, new `alt+e` chord; footer, help overlay, and keybinding bar gain the page's keys; `?` overlay lists them (existing mechanism) |
| 4 — add/edit/delete + masked input | modal components (pattern: `GroupNameModal`/`ConfirmModal`), edit-command plumbing | Reuses `OpenConfirmModal`; new value-input component |
| 5 — write path: symlink, mode, line-preserving writes, dupes | `src/utils/ApplyEnvEdit.go` (+tests), `AtomicWrite` gains a mode parameter | **`ReplaceFileAtomically`'s default-mode callers must be swept** — changing its signature touches every existing writer (`ApplyServiceFragment`, group writes); keep the old function and add the mode variant to limit blast radius |
| 6 — reload-on-write + status line | `src/model/Update.go` | Background reload already exists; re-interpolation failures go down the existing error path |
| 7 — docs + demo | `README.md`, `docs/DESIGN.md`, `TODO.md`, demo tape | README keybindings table, DESIGN §The Env page, fixture `.env` in `demo/fixtures` |

## Implementation order

1. **Step 1 + 2 first** — the helper and the load-path truth. Small, isolated,
   testable, and everything else depends on them. (`go build ./... && go vet
   ./... && go test ./...` green at every commit, per ROADMAP conventions.)
2. **Step 3** — the page with a *read-only* masked table + reveal + copy.
   This is the visible core; demo it before building writes.
3. **Steps 4 + 5 together** — the operations and the write path are one
   deliverable (add/edit/delete is meaningless without the safe writer), and
   the writer's edge cases (symlink, mode, dupes, round-trip) are a single
   test suite.
4. **Step 6** — reload-on-write, then the status-line truth (D7) once the
   reload message carries the fields.
5. **Step 7** — docs + demo tape, last.

## Acceptance criteria

1. A new "Env" tab (digit `4`, `alt+e`) shows the project `.env` as a
   key/value table with every value masked at a fixed width; no length leak
   (assert the mask string in a test).
2. Reveal is per-row and re-masks on any navigation, page switch, or modal
   open; `c` copies with confirmation.
3. Add/edit/delete round-trip through `ReplaceFileAtomically` (mode-aware):
   comments, blank lines, quoting, key order, `export` prefixes, CRLF and BOM
   survive; parse → write → parse is identity for every fixture in the round-
   trip test.
4. A symlinked `.env` is written through to its target; the symlink survives;
   the status line says the file is a symlink.
5. A new `.env` is created `0600`; a world-readable `.env` warns and `m`
   chmods it.
6. Editing `.env` re-interpolates: a value that feeds `${VAR}` in the compose
   file changes the Services page after the write; a write that breaks
   interpolation shows the error and keeps the last good project (no file
   revert).
7. `COMPOSE_DISABLE_ENV_FILE` and shell-shadowing are reflected on the status
   line, not hidden.
8. Unparsable/encrypted `.env` files show parse-error rows, never crash, and
   are byte-identical after a write that touched another key.
9. `secretHint` exists, is tested, and nothing in the app renders a
   secret-hinted env value unmasked (sweep the two existing PUID/PGID sites
   and any new render site).
10. `README.md` documents the masking boundary (editors, Files raw view, logs
    deliberately unmasked) and the security notes in §Security drawbacks.
11. Demo tape renders the masked Env page with the fixture `.env`; no real
    secret appears in any committed fixture.

## Review pass — 2026-07-31 (changes to make before implementing)

A second reader went through this plan against the code it names. The plan
stands; nine things in it are wrong, unstated, or cheaper done another way.
**These override the sections above where they conflict.**

### R1. `a` for "add variable" collides with a global key

`a` is `Global.About` (`src/keys/Keys.go:161`), live on every page. The D4
table cannot have it.

The right key is already declared: **`n` — `List.New`, "new"**
(`Keys.go:172`). It is bound on Home only (`src/model/Update.go:441` gates it
with `m.activePage == "Home"`), so the Services and Env pages leave it free,
and "one verb is one binding" (the rule at the top of `src/keys/Keys.go`) says
the verb *new* is `n` wherever it appears.

### R2. `r` for "reveal" is `Details.Restart`

`r` is restart (`Keys.go:196`). Reusing it for reveal is exactly the collision
the `R`-for-rename comment was written to avoid ("uppercase so it does not
collide with the details panel's lowercase r").

Use **`v`** (view/reveal) — unbound anywhere today. The revised D4 table:

| Key | Action | Why this key |
|---|---|---|
| `v` | reveal selected value | free; "view" |
| `c` | copy revealed value | free |
| `n` | new variable | `List.New`, R1 |
| `e` | edit value | `List.Edit`/`Details.EditService` — same verb |
| `d` | delete key | `List.Delete` — same verb |
| `o` | raw edit of the whole file | free |
| `E` | open in `$EDITOR` | `Details.EditFile` — same verb |
| `m` | chmod 600 | free (see R7 for Windows) |

Note for whoever writes the keymap: `h` is spoken for by
`docs/plans/healthcheck-insertion.md`. Do not spend it here.

### R3. Use `tea.SetClipboard`, not `github.com/atotto/clipboard`

D2 proposes atotto. atotto shells out to `pbcopy` on macOS and
`xclip`/`xsel` on Linux — neither of which exists on a headless server reached
over SSH, which is this app's core audience. `c` would fail precisely where the
app is most used, and the failure would look like a bug in the feature.

Bubble Tea v2 already ships the right mechanism: `tea.SetClipboard` writes the
value with **OSC 52**, an escape sequence the *terminal* interprets — so over
SSH it lands in the clipboard of the machine the human is sitting at, which is
what they actually wanted. Verified present in
`charm.land/bubbletea/v2@v2.0.7/clipboard.go` (`SetClipboard`, `ClipboardMsg`).
It is a `tea.Cmd`, so it also tests like every other command.

Two costs, both worth stating in §Security drawbacks rather than hiding:
OSC 52 puts the secret into the terminal's *output stream* (a logged or
recorded session captures it), and some terminals ship it disabled — tmux
needs `set -g set-clipboard on`. So `c` needs a status-line confirmation that
says the request was *sent*, not that it succeeded; OSC 52 gives no
acknowledgement. Also: this removes a dependency from the plan rather than
adding one — atotto stays indirect.

### R4. The 0600 mode must be set before the bytes are written

D5.3 says "New `.env` created `0600`". Read `src/utils/AtomicWrite.go:39-49`
before implementing: `os.CreateTemp` opens at 0600, and
`ReplaceFileAtomically` then **chmods the temp file up to 0644 when the target
does not exist**, before `temp.Write`. A mode variant that chmods after the
rename leaves the secret world-readable for the whole write — and on a busy
box that window is real.

So: `ReplaceFileAtomicallyWithMode(fileName string, contents []byte, mode
os.FileMode)`, applying the mode at the same point the existing function does
(before the write), with the existing function calling it with its current
0644/preserve behaviour so no existing caller changes. Test it by asserting
the mode of a `.env` created by the app is `0600`, and that a pre-existing
`0644` `.env` keeps `0644` (D5.3 preserves; `m` is how the user tightens it).

### R5. Symlink resolution has two consequences the plan doesn't state

D5.1 resolves with `filepath.EvalSymlinks` and writes the target. Correct — and
it moves the write into the *target's* directory, because the temp file is
created next to the file being replaced (`AtomicWrite.go:26`). Therefore:

1. The failure surface moves. A read-only secrets directory fails at
   `CreateTemp`, not at rename, and the error must name the **resolved** path
   or the user will not understand which file the app could not write.
2. The mode carried over is the target's, not the symlink's. That is right,
   and worth a comment so nobody "fixes" it later.
3. `EvalSymlinks` errors on a **dangling** symlink. Treat that as "present but
   broken": show it on the status line, offer no write, and do **not** create a
   regular file at the symlink's path — that would silently detach the user's
   dotfiles setup.

### R6. A fourth tab needs the owner to reopen a recorded decision

`docs/ROADMAP.md` §Decisions already taken: *"Tabs for the alpha are Groups,
Services, Files. No dead placeholder tabs."* The alpha shipped, so the
constraint has arguably expired — but it is written down, and D1 adds a tab, so
the decision has to be re-taken and the ROADMAP line updated in the same phase.
That is a doc edit, not a discussion, but skipping it leaves the repo
contradicting itself.

The mechanics D1 claims do hold, verified: `Global.Page`'s help text is
`fmt.Sprintf("1-%d", len(apptypes.PageTitles))` (`Keys.go:153-156`), so the
digit hint becomes `1-4` on its own, and `PageShortcut` derives the chord from
the label's first letter (`Pages.go:33-46`) — Groups/Services/Files are g/s/f,
so **"Env" takes `alt+e` with no collision**.

### R7. `m` (chmod) is meaningless on Windows

`docs/plans/release-distribution.md` ships Windows binaries. Unix permission
bits do not survive there: `os.Stat().Mode().Perm()` reports something
plausible and `Chmod` mostly does nothing. Gate both the world-readable
warning and the `m` action behind `runtime.GOOS != "windows"`, and say so in
the footer's absence rather than showing a control that lies. One `if`, but it
has to be in the plan or it will ship as a bug report from the first Windows
user.

### R8. Split the phase: read-only Env page is a shippable unit

§Implementation order already sequences steps 1–2, then 3, then 4+5. Make that
split formal: **steps 1–3 are one phase** (branch, `--no-ff` merge, ROADMAP
row) and **steps 4–7 are the second**. A masked, revealable, copyable view of
`.env` is useful on its own, it is where all the visual risk lives, and it
lets the write path — the part with symlinks, modes and round-tripping — land
against a UI that already exists rather than at the same time as one.

### R9. Two additions to the "considered and rejected" list

So they do not get invented mid-implementation:

- **Warning that `.env` is tracked by git.** Tempting and cheap-looking
  (`git ls-files --error-unmatch .env`), but it makes the app shell out to a
  tool it otherwise never needs, and it is wrong in a worktree without git.
  A sentence in the README (".env holds secrets; add it to .gitignore") does
  the same job for free.
- **Reading `.env.example` to prefill keys on the add form.** That is
  dotenv-tui's feature, it is genuinely nice, and it is a follow-up — listed
  in §Scope as out, repeated here because "we already parse dotenv files" makes
  it look like a two-line addition at exactly the wrong moment.

### Unchanged by this review

Mask-all with a fixed width (D2), the raw-lines-not-parsed-map table (D2), the
`secretHint` boundary and its deliberately-unmasked list (D3), reload-on-write
(D6), the status-line truth including `COMPOSE_DISABLE_ENV_FILE` (D7), and
every entry in §Edge cases. Those were checked against the same code and hold.

## Do not

- Do not classify secrets by value shape on the Env page (D2); do not add
  `.env.*` variant scanning, `.env.example` generation, dotenvx decryption, or
  `env_file:` service files in v1 — each is a listed follow-up, and adding any
  of them mid-implementation is how this scope doubles.
- Do not change `ReplaceFileAtomically`'s existing callers' behavior — add the
  mode variant (blast-radius table, step 5).
- Do not write `.env` values onto any docker argv (Security drawbacks §3).
- Do not render a length-matched mask (D2).
- Do not put an auto-timeout re-mask timer in v1 (Alternatives §6).
- Do not touch `secrets:`/docker-swarm secret handling — out of scope by
  decision, not by accident.
- Do not bind `a` or `r` on the Env page (review R1, R2), and do not add
  `github.com/atotto/clipboard` as a direct dependency (R3).
- Do not chmod after the rename (R4), and do not create a regular file where a
  dangling symlink was (R5).
- Do not add git-awareness or `.env.example` prefilling (R9).
