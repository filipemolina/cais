# Plan: .env Secrets Support — Management Surface + Masking Discipline

> **Before you start.** Work on a feature branch of small commits, merged
> `--no-ff`; `go build ./... && go vet ./... && go test ./... && gofmt -l .`
> green at **every** commit, not just at the tip — `docs/ROADMAP.md`
> §Conventions is the full contract and `CONTRIBUTING.md` explains how a TUI
> gets tested. Behaviour that only shows on screen gets checked in the real app
> with VHS before it is committed. **Step 5 of the post-alpha order,** and the largest of them — it is two branches, not one; see §Implementation order.

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

Every decision below is **decided**, not proposed. Where this plan once
offered a recommendation in parentheses, the recommendation is now the
instruction; §Alternatives records what was rejected and why, so the reasoning
survives without reopening the choice.

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

**A fourth tab contradicts a written decision; re-take it in this phase.**
`docs/ROADMAP.md` §Decisions already taken says *"Tabs for the alpha are
Groups, Services, Files. No dead placeholder tabs."* The alpha has shipped, so
the constraint has expired — but it is written down, so the phase that adds
the tab **must also update that line in `docs/ROADMAP.md`**, or the repo
contradicts itself. A doc edit, not a discussion.

The mechanics D1 claims are verified: `Global.Page`'s help text is
`fmt.Sprintf("1-%d", len(apptypes.PageTitles))` (`Keys.go:153-156`), so the
digit hint becomes `1-4` by itself, and `PageShortcut` derives the chord from
the label's first letter (`Pages.go:33-46`) — Groups/Services/Files are g/s/f,
so **"Env" takes `alt+e` with no collision.**

### D2. The view: key/value table, mask-by-default, no classification here

The page body is a table: KEY | VALUE | (mode of the file, one line in the
header). Key facts:

- **Every value renders as a fixed-width mask** (`••••••••`), regardless of
  key name or value shape. Fixed width, not length-matched: mask length must
  not leak `sk_live_…` vs `sk_test_…` (the k9s lesson — masks hide content,
  and length is content).
- **`Enter` / `v` reveals the selected row.** Reveal is per-row, transient:
  any navigation (arrow keys), page switch, modal open, or focus change
  re-masks. No auto-timeout timer in v1: a timer is untestable in the rig and
  the navigation rule already closes the leak window (the value is only on
  screen while the row is selected). A config knob (`mask: false` for users
  who want plain view) is a follow-up. (`v`, not `r` — `r` is
  `Details.Restart`; see D4.)
- **`c` copies the revealed value using `tea.SetClipboard`** — *not*
  `github.com/atotto/clipboard` — with a transient confirmation on the status
  line. This matters more than it looks: atotto shells out to `pbcopy` on
  macOS and `xclip`/`xsel` on Linux, and **neither exists on a headless server
  reached over SSH**, which is this app's core audience. `c` would fail exactly
  where the app is most used, and it would look like a bug in the feature.
  `tea.SetClipboard` writes the value with **OSC 52**, an escape sequence the
  *terminal* interprets, so over SSH it lands in the clipboard of the machine
  the human is actually sitting at. Verified present in
  `charm.land/bubbletea/v2@v2.0.7/clipboard.go` (`SetClipboard`,
  `ClipboardMsg`); it is a `tea.Cmd`, so it tests like every other command,
  and it removes a dependency from this plan rather than adding one.
  OSC 52 gives **no acknowledgement**, so the confirmation must say the copy
  was *sent*, not that it succeeded (some terminals ship it disabled — tmux
  needs `set -g set-clipboard on`). Its two costs are stated in §Security
  drawbacks; do not paper over them.
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

```go
// SecretHint reports whether a value under this env key should be masked by
// default when something other than the Env page renders it. Key name only -
// value-shape heuristics are deliberately not used (see D3).
//
// It is a hint, not a guarantee: the Env page masks everything regardless,
// and this exists so the next panel to render arbitrary env values does the
// safe thing without its author having to think about it.
func SecretHint(key string) bool
```

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

