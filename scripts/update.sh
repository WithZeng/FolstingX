#!/usr/bin/env bash
set -euo pipefail

APP_NAME="folstingx"
INSTALL_DIR="/opt/${APP_NAME}"
BACKUP_DIR="/opt/${APP_NAME}-backup-$(date +%Y%m%d%H%M%S)"

if [[ $EUID -ne 0 ]]; then
  echo "请使用 root 运行 update.sh"
  exit 1
fi

echo "开始备份: ${BACKUP_DIR}"
mkdir -p "$BACKUP_DIR"
cp -r "$INSTALL_DIR"/* "$BACKUP_DIR"/

rollback() {
  echo "更新失败，回滚中..."
  systemctl stop "$APP_NAME" || true
  rm -rf "$INSTALL_DIR"/*
  cp -r "$BACKUP_DIR"/* "$INSTALL_DIR"/
  systemctl start "$APP_NAME" || true
}
trap rollback ERR

systemctl stop "$APP_NAME"
cp -r backend "$INSTALL_DIR"/
cp -r frontend/dist "$INSTALL_DIR"/frontend-dist || true

systemctl start "$APP_NAME"

echo "更新完成"
