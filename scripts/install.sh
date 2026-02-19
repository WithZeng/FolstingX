#!/usr/bin/env bash
set -euo pipefail

APP_NAME="folstingx"
INSTALL_DIR="/opt/${APP_NAME}"
SERVICE_FILE="/etc/systemd/system/${APP_NAME}.service"
REPO_URL="${REPO_URL:-https://github.com/WithZeng/FolstingX.git}"
REPO_REF="${REPO_REF:-main}"
TMP_DIR="/tmp/${APP_NAME}-src-$$"

if [[ "${EUID}" -ne 0 ]]; then
  echo "请使用 root 运行 install.sh (仅安装需要 root, 运行使用 folstingx 用户)"
  exit 1
fi

cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

install_packages() {
  local os_id=""
  if [[ -f /etc/os-release ]]; then
    # shellcheck disable=SC1091
    source /etc/os-release
    os_id="${ID:-}"
  fi

  echo "检测系统: ${os_id}"
  case "${os_id}" in
    ubuntu | debian)
      apt update
      apt install -y curl wget nginx tar git ca-certificates build-essential golang-go nodejs npm openssl
      ;;
    centos | rhel | rocky | almalinux)
      yum install -y curl wget nginx tar git ca-certificates gcc gcc-c++ make golang nodejs npm openssl
      ;;
    *)
      echo "不支持的发行版: ${os_id}"
      exit 1
      ;;
  esac
}

prepare_source() {
  echo "拉取源码: ${REPO_URL} (${REPO_REF})"
  rm -rf "${TMP_DIR}"
  git clone --depth 1 --branch "${REPO_REF}" "${REPO_URL}" "${TMP_DIR}"
}

build_backend() {
  echo "编译后端..."
  mkdir -p "${INSTALL_DIR}/bin" "${INSTALL_DIR}/backend" "${INSTALL_DIR}/config" "${INSTALL_DIR}/logs"
  cp -r "${TMP_DIR}/backend/"* "${INSTALL_DIR}/backend/"
  cp "${TMP_DIR}/config/config.example.yaml" "${INSTALL_DIR}/config/config.yaml"
  (
    cd "${INSTALL_DIR}/backend"
    go mod tidy
    go build -o "${INSTALL_DIR}/bin/folstingx-server" ./cmd/server
  )
}

build_frontend() {
  echo "编译前端..."
  (
    cd "${TMP_DIR}/frontend"
    if [[ -f package-lock.json ]]; then
      npm ci
    else
      npm install
    fi
    npm run build
  )
  rm -rf "${INSTALL_DIR}/frontend-dist"
  cp -r "${TMP_DIR}/frontend/dist" "${INSTALL_DIR}/frontend-dist"
}

setup_config() {
  # 自动生成 JWT Secret，避免默认密钥上线。
  local jwt_secret
  jwt_secret="$(openssl rand -hex 32)"
  sed -i "s/^  jwt_secret:.*/  jwt_secret: ${jwt_secret}/" "${INSTALL_DIR}/config/config.yaml"
}

setup_systemd() {
  # 创建专用运行用户 (非 root)
  if ! id "folstingx" &>/dev/null; then
    useradd -r -m -s /bin/bash -d /home/folstingx folstingx || true
    echo "创建服务用户: folstingx"
  fi
  # 授权安装目录
  chown -R folstingx:folstingx "${INSTALL_DIR}"

  cat > "${SERVICE_FILE}" <<EOF
[Unit]
Description=FolstingX Service
After=network.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${INSTALL_DIR}/backend
ExecStart=${INSTALL_DIR}/bin/folstingx-server
Restart=always
RestartSec=5
User=folstingx
Group=folstingx
LimitNOFILE=65535
# 允许绑定低端口
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable --now "${APP_NAME}"
}

setup_nginx() {
  mkdir -p /etc/nginx/conf.d
  # Disable distro default site to avoid hitting "Welcome to nginx".
  if [[ -f /etc/nginx/sites-enabled/default ]]; then
    rm -f /etc/nginx/sites-enabled/default
  fi
  if [[ -f /etc/nginx/conf.d/default.conf ]]; then
    rm -f /etc/nginx/conf.d/default.conf
  fi
  cat > /etc/nginx/conf.d/folstingx.conf <<EOF
server {
    listen 80 default_server;
    listen [::]:80 default_server;
    server_name _;

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
    }

    location /ws/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    location / {
        root ${INSTALL_DIR}/frontend-dist;
        try_files \$uri /index.html;
    }
}
EOF

  nginx -t
  systemctl enable --now nginx
  systemctl restart nginx
}

main() {
  install_packages
  prepare_source
  build_backend
  build_frontend
  setup_config
  setup_systemd
  setup_nginx

  echo "安装完成"
  echo "访问地址: http://$(hostname -I | awk '{print $1}')"
  echo "默认账号: admin / admin123"
}

main "$@"
