# FolstingX 转发面板

> 高性能、可视化、多协议转发管理面板

---

## 目录

- [项目简介](#项目简介)
- [功能特性](#功能特性)
- [技术栈](#技术栈)
- [快速开始](#快速开始)
- [分步开发指南](#分步开发指南)
- [部署文档](#部署文档)
- [API文档](#api文档)

---

## 项目简介

FolstingX 是一个专为网络转发场景设计的高性能管理面板，支持多协议、多链路、多用户管理。参考 ZeroForwarder、NyaPass、Flux 等成熟产品的设计理念，提供完整的转发规则配置、实时监控、流量统计、热更新等功能。

---

## 功能特性

| 功能模块 | 状态 | 描述 |
|---------|------|------|
| 多协议转发 | 🔲 开发中 | TCP/UDP/HTTP/HTTPS/SOCKS5 |
| 入站代理（可选） | 🔲 开发中 | 默认不开启，开启后配置客户端入站协议 |
| VLESS+Reality 入站 | 🔲 开发中 | 海外直连开启入站代理时必须使用 |
| 隧道转发 / 隧道+SS | 🔲 开发中 | 国内中转开启入站代理时可选 |
| 热更新规则 | 🔲 开发中 | 无需重启即可更新转发规则 |
| 实时流量监控 | 🔲 开发中 | WebSocket 推送实时数据 |
| 多用户权限管理 | 🔲 开发中 | RBAC 权限模型 |
| 负载均衡 & 故障转移 | 🔲 开发中 | 多出口自动切换 |
| 带宽限速 | 🔲 开发中 | 单用户/全局限速 |
| 单线程带宽聚合 | 🔲 开发中 | 多链路带宽合并 |
| 批量导入/导出规则 | 🔲 开发中 | JSON/CSV 格式 |
| 自动化部署 | 🔲 开发中 | 一键安装脚本 |
| REST API | 🔲 开发中 | 完整的编程接口 |

---

## 技术栈

### 后端
- **语言**: Go 1.21+
- **Web框架**: Gin
- **数据库**: SQLite（单机）/ PostgreSQL（集群）
- **ORM**: GORM
- **转发引擎**: 内置 TCP/UDP 转发 + Xray-core（海外直连入站）+ gost（节点间加密隧道）
- **认证**: JWT + RBAC
- **实时通信**: WebSocket (gorilla/websocket)
- **任务调度**: 內置协程池

### 前端
- **框架**: Vue 3 + TypeScript
- **UI库**: Naive UI / Arco Design
- **状态管理**: Pinia
- **HTTP客户端**: Axios
- **图表**: ECharts
- **构建工具**: Vite

### 运维
- **容器**: Docker + Docker Compose
- **反向代理**: Nginx
- **进程管理**: Systemd / Supervisor
- **CI/CD**: GitHub Actions

---

## 快速开始

### 环境要求
- Linux (Ubuntu 20.04+ / Debian 11+ / CentOS 7+)
- Go 1.21+
- Node.js 18+
- Git

### 一键安装（推荐）

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/yourrepo/folstingx/main/scripts/install.sh)
```

### 手动安装

```bash
# 1. 克隆仓库
git clone https://github.com/yourrepo/folstingx.git
cd folstingx

# 2. 编译后端
cd backend
go build -o folstingx-server ./cmd/server
cd ..

# 3. 编译前端
cd frontend
npm install
npm run build
cd ..

# 4. 配置文件
cp config/config.example.yaml config/config.yaml
# 编辑 config.yaml 设置数据库路径、端口等

# 5. 启动服务
./backend/folstingx-server
```

---

## 分步开发指南

项目开发分为以下 **12个阶段**，每个阶段完成后需通过自动化测试才能进入下一阶段：

### Phase 1: 项目骨架搭建
- [ ] 初始化 Go Modules 项目结构
- [ ] 初始化 Vue3 + Vite 前端项目
- [ ] 配置 Docker Compose 开发环境
- [ ] 建立 CI/CD 流水线基础

### Phase 2: 用户认证系统
- [ ] 用户注册/登录接口
- [ ] JWT Token 颁发与刷新
- [ ] RBAC 权限模型（超级管理员/管理员/普通用户）
- [ ] 前端登录页、权限路由守卫

### Phase 3: 核心转发引擎（TCP/UDP）
- [ ] 基础 TCP 端口转发
- [ ] UDP 端口转发
- [ ] 转发规则CRUD API
- [ ] 规则热加载机制（无需重启）

### Phase 4: 高级协议支持
- [ ] HTTP 反向代理转发
- [ ] SOCKS5 代理协议支持
- [ ] VLESS+Reality 入站加密（集成 Xray-core）
- [ ] 协议自动识别

### Phase 5: 转发模式配置
- [ ] 国内-IX专线转发模式
- [ ] 国内-海外机中转模式
- [ ] 海外机直连模式
- [ ] 链式转发（入口→中间节点→出口）
- [ ] 单线程带宽聚合

### Phase 6: 负载均衡 & 故障转移
- [ ] 多出口轮询/加权轮询
- [ ] 健康检查（主动探测）
- [ ] 故障自动切换
- [ ] 带宽限速（令牌桶算法）

### Phase 7: 监控 & 统计
- [ ] 实时流量采集
- [ ] WebSocket 推送到前端
- [ ] 历史流量统计（按天/周/月）
- [ ] 机器状态监控（CPU/内存/带宽）
- [ ] 连接数统计

### Phase 8: 前端管理界面
- [ ] 仪表盘（实时监控）
- [ ] 转发规则管理页
- [ ] 用户管理页
- [ ] 节点管理页
- [ ] 日志查看页
- [ ] 系统设置页

### Phase 9: 日志系统
- [ ] 结构化日志记录
- [ ] 日志分级（Debug/Info/Warn/Error）
- [ ] 日志轮转（按天）
- [ ] 前端日志查询 & 过滤

### Phase 10: 批量操作 & 导入导出
- [ ] 转发规则批量导出（JSON/CSV）
- [ ] 批量导入规则
- [ ] 节点批量管理
- [ ] 配置备份恢复

### Phase 11: REST API 完善
- [ ] 完整的 API 文档（Swagger）
- [ ] API 密钥管理
- [ ] Webhook 通知
- [ ] 速率限制

### Phase 12: 自动化部署
- [ ] 一键安装脚本（install.sh）
- [ ] 自动更新脚本（update.sh）
- [ ] Systemd 服务配置
- [ ] Docker 镜像构建
- [ ] Nginx 反向代理配置

---

## 转发模式说明

### 入站代理：可选功能

> **入站代理默认不开启。** 不开启时，面板仅做系统级 TCP/UDP 端口转发，客户端用什么协议连接入口端口与面板无关。
> 开启入站代理后，面板托管客户端入站协议，此时需根据转发模式选择对应的入站类型。

---

### 两套独立系统

> 本项目使用两套完全独立的工具，各司其职，切勿混淆：
>
> | 工具 | 负责范围 | 使用场景 |
> |------|---------|--------|
> | **Xray-core** | 海外直连的**入站代理** | 仅用于 `海外直连` 模式开启入站代理时 |
> | **gost** | **节点间加密隧道** | 所有需要跨节点中转的链路（过墙/不过墙均用） |

---

### 模式一：海外直连

面板在海外机上监听端口，内地用户直接连接。流量需穿越防火长城，**必须开启入站代理**，使用 Xray-core 的 VLESS+Reality 做流量伪装。节点间无需隧道（直连目标）。

```
【必须开启入站代理 - vless_reality（Xray-core）】
内地用户 --[VLESS+Reality]--> 海外机 Xray 入站 --> 目标服务
```

---

### 模式二：国内-海外机中转

面板在国内入口机监听，流量经 **gost mwss 加密隧道**中转至海外出口机。入站代理**可选**。

```
【不开启入站代理】
用户（任意协议）--> 国内入口机端口
                        ↓
               gost mwss 加密隧道（过墙）
                        ↓
                   海外出口机 --> 目标

【开启入站代理 - SS（gost 提供 SS 入站）】
用户 --[Shadowsocks]--> 国内入口机 gost SS入站
                               ↓
                      gost mwss 加密隧道（过墙）
                               ↓
                          海外出口机 --> 目标
```

---

### 模式三：国内-IX专线

与中转模式相同，节点间隧道同样使用 **gost mwss**，区别仅在于国内到海外段走 IX 高速专线线路。

```
用户 --> 国内入口机 --[gost mwss / IX专线]--> 海外出口机 --> 目标
入站代理选项同模式二。
```

---

### 模式四：链式中转

多跳转发，每一跳节点之间均使用 **gost mwss 加密隧道**。入站代理**可选**，选项同模式二。

```
用户 --> 入口节点 --[gost mwss]--> 中继节点 --[gost mwss]--> 出口节点 --> 目标
```

---

### 入站代理速查

| 转发模式 | 不开启入站代理 | 可选入站类型 | 工具 |
|---------|-------------|------------|------|
| 海外直连 | ❌ 必须开启 | `vless_reality` 唯一选项 | Xray-core |
| 国内中转 | ✅ 可用 | 无 / `shadowsocks` | gost |
| IX专线 | ✅ 可用 | 无 / `shadowsocks` | gost |
| 链式中转 | ✅ 可用 | 无 / `shadowsocks` | gost |

> **节点间隧道始终是 gost mwss（或同类 gost 加密传输），与入站代理无关。**

---

## 部署文档

详见 [DEPLOYMENT.md](./DEPLOYMENT.md)

## API文档

详见 [API.md](./API.md) 或启动后访问 `http://your-server:8080/swagger/index.html`

---

## 许可证

MIT License
