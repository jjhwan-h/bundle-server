#!/bin/bash

set -e

APP_NAME="bundleserver"
APP_USER="bundleserver"
APP_DIR="/mnt/d/opa_bundle_server"
ENV_FILE="$APP_DIR/.env"
UNIT_FILE="/etc/systemd/system/$APP_NAME.service"

echo "[INFO] Copying systemd unit file..."
cat <<EOF | sudo tee "$UNIT_FILE" > /dev/null
[Unit]
Description=OPA-BUNDLE-SERVER
After=network.target

[Service]
EnvironmentFile=$ENV_FILE
WorkingDirectory=$APP_DIR
ExecStart=$APP_DIR/opa_bundle_server serve -p 4001
User=$APP_USER
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

echo "[INFO] Reloading systemd..."
sudo systemctl daemon-reload

echo "[INFO] Enabling and starting service..."
sudo systemctl enable "$APP_NAME"
sudo systemctl start "$APP_NAME"

