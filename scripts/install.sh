#!/usr/bin/env bash
set -euo pipefail

APP_NAME="folstingx"
INSTALL_DIR="/opt/${APP_NAME}"
SERVICE_FILE="/etc/systemd/system/${APP_NAME}.service"

if [[ $EUID -ne 0 ]]; then
  echo "请使用 root 运行 install.sh"
  exit 1
fi

OS_ID=""
if [[ -f /etc/os-release ]]; then
  source /etc/os-release
  OS_ID="${ID:-}"
fi

echo "检测系统: ${OS_ID}"
case "$OS_ID" in
  ubuntu|debian)
    apt update
    apt install -y curl wget nginx tar git build-essential
    ;;
  centos|rhel|rocky|almalinux)
    yum install -y curl wget nginx tar git gcc gcc-c++ make
    ;;
  *)
    echo "不支持的发行版: $OS_ID"
    exit 1
    ;;
esac

mkdir -p "$INSTALL_DIR" "$INSTALL_DIR"/bin "$INSTALL_DIR"/config
cp -r backend "$INSTALL_DIR"/
cp -r frontend/dist "$INSTALL_DIR"/frontend-dist || true
cp config/config.example.yaml "$INSTALL_DIR"/config/config.yaml

JWT_SECRET=$(openssl rand -hex 32)
sed -i "s/^  jwt_secret:.*/  jwt_secret: ${JWT_SECRET}/" "$INSTALL_DIR"/config/config.yaml

cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=FolstingX Service
After=network.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${INSTALL_DIR}/backend
ExecStart=/usr/bin/env bash -lc 'go run cmd/server/main.go'
Restart=always
RestartSec=5
User=root

[Install]
WantedBy=multi-user.target
EOF

cat > /etc/nginx/conf.d/folstingx.conf <<EOF
server {
    listen 80;
    server_name _;

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /ws/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    location / {
        root ${INSTALL_DIR}/frontend-dist;
        try_files $uri /index.html;
    }
}
EOF

systemctl daemon-reload
systemctl enable --now "$APP_NAME"
systemctl restart nginx

echo "安装完成"
echo "访问地址: http://$(hostname -I | awk '{print $1}')"
echo "默认账号: admin / admin123"
