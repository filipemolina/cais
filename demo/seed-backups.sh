#!/usr/bin/env bash
# Seed the demo backup store so the Backups page (tab 4) has real history to
# show when the README demo or screenshots are recorded.
#
# The app's backup store lives next to the compose file, at
# demo/fixtures/.cais/backups/<slug>/<utc-ts>.<sha8>.bak, where slug is the
# source basename with dots turned to underscores (compose.yaml -> compose_yaml,
# .env -> _env) and sha8 is the first 8 hex chars of the content's SHA-256.
# ListBackups parses the timestamp and sha8 straight from the filename, so the
# names here must match that exact shape or the rows will not parse.
#
# The seeded copies are not real edit history: they are hand-written variants
# of the fixture compose file and a sample .env, so the preview pane has
# something legible to show. They live under demo/fixtures/.cais/, which the
# repo .gitignore excludes, so they are never committed.
#
# Usage: ./demo/seed-backups.sh   (run from the repo root)
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FIXTURES="$REPO_ROOT/demo/fixtures"
STORE="$FIXTURES/.cais/backups"

# Wipe any prior seed so re-runs are deterministic.
rm -rf "$STORE"
mkdir -p "$STORE/compose_yaml" "$STORE/_env"

sha8_of() {
  # $1 = file path; prints the first 8 hex chars of its SHA-256.
  sha256sum "$1" | cut -c1-8
}

# --- Compose variants -------------------------------------------------------
# A small earlier draft (fewer services), an intermediate one, and the live
# file as the most recent copy. The live file is copied verbatim so its sha8
# matches whatever the app would compute, but it is still seeded here for a
# deterministic newest-first order with the two older drafts.

EARLY="$STORE/compose_yaml/early.compose.yaml"
INTERMEDIATE="$STORE/compose_yaml/intermediate.compose.yaml"

# Early draft: the media + home groups, no infra yet.
cat > "$EARLY" <<'YAML'
name: homelab

services:
  navidrome:
    image: deluan/navidrome:latest
    profiles: ["media"]
    ports:
      - "14533:4533"
    volumes:
      - navidrome-data:/data
    restart: unless-stopped

  homeassistant:
    image: homeassistant/home-assistant:stable
    profiles: ["home"]
    ports:
      - "14123:8123"
    volumes:
      - ha-config:/config
    restart: unless-stopped

  calibre:
    image: lscr.io/linuxserver/calibre:latest
    profiles: ["home"]
    ports:
      - "14809:8083"
    volumes:
      - calibre-config:/config
      - calibre-books:/books
    restart: unless-stopped

volumes:
  navidrome-data:
  ha-config:
  calibre-config:
  calibre-books:
YAML

# Intermediate draft: adds audiobookshelf + kavita, still no infra.
cat > "$INTERMEDIATE" <<'YAML'
name: homelab

services:
  navidrome:
    image: deluan/navidrome:latest
    profiles: ["media"]
    ports:
      - "14533:4533"
    volumes:
      - navidrome-data:/data
    restart: unless-stopped

  audiobookshelf:
    image: ghcr.io/advplyr/audiobookshelf:latest
    profiles: ["media"]
    ports:
      - "14378:80"
    volumes:
      - abs-config:/config
      - abs-metadata:/metadata
    restart: unless-stopped

  kavita:
    image: lscr.io/linuxserver/kavita:latest
    profiles: ["media"]
    ports:
      - "14230:5000"
    volumes:
      - kavita-config:/config
    restart: unless-stopped

  homeassistant:
    image: homeassistant/home-assistant:stable
    profiles: ["home"]
    ports:
      - "14123:8123"
    volumes:
      - ha-config:/config
    restart: unless-stopped

volumes:
  navidrome-data:
  abs-config:
  abs-metadata:
  kavita-config:
  ha-config:
YAML

# The live file, copied verbatim, as the newest copy.
cp "$FIXTURES/compose.yaml" "$STORE/compose_yaml/live.compose.yaml"

# --- .env variant -----------------------------------------------------------
# A sample earlier .env with fewer variables than the live file would hold.
EARLY_ENV="$STORE/_env/early.env"
cat > "$EARLY_ENV" <<'ENV'
TZ=Europe/Lisbon
PUID=1000
PGID=1000
ENV

# Copy the live .env if it exists (so the page shows both sources), else make
# a representative newest copy. The demo fixtures have no .env by default, so
# fabricate one with the secrets the README mentions.
LIVE_ENV="$STORE/_env/live.env"
cat > "$LIVE_ENV" <<'ENV'
TZ=Europe/Lisbon
PUID=1000
PGID=1000
PAPERLESS_SECRET_KEY=cais-demo-secret
PAPERLESS_DBPASS=demo
ENV

# --- Stamp the names with deterministic timestamps (oldest -> newest) -------
# Rename the staged copies to the app's <utc-ts>.<sha8>.bak name. Fixed
# timestamps keep the list order stable across re-runs and look plausible in
# the recording (a few days of history).

rename_to_bak() {
  # $1 = staged file, $2 = utc timestamp
  local staged="$1" ts="$2" dir
  dir="$(dirname "$staged")"
  local sha8
  sha8="$(sha8_of "$staged")"
  mv "$staged" "$dir/${ts}.${sha8}.bak"
}

rename_to_bak "$EARLY"            "20260808T091500"
rename_to_bak "$INTERMEDIATE"     "20260809T174500"
# One minute after the live .env copy, so the newest compose copy (which
# renders as highlighted YAML in the preview) sorts to the top of the list
# and is the one selected by default in the screenshot.
rename_to_bak "$STORE/compose_yaml/live.compose.yaml" "20260811T014501"
rename_to_bak "$EARLY_ENV"        "20260808T091500"
rename_to_bak "$LIVE_ENV"         "20260811T014500"

# The store sits next to a compose file, so give it the same .gitignore the
# app would, even though the repo already ignores the whole directory.
printf 'backups/*\n' > "$FIXTURES/.cais/.gitignore"

echo "Seeded demo backup store at $STORE"
ls -R "$STORE"
