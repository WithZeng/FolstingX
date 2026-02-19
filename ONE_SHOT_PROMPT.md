# FolstingX 一键开发 Prompt

> 将以下内容完整复制，一次性发送给 AI（如 Copilot / Claude / Cursor 等）。
> AI 将自动执行所有步骤，每步完成后自检修 bug，全部完成后输出验收报告。

---

## 使用方法

1. 打开 AI 开发工具（VS Code Copilot Chat / Cursor / Claude 等）
2. 将下方 `---START---` 到 `---END---` 之间的全部内容复制粘贴发送
3. 等待 AI 自动完成所有开发
4. 如果 AI 中途因上下文限制暂停，发送"继续"即可

---START---

你是 FolstingX 转发面板项目的全栈开发工程师。请按照以下指令，从零开始完成整个项目的开发。

## 核心规则（必须严格遵守）

1. **自动推进**：完成一个步骤后，立即开始下一个步骤，不要停下来问我。
2. **自检修 bug**：每个步骤完成后，你必须：
   - 检查代码是否有编译错误
   - 运行测试（如有）
   - 发现问题立即修复，修复后再次检查，直到无 bug
   - 输出简短的检查结果（✅ 通过 / 🔧 已修复 xxx）
3. **进度报告**：每完成一个 STEP，输出一行进度，例如：`[4/12] ✅ STEP 4 完成 - 核心转发引擎`
4. **不要跳步**：严格按顺序 STEP 1 → STEP 12 执行。
5. **不要问我**：遇到不确定的选择，按工程最佳实践自行决定。
6. **中文注释**：代码关键位置用中文注释。

---

## 项目概述

FolstingX 是一个高性能网络转发管理面板，核心功能：
- TCP/UDP 端口转发，支持热更新规则
- 两套独立工具：Xray-core（海外直连入站代理）+ gost（节点间加密隧道 mwss）
- 入站代理为**可选功能**，默认不开启
- 多用户权限管理（RBAC），流量限速，实时监控
- 前后端分离，支持 Linux VPS 一键部署

## 技术栈（固定，不可更改）

**后端**：Go 1.21+ / Gin / GORM / SQLite / JWT / WebSocket / zap日志
**前端**：Vue 3 / TypeScript / Vite / Naive UI / Pinia / ECharts / Axios
**外部工具**：Xray-core（仅海外直连入站）/ gost（节点间隧道）
**部署**：Docker / Systemd / Nginx

---

## STEP 1: 项目骨架

在当前工作区创建以下结构并初始化：

**后端 `backend/`**：
- `go mod init github.com/folstingx/server`
- 安装依赖：gin, gorm, sqlite-driver, jwt/v5, viper, zap, gorilla/websocket, gopsutil/v3, crypto
- 创建目录：cmd/server/, internal/api/, internal/models/, internal/services/, internal/middleware/, internal/database/, pkg/forwarder/, pkg/utils/, config/
- `cmd/server/main.go`：初始化 Gin，CORS 中间件，路由占位，监听 :8080
- `GET /health` → `{"status":"ok","version":"1.0.0"}`
- `config/config.go`：用 viper 读取 YAML 配置文件

**前端 `frontend/`**：
- Vite + Vue3 + TypeScript 初始化
- 安装依赖：vue-router@4, pinia, axios, naive-ui, echarts, vue-echarts
- 基础布局组件：侧边栏（Logo + 导航菜单）+ 顶部栏（用户信息）+ 主内容区
- 路由：/login, /dashboard, /rules, /nodes, /users, /logs, /settings
- Vite 代理 /api → http://localhost:8080, /ws → ws://localhost:8080
- 暗色主题（Naive UI darkTheme）

**配置文件**：
- `config/config.example.yaml`：完整注释的配置示例

自检：go build 成功 / 前端 npm run build 成功 / health 接口可访问

---

## STEP 2: 用户认证系统

