#!/usr/bin/env bash

# detect location of XCADDY
XCADDY=xcaddy
if [ -f ~/go/bin/xcaddy ]; then
  XCADDY=~/go/bin/xcaddy
fi

# build caddy server if needed
if [ ! -f caddy ]; then
  echo "Building Caddy with NATS bridge..."
  # Calculate project root (assuming we're in examples/header-logging)
  PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
  cd "$PROJECT_ROOT"
  $XCADDY build --with github.com/CoverWhale/caddy-nats-bridge
  cp caddy examples/header-logging/
  cd examples/header-logging
fi

echo "Starting NATS server..."
nats-server &
sleep 1
trap 'killall nats-server' EXIT

echo ""
echo "Starting Caddy with header logging example..."
echo ""
echo "To test WITHOUT header redaction, just run:"
echo "  ./caddy run --config Caddyfile"
echo ""
echo "To test WITH header redaction, run:"
echo "  LOGGER_REDACT_HEADERS=\"Authorization,X-API-Key\" ./caddy run --config Caddyfile"
echo ""
echo "Then in another terminal, create a NATS responder:"
echo "  nats reply 'test.subject.>' --command \"echo 'Response for {{1}}'\""
echo ""
echo "And make test requests:"
echo "  curl -H 'Authorization: Bearer token123' -H 'X-API-Key: key456' http://127.0.0.1:8888/test/hello"
echo ""
echo "Check the Caddy logs for 'publishing NATS message' to see the headers."
echo ""

./caddy run --config Caddyfile

