#!/bin/bash
# Usage: ./deploy.sh user@yourserver.example.com
set -e

SERVER=${1:-peter@fimblefowl.co.uk}

HASH=$(git rev-parse --short HEAD)

echo "Building for linux/amd64 (static, hash=$HASH)..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-X main.buildHash=$HASH" \
  -o /tmp/qrzlook_deploy .

echo "Copying binary to $SERVER..."
scp /tmp/qrzlook_deploy "$SERVER":/tmp/qrzlook_new

echo "Installing on $SERVER..."
ssh "$SERVER" "mv /tmp/qrzlook_new /home/peter/qrzlook && systemctl --user restart qrzlook && systemctl --user is-active qrzlook"

echo "Done — deployed $HASH to $SERVER"