**后端**：
- `internal/models/user.go`：User 模型（id, username, password_hash, role[super_admin/admin/user], api_key, bandwidth_limit, traffic_limit, traffic_used, is_active, expire_at, created_at, updated_at）
- `internal/database/db.go`：数据库初始化 + AutoMigrate + 默认管理员 admin/admin123
- `pkg/utils/jwt.go`：JWT 工具（AccessToken 2h / RefreshToken 7d）
- `pkg/utils/crypto.go`：bcrypt 密码哈希
- `internal/middleware/auth.go`：JWT 认证中间件
- `internal/middleware/rbac.go`：角色权限检查中间件
- API 接口：
  - POST /api/v1/auth/login → 返回 access_token + refresh_token
  - POST /api/v1/auth/refresh → 刷新 token
  - GET /api/v1/auth/profile → 当前用户信息（需JWT）
  - PUT /api/v1/auth/password → 修改密码（需JWT）

**前端**：
- `views/Login.vue`：登录表单（Naive UI），登录成功保存 token 到 localStorage，跳转 /dashboard
- `stores/auth.ts`：Pinia store 管理 token 和用户信息
- `api/index.ts`：Axios 实例，自动附加 Bearer token，401 自动跳转登录页
- 路由守卫：未登录重定向 /login

自检：admin/admin123 能登录 / token 刷新正常 / 无 token 返回 401 / 前端登录跳转正常

---

## STEP 3: 节点管理

**后端**：
- `internal/models/node.go`：Node 模型（id, name, host, ssh_port, ssh_user, ssh_key[加密存储], location, node_type[entry/relay/exit], is_active, last_check, latency_ms, created_at）
- `internal/api/nodes.go`：节点 CRUD（需管理员权限）
  - GET/POST /api/v1/nodes, GET/PUT/DELETE /api/v1/nodes/:id
  - POST /api/v1/nodes/:id/check → TCP ping 健康检查
- `internal/services/node_checker.go`：
  - TCP 连接测延迟，结果写 latency_ms + last_check
  - 后台定时任务每 60 秒检查所有活跃节点
  - 状态变化写 SystemLog

**前端 `views/Nodes.vue`**：
- 节点列表表格（名称、地址、类型、位置、延迟、状态）
- 状态颜色：绿 <100ms / 黄 100-300ms / 红 离线
- 添加/编辑节点弹窗（Modal + Form）
- 手动健康检查按钮、删除确认

自检：CRUD 全部正常 / 健康检查更新延迟 / 定时任务运行

---

## STEP 4: 核心转发引擎（TCP/UDP）

**后端 `pkg/forwarder/`**：
- `tcp.go`：TCP 转发器 - 监听端口→接受连接→双向 io.Copy→统计流量字节数
- `udp.go`：UDP 转发器
- `ratelimit.go`：令牌桶限速器（goroutine 安全）
- `forwarder.go`：统一接口 Forwarder { Start() / Stop() / Stats() }

**`internal/services/forward_manager.go`**：
- map[ruleID]Forwarder 管理所有活跃转发
- Start(rule) / Stop(ruleID) / Reload(rule) 热更新（优雅停止旧→启新）
- StartAll() 启动时加载所有 is_active 规则
- 流量统计每 5 秒写入数据库

**`internal/models/rule.go`**：ForwardRule 模型（id, name, mode[direct/relay/chain/ix], listen_node_id, listen_port, protocol[tcp/udp/both], inbound_proxy_enabled[默认false], inbound_type[vless_reality/shadowsocks], target_address, target_port, chain_nodes[JSON], lb_strategy, lb_targets[JSON], bandwidth_limit, is_active, traffic_up, traffic_down, connections, owner_id, created_at, updated_at）

**`internal/api/rules.go`**：
- CRUD + PUT 热更新 + enable/disable + stats
- POST /api/v1/rules/import, GET /api/v1/rules/export

自检：创建 TCP 规则能实际转发（用 curl 测试）/ 热更新生效 / 限速生效

---

## STEP 5: 入站代理 & 节点间隧道

**两套独立工具，各司其职：**

