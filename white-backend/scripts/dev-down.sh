#!/usr/bin/env bash
# Stops what dev-up.sh started.
cd "$(dirname "$0")/.."
for f in .run/*.pid; do
  [ -e "$f" ] || continue
  kill "$(cat "$f")" 2>/dev/null || true
  rm -f "$f"
done
