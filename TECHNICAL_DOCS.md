# FolstingX 技术架构文档

## 1. 系统架构概览

```
┌─────────────────────────────────────────────────────────┐
│                      用户浏览器                          │
│                   Vue3 前端面板                          │
└──────────────────────┬──────────────────────────────────┘
                       │ HTTP/WebSocket
                       ▼
┌─────────────────────────────────────────────────────────┐
│                   Nginx 反向代理                         │
│              (TLS终止 / 静态文件服务)                    │
└──────────────────────┬──────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────┐
│              FolstingX 后端服务 (Go/Gin)                 │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────────┐  │
│  │  REST API   │  │  WebSocket   │  │  转发引擎管理  │  │
│  │  /api/v1    │  │  /ws/monitor │  │  ForwardMgr   │  │
│  └─────────────┘  └──────────────┘  └───────┬───────┘  │
│                                             │           │
│  ┌──────────────────────────────────────────▼────────┐  │
│  │              转发规则执行层                        │  │
│  │  ┌─────────┐ ┌─────────┐ ┌──────────────────┐   │  │
│  │  │TCP转发  │ │UDP转发  │ │Xray-core         │   │  │
│  │  │Go原生   │ │Go原生   │ │VLESS+Reality     │   │  │
│  │  └─────────┘ └─────────┘ └──────────────────┘   │  │
│  └────────────────────────────────────────────────┘  │
│                                                        │
│  ┌──────────────────────────────────────────────────┐  │
│  │              数据层                               │  │
│  │  SQLite/PostgreSQL + GORM ORM                    │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

---

## 2. 数据库设计

### 2.1 用户表 `users`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER PK | 用户ID |
| username | VARCHAR(64) UNIQUE | 用户名 |
| password_hash | VARCHAR(256) | bcrypt 哈希密码 |
| role | ENUM(super_admin, admin, user) | 角色 |
| api_key | VARCHAR(64) | API密钥 |
| bandwidth_limit | BIGINT | 带宽限制(bytes/s，0=无限) |
| traffic_limit | BIGINT | 流量限制(bytes，0=无限) |
| traffic_used | BIGINT | 已用流量 |
| is_active | BOOLEAN | 是否启用 |
| expire_at | TIMESTAMP | 过期时间(NULL=永不过期) |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

### 2.2 节点表 `nodes`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER PK | 节点ID |
| name | VARCHAR(128) | 节点名称 |
| host | VARCHAR(256) | 节点地址(IP/域名) |
| ssh_port | INTEGER | SSH端口 |
| ssh_user | VARCHAR(64) | SSH用户名 |
| ssh_key | TEXT | SSH私钥(加密存储) |
| location | VARCHAR(64) | 地理位置(如: 香港/日本) |
| node_type | ENUM(entry,relay,exit) | 节点类型 |
| is_active | BOOLEAN | 是否启用 |
| last_check | TIMESTAMP | 最后健康检查时间 |
| latency_ms | INTEGER | 延迟(毫秒) |
| created_at | TIMESTAMP | 创建时间 |

### 2.3 转发规则表 `forward_rules`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER PK | 规则ID |
| name | VARCHAR(128) | 规则名称 |
| mode | ENUM(direct,relay,chain,ix) | 转发模式 |
| listen_node_id | INTEGER FK | 入站节点 |
| listen_port | INTEGER | 监听端口 |
| protocol | ENUM(tcp,udp,both) | 协议 |
| inbound_proxy_enabled | BOOLEAN | 是否开启入站代理（默认 false） |
| inbound_type | ENUM(vless_reality,shadowsocks) | 入站代理类型（仅当 inbound_proxy_enabled=true 时有效） |
| target_address | VARCHAR(256) | 目标地址 |
| target_port | INTEGER | 目标端口 |
| chain_nodes | JSON | 链式节点列表 [node_id, ...] |
| lb_strategy | ENUM(none,roundrobin,weighted,random) | 负载均衡策略 |
| lb_targets | JSON | 负载均衡目标列表 |
| bandwidth_limit | BIGINT | 带宽限制(bytes/s) |
| is_active | BOOLEAN | 是否启用 |
| traffic_up | BIGINT | 上行流量 |
| traffic_down | BIGINT | 下行流量 |
| connections | INTEGER | 当前连接数 |
| owner_id | INTEGER FK | 所属用户 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

### 2.4 流量统计表 `traffic_stats`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER PK | ID |
| rule_id | INTEGER FK | 关联规则 |
| user_id | INTEGER FK | 关联用户 |
| date | DATE | 统计日期 |
| traffic_up | BIGINT | 当日上行流量 |
| traffic_down | BIGINT | 当日下行流量 |
| connections | INTEGER | 当日总连接数 |
| peak_bandwidth | BIGINT | 峰值带宽 |

### 2.5 系统日志表 `system_logs`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER PK | ID |
| level | ENUM(debug,info,warn,error) | 日志级别 |
| module | VARCHAR(64) | 模块名 |
| message | TEXT | 日志内容 |
| rule_id | INTEGER NULL | 关联规则 |
| user_id | INTEGER NULL | 关联用户 |
| created_at | TIMESTAMP | 时间 |

---

## 3. API 设计

### 3.1 认证模块 `/api/v1/auth`

```
POST /api/v1/auth/login          # 用户登录，返回JWT
POST /api/v1/auth/refresh        # 刷新Token
POST /api/v1/auth/logout        # 注销
GET  /api/v1/auth/profile       # 获取当前用户信息
PUT  /api/v1/auth/password      # 修改密码
```

### 3.2 用户管理 `/api/v1/users`（需要管理员权限）

```
GET    /api/v1/users            # 用户列表
POST   /api/v1/users            # 创建用户
GET    /api/v1/users/:id        # 用户详情
PUT    /api/v1/users/:id        # 更新用户
DELETE /api/v1/users/:id        # 删除用户
POST   /api/v1/users/:id/reset-traffic  # 重置流量
```

### 3.3 节点管理 `/api/v1/nodes`

```
GET    /api/v1/nodes            # 节点列表
POST   /api/v1/nodes            # 添加节点
GET    /api/v1/nodes/:id        # 节点详情
PUT    /api/v1/nodes/:id        # 更新节点
DELETE /api/v1/nodes/:id        # 删除节点
POST   /api/v1/nodes/:id/check  # 手动健康检查
GET    /api/v1/nodes/:id/status # 节点实时状态
```

### 3.4 转发规则 `/api/v1/rules`

```
GET    /api/v1/rules            # 规则列表
POST   /api/v1/rules            # 创建规则
GET    /api/v1/rules/:id        # 规则详情
PUT    /api/v1/rules/:id        # 更新规则（支持热更新）
DELETE /api/v1/rules/:id        # 删除规则
POST   /api/v1/rules/:id/enable # 启用规则
POST   /api/v1/rules/:id/disable # 禁用规则
GET    /api/v1/rules/:id/stats  # 规则流量统计
POST   /api/v1/rules/import     # 批量导入（JSON/CSV）
GET    /api/v1/rules/export     # 批量导出
```

### 3.5 监控 `/api/v1/monitor`

```
GET    /api/v1/monitor/overview        # 系统概览
GET    /api/v1/monitor/traffic         # 全局流量统计
GET    /api/v1/monitor/rules/traffic   # 各规则流量
GET    /api/v1/monitor/nodes/status    # 各节点状态
WS     /ws/monitor                     # WebSocket 实时推送
```

### 3.6 日志 `/api/v1/logs`

```
GET    /api/v1/logs             # 日志列表（支持过滤分页）
DELETE /api/v1/logs             # 清空日志
```

---

## 4. WebSocket 消息格式

### 订阅实时数据

```json
// 客户端发送
{"type": "subscribe", "topics": ["traffic", "connections", "node_status"]}

