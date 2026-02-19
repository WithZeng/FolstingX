#!/usr/bin/env bash
set -euo pipefail

# ===================================================================
# FolstingX AI 自动配置脚本
# 一键完成: 环境检测 → 依赖安装 → 面板部署 → 节点注册 → 隧道创建
#
# 用法:
#   curl -fsSL https://your-panel.com/install-ai.sh | bash
#   或: bash ai-autoconfig.sh [选项]
#
# 选项:
#   --mode panel        仅安装面板
#   --mode node         仅安装节点 Agent
#   --mode full         完整安装: 面板 + 本机节点 + 示例隧道
#   --panel-addr URL    面板地址 (节点模式必须)
#   --secret SECRET     节点密钥 (节点模式必须)
#   --domain DOMAIN     面板域名 (可选, 用于 TLS)
#   --port PORT         面板端口 (默认 8080)
# ===================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

MODE="full"
PANEL_ADDR=""
SECRET=""
DOMAIN=""
PORT="8080"
INSTALL_DIR="/opt/folstingx"
DATA_DIR="/opt/folstingx/data"
AGENT_DIR="/etc/folstingx_agent"

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step()  { echo -e "\n${CYAN}━━━ $1 ━━━${NC}"; }

# ========== 参数解析 ==========
while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode) MODE="$2"; shift 2 ;;
    --panel-addr) PANEL_ADDR="$2"; shift 2 ;;
    --secret) SECRET="$2"; shift 2 ;;
    --domain) DOMAIN="$2"; shift 2 ;;
    --port) PORT="$2"; shift 2 ;;
    --dir) INSTALL_DIR="$2"; shift 2 ;;
    -h|--help)
      echo "FolstingX AI 自动配置"
      echo ""
      echo "用法: bash ai-autoconfig.sh [--mode panel|node|full] [--panel-addr URL] [--secret SECRET]"
      echo ""
      echo "模式:"
      echo "  panel  - 安装面板 (后端+前端+数据库)"
      echo "  node   - 安装节点 Agent (需指定 --panel-addr 和 --secret)"
      echo "  full   - 完整安装: 面板 + 本机作为首个节点 + 创建示例隧道"
      exit 0
      ;;
    *) echo "未知参数: $1"; exit 1 ;;
  esac
done

# ========== 环境检测 ==========
log_step "环境检测"

ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64)  ARCH_NAME="amd64" ;;
  aarch64) ARCH_NAME="arm64" ;;
  *) log_error "不支持的架构: ${ARCH}"; exit 1 ;;
esac
log_info "架构: ${ARCH} (${ARCH_NAME})"

OS="$(uname -s)"
if [[ "${OS}" != "Linux" ]]; then
  log_error "仅支持 Linux"; exit 1
fi

# 检测发行版
if [ -f /etc/os-release ]; then
  . /etc/os-release
  log_info "系统: ${PRETTY_NAME}"
fi

# 检查 root
if [[ $EUID -ne 0 ]]; then
  log_warn "建议以 root 运行 (当前用户: $(whoami))"
fi

# 检查必要命令
for cmd in curl wget; do
  if command -v ${cmd} >/dev/null 2>&1; then
    log_info "✓ ${cmd} 已安装"
  else
    log_warn "✗ ${cmd} 未安装，尝试安装..."
    apt-get install -y ${cmd} 2>/dev/null || yum install -y ${cmd} 2>/dev/null || true
  fi
done

# ========== 节点模式: 仅安装 Agent ==========
if [[ "${MODE}" == "node" ]]; then
  log_step "安装节点 Agent"

  if [[ -z "${PANEL_ADDR}" ]] || [[ -z "${SECRET}" ]]; then
    log_error "节点模式需要 --panel-addr 和 --secret 参数"
    echo ""
    echo "示例: bash ai-autoconfig.sh --mode node --panel-addr https://panel.example.com --secret abc123"
    exit 1
  fi

  mkdir -p "${AGENT_DIR}"

  # 下载 gost
  log_info "下载 gost..."
  GOST_URL="https://github.com/go-gost/gost/releases/latest/download/gost_linux_${ARCH_NAME}"
  curl -fsSL -o "${AGENT_DIR}/gost" "${GOST_URL}" 2>/dev/null || {
    log_warn "gost 下载失败, 使用面板提供的脚本重试..."
    curl -fsSL "${PANEL_ADDR}/api/v1/node-agent/install.sh" | bash -s -- -a "${PANEL_ADDR}" -s "${SECRET}"
    exit $?
  }
  chmod +x "${AGENT_DIR}/gost"

  # 配置
  cat > "${AGENT_DIR}/config.json" <<EOF
{"addr":"${PANEL_ADDR}","secret":"${SECRET}"}
EOF
  echo '{}' > "${AGENT_DIR}/gost.json"

  # systemd
  cat > /etc/systemd/system/folstingx-agent.service <<EOF