**工具一：Xray-core 管理器** `internal/services/xray_manager.go`
- 仅用于：海外直连 + 开启入站代理 → VLESS+Reality 入站
- 下载管理 xray-core 二进制
- 生成 Xray 配置：VLESS+Reality 入站 → 直连出站至目标
- 通过 Xray gRPC API 动态管理入站
- UUID/Reality keys 自动生成存数据库
- 前端显示 vless:// 分享链接和二维码

**工具二：gost 管理器** `internal/services/gost_manager.go`
- 用于：所有需要跨节点传输的链路
- 面板通过 SSH 向各节点下发 gost 二进制并启动
- 跨境节点（过墙）：gost mwss（WebSocket over TLS）传输
- 纯境外节点间：gost mws（无 TLS）
- inbound_type=shadowsocks 时：gost SS 入站 + mwss 出站
- 热更新：发送 SIGHUP 或重启 gost 进程

**入站代理规则：**
- 入站代理默认不开启，不开启时面板仅做端口转发
- 海外直连 + 开入站 → 必须 vless_reality（Xray-core）
- 国内中转/IX/链式 + 开入站 → 可选 shadowsocks（gost）

自检：不开入站的中转规则 gost mwss 隧道正常 / 海外直连 vless_reality Xray 配置正确 / 中转 SS 入站+mwss 正常

---

## STEP 6: 实时监控 & WebSocket

**`internal/services/traffic_collector.go`**：
- 每秒从 ForwardManager 采集所有规则流量和连接数
- 用 gopsutil 采集 CPU/内存/网络 IO
- 每 5 秒写入 traffic_stats 表（按天聚合）
- 维护最近 60 秒时间序列

**`internal/api/monitor.go`**：
- WS /ws/monitor：每秒推送实时数据（系统指标 + 各规则流量 + 总计）
- GET /api/v1/monitor/overview：总流量、活跃规则数、在线节点数
- GET /api/v1/monitor/traffic：历史图表数据（period=day/week/month）

**前端 `views/Dashboard.vue`**：
- 顶部统计卡片：总上行/下行、活跃规则、在线节点、当前连接
- ECharts 实时折线图（最近 60 秒流量）
- CPU/内存仪表盘
- 各规则流量排行表
- 全部通过 WebSocket 实时更新，断线 3 秒自动重连

自检：WebSocket 正常建立 / 图表每秒更新 / 断线重连正常

---

## STEP 7: 前端规则管理页

**`views/Rules.vue`**：
- 规则列表表格：名称、模式、端口、目标、协议、入站类型、状态（运行/停止/错误）、连接数、今日流量、操作按钮
- 搜索过滤
- 新建规则按钮

**`components/RuleEditor.vue`**：分步表单（Stepper）：
- 第1步 基本信息：规则名、转发模式（海外直连/中转/IX/链式）、入站代理开关
  - 海外直连开入站：自动锁定 vless_reality（Xray-core），配置伪装域名
  - 中转开入站：可选 shadowsocks（gost），配置 SS 密码+加密算法
- 第2步 入站配置：监听节点、端口、协议
- 第3步 出站配置：目标地址端口（链式可添加多中间节点拖拽排序、LB 多出口配置）
- 第4步 高级设置：带宽限速
- 第5步 确认保存，vless_reality 显示分享链接

自检：规则列表正确 / 分步表单逻辑通畅 / 保存后热加载生效 / 表单验证完整

---

## STEP 8: 用户管理 & 权限

**后端 `internal/api/users.go`**：
- 用户 CRUD（admin 以上权限）
- POST /api/v1/users/:id/reset-traffic：重置流量
- 权限控制：super_admin 管所有 / admin 管自己创建的 user / user 管被分配的规则
- 流量/过期检查中间件：超限返回 403
- API Key 认证：X-API-Key header

**前端 `views/Users.vue`**：
- 用户列表（角色徽章、流量进度条、过期时间、状态）
- 创建/编辑用户（角色、带宽限制、流量限制、过期时间）
- 重置流量按钮