// 服务端推送（每秒）
{
  "type": "traffic",
  "timestamp": 1708000000,
  "data": {
    "total_upload": 1024000,
    "total_download": 5120000,
    "active_connections": 42,
    "rules": [
      {"id": 1, "upload": 512000, "download": 2048000, "connections": 10}
    ]
  }
}
```

---

## 5. 转发引擎设计

### 5.1 规则热更新机制

```
前端修改规则 → API接收 → 更新数据库 → 通知ForwardManager
                                              ↓
ForwardManager持有所有活跃规则的goroutine map
                                              ↓
找到对应规则的goroutine → 优雅停止（等待已有连接完成）→ 用新配置重启
```

---

### 5.2 两套独立工具分工

| 工具 | 职责 | 使用场景 |
|------|------|--------|
| **Xray-core** | 海外直连的入站代理 | 仅当开启入站代理且模式为海外直连时 |
| **gost** | 节点间加密隧道 | 所有需要跨节点传输的链路（包括过墙段） |

#### Xray-core（入站代理，仅海外直连模式使用）
- `inbound_type = vless_reality` 时启动
- Xray 配置：VLESS+Reality 入站 → 直连出站至目标
- 面板通过 Xray gRPC API 动态管理入站配置
- 前端生成 `vless://` 分享链接和二维码

#### gost（节点间加密隧道，所有中转模式均使用）
- 面板通过 SSH 向各节点下发 gost 二进制和配置
- 跨境节点（过墙段）：使用 `mwss`（WebSocket over TLS）传输
- 纯境外节点间：可使用 `mws`（无 TLS）降低延迟
- `inbound_type = shadowsocks` 时：gost 提供 SS 入站 + mwss 出站的组合配置
- 规则变更时发送 SIGHUP 或重启 gost 进程实现热更新

**入站代理与节点间隧道的关系：**

| | 海外直连 | 国内中转/IX/链式 |
|---|---|---|
| 不开入站 | 端口转发，节点间 gost mwss | 端口转发，节点间 gost mwss |
| 开入站 vless_reality | Xray-core（必须） | 不适用 |
| 开入站 shadowsocks | 不适用 | gost SS入站 + mwss隧道 |

---

### 5.3 带宽限速实现

