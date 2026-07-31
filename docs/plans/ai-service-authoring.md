# Plan: Free AI Help When Adding a Service

Feature request: *"adding free AI features, namely, AI help when adding a new
service, so the AI would fill the service configuration options with sensible
defaults."*

Researched 2026-07-31; every provider fact below carries its source and date,
and the two environment claims (what is installed on the maintainer's machine,
what Docker ships) were measured rather than assumed.

## Status of the feature — an honest reframing

The ask contains a hidden dependency and a hidden question.

**The dependency:** the app cannot add a service at all today.
`ApplyServiceFragment` refuses a name that is not already in the file
(`src/utils/ServiceFragment.go:91`), `WriteNewComposeFile` only seeds a
brand-new file (`src/utils/GroupTags.go:249`), and `docs/DESIGN.md` §Editing
services says `E` in `$EDITOR` is *"the only way to add a service"*. So "AI
help when adding a new service" needs *adding a new service* to exist first.
That is Phase 1 of `docs/plans/image-search.md`, and this plan **starts from
it**: `n` on the Services page, `utils.AddServiceFragment`, review in the
inline editor. Do not build a second insert path.

**The question:** *what does "free" have to mean?* There are three different
answers and they have different consequences, so this plan takes all three and
lets the user pick, because the honest truth is that no single one is free for
everybody:

| Kind | Free how | Costs the user |
|---|---|---|
| **Local** (Ollama, llama.cpp, LM Studio) | genuinely, forever | disk (2–5 GB), RAM, and a slower, weaker model |
| **Hosted free tier** (Gemini, Groq, OpenRouter, Cerebras) | free of money | an account, an API key, rate limits, and — on Gemini's unpaid tier — their prompts train the model |
| **Nothing configured** | — | the app must still work, which is what makes the built-in template catalog (Phase 1) non-optional |

**Verdict up front: worth doing, as an opt-in that is off by default and
useless-but-harmless when unconfigured.** The feature is a *drafting aid* that
writes YAML into an editor the user then reads. It must never write to the
compose file directly, never run a docker command of its own, and never ship a
key. Under those three constraints it is a genuinely good fit for this app —
"what does a sensible Jellyfin service look like" is exactly the question a
self-hoster asks, and the answer is 15 lines of YAML the model has seen ten
thousand times.

## Research

### One protocol covers every provider worth supporting

Every option below speaks **OpenAI-compatible `POST {base}/chat/completions`**.
That means one HTTP client, `net/http` + `encoding/json`, **no new
dependency**, and provider support becomes a base URL plus a model name in
config:

| Provider | Base URL | Free-tier limits (retrieved 2026-07-31) |
|---|---|---|
| **Ollama** (local) | `http://localhost:11434/v1` | none — it is your machine |
| **Gemini** | `https://generativelanguage.googleapis.com/v1beta/openai/` | published per-project in AI Studio, no longer in the docs (see below) |
| **Groq** | `https://api.groq.com/openai/v1` | `llama-3.1-8b-instant`: 30 RPM / 14,400 RPD / 6K TPM; `llama-3.3-70b-versatile` and `gpt-oss-120b`: 30 RPM / 1,000 RPD (console.groq.com/docs/rate-limits) |
| **OpenRouter** | `https://openrouter.ai/api/v1` | `:free` models: 20 RPM, **50 requests/day**, raised to 1,000/day only after $10 of credits has *ever* been purchased (openrouter.ai/docs/api-reference/limits) |
| anything else | user-supplied | user's problem, and that is the point |

Sources: Gemini's OpenAI-compatibility layer is documented at
`ai.google.dev/gemini-api/docs/openai` (base URL above, supports chat
completions, streaming, function calling and structured outputs; described as
beta). Ollama's OpenAI compatibility and its `format` / `response_format`
structured outputs are documented at `docs.ollama.com/capabilities/structured-outputs`,
which also notes **Ollama Cloud does not support structured outputs** — only
local models do.

**Gemini's free tier has a privacy cost that must be in the UI, not the
footnotes.** Google's Gemini API terms (`ai.google.dev/gemini-api/terms`,
retrieved 2026-07-31) say of the unpaid tier: *"Google uses the content you
submit to the Services and any generated responses to provide, improve, and
develop Google products and services"*, that *"human reviewers may read,
annotate, and process your API input and output"*, and *"Do not submit
sensitive, confidential, or personal information to the Unpaid Services."*
This app's prompt would contain the user's compose file. Compose files contain
host paths, internal hostnames, port maps and sometimes `${VAR}` names that
describe the shape of a homelab. That is exactly what "do not submit" means,
and it is why **the prompt sends a summary, not the file** (D4).