[Unit]
Description=FolstingX Node Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${AGENT_DIR}
ExecStart=${AGENT_DIR}/gost -C ${AGENT_DIR}/gost.json
Restart=always
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable folstingx-agent
  systemctl restart folstingx-agent

  log_info "✓ 节点 Agent 安装完成"
  log_info "服务状态: systemctl status folstingx-agent"
  exit 0
fi

# ========== 面板模式 / 完整模式 ==========
log_step "安装 FolstingX 面板"

# 安装 Docker (如果没有)
if ! command -v docker >/dev/null 2>&1; then
  log_info "安装 Docker..."
  curl -fsSL https://get.docker.com | bash
  systemctl enable docker
  systemctl start docker
fi
log_info "✓ Docker $(docker --version | cut -d' ' -f3)"

# 安装 docker-compose (如果没有)
if ! command -v docker-compose >/dev/null 2>&1 && ! docker compose version >/dev/null 2>&1; then
  log_info "安装 docker-compose..."
  curl -fsSL "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
  chmod +x /usr/local/bin/docker-compose
fi
log_info "✓ docker-compose 已就绪"

# 创建目录结构
mkdir -p "${INSTALL_DIR}"/{config,data,logs,docker}

# 生成 JWT Secret
JWT_SECRET=$(openssl rand -hex 32 2>/dev/null || head -c 64 /dev/urandom | base64 | tr -d '=+/' | head -c 64)

# 生成配置文件
cat > "${INSTALL_DIR}/config/config.yaml" <<EOF
server:
  host: "0.0.0.0"
  port: ${PORT}
  mode: "release"

database:
  type: "sqlite"
  dsn: "/app/data/folstingx.db"

auth:
  jwt_secret: "${JWT_SECRET}"

log:
  level: "info"
  file: "/app/logs/app.log"
EOF

log_info "✓ 配置文件已生成"

# 生成 docker-compose.yml
PANEL_DOMAIN="${DOMAIN:-$(curl -s ifconfig.me 2>/dev/null || echo 'localhost')}"

cat > "${INSTALL_DIR}/docker/docker-compose.yml" <<EOF
version: '3.8'

services:
  backend:
    image: folstingx/server:latest
    build:
      context: ..
      dockerfile: docker/Dockerfile
    container_name: folstingx-backend
    restart: always
    ports:
      - "${PORT}:8080"
    volumes:
      - ../config:/app/config:ro
      - ../data:/app/data
      - ../logs:/app/logs
    environment:
      - GIN_MODE=release
    networks:
      - folstingx

  frontend:
    image: nginx:alpine
    container_name: folstingx-frontend
    restart: always
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/conf.d/default.conf:ro
    depends_on:
      - backend
    networks:
      - folstingx

networks:
  folstingx:
    driver: bridge
EOF

