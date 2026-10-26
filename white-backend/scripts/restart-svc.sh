#!/usr/bin/env bash
# Rebuilds and restarts one locally running service (started by dev-up.sh).
# Usage: ./scripts/restart-svc.sh <service>   e.g. finance, hr, gateway
# A mkdir lock serializes restarts, so parallel callers never start two
# copies on the same port.
set -euo pipefail
cd "$(dirname "$0")/.."
svc=${1:?service name}

# macOS ships bash 3.2 (no associative arrays).
case $svc in
  identity) port=9001 ;; integrations) port=9002 ;; analytics) port=9003 ;; publishing) port=9004 ;;
  seosuite) port=9005 ;; reports) port=9006 ;; finance) port=9007 ;; hr) port=9008 ;;
  recruitment) port=9009 ;; sales) port=9010 ;; content) port=9011 ;; performance) port=9012 ;;
  gateway | scheduler) port= ;;
  *) echo "unknown service: $svc" >&2; exit 1 ;;
esac

if [ -f .env ]; then set -a; . ./.env; set +a; fi
export NATS_URL="${NATS_URL:-nats://127.0.0.1:4222}" LOG_FORMAT="${LOG_FORMAT:-text}"

lock=".run/restart-$svc.lock"
until mkdir "$lock" 2>/dev/null; do sleep 1; done
trap 'rmdir "$lock"' EXIT

go build -o "bin/$svc" "./cmd/$svc"
[ -f ".run/$svc.pid" ] && kill "$(cat ".run/$svc.pid")" 2>/dev/null || true
sleep 1
if [ "$svc" = gateway ] || [ "$svc" = scheduler ]; then
  nohup "./bin/$svc" worker >".run/$svc.log" 2>&1 &
else
  GRPC_ADDR=":$port" nohup "./bin/$svc" serve >".run/$svc.log" 2>&1 &
fi
echo $! >".run/$svc.pid"
sleep 2
tail -1 ".run/$svc.log"
