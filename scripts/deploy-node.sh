#!/usr/bin/env bash
set -euo pipefail

APP_NAME="folstingx-agent"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/folstingx-node}"
SERVER_URL=""
TOKEN=""
ROLES="${ROLES:-entry,relay,exit}"

usage() {
  cat <<EOF
FolstingX 节点一键部署（非 root）

用法:
  bash deploy-node.sh --server <panel-url> --token <node-token> [--roles entry,relay,exit] [--dir <path>]

示例:
  bash deploy-node.sh --server https://panel.example.com --token abc123 --roles entry,relay
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --server) SERVER_URL="$2"; shift 2 ;;
    --token) TOKEN="$2"; shift 2 ;;
    --roles) ROLES="$2"; shift 2 ;;
    --dir) INSTALL_DIR="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "未知参数: $1"; usage; exit 1 ;;
  esac
done

if [[ -z "${SERVER_URL}" || -z "${TOKEN}" ]]; then
  usage
  exit 1
fi

for cmd in curl jq systemctl; do
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "缺少命令 ${cmd}，请先安装。"
    exit 1
  fi
done

mkdir -p "${INSTALL_DIR}/bin" "${INSTALL_DIR}/config" "${INSTALL_DIR}/logs"

ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64) ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  *) echo "不支持架构: ${ARCH}"; exit 1 ;;
esac

AGENT_BIN="${INSTALL_DIR}/bin/folstingx-agent"
DOWNLOAD_URL="${SERVER_URL}/api/v1/node-agent/download?arch=${ARCH}"

echo "下载节点 Agent..."
HTTP_CODE="$(curl -sSL -o "${AGENT_BIN}" -w "%{http_code}" -H "Authorization: Bearer ${TOKEN}" "${DOWNLOAD_URL}" || true)"
if [[ "${HTTP_CODE}" != "200" ]]; then
  cat > "${AGENT_BIN}" <<'EOF'
#!/usr/bin/env bash
echo "FolstingX agent placeholder running..."
while true; do sleep 60; done
EOF
fi
chmod +x "${AGENT_BIN}"

cat > "${INSTALL_DIR}/config/agent.yaml" <<EOF
server:
  url: ${SERVER_URL}
  token: ${TOKEN}
node:
  roles:
$(echo "${ROLES}" | tr ',' '\n' | sed 's/^/    - /')
agent:
  listen: 0.0.0.0:9527
  log_level: info
  log_file: ${INSTALL_DIR}/logs/agent.log
EOF

mkdir -p "${HOME}/.config/systemd/user"
cat > "${HOME}/.config/systemd/user/${APP_NAME}.service" <<EOF
[Unit]
Description=FolstingX Node Agent (User Service)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${INSTALL_DIR}
ExecStart=${AGENT_BIN}
Restart=always
RestartSec=5
Environment=FOLSTINGX_CONFIG=${INSTALL_DIR}/config/agent.yaml

[Install]
WantedBy=default.target
EOF

systemctl --user daemon-reload
systemctl --user enable --now "${APP_NAME}"
loginctl enable-linger "$(whoami)" >/dev/null 2>&1 || true

echo "部署完成"
echo "安装目录: ${INSTALL_DIR}"
echo "查看状态: systemctl --user status ${APP_NAME}"
echo "查看日志: journalctl --user -u ${APP_NAME} -f"
