# Interactive TUI Features for Docker Compose Profiles

Building a bespoke Terminal User Interface (TUI) around a single, unified Docker Compose file is a highly pragmatic pattern for managing self-hosted infrastructure. Native Docker Compose profiles provide a strong foundation for service segregation, but extending them with custom TUI logic unlocks capabilities tailored to self-hosting enthusiasts.

---

## 1. Dynamic "Profile Modes" (Smart Presets)

Rather than managing individual profiles manually, group profile combinations into context-aware operational states:

* **Maintenance Mode:** Temporarily downs public-facing services (e.g., `ingress`, `arr-stack`) while spinning up operational tools (`backups`, `db-tools`).
* **Low Power / Eco Mode:** Drops resource-intensive background workers (AI model inference, transcoding, local search indexing) while preserving core networking and primary storage.
* **Family / Scheduled Mode:** Toggles specific content or routing profiles on demand (e.g., swapping DNS rules, adjusting media library availability during specific time windows).

---

## 2. Resource-Aware Profile Toggling

Standard Compose does not inspect system limits before starting containers. Leveraging host runtime metrics (`/proc`, `docker stats`) inside your TUI allows for proactive resource budgeting:

* **Pre-flight RAM & CPU Checks:** Calculate estimated resource requirements before bringing a profile online. Warn or require confirmation if system memory utilization exceeds standard thresholds (e.g., >80%).
* **Aggregated Profile Metrics:** Display real-time CPU, RAM, and storage utilization rolled up by **Profile**, rather than solely at the individual container level.

---

## 3. Advanced Dependency Mapping

While standard Compose manages service-level `depends_on` directives, cross-profile relationships are traditionally manual. The TUI layer can handle these implicit links:

* **Profile Chaining:** Automatically prompt or bring up base profiles when a dependent profile is selected (e.g., enabling `monitoring` auto-detects that `core-infra` must be active).
* **Cross-Profile Conflict Prevention:** Warn the user if stopping a profile will break running services in an active profile (e.g., stopping a shared database profile while application frontends remain up).

---

## 4. One-Shot "Task Profiles"

Extend profiles to handle ephemeral administration tasks rather than persistent background daemons:

* **On-Demand Utility Execution:** Map ephemeral profiles (`profile: ["task-backup"]`, `profile: ["task-cleanup"]`) to interactive action triggers within the UI.
* **Integrated Log Streaming:** Execute `docker compose run --rm <service>` directly, piping standard output to an embedded log viewer window before auto-cleaning container artifacts upon exit.

---

## 5. Dynamic Overlay & Environment Management

Pair Compose profiles with runtime configuration overlays directly within the interface:

* **Per-Profile `.env` Presets:** Toggle specific variable overlays on the fly (e.g., launching a `media` profile with `VPN_ENABLED=true` versus local debugging routes).
* **Pre-flight Secret Validation:** Verify that required environment variables and path mounts exist before permitting a profile to transition to the `UP` state.

---

## Interface Layout Concept

```text
┌────────────────────────────────────────────────────────────────────────┐
│ Profile: [x] Core  [x] Media  [ ] Monitoring  [ ] Backup (Task)       │
├────────────────────────────────────────────────────────────────────────┤
│ Active RAM: 4.2 / 16 GB  │ CPU: 12%  │ Core Temp: 42°C                 │
├──────────────────────────┬─────────────────────────────────────────────┤
│ Services (Media Profile) │ Logs (Selected: Radarr)                     │
│  ● radarr    (Running)   │ [INFO] Searching for updates...             │
│  ● sonarr    (Running)   │ [INFO] Database vacuum complete.            │
│  ○ plex      (Stopped)   │                                             │
└──────────────────────────┴─────────────────────────────────────────────┘
```
