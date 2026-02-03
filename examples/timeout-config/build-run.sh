#!/usr/bin/env bash

# detect location of XCADDY
XCADDY=xcaddy
if [ -f ~/go/bin/xcaddy ]; then
  XCADDY=~/go/bin/xcaddy
fi

# build caddy server if needed
if [ ! -f caddy ]; then
  echo "Building Caddy with NATS bridge..."
  cd /Users/buarki/projects/caddy-nats-bridge
  $XCADDY build --with github.com/CoverWhale/caddy-nats-bridge
  cp caddy examples/timeout-config/
  cd examples/timeout-config
fi

echo "Starting NATS server..."
nats-server &
sleep 1
trap 'killall nats-server' EXIT

echo "Starting Caddy with timeout configuration..."
echo "You can set NATS_REQUEST_DEFAULT_TIMEOUT=30s to test environment variable timeout"
./caddy run --config Caddyfile 