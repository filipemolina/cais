---
title: Backups
description: How cais keeps your compose file and .env safe — atomic writes, snapshots, and restore.
sidebar:
  order: 8
---

# Backups

Every change cais makes reaches your compose or `.env` file through a single atomic write path. The existing file is snapshotted into `.cais/backups/` before it is replaced, so a bad edit is always reversible.

![The Backups page: every stored copy of the compose file and the .env, with a live preview](/screenshot-backups.png)

## The safety model

- **Atomic writes.** The new contents are written to a temporary file alongside the target and renamed into place, carrying the original's permissions across. The replacement lands in full or not at all — no half-written file, no lost original if the disk fills or the process is killed mid-write.
- **Snapshot before replace.** The existing file is copied into `.cais/backups/` before it is replaced, so the previous state always exists.
- **The `.env` is backed up alongside** the compose file.
- **A `.cais/.gitignore`** keeps the store out of `git status`.

## The backup store

Backups live in `.cais/backups/` beside your compose file. Each backup is named `UTC-timestamp.sha8-of-content.bak` — the timestamp sorts them, and the content hash deduplicates identical copies. The store is pruned to 500 backups per source.

## Restoring

Tab `4` (Backups) lists every stored copy of the compose file and the `.env`, newest first, with a live preview beside the list — the exact bytes a restore would put back. Compose copies are shown syntax-highlighted; `.env` copies are shown raw.

`enter` or `r` restores the chosen copy over the live file, through a confirm modal. Because the write is atomic, the live file is snapshotted first — **so a restore is itself undoable**: the copy you restored from still sits in the store, and a later restore of the post-restore snapshot brings the file you had back.

A `.env` restore brings the secrets back too, which the confirm makes clear.

## What this means in practice

- A bad inline edit, a bad `$EDITOR` save, or a bad healthcheck insertion can always be undone from the Backups page.
- The compose file is the one piece of state the app cannot recreate — which is why every write to it goes through the atomic path, and why nothing may write to it with a plain truncating write.
- Restoring reloads the config, so the panels and any `.env`-derived interpolation reflect the restored file immediately.