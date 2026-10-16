#!/usr/bin/env bash
# Start the Dailzo backend for local/emulator testing.
#
# The apps talk to http://10.0.2.2:2193 (the Android emulator's alias for this
# machine), so if this isn't running the customer app shows
# "Couldn't load products" and the vendor app shows empty lists.
#
#   ./run-backend.sh          # run in the foreground (Ctrl-C to stop)
#   ./run-backend.sh --bg     # run in the background, log to /tmp/dailzo-server.log
set -euo pipefail
cd "$(dirname "$0")"

if lsof -nP -iTCP:2193 -sTCP:LISTEN >/dev/null 2>&1; then
  echo "Already running on :2193"; exit 0
fi

go build -o /tmp/dailzo-server .
if [[ "${1:-}" == "--bg" ]]; then
  nohup /tmp/dailzo-server > /tmp/dailzo-server.log 2>&1 &
  sleep 5
  curl -sf -m 5 http://localhost:2193/api/v1/health && echo "  <- backend up (log: /tmp/dailzo-server.log)"
else
  exec /tmp/dailzo-server
fi