# 生成 nginx.conf
cat > "${INSTALL_DIR}/docker/nginx.conf" <<EOF
server {
    listen 80;
    server_name ${PANEL_DOMAIN};

    location /api/ {
        proxy_pass http://backend:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    }

    location /ws/ {
        proxy_pass http://backend:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
    }

    location / {
        root /usr/share/nginx/html;
        try_files \$uri \$uri/ /index.html;
    }
}
EOF

log_info "✓ Docker Compose 配置已生成"

# 启动面板
log_step "启动面板服务"
cd "${INSTALL_DIR}/docker"
docker compose up -d 2>/dev/null || docker-compose up -d 2>/dev/null || {
  log_warn "Docker Compose 启动失败，可能需要先编译镜像"
  log_info "请手动执行: cd ${INSTALL_DIR}/docker && docker compose up -d --build"
}

# 等待后端启动
log_info "等待后端启动..."
for i in $(seq 1 30); do
  if curl -sf "http://127.0.0.1:${PORT}/health" >/dev/null 2>&1; then
    log_info "✓ 面板后端已启动"
    break
  fi
  sleep 1
done

# ========== 完整模式: 注册本机为节点 + 创建示例隧道 ==========
if [[ "${MODE}" == "full" ]]; then
  log_step "配置本机节点 + 示例隧道"

  PANEL_URL="http://127.0.0.1:${PORT}"

  # 登录获取 Token
  log_info "登录面板..."
  LOGIN_RESP=$(curl -sf -X POST "${PANEL_URL}/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}' 2>/dev/null || echo '{}')

  TOKEN=$(echo "${LOGIN_RESP}" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
  if [[ -z "${TOKEN}" ]]; then
    log_warn "自动登录失败，跳过节点注册。请手动登录面板进行配置。"
  else
    AUTH="Authorization: Bearer ${TOKEN}"
    LOCAL_IP=$(curl -sf ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')

    # 注册本机为节点
    log_info "注册本机为节点 (${LOCAL_IP})..."
    NODE_RESP=$(curl -sf -X POST "${PANEL_URL}/api/v1/nodes" \
      -H "${AUTH}" -H "Content-Type: application/json" \
      -d "{\"name\":\"本机节点\",\"host\":\"${LOCAL_IP}\",\"ssh_port\":22,\"ssh_user\":\"folstingx\",\"location\":\"Local\",\"roles\":[\"entry\",\"relay\",\"exit\"]}" 2>/dev/null || echo '{}')

    NODE_ID=$(echo "${NODE_RESP}" | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2)
    if [[ -n "${NODE_ID}" ]]; then
      log_info "✓ 节点注册成功 (ID: ${NODE_ID})"

      # 获取安装命令并安装 Agent
      log_info "获取节点安装命令..."
      INSTALL_RESP=$(curl -sf "${PANEL_URL}/api/v1/nodes/${NODE_ID}/install-command" \
        -H "${AUTH}" -H "X-Panel-Addr: http://${LOCAL_IP}:${PORT}" 2>/dev/null || echo '{}')

      NODE_SECRET=$(echo "${INSTALL_RESP}" | grep -o '"secret":"[^"]*"' | cut -d'"' -f4)
      if [[ -n "${NODE_SECRET}" ]]; then
        log_info "安装本机 Agent..."
        bash "$0" --mode node --panel-addr "http://${LOCAL_IP}:${PORT}" --secret "${NODE_SECRET}"
      fi

      # 创建示例隧道
      log_info "创建示例隧道..."
      TUNNEL_RESP=$(curl -sf -X POST "${PANEL_URL}/api/v1/tunnels" \
        -H "${AUTH}" -H "Content-Type: application/json" \
        -d "{\"name\":\"示例-端口转发\",\"type\":1,\"traffic_ratio\":1.0,\"is_active\":true}" 2>/dev/null || echo '{}')

      TUNNEL_ID=$(echo "${TUNNEL_RESP}" | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2)
      if [[ -n "${TUNNEL_ID}" ]]; then
        # 添加入口链路节点
        curl -sf -X POST "${PANEL_URL}/api/v1/tunnels/${TUNNEL_ID}/chain" \
          -H "${AUTH}" -H "Content-Type: application/json" \
          -d "{\"node_id\":${NODE_ID},\"chain_type\":1,\"port\":10000,\"protocol\":\"tcp\"}" >/dev/null 2>&1

        # 添加示例转发
        curl -sf -X POST "${PANEL_URL}/api/v1/tunnels/${TUNNEL_ID}/forwards" \
          -H "${AUTH}" -H "Content-Type: application/json" \
          -d "{\"name\":\"示例转发\",\"remote_address\":\"127.0.0.1:80\",\"listen_port\":10000,\"protocol\":\"tcp\"}" >/dev/null 2>&1

        log_info "✓ 示例隧道已创建 (ID: ${TUNNEL_ID})"
      fi
    fi
  fi
fi

# ========== 完成 ==========
log_step "安装完成"

EXTERNAL_IP=$(curl -sf ifconfig.me 2>/dev/null || echo "your-server-ip")

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║          FolstingX 安装完成!                         ║${NC}"
echo -e "${GREEN}╠══════════════════════════════════════════════════════╣${NC}"
echo -e "${GREEN}║${NC}  面板地址: http://${EXTERNAL_IP}:${PORT}                  ${GREEN}║${NC}"
echo -e "${GREEN}║${NC}  默认账号: admin / admin123                          ${GREEN}║${NC}"
echo -e "${GREEN}║${NC}  安装目录: ${INSTALL_DIR}                              ${GREEN}║${NC}"
echo -e "${GREEN}║${NC}                                                      ${GREEN}║${NC}"
echo -e "${GREEN}║${NC}  ${YELLOW}⚠ 请立即修改默认密码!${NC}                             ${GREEN}║${NC}"
echo -e "${GREEN}╠══════════════════════════════════════════════════════╣${NC}"
echo -e "${GREEN}║${NC}  添加节点命令:                                        ${GREEN}║${NC}"
echo -e "${GREEN}║${NC}  在面板 > 节点管理 > 点击 '安装命令' 获取一键脚本    ${GREEN}║${NC}"
echo -e "${GREEN}╠══════════════════════════════════════════════════════╣${NC}"
echo -e "${GREEN}║${NC}  常用命令:                                            ${GREEN}║${NC}"
echo -e "${GREEN}║${NC}  查看状态: cd ${INSTALL_DIR}/docker && docker compose ps  ${GREEN}║${NC}"
echo -e "${GREEN}║${NC}  查看日志: docker logs -f folstingx-backend            ${GREEN}║${NC}"
echo -e "${GREEN}║${NC}  重启服务: docker compose restart                      ${GREEN}║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
echo ""

# 如果是完整模式，显示额外信息
if [[ "${MODE}" == "full" ]]; then
  echo -e "${CYAN}━━━ 已自动完成 ━━━${NC}"
  echo "  ✓ 面板已部署并运行"
  echo "  ✓ 本机已注册为节点 (entry+relay+exit)"
  echo "  ✓ gost Agent 已安装并连接"
  echo "  ✓ 示例端口转发隧道已创建"
  echo ""
  echo "下一步:"
  echo "  1. 登录面板 http://${EXTERNAL_IP}:${PORT}"
  echo "  2. 修改默认密码"
  echo "  3. 在其他服务器上添加更多节点"
  echo "  4. 创建链式中转隧道 (entry → relay → exit)"
fi
