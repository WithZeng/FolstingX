#!/usr/bin/env bash
set -euo pipefail

APP_NAME="folstingx"
INSTALL_DIR="/opt/${APP_NAME}"
REPO_URL="${REPO_URL:-https://github.com/WithZeng/FolstingX.git}"
REPO_REF="${REPO_REF:-main}"
TMP_DIR="/tmp/${APP_NAME}-update-$$"
BACKUP_DIR="/opt/${APP_NAME}-backup-$(date +%Y%m%d%H%M%S)"

if [[ "${EUID}" -ne 0 ]]; then
  echo "请使用 root 运行 update.sh"
  exit 1
fi

cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

rollback() {
  echo "更新失败，开始回滚..."
  systemctl stop "${APP_NAME}" || true
  rm -rf "${INSTALL_DIR}"/*
  cp -r "${BACKUP_DIR}"/* "${INSTALL_DIR}"/
  systemctl start "${APP_NAME}" || true
}
trap rollback ERR

echo "[1/7] 备份当前版本: ${BACKUP_DIR}"
mkdir -p "${BACKUP_DIR}"
cp -r "${INSTALL_DIR}"/* "${BACKUP_DIR}"/

echo "[2/7] 拉取最新代码: ${REPO_URL} (${REPO_REF})"
git clone --depth 1 --branch "${REPO_REF}" "${REPO_URL}" "${TMP_DIR}"

echo "[3/7] 停止服务"
systemctl stop "${APP_NAME}"

echo "[4/7] 更新后端并编译"
rm -rf "${INSTALL_DIR}/backend"
mkdir -p "${INSTALL_DIR}/backend" "${INSTALL_DIR}/bin"
cp -r "${TMP_DIR}/backend/"* "${INSTALL_DIR}/backend/"
(
  cd "${INSTALL_DIR}/backend"
  go mod tidy
  go build -o "${INSTALL_DIR}/bin/folstingx-server" ./cmd/server
)

echo "[5/7] 更新前端并构建"
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

echo "[6/7] 更新配置模板和脚本"
mkdir -p "${INSTALL_DIR}/scripts" "${INSTALL_DIR}/config"
cp -r "${TMP_DIR}/scripts/"* "${INSTALL_DIR}/scripts/" || true
cp "${TMP_DIR}/config/config.example.yaml" "${INSTALL_DIR}/config/config.example.yaml" || true

echo "[7/7] 启动服务"
systemctl daemon-reload
systemctl start "${APP_NAME}"
systemctl status "${APP_NAME}" --no-pager -l || true

echo "更新完成"
