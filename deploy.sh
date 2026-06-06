#!/bin/bash
# Usage: ./deploy.sh user@yourserver.example.com
set -e

SERVER=${1:?Usage: ./deploy.sh user@host}

echo "Building for linux/amd64..."
GOOS=linux GOARCH=amd64 go build -o qrzlook .

echo "Copying binary to $SERVER..."
scp qrzlook "$SERVER":/tmp/qrzlook

echo "Installing on $SERVER..."
ssh "$SERVER" bash <<'EOF'
  sudo mv /tmp/qrzlook /usr/local/bin/qrzlook
  sudo chmod 755 /usr/local/bin/qrzlook
EOF

echo "Done. On the server:"
echo "  1. Edit /etc/systemd/system/qrzlook.service and set QRZ_USERNAME/QRZ_PASSWORD"
echo "  2. sudo systemctl daemon-reload"
echo "  3. sudo systemctl enable --now qrzlook"
echo "  4. Add the lines from apache-proxy.conf to your VirtualHost and reload Apache"
