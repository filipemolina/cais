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
# of the fixture compose file and the fixture .env, so the preview pane has
# something legible to show. They live under demo/fixtures/.cais/, which the
# repo .gitignore excludes, so they are never committed.
#
# The variants are written for the diff preview: the newest copy of each
# source is the live file verbatim, so it carries the (current) marker and
# answers with the "Identical to the live file" card, and the drafts beneath
# it each disagree with the live file in ways that paint both washes - a
# changed line (red removed, green added) and a service the live file no
# longer has. An all-red or all-green draft would not show what the feature
# actually looks like.
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

# Early draft: media + home only, and the state the stack had outgrown —
# navidrome still on its old pinned image and port, homeassistant on its
# 2025.7 pin, and jellyfin, which the stack later dropped for navidrome +
# audiobookshelf. Diffing this against the live file paints both washes: the
# pins and the dropped services as red (restoring would remove what the live
# file has since gained), jellyfin as green (restoring would bring it back).
cat > "$EARLY" <<'YAML'
name: homelab

services:
  navidrome:
    image: deluan/navidrome:0.53.3
    profiles: ["media"]
    ports:
      - "1533:4533"
    volumes:
      - navidrome-data:/data
    restart: unless-stopped

  jellyfin:
    image: jellyfin/jellyfin:10.9.11
    profiles: ["media"]
    ports:
      - "18096:8096"
    volumes:
      - jellyfin-config:/config
    restart: unless-stopped

  homeassistant:
    image: homeassistant/home-assistant:2025.7
    profiles: ["home"]
    ports:
      - "14123:8123"
    volumes:
      - ha-config:/config
    restart: unless-stopped

volumes:
  navidrome-data:
  jellyfin-config:
  ha-config:
YAML

# Intermediate draft: the jellyfin era is over and audiobookshelf + kavita
# have arrived, but navidrome still logs at debug and the infra profile
# (paperless's db and cache, the proxy) plus calibre do not exist yet. The
# diff against the live file opens on the ND_LOGLEVEL red/green pair and
# runs red from paperless onward.
cat > "$INTERMEDIATE" <<'YAML'
name: homelab

services:
  navidrome:
    image: deluan/navidrome:latest
    profiles: ["media"]
    ports:
      - "14533:4533"
    environment:
      ND_LOGLEVEL: debug
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
    environment:
      PUID: 1000
      PGID: 1000
      TZ: Europe/Lisbon
    volumes:
      - kavita-config:/config
    restart: unless-stopped

  homeassistant:
    image: homeassistant/home-assistant:stable
    profiles: ["home"]
    ports:
      - "14123:8123"
    environment:
      TZ: Europe/Lisbon
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
# An earlier .env from before paperless joined the stack: the timezone had
# not settled on Lisbon yet, and the two paperless secrets did not exist.
# The live file (demo/fixtures/.env) is copied verbatim as the newest copy,
# so its row carries the (current) marker the same way the compose one does.

EARLY_ENV="$STORE/_env/early.env"
cat > "$EARLY_ENV" <<'ENV'
TZ=Europe/Madrid
PUID=1000
PGID=1000
ENV

# The live .env fixture is tracked, so the demo always has a live side to
# diff the .env copies against.
LIVE_ENV="$FIXTURES/.env"
if [ ! -f "$LIVE_ENV" ]; then
  echo "demo/fixtures/.env is missing; the .env copies would have no live side" >&2
  exit 1
fi
cp "$LIVE_ENV" "$STORE/_env/live.env"
# The rename below must hit the staged copy, never the fixture itself.
STAGED_LIVE_ENV="$STORE/_env/live.env"

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

rename_to_bak "$EARLY"            "20260904T091500"
rename_to_bak "$INTERMEDIATE"     "20260906T174500"
# One minute after the live .env copy, so the newest compose copy (which
# renders as highlighted YAML in the preview) sorts to the top of the list
# and is the one selected by default in the recording.
rename_to_bak "$STORE/compose_yaml/live.compose.yaml" "20260908T014501"
rename_to_bak "$EARLY_ENV"        "20260904T091505"
rename_to_bak "$STAGED_LIVE_ENV"  "20260908T014500"

# The store sits next to a compose file, so give it the same .gitignore the
# app would, even though the repo already ignores the whole directory.
printf 'backups/*\n' > "$FIXTURES/.cais/.gitignore"

echo "Seeded demo backup store at $STORE"
ls -R "$STORE"
