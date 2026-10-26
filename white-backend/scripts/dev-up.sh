#!/usr/bin/env bash
# Builds every service and starts it locally with NATS, logging to .run/<svc>.log.
# Needs DATABASE_URL, AUTH_SECRET and ENCRYPTION_KEY in the environment or .env.
# Stop everything with ./scripts/dev-down.sh.
set -euo pipefail
cd "$(dirname "$0")/.."

if [ -f .env ]; then set -a; . ./.env; set +a; fi
: "${DATABASE_URL:?DATABASE_URL belum diset}"
: "${AUTH_SECRET:?AUTH_SECRET belum diset}"
: "${ENCRYPTION_KEY:?ENCRYPTION_KEY belum diset}"
export NATS_URL="${NATS_URL:-nats://127.0.0.1:4222}"
export LOG_FORMAT="${LOG_FORMAT:-text}"

mkdir -p .run bin
[ -n "${SKIP_BUILD:-}" ] || make build >/dev/null

if ! nc -z 127.0.0.1 4222 2>/dev/null; then
  nats-server -js -sd .run/nats -a 127.0.0.1 -p 4222 >.run/nats.log 2>&1 &
  echo $! >.run/nats.pid
  sleep 1
fi

start() {
  local svc=$1 port=$2
  GRPC_ADDR=":$port" ./bin/"$svc" serve >".run/$svc.log" 2>&1 &
  echo $! >".run/$svc.pid"
  echo "$svc :$port pid $!"
}

# identity first: it owns the client table other services' imports refer to.
start identity 9001
sleep 1
start integrations 9002
start analytics 9003
start publishing 9004
start seosuite 9005
[ -x bin/reports ] && start reports 9006
[ -x bin/content ] && start content 9011
# ERP services; hr before the ones that read its directory.
for erp in "hr 9008" "finance 9007" "recruitment 9009" "sales 9010" "performance 9012"; do
  set -- $erp
  [ -x "bin/$1" ] && start "$1" "$2"
done
./bin/scheduler worker >.run/scheduler.log 2>&1 &
echo $! >.run/scheduler.pid
./bin/gateway worker >.run/gateway.log 2>&1 &
echo $! >.run/gateway.pid
echo "gateway ${HTTP_ADDR:-:8080}"