自检：不同角色权限隔离正确 / 流量超限拒绝 / 过期用户无法登录 / API Key 认证正常

---

## STEP 9: 负载均衡 & 故障转移

**`pkg/forwarder/loadbalancer.go`**：
- 策略：RoundRobin / WeightedRoundRobin / Random / LeastConn / Failover
- LBTarget 结构：Address, Port, Weight, IsBackup, IsHealthy
- 每个目标独立健康检查（TCP 连接，超时 3 秒）
- 连续 3 次失败标记不健康，从轮询移除
- 每 30 秒重试不健康目标，恢复后重新加入
- 状态变化写 SystemLog + WebSocket 通知

**前端**：RuleEditor 出站步骤支持添加多个 LB 目标，选择策略

自检：轮询分配正确 / 目标下线自动移除 / 恢复后重新加入

---

## STEP 10: 日志系统 & 批量导入导出

**`internal/api/logs.go`**：
- GET /api/v1/logs（支持 level/module/rule_id/时间范围过滤分页）
- DELETE /api/v1/logs
- 日志双写：数据库（7 天）+ 文件 logs/app-YYYY-MM-DD.log（30 天轮转）

**批量导入导出**：
- GET /api/v1/rules/export?format=json&ids=1,2,3
- POST /api/v1/rules/import（multipart/form-data，支持 skip/overwrite/rename 冲突策略）

**前端**：
- `views/Logs.vue`：日志列表 + 筛选 + WebSocket 实时追尾 + 清空确认
- Rules 页：多选 checkbox + 批量导出按钮 + 导入按钮 + 导入预览弹窗

自检：日志过滤正常 / 导出 JSON 可重新导入 / 冲突处理正确

---

## STEP 11: API 文档 & 安全加固

**Swagger**：用 swaggo/swag 为所有接口添加注释，/swagger/index.html 可访问

**速率限制 `internal/middleware/ratelimit.go`**：
- 未认证：每 IP 每分钟 30 次
- 已认证：每用户每分钟 1000 次
- 登录：每 IP 每分钟 5 次
- 连续 5 次失败锁定 15 分钟

自检：Swagger 页面可访问 / 超限返回 429 / 账号锁定正常

---

## STEP 12: 自动化部署

**`scripts/install.sh`**：
- 检测发行版（Ubuntu/Debian/CentOS）
- 安装依赖 → 下载编译 → /opt/folstingx/ 安装 → 随机 JWT Secret →  Systemd 服务 → Nginx 反向代理 → 显示访问地址和默认密码

**`scripts/update.sh`**：备份 → 下载新版 → 停止 → 替换 → 迁移数据库 → 启动 → 失败回滚

**`docker/Dockerfile`**：多阶段构建（Go + Node 编译 → 最小运行镜像）
**`docker/docker-compose.yml`**：server + nginx + 数据卷

**`/etc/systemd/system/folstingx.service`**：自动重启、网络就绪后启动

自检：Docker 构建成功 / install.sh 语法检查通过 / Systemd 配置正确

---

## 完成后

所有 12 个 STEP 完成后，输出最终验收报告：

```
========== FolstingX 开发完成报告 ==========

[1/12] ✅ 项目骨架
[2/12] ✅ 用户认证
[3/12] ✅ 节点管理
[4/12] ✅ 转发引擎
[5/12] ✅ 入站代理 & 隧道
[6/12] ✅ 实时监控
[7/12] ✅ 规则管理页
[8/12] ✅ 用户管理
[9/12] ✅ 负载均衡
[10/12] ✅ 日志 & 导入导出
[11/12] ✅ API文档 & 安全
[12/12] ✅ 自动化部署

修复的 bug 数量：X
创建的文件数量：X
总代码行数：约 X 行

启动方式：
  后端：cd backend && go run cmd/server/main.go
  前端：cd frontend && npm run dev
  默认账号：admin / admin123
```

现在开始执行 STEP 1。

---END---