Google also stopped publishing free-tier RPM/RPD in the docs; the rate-limits
page now says limits *"can be viewed in Google AI Studio"* and links to a
per-project page. So the plan must not hardcode or document specific Gemini
numbers — it shows the provider's own 429 message when one arrives.

### What is actually on this machine (measured 2026-07-31)

- **Ollama 0.24.0 is installed and serving** — `/api/version` and the
  OpenAI-compatible `/v1/models` both answer on `localhost:11434`. Good: the
  recommended default is testable on the maintainer's own box.
- **No local model is pulled.** `/v1/models` lists exactly one entry,
  `qwen3-coder-next:cloud`, which is an Ollama *Cloud* alias — so it is neither
  local nor structured-output-capable. Before Phase 3 can be tested honestly,
  pull a real local model (`ollama pull qwen3:4b` or `gemma3:4b`, ~2–3 GB).
- **`docker ai` (Gordon) is not available.** Docker 29.6.0 here has exactly two
  CLI plugins, `buildx v0.34.1` and `compose v5.1.4`; `docker ai` is
  unrecognised. Gordon is a Docker Desktop feature, and the self-hosting
  audience this app targets runs Docker Engine from a package manager on a
  headless box. See Alternatives — it is a good idea that does not reach the
  users.

### Prior art

Docker ships its own agent, **Gordon** (`docker ai`), GA in Docker Desktop 4.74
(May 2026), free with a Docker account, and it explicitly generates compose
configurations from a project directory (docker.com/products/gordon and the
Docker blog, retrieved 2026-07-31). It is the strongest argument *against*
building this — and the measurement above is the argument against that
argument: on a Debian server with `docker-ce` from apt, it is not there.

## Scope

