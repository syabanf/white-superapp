#!/usr/bin/env bash
# Pulls the backend images and restarts the stack, then waits for the gateway.
# Run on the server from the folder holding docker-compose.prod.yml, Caddyfile and .env.
#   ./deploy.sh              # newest images (tag latest)
#   TAG=<commit-sha> ./deploy.sh   # a specific build, e.g. to roll back
set -euo pipefail
cd "$(dirname "$0")"

if [ ! -f .env ]; then
  echo ".env belum ada: salin .env.prod.example ke .env lalu isi nilainya." >&2
  exit 1
fi

export TAG="${TAG:-latest}"
compose() { docker compose -f docker-compose.prod.yml "$@"; }

compose pull --quiet
compose up -d --remove-orphans
docker image prune -f >/dev/null

for _ in $(seq 1 45); do
  if compose exec -T caddy wget -qO- http://gateway:8080/healthz >/dev/null 2>&1; then
    echo "Gateway sehat (tag $TAG)."
    exit 0
  fi
  sleep 2
done
echo "Gateway belum menjawab setelah 90 detik. Cek: docker compose -f docker-compose.prod.yml logs gateway identity" >&2
exit 1