These letters are final. Two earlier drafts of this table used `a` and `r`;
both collide with live global bindings, so they are listed under *Do not*.

| Key | Action | Why this key |
|---|---|---|
| `Enter` / `v` | reveal selected value (re-mask on any navigation) | `v` for view; unbound anywhere today. **Not `r`** — `r` is `Details.Restart` (`Keys.go:196`), and reusing it is the exact collision the `R`-for-rename comment exists to avoid |
| `c` | copy revealed value (OSC 52) + confirm | free |
| `n` | new variable (modal: key field + masked value field) | already declared as `List.New` (`Keys.go:172`) and gated to Home (`src/model/Update.go:441`), so Env is free to use it. **Not `a`** — `a` is `Global.About` (`Keys.go:161`), live on every page. "One verb is one binding": the verb *new* is `n` wherever it appears |
| `e` | edit value of selected key (inline masked input) | `List.Edit` / `Details.EditService` — same verb |
| `d` | delete key (confirm modal; deletes *all* occurrences, count shown) | `List.Delete` — same verb |
| `o` | raw inline edit of the whole file (textarea, dotenv validation on save) | free |
| `E` | open the file in `$EDITOR` (reuses the existing handover) | `Details.EditFile` — same verb |
| `m` | chmod 600 (offered only when the file is world-readable) | free. **Unix only** — see below |

**`m` and the world-readable warning are gated behind `runtime.GOOS !=
"windows"`.** `docs/plans/release-distribution.md` ships Windows binaries, and
Unix permission bits do not survive there: `os.Stat().Mode().Perm()` reports
something plausible and `Chmod` mostly does nothing. Show neither the warning
nor the key on Windows rather than shipping a control that lies. One `if` —
but written down, because otherwise it arrives as a bug report from the first
Windows user.

Note for whoever writes the keymap: `h` is spoken for by
`docs/plans/healthcheck-insertion.md`. Do not spend it here.

`n`/`e`'s value input uses a masked text field (renders `••`, stores real
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

   Three consequences, all of which must be handled:

   - **The failure surface moves.** The temp file is created next to the file
     being replaced (`AtomicWrite.go:26`), so resolving moves the write into
     the *target's* directory. A read-only secrets directory now fails at
     `CreateTemp`, not at rename, and the error must name the **resolved**
     path or the user cannot tell which file could not be written.
   - **The mode carried over is the target's, not the symlink's.** That is
     correct; say so in a comment so nobody "fixes" it later.
   - **`EvalSymlinks` errors on a dangling symlink.** Treat that as "present
     but broken": report it on the status line, offer no write, and **do not
     create a regular file at the symlink's path** — that would silently
     detach the user's dotfiles setup.