**In:** an opt-in AI draft step inside the existing add-a-service flow. The
user types a name and a description of what they want ("jellyfin with hardware
transcoding and my media on /srv/media"); the app asks a model for a compose
service fragment; the fragment lands **in the inline editor** for review; the
user saves it through the existing validated write path or throws it away.

**Out, stated:**

- **A chat panel.** Different product. This is one request, one answer, in a
  flow that already exists.
- **AI that acts.** No generated shell command is ever run, no docker action is
  ever triggered by a model, nothing is written to disk without the user
  reading it in the editor and pressing `ctrl+s`.
- **Anything bundled.** No API key ships in the binary, no default hosted
  endpoint, no telemetry, no "try it free" account.
- **Explaining logs / diagnosing failures / suggesting fixes.** Each is a
  plausible follow-up and each doubles the prompt surface. Not now.
- **Editing an existing service by prompt.** `e` edits. Revisit after this
  ships and gets used.
- **Streaming tokens into the editor.** Nice demo, real complexity (partial
  YAML is invalid YAML). One request, one spinner, one result.

## Design decisions

### D1. Off by default, and the app never mentions it until it is configured

No provider configured → the AI step does not appear in the modal, no keys are
advertised, no network is touched. The feature announces itself in the README
and nowhere else. A TUI for managing containers that nags about AI is a TUI
people uninstall.

### D2. Configuration: endpoint in the config file, **key in the environment**

`~/.config/stack-stitcher/config.yaml` gains:

```yaml
ai:
  base_url: http://localhost:11434/v1   # OpenAI-compatible endpoint
  model: qwen3:4b
  api_key_env: OLLAMA_API_KEY           # name of an env var, never the key
  timeout_seconds: 60
```

**The API key is read from the environment, never stored by the app.** The
reason is in the code: `config.SaveConfig` writes the file with
`os.WriteFile(..., 0o644)` (`src/config/config.go`, last line) — world-readable.
Storing a hosted-provider key there would be a credential leak the app created
itself, and it would contradict `docs/plans/env-secrets.md`, which is a whole
plan about not doing that. `api_key_env` names the variable; the app reads it
with `os.Getenv` at request time and holds it no longer than the request.

`config.Config` absorbs this by design — its doc comment says *"Add a field,
tag it, and LoadConfig/SaveConfig round-trip it automatically."* Add
`AI *AIConfig \`yaml:"ai,omitempty"\``; a nil pointer is "not configured",
which is the check D1 needs.

### D3. One client, one interface, no dependency

```go
// src/ai/Client.go
package ai

// Draft is what the app asks for and what a provider must return: the YAML
// body of exactly one compose service, without the service-name key.
type Draft struct {
    YAML  string
    Notes string // one line the model may use to flag an assumption
}

type Client interface {
    DraftService(ctx context.Context, req Request) (Draft, error)
}

type Request struct {
    ServiceName string
    Prompt      string   // what the user typed
    Image       string   // optional, if they already picked one
    Existing    []string // names of services already in the file (for depends_on)
}
```

One implementation, `openaiCompatible`, posting:

```json
{
  "model": "<config.model>",
  "temperature": 0,
  "messages": [
    {"role": "system", "content": "<the system prompt, D4>"},
    {"role": "user",   "content": "<the rendered request>"}
  ],
  "response_format": {"type": "json_schema", "json_schema": {...}}
}
```

and reading `choices[0].message.content`. `temperature: 0` because this is a
lookup dressed as generation, and because a deterministic answer is a testable
one.

**Structured output, with a fallback that is not optional.** Ollama (local),
Gemini and OpenAI-compatible hosted providers accept `response_format`, but
coverage is uneven and Ollama *Cloud* does not support it at all
(docs.ollama.com, above). So: send `response_format`; if the response is a
400 mentioning the field, retry once without it; either way parse the content
defensively — accept a bare JSON object, a ```json fence, or a ```yaml fence,
in that order. Every model will eventually return one of the three, and a
feature that breaks on a code fence will break for half the free models.

**Testing.** `Client` is an interface, so `src/model` and the components take a
fake. The real client is tested against `httptest.Server` with recorded
payloads: a good response, a fenced response, a `response_format`-rejecting
400, a 429, a 500 with an HTML body (a proxy error page — very common), and a
context timeout. **No test may reach the network**, and CI must stay green on a
machine with no model and no key.

### D4. The prompt sends a summary of the project, never the file

The model needs enough context to be useful and no more. Send:

- the service name the user chose;
- what the user typed;
- the names of existing services (so `depends_on:` can be right) — **names
  only**;
- whether a `.env` exists (so it can reference `${VAR}` instead of inventing
  literals) — **not its contents, not its keys**;
- the host OS/arch, so volume-path advice is not absurd.

Do **not** send: the compose file, host paths, port numbers, image tags in use,
`.env` values or key names, container names, or logs. This is what makes the
Gemini unpaid-tier terms survivable and what makes the feature safe to use on a
work machine. It is also a testable rule: assert the request body contains no
substring from a fixture compose file's paths or environment values.

The system prompt states the contract: output one compose service body only, no
service-name key, no `version:`, no markdown, prefer a pinned image tag over
`latest`, prefer named volumes over host paths unless the user gave a path,
use `${VAR}` for anything secret-shaped, and say so in `Notes` when guessing.

### D5. The output lands in the editor. Always.

The returned YAML is written into the *inline editor buffer*, not the file. The
user sees it in the same `textarea` they would have typed it into, and
`ctrl+s` runs the same `ValidateComposeCandidate` → `ReplaceFileAtomically`
path as every other edit. If the model returns something that does not parse,
the editor's existing live YAML validation shows it and the user fixes or
discards it — no special error path, because the app already has one.

This is the whole safety story and it is not negotiable:

- **Model output is untrusted input.** It is influenced by whatever the model
  absorbed in training and by whatever the user typed, and it becomes
  configuration that runs containers on the user's host. A generated
  `volumes: - /:/host` or a `privileged: true` must be something a human
  looked at.
- **Typosquatted or hallucinated images.** Models invent plausible image names.
  Phase 4 adds a cheap check — `docker search` for the exact repository (the
  same `utils.SearchImages` from the image-search plan) and a status-line
  warning when the image does not exist on Hub — but the editor review is the
  real defence.
- **No generated string is ever executed.** Not as a shell command, not as a
  docker argument. The only sink for a model's output is the editor buffer.

### D6. Where it sits in the flow, and what it costs when it fails

```
n ──> AddServiceModal
        name  ──> image (or "/" to search, or blank)
                    └─ ctrl+a ──> "describe what you want" field
                                    └─ Enter ──> spinner ──> editor pre-filled
```

`ctrl+a` (unbound; `a` alone is `Global.About` — see `src/keys/Keys.go:161`)
opens the description field, and only when D1's config exists. Every failure —
no key, timeout, 429, garbage response — closes back to the plain flow with the
message on the status line and the typed name and image intact. The user
finishes by hand. The AI step is a shortcut, never a gate.

Timeout: the config's `timeout_seconds`, default 60. A local 4B model on a
laptop CPU can take 30 s for 200 tokens; a hosted one takes 2. The spinner is
`components/spinner.go`, already built for exactly this.

## Phases

Feature branch per phase, small commits, `--no-ff` merge, ROADMAP row, green
`go build ./... && go vet ./... && go test ./...` + `gofmt -l .` at every
commit.

**Phase 0 (prerequisite, not part of this plan):** Phase 1 of
`docs/plans/image-search.md` — `n` on the Services page,
`utils.AddServiceFragment`, the add-service modal, the editor opening on the
new service. Nothing here works without it.

### Phase 1 — The offline catalog (no AI, and it ships first)

A table of ~20 known self-hosted services (jellyfin, plex, nextcloud, sonarr,
radarr, prowlarr, qbittorrent, gitea, forgejo, immich, paperless-ngx, vaultwarden,
home-assistant, uptime-kuma, adguardhome, pihole, grafana, prometheus, postgres,
redis, traefik, caddy), each with a reviewed compose fragment: pinned image,
`restart: unless-stopped`, named volumes, the ports it actually needs, `PUID`/
`PGID`/`TZ` for LinuxServer images, and `${VAR}` placeholders for anything
secret-shaped.

In the modal, typing a name that matches an entry offers "use the template".
Same as the healthcheck catalog in `docs/plans/healthcheck-insertion.md`: a data
table, pure, no network, testable without Docker.

Why first: it is the answer for the unconfigured user (which is every user on
day one), it is deterministic, it is the *quality bar* the AI output is judged
against, and it doubles as the fixture set for Phase 3's tests. Each row is a
correctness claim about an image — keep the list short and verify each fragment
with `docker compose config` before committing it.

| File | Change |
|---|---|
| `src/utils/ServiceTemplate.go` (+test) | the catalog, `Lookup(name string) (ServiceTemplate, bool)`, fragment rendering |
| `src/components/AddServiceModal.go` | offer the template when the typed name matches |
| `README.md`, `docs/DESIGN.md` | the catalog and its limits |

Acceptance: typing `jellyfin` offers a template; accepting it opens the editor
with a fragment that `ValidateComposeCandidate` accepts; every catalog entry has
a test asserting it parses as compose.

### Phase 2 — Config plumbing, no network

`src/config`: the `AI` struct, `api_key_env` (D2), round-trip test, and a test
asserting the API key itself is never written to the file. `src/ai`: the
`Client` interface, `Request`, `Draft`, and a `NoopClient` used when
unconfigured. Nothing in the UI changes.

Acceptance: a config with an `ai:` block round-trips; a config without one
yields "not configured"; `gofmt`/`vet`/tests green with no network and no key.

### Phase 3 — The OpenAI-compatible client

`src/ai/OpenAICompatible.go` implementing D3, with the parsing fallbacks and
the full `httptest` matrix. Still no UI.

Acceptance: the six recorded scenarios in D3 all produce either a `Draft` or a
legible error; the request body assertion of D4 (no compose-file content) is a
test, not a comment; `go test -race ./...` green offline.

### Phase 4 — The UI step

`ctrl+a` in the modal (D6), the description field, the spinner, the editor
pre-fill, and the "image not found on Docker Hub" warning using
`utils.SearchImages` from the image-search plan (skipped silently if that phase
has not landed). Foreground errors go down the existing
`reportForegroundError` path.

Acceptance: with a fake client injected, `ctrl+a` → text → Enter opens the
editor containing the fake's YAML; with the fake returning an error the flow
falls back to manual entry with the typed values intact; with no `ai:` config
`ctrl+a` does nothing and is not advertised in the footer or `?`.

### Phase 5 — Docs, and one honest paragraph

README: how to point it at Ollama (with the `ollama pull` line, since the
maintainer's own box has no local model), how to point it at a hosted free
tier, that the key comes from an env var and why, what is sent (D4) and what is
not, and that **Gemini's unpaid tier trains on what you send** with the link.
`docs/DESIGN.md`: why the output goes to the editor and never to the file.
A VHS tape only if a local model is pulled — recording a spinner is not worth a
network dependency in the demo.

## Edge cases and unknowns

1. **Unconfigured** — the whole feature is invisible (D1).
2. **Key env var set but empty** — treated as unconfigured, with a status-line
   note naming the variable. Silent failure here wastes an afternoon.
3. **Local Ollama not running** — connection refused in under a second; message
   says "no answer from <base_url>", not a Go error string.
4. **Model not pulled** — Ollama answers 404 with a "model not found" body.
   Surface the body; it already tells the user to run `ollama pull`.
5. **429** — show the provider's message verbatim. Do not retry; do not
   implement backoff for a human-triggered single request.
6. **Response is prose, not YAML** — the fence/JSON parser fails, the error says
   "the model did not return YAML", the flow falls back to manual.
7. **Response is valid YAML but not a service** (a whole compose file with
   `services:`, a common failure) — detect the `services:` key and unwrap the
   first entry rather than rejecting; models do this constantly.
8. **Response includes a service-name key** (`jellyfin:` at the top) — unwrap it
   and *keep the user's chosen name*. The user named the service; the model did
   not.
9. **Huge response** — cap the accepted content at, say, 16 KB and refuse
   beyond it. A 2 MB answer into a `textarea` is a hang.
10. **Non-UTF-8 / control characters** in the response — strip before it
    reaches the editor buffer; a stray escape sequence in a TUI is a corrupted
    screen at best.
11. **`${VAR}` in generated YAML with no `.env`** — compose interpolation will
    warn or fail at load. This is the *correct* generated output, so the editor
    save shows the real compose error and `docs/plans/env-secrets.md` is where
    the user goes next. Cross-reference it in the docs.
12. **Proxy/corporate TLS** — `net/http` honours `HTTPS_PROXY`; nothing to do,
    worth one README line.
13. **Windows** — nothing platform-specific here; the client is stdlib HTTP.
14. **Offline entirely** — Phase 1's catalog is the answer, which is the reason
    Phase 1 exists.

**Unknowns to settle before Phase 4:** whether the inline editor can be opened
pre-filled with content that is not yet in the file (the same unknown the
image-search plan lists — it is the same mechanism, so answer it once); and
which model the maintainer will actually pull locally, because the system
prompt in D4 should be tuned against a small model, not a large one — a 4B
model needs a blunter prompt than a 70B.

## Alternatives considered

1. **Do nothing; ship the catalog only (Phase 1).** Deterministic, offline,
   free, and covers the top 20 services that are 80% of what a self-hoster
   installs. This is genuinely competitive with the AI path and it is why
   Phase 1 comes first. It fails on the long tail — the twenty-first service —
   which is exactly where a model earns its place.
2. **Shell out to `docker ai` (Gordon).** Free with a Docker account, no key
   handling, generates compose configs, and the app already shells out to
   `docker`. **Rejected as the primary path on measurement:** not present on
   Docker Engine 29.6.0 on this machine (only `buildx` and `compose` plugins),
   because it is a Docker Desktop feature — and the target audience runs
   headless Engine. Worth keeping as an *additional provider* later: if
   `docker ai` exists, offer it and skip all key handling. Cheap to add once
   `ai.Client` is an interface (D3), which is a reason to keep that interface.
3. **Bundle a small model with the binary.** Turns a 12 MB Go binary into a
   gigabyte. No.
4. **A hosted key owned by the project** ("free for our users"). Someone pays,
   the key leaks, the quota is drained in a week, and the project inherits an
   abuse problem. Never.
5. **Let the AI write to the compose file directly, with a confirm dialog.**
   Faster by one keypress and strictly worse: a diff in a modal is a worse
   review surface than the editor the user already knows, and the editor gives
   undo, `ctrl+o` to `$EDITOR`, and save-time validation for free.
6. **MCP / tool-calling so the model can inspect the project.** A much larger
   surface (the model gets to *read* things), a much larger prompt-injection
   question, and no added value for "write me a jellyfin service". Revisit
   never, or at least not for this feature.

## Blast radius

| Area | Effect |
|---|---|
| Dependencies | **none** — `net/http`, `encoding/json`, `context` are stdlib |
| `src/config` | one optional struct field; nil = off; existing configs unaffected |
| `src/keys` | `ctrl+a`, advertised only inside the add-service modal when configured |
| Network | a second network destination after the image-search plan's Hub calls, and this one is **user-specified** — the README must be explicit that the app talks to whatever endpoint the config names |
| Privacy | the first feature that can send anything off the machine. D4's summary-not-file rule is the mitigation, and it needs a test, not a promise |
| Binary size | negligible |
| CI | unchanged: everything is faked or `httptest` |

## Do not

- Do not ship an API key, a default hosted endpoint, or a project-owned quota.
- Do not store the API key in `config.yaml` — it is written `0644` (D2).
- Do not send the compose file, `.env` contents, `.env` key names, host paths
  or logs to any model (D4). Test it.
- Do not write model output to the compose file. It goes to the editor (D5).
- Do not execute anything a model produced — no shell, no docker argv.
- Do not build a chat panel, streaming, or log explanation in this feature.
- Do not make the feature visible when it is not configured (D1).
- Do not skip Phase 1. The catalog is the fallback for every failure mode in
  this document, and there are fourteen of them.
