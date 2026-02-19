# FolstingX 转发面板

高性能网络转发管理面板，支持 TCP/UDP 转发、可选入站代理、节点间隧道、实时监控与多用户权限管理。

## 当前版本
- `v1.0.1`

## 核心功能
- TCP/UDP 端口转发，支持规则热更新
- 两套独立工具：`Xray-core`（海外直连入站）+ `gost`（节点间隧道）
- 入站代理可选，默认关闭
- RBAC 权限模型（`super_admin/admin/user`）
- 实时监控（WebSocket + ECharts）
- 规则导入导出、日志查询、限流与基础安全中间件
- Docker / Systemd / Nginx 部署

## 技术栈
- 后端：Go 1.21+ / Gin / GORM / SQLite / JWT / WebSocket
- 前端：Vue 3 / TypeScript / Vite / Naive UI / Pinia / ECharts
- 部署：Docker / Docker Compose / Systemd / Nginx

## 快速开始

### 环境要求
- Linux（Ubuntu 20.04+ / Debian 11+ / CentOS 7+）
- Go 1.21+
- Node.js 18+
- Git

### 一键安装（推荐）
```bash
bash <(curl -fsSL https://raw.githubusercontent.com/WithZeng/FolstingX/main/scripts/install.sh)
```

### 一键更新到最新版本
```bash
bash <(curl -fsSL https://raw.githubusercontent.com/WithZeng/FolstingX/main/scripts/update.sh)
```

### 手动安装
```bash
git clone https://github.com/WithZeng/FolstingX.git
cd FolstingX

# 前端构建
cd frontend
npm install
npm run build
cd ..

# 后端运行
cd backend
go run cmd/server/main.go
```

## Docker 部署
```bash
cd docker
docker compose up -d --build
```

## 本地开发
```bash
# 后端
cd backend
go run cmd/server/main.go

# 前端（新终端）
cd frontend
npm install
npm run dev
```

## 默认账号
- 用户名：`admin`
- 密码：`admin123`

## 项目结构
```text
backend/                后端代码
frontend/               前端代码
config/                 配置示例
scripts/                安装与更新脚本
docker/                 Docker 与 Nginx 配置
deploy/systemd/         Systemd 服务文件
```

## 说明
- 若你在 Windows 编辑后出现乱码，请确保文件以 **UTF-8** 保存。
- 若服务部署到公网，请务必修改默认密码与 JWT 密钥。