2. **Line-preserving edits.** Only the lines that changed are rewritten.
   Comments, blank lines, key order, quoting, `export ` prefixes, CRLF
   endings and BOM survive — the same house value the compose writer already
   honors ("your comments, quoting and key order are kept").
   A small quoting/escaping util round-trips values per the godotenv rules
   compose-go parses (quote when the value contains `#`, a space, or is empty;
   escape `\`, `"` inside quotes) — with tests that a value survives
   parse → write → parse unchanged.
3. **File mode — and the mode must be applied *before* the bytes are
   written.** New `.env` created `0600`; existing mode preserved; when the
   file is world-readable the status line warns and `m` offers the chmod (Unix
   only, D4).

   Read `src/utils/AtomicWrite.go:39-49` before touching this. `os.CreateTemp`
   opens at 0600, and `ReplaceFileAtomically` then **chmods the temp file up to
   0644 when the target does not exist, before `temp.Write`**. A mode variant
   that chmods *after* the rename therefore leaves the secret world-readable
   for the entire duration of the write, and on a busy box that window is real.

   Add a variant rather than changing the existing function, so no current
   caller changes behaviour:

   ```go
   // ReplaceFileAtomicallyWithMode is ReplaceFileAtomically with an explicit
   // mode for a file that does not exist yet. The mode is applied at the same
   // point the 0644 default is applied today - before the write, not after
   // the rename - so a 0600 file is never briefly world-readable.
   func ReplaceFileAtomicallyWithMode(fileName string, contents []byte, mode os.FileMode) error
   ```

   `ReplaceFileAtomically` becomes a one-line call into it with its current
   0644/preserve behaviour. Tests: a `.env` the app creates is `0600`; a
   pre-existing `0644` `.env` is still `0644` after a write (D5.3 preserves —
   `m` is how the user tightens it).
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
   OSC 52 (D2) adds two of its own, and both belong in the README: the secret
   travels in the terminal's **output stream**, so a logged or recorded session
   captures it; and the sequence is **unacknowledged and sometimes disabled**
   (tmux needs `set -g set-clipboard on`), so the app can only report that the
   copy was sent.
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
7. **`github.com/atotto/clipboard` for `c`.** Rejected in favour of
   `tea.SetClipboard` (D2): atotto needs `pbcopy`/`xclip`/`xsel` on the
   machine running the app, which a headless server does not have.
8. **Warning that `.env` is tracked by git.** Tempting and cheap-looking
   (`git ls-files --error-unmatch .env`), but it makes the app shell out to a
   tool it otherwise never needs, and it is wrong in a worktree without git.
   One sentence in the README (".env holds secrets; add it to .gitignore")
   does the same job for free.
9. **Reading `.env.example` to prefill keys on the add form.** dotenv-tui's
   feature, genuinely nice, and a follow-up — already listed as out of scope,
   repeated here because "we already parse dotenv files" makes it look like a
   two-line addition at exactly the wrong moment.

## Decisions taken — nothing here is open

This section used to list five choices for the owner. They are all decided, in
each case the way the plan already recommended. Recorded here so an
implementer never has to ask, and so the reasoning is not re-litigated:

| Question | Decision | Where |
|---|---|---|
| New "Env" tab, or a toggle on the Files page? | **New tab.** The Files page is a raw-bytes viewport; this needs a value table with per-row operations | D1 (and update the ROADMAP's three-tab line in the same phase) |
| Mask everything, or classify by heuristic? | **Mask everything, fixed width.** Zero false negatives in the one place that exists to show secrets | D2 |
| `m` to chmod, or warn only? | **`m` chmods,** Unix only | D4, D5.3 |
| Reload the project after a write, or leave it to the user? | **Reload.** Manual would be a stale-view bug the moment someone edits `.env` and wonders why nothing changed | D6 |
| Which letters? | **`v` `c` `n` `e` `d` `o` `E` `m`** — final, and `a`/`r` are forbidden | D4 |

**No external sign-off.** Everything is repo-local: no money, no accounts, and
no new direct dependencies (D2's clipboard decision removes one).

## Blast radius per step

| Step | Files touched | Other effects |
|---|---|---|
| 1 — `SecretHint` helper + tests | `src/utils/SecretHint.go` (+test) | None (nothing calls it yet; grep after adding) |
| 2 — load-path truth | `src/utils/ReadConfigFile.go`, `src/cmds/GetConfig.go`, reload messages | The compose-load message carries `.env` path + loaded flag; `src/model` message structs grow fields — all existing consumers compile-checked by the compiler |
| 3 — Env page, masked table, reveal/copy | `src/apptypes/Pages.go`, `src/keys/Keys.go`, new `src/components/EnvPanel.go`, `src/cmds/GetEnvFileContents.go`(+`SaveEnvFile.go`), `src/model/Update.go`, `src/model/View.go` | Fourth tab: digits `1`–`4`, `[`/`]`, new `alt+e` chord; footer, help overlay, and keybinding bar gain the page's keys; `?` overlay lists them (existing mechanism) |
| 4 — add/edit/delete + masked input | modal components (pattern: `GroupNameModal`/`ConfirmModal`), edit-command plumbing | Reuses `OpenConfirmModal`; new value-input component |
| 5 — write path: symlink, mode, line-preserving writes, dupes | `src/utils/ApplyEnvEdit.go` (+tests), `src/utils/AtomicWrite.go` gains `ReplaceFileAtomicallyWithMode` (D5.3) | **No existing caller changes.** Adding the variant and leaving `ReplaceFileAtomically` as a one-line call into it is what keeps this step off every current writer (`ApplyServiceFragment`, the group writes). Changing the existing signature instead would touch all of them |
| 6 — reload-on-write + status line | `src/model/Update.go` | Background reload already exists; re-interpolation failures go down the existing error path |
| 7 — docs + demo | `README.md`, `docs/DESIGN.md`, `TODO.md`, demo tape | README keybindings table, DESIGN §The Env page, fixture `.env` in `demo/fixtures` |

## Implementation order — two phases, two branches

This is **two** feature branches, each merged `--no-ff` with its own ROADMAP
row, not one long one. The split is load-bearing: a masked, revealable,
copyable view of `.env` is useful on its own, it is where all the visual risk
lives, and it lets the write path — symlinks, modes, round-tripping — land
against a UI that already exists instead of at the same time as one.

**Phase A — the read-only page (steps 1–3).** Shippable alone.

1. **Steps 1 + 2** — the `secretHint` helper and the load-path truth. Small,
   isolated, testable, and everything else depends on them.
2. **Step 3** — the page: masked table, reveal, copy. Demo it before building
   any write path.

**Phase B — the write path (steps 4–7).**

3. **Steps 4 + 5 together** — the operations and the writer are one
   deliverable (add/edit/delete is meaningless without the safe writer), and
   the writer's edge cases (symlink, mode, dupes, round-trip) are a single
   test suite.
4. **Step 6** — reload-on-write, then the status-line truth (D7) once the
   reload message carries the fields.
5. **Step 7** — docs + demo tape, last.

`go build ./... && go vet ./... && go test ./... && gofmt -l .` green at every
commit in both phases, per `docs/ROADMAP.md` §Conventions.

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

## Review pass — 2026-07-31

A second reader went through this plan against the code it names and found
nine things that were wrong, unstated, or cheaper done another way. **All nine
are now folded into the sections above**, so this plan says one thing once and
can be read straight through:

| Was | Now lives in |
|---|---|
| `a` for add collides with `Global.About` | D4 — the key is `n` |
| `r` for reveal collides with `Details.Restart` | D2, D4 — the key is `v` |
| atotto/clipboard fails over SSH | D2 — `tea.SetClipboard` (OSC 52), costs in §Security drawbacks |
| 0600 must be applied before the write, not after the rename | D5.3, with the `ReplaceFileAtomicallyWithMode` signature |
| Symlink resolution moves the failure surface, carries the target's mode, and breaks on a dangling link | D5.1 |
| A fourth tab contradicts a recorded ROADMAP decision | D1 — re-take it and edit the line in the same phase |
| `m` (chmod) is meaningless on Windows | D4 — gated on `runtime.GOOS` |
| The read-only page is a shippable unit | §Implementation order — two phases, two branches |
| Two more things to reject explicitly | §Alternatives 8 and 9 |

Checked against the same code and unchanged: mask-all at fixed width (D2), the
raw-lines-not-parsed-map table (D2), the `secretHint` boundary and its
deliberately-unmasked list (D3), reload-on-write (D6), the status-line truth
including `COMPOSE_DISABLE_ENV_FILE` (D7), and every entry in §Edge cases.

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
- Do not bind `a` or `r` on the Env page (D4 — they are `Global.About` and
  `Details.Restart`), and do not add `github.com/atotto/clipboard` as a direct
  dependency (D2).
- Do not chmod after the rename (D5.3), and do not create a regular file where
  a dangling symlink was (D5.1).
- Do not show the world-readable warning or the `m` key on Windows (D4).
- Do not add git-awareness or `.env.example` prefilling (Alternatives 8, 9).