使用令牌桶（Token Bucket）算法：
- 每个规则独立的令牌桶
- goroutine 安全的速率限制器
- 支持突发流量（burst）配置

### 5.4 负载均衡策略

| 策略 | 说明 |
|------|------|
| roundrobin | 轮询，按顺序分配 |
| weighted | 加权轮询，按权重比例分配 |
| random | 随机选择 |
| leastconn | 最少连接数优先 |
| failover | 主备模式，主故障切换到备 |

---

## 6. 安全设计

### 6.1 认证安全
- JWT Access Token（有效期 2小时）+ Refresh Token（有效期 7天）
- bcrypt 密码哈希（cost=12）
- API Key 用于程序化访问

### 6.2 传输安全
- HTTPS/TLS 1.3（通过 Nginx 终止）
- WebSocket 升级走 HTTPS
- Xray-core VLESS+Reality：海外直连入站伪装，防止 GFW 主动探测
- gost mwss：节点间跨境隧道加密，防止造路盗听

### 6.3 SSH 密钥安全
- SSH 私钥使用 AES-256-GCM 加密存储
- 加密密钥从环境变量获取

### 6.4 访问控制
- RBAC 三级权限：super_admin > admin > user
- 管理员只能管理自己创建的子用户
- 普通用户只能查看/管理被分配的规则

---

## 7. 部署架构

### 单机部署（推荐入门）

```
Internet → Nginx(:443) → FolstingX Backend(:8080)
                      → 静态前端文件
```

### 高可用部署

```
Internet → CDN → 多台 Nginx (Load Balancer)
                    ↓
              FolstingX 集群（共享 PostgreSQL）
```

---

## 8. 目录结构

```
FolstingX/
├── README.md                  # 项目说明
├── TECHNICAL_DOCS.md          # 本技术文档
├── DEVELOPMENT_STEPS.md       # AI 分步开发指南
├── API.md                     # API 详细文档
├── DEPLOYMENT.md              # 部署文档
│
├── backend/                   # Go 后端
│   ├── cmd/
│   │   └── server/
│   │       └── main.go        # 程序入口
│   ├── internal/
│   │   ├── api/               # HTTP 路由和处理器
│   │   │   ├── auth.go
│   │   │   ├── users.go
│   │   │   ├── nodes.go
│   │   │   ├── rules.go
│   │   │   ├── monitor.go
│   │   │   └── logs.go
│   │   ├── models/            # 数据模型
│   │   │   ├── user.go
│   │   │   ├── node.go
│   │   │   ├── rule.go
│   │   │   └── stats.go
│   │   ├── services/          # 业务逻辑
│   │   │   ├── auth_service.go
│   │   │   ├── forward_manager.go  # 核心转发引擎
│   │   │   ├── node_checker.go     # 节点健康检查
│   │   │   ├── traffic_collector.go # 流量采集
│   │   │   └── xray_manager.go     # Xray-core 管理
│   │   ├── middleware/
│   │   │   ├── auth.go        # JWT 认证中间件
│   │   │   ├── rbac.go        # 权限检查
│   │   │   └── ratelimit.go   # 速率限制
│   │   └── database/
│   │       ├── db.go          # 数据库初始化
│   │       └── migrations/    # 数据库迁移
│   ├── pkg/
│   │   ├── forwarder/         # TCP/UDP 转发器
│   │   │   ├── tcp.go
│   │   │   ├── udp.go
│   │   │   └── ratelimit.go
│   │   ├── xray/              # Xray-core 封装
│   │   │   └── client.go
│   │   └── utils/
│   │       ├── jwt.go
│   │       ├── crypto.go
│   │       └── network.go
│   ├── config/
│   │   └── config.go          # 配置读取
│   ├── go.mod
│   └── go.sum
│
├── frontend/                  # Vue3 前端
│   ├── src/
│   │   ├── main.ts
│   │   ├── App.vue
│   │   ├── router/
│   │   │   └── index.ts
│   │   ├── stores/
│   │   │   ├── auth.ts
│   │   │   └── monitor.ts
│   │   ├── views/
│   │   │   ├── Dashboard.vue  # 仪表盘
│   │   │   ├── Rules.vue      # 转发规则管理
│   │   │   ├── Nodes.vue      # 节点管理
│   │   │   ├── Users.vue      # 用户管理
│   │   │   ├── Logs.vue       # 日志
│   │   │   ├── Settings.vue   # 设置
│   │   │   └── Login.vue      # 登录页
│   │   ├── components/
│   │   │   ├── RuleEditor.vue
│   │   │   ├── TrafficChart.vue
│   │   │   └── StatusBadge.vue
│   │   └── api/
│   │       └── index.ts       # API 客户端
│   ├── package.json
│   ├── vite.config.ts
│   └── tsconfig.json
│
├── scripts/
│   ├── install.sh             # 一键安装
│   ├── update.sh              # 一键更新
│   └── uninstall.sh           # 卸载
│
├── docker/
│   ├── Dockerfile
│   └── docker-compose.yml
│
└── config/
    └── config.example.yaml    # 配置文件示例
```
