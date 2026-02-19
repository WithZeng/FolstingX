# FolstingX AI 分步开发指南

> 将此文档发送给 AI 时，请每次只发送一个 STEP 的内容。
> AI 完成开发并通过检查后，再发送下一个 STEP。

---

## 使用方法

每个 STEP 包含：
1. **目标**：本步骤要实现什么
2. **Prompt**：发送给 AI 的指令（可直接复制）
3. **验证清单**：检查 AI 是否完成了所有要求

---

## STEP 1: 项目骨架初始化

### 目标
初始化 Go 后端项目结构和 Vue3 前端项目结构，建立基础配置。

### 发送给 AI 的 Prompt

```
你是一名资深 Go 和 Vue3 全栈工程师。

请为名为 FolstingX 的转发面板项目初始化项目骨架：

【后端 - Go】
1. 在 backend/ 目录下初始化 Go Modules（module 名: github.com/folstingx/server）
2. 安装以下依赖：
   - github.com/gin-gonic/gin（Web框架）
   - gorm.io/gorm + gorm.io/driver/sqlite（数据库/ORM）
   - github.com/golang-jwt/jwt/v5（JWT）
   - github.com/spf13/viper（配置文件）
   - go.uber.org/zap（日志）
   - github.com/gorilla/websocket（WebSocket）
   - golang.org/x/crypto（bcrypt）
3. 创建以下目录结构：
   backend/cmd/server/main.go
   backend/internal/api/
   backend/internal/models/
   backend/internal/services/
   backend/internal/middleware/
   backend/internal/database/
   backend/pkg/forwarder/
   backend/config/
4. 在 main.go 中初始化 Gin，监听 :8080，返回健康检查接口 GET /health -> {"status":"ok"}
5. 创建 config/config.example.yaml 配置文件示例

【前端 - Vue3】
1. 在 frontend/ 目录下用 Vite 初始化 Vue3 + TypeScript 项目
2. 安装以下依赖：
   - vue-router@4（路由）
   - pinia（状态管理）
   - axios（HTTP客户端）
   - @vicons/ionicons5（图标）
   - naive-ui（UI组件库）
   - echarts + vue-echarts（图表）
3. 创建基础路由：/login, /dashboard, /rules, /nodes, /users, /logs, /settings
4. 创建基础布局组件（侧边栏导航 + 顶部栏 + 内容区）
5. 前端开发时代理 /api 到 http://localhost:8080

完成后，请按以下清单自检：
- [ ] go.mod 文件存在且依赖正确
- [ ] backend 可以 go run 启动
- [ ] GET http://localhost:8080/health 返回 200
- [ ] frontend/package.json 存在且依赖正确
- [ ] npm run dev 可以启动前端
- [ ] 前端可以看到基础布局（侧边栏+顶部栏）
如有 bug 请自动修复后再报告完成。
```

### 验证清单
- [ ] `cd backend && go run cmd/server/main.go` 能正常启动
- [ ] `GET localhost:8080/health` 返回 `{"status":"ok"}`
- [ ] `cd frontend && npm run dev` 能正常启动
- [ ] 浏览器访问前端能看到基础布局

---

## STEP 2: 数据库建模 & 用户认证

### 目标
实现数据库模型定义、自动迁移、用户注册/登录、JWT 认证。

### 发送给 AI 的 Prompt

```
基于已有的 FolstingX 项目骨架，实现数据库建模和用户认证系统：

【数据库模型 - backend/internal/models/】
创建以下 GORM 模型（参考 TECHNICAL_DOCS.md 中的数据库设计）：
1. user.go - User 模型（id, username, password_hash, role, api_key, bandwidth_limit, traffic_limit, traffic_used, is_active, expire_at, created_at, updated_at）
   - role 为枚举：super_admin, admin, user
2. node.go - Node 模型
3. rule.go - ForwardRule 模型（包含 JSON 字段 chain_nodes, lb_targets）
4. stats.go - TrafficStat 和 SystemLog 模型
5. 数据库初始化时自动迁移所有模型
6. 种子数据：创建默认超级管理员账号 admin/admin123（首次运行时）

【认证系统 - backend/internal/】
1. pkg/utils/jwt.go - JWT 工具函数（生成/验证 AccessToken 和 RefreshToken）
   - AccessToken 有效期 2 小时
   - RefreshToken 有效期 7 天
2. pkg/utils/crypto.go - bcrypt 密码哈希和验证
3. middleware/auth.go - JWT 认证中间件
4. middleware/rbac.go - RBAC 权限检查中间件（RequireRole 函数）
5. 实现以下 API：
   POST /api/v1/auth/login    -> 返回 access_token 和 refresh_token
   POST /api/v1/auth/refresh  -> 用 refresh_token 换新 access_token
   POST /api/v1/auth/logout   -> 注销（客户端丢弃token，服务端可选黑名单）
   GET  /api/v1/auth/profile  -> 返回当前用户信息（需要 JWT）
   PUT  /api/v1/auth/password -> 修改密码（需要 JWT）

【前端登录页 - frontend/src/views/Login.vue】
1. 美观的登录表单（用户名+密码）
2. 使用 Naive UI 组件
3. 登录成功后保存 token 到 localStorage 并跳转到 /dashboard
4. 路由守卫：未登录自动重定向到 /login

完成后，请按以下清单自检：
- [ ] POST /api/v1/auth/login 用 admin/admin123 能返回 token
- [ ] GET /api/v1/auth/profile 带 Bearer token 能返回用户信息
- [ ] 无 token 访问受保护接口返回 401
- [ ] 前端登录页能正常显示并完成登录跳转
- [ ] 刷新页面后 token 不丢失，仍然保持登录状态
如有 bug 请自动修复。
```

---

## STEP 3: 节点管理 & 前端节点管理页

### 目标
实现节点的增删改查、SSH 连通性检测、前端节点管理界面。

### 发送给 AI 的 Prompt

```
基于已有的 FolstingX 项目，实现节点管理功能：

【后端 API - backend/internal/api/nodes.go】
实现以下节点管理接口（所有接口需要管理员权限）：
GET    /api/v1/nodes            # 节点列表（分页+搜索）
POST   /api/v1/nodes            # 添加节点
GET    /api/v1/nodes/:id        # 节点详情
PUT    /api/v1/nodes/:id        # 更新节点
DELETE /api/v1/nodes/:id        # 删除节点
POST   /api/v1/nodes/:id/check  # 触发健康检查（ping延迟测试）

【节点健康检查服务 - backend/internal/services/node_checker.go】
1. 实现 TCP ping 方式检测节点连通性（连接节点SSH端口，测量响应时间）
2. 健康检查结果写入数据库（latency_ms, last_check 字段）
3. 启动后台定时任务（每60秒）检查所有活跃节点
4. 节点状态变化时写入 SystemLog

【前端节点管理页 - frontend/src/views/Nodes.vue】
1. 节点列表表格（显示名称、地址、类型、位置、延迟、状态）
2. 添加/编辑节点表单（Modal弹窗）
3. 节点状态用颜色区分：绿色(在线<100ms)、黄色(延迟高100-300ms)、红色(离线)
4. 手动触发健康检查按钮
5. 删除确认对话框

完成后自检：
- [ ] 节点CRUD接口全部正常工作
- [ ] 健康检查能正确更新节点延迟
- [ ] 后台定时健康检查每60秒运行一次
- [ ] 前端页面能正常展示节点列表
- [ ] 添加节点表单验证正确
如有 bug 请自动修复。
```

---

## STEP 4: 核心转发引擎（TCP/UDP）

### 目标
实现 TCP/UDP 端口转发引擎，支持热更新规则。

### 发送给 AI 的 Prompt

```
基于已有的 FolstingX 项目，实现核心 TCP/UDP 转发引擎：

【TCP/UDP 转发器 - backend/pkg/forwarder/】
1. tcp.go - TCP 转发实现：
   - 监听本地端口
   - 接受连接后创建到目标的连接
   - 双向数据流转发（io.Copy）
   - 统计上行/下行流量字节数
   - 支持带宽限速（令牌桶算法）
2. udp.go - UDP 转发实现
3. ratelimit.go - 令牌桶速率限制器（线程安全）
4. forwarder.go - 统一接口：
   type Forwarder interface {
     Start() error
     Stop() error
     Stats() ForwardStats  // 返回当前流量和连接数
   }

【转发规则管理器 - backend/internal/services/forward_manager.go】
核心功能：
1. 维护所有活跃规则的 Forwarder 实例 map（map[ruleID]Forwarder）
2. Start(rule) - 启动规则转发
3. Stop(ruleID) - 停止规则转发
4. Reload(rule) - 热更新规则（停止旧实例 -> 等待存量连接结束 -> 启动新实例）
5. StartAll() - 程序启动时加载所有 is_active=true 的规则
6. GetStats(ruleID) - 获取规则实时统计

【转发规则 API - backend/internal/api/rules.go】
GET    /api/v1/rules
POST   /api/v1/rules
GET    /api/v1/rules/:id
PUT    /api/v1/rules/:id    # 调用 ForwardManager.Reload() 热更新
DELETE /api/v1/rules/:id
POST   /api/v1/rules/:id/enable
POST   /api/v1/rules/:id/disable
GET    /api/v1/rules/:id/stats  # 返回实时流量和连接数

重要设计要求：
- 热更新时不中断已有连接
- 每个规则独立 goroutine，互不影响
- 规则异常时自动记录日志，不影响其他规则
- 流量统计每5秒写入数据库

完成后自检：
- [ ] 创建 TCP 转发规则后能真正转发流量（用 curl 或 netcat 测试）
- [ ] 修改规则配置后热更新生效（不需要重启服务）
- [ ] 禁用规则后立即停止转发
- [ ] /api/v1/rules/:id/stats 能返回实时流量数据
- [ ] 带宽限速生效（配置限速后用测速工具验证）
如有 bug 请自动修复。
```

---

## STEP 5: 入站代理 & 节点间加密隧道

### 目标
实现两套独立系统：
1. Xray-core 管理器：仅用于海外直连模式的 VLESS+Reality 入站代理
2. gost 管理器：用于所有转发模式的节点间加密隧道

### 发送给 AI 的 Prompt

```
基于已有的 FolstingX 项目，实现入站代理和节点间加密隧道。

这里有两套完全独立的工具，各司其职：

【工具一：Xray-core 管理器】
仅用于海外直连模式 + 开启入站代理时，提供 VLESS+Reality 入站

实现 backend/internal/services/xray_manager.go：
1. 下载并管理 xray-core 二进制文件
2. 为 enable inbound_proxy + mode=direct 的规则生成 Xray 配置：
   - 入站：VLESS+Reality（监听指定端口）
   - 出站：直连至目标服务
3. 通过 Xray gRPC API 动态添加/删除入站配置
4. UUID、Reality keys 自动生成并存入数据库
5. 前端生成 vless:// 分享链接和二维码

【工具二：gost 管理器】
用于所有需要跨节点传输的场景，包括过墙段和出境段

实现 backend/internal/services/gost_manager.go：
1. 下载并管理 gost 二进制文件
2. 面板通过 SSH 向各节点自动下发 gost 二进制并启动
3. 为每条跨节点转发链路生成 gost 命令：
   - 跨境节点（过墙）：使用 mwss（WebSocket over TLS）传输
   - 纯境外节点间：可使用 mws（无 TLS）降低延迟
4. 当 inbound_type=shadowsocks 时，生成 gost SS 入站 + mwss 出站的组合配置
5. 节点配置变更时热更新 gost 进程（发送 SIGHUP 或重启进程）

【入站代理可选项设计】
数据库字段：
- inbound_proxy_enabled BOOLEAN 默认 false
- inbound_type ENUM(vless_reality, shadowsocks) 仅当开启时有效

对应关系：
- mode=direct + inbound_proxy_enabled=true → inbound_type 必须为 vless_reality ，工具为 Xray-core
- mode=relay/ix/chain + inbound_proxy_enabled=true → inbound_type 可为 shadowsocks，工具为 gost
- inbound_proxy_enabled=false → 不启动任何入站工具

【前端规则编辑器更新】
1. 添加“入站代理”开关（默认关）
2. 开启后根据转发模式显示：
   - 海外直连：自动决定为 vless_reality，要求填入伪装域名
   - 中转/IX/链式：可选 shadowsocks，填入 SS 密码和加密方式
3. vless_reality 模式创建后显示 vless:// 分享链接和二维码

完成后自检：
- [ ] 不开入站的转发规则，节点间 gost mwss 隧道正常工作
- [ ] 海外直连 + vless_reality 能生成正确 Xray 配置
- [ ] Xray 进程正常启动，vless:// 分享链接生成正确
- [ ] 中转 + shadowsocks 能生成正确 gost SS入站+mwss出站配置
- [ ] 修改规则时 Xray/gost 配置热更新
- [ ] 两种工具长期运行无内存泄漏
如有 bug 请自动修复。
```

---

## STEP 6: 实时监控 & WebSocket

### 目标
实现实时流量监控，通过 WebSocket 推送数据到前端仪表盘。

### 发送给 AI 的 Prompt

```
基于已有的 FolstingX 项目，实现实时监控系统：

【流量采集服务 - backend/internal/services/traffic_collector.go】
1. 每秒从 ForwardManager 采集所有规则的实时流量和连接数
2. 采集系统指标（CPU使用率、内存使用率、网络IO）使用 github.com/shirou/gopsutil/v3
3. 每5秒将数据写入 traffic_stats 表（按天聚合）
4. 维护最近60秒的时间序列数据（供WebSocket推送）

【WebSocket 服务 - backend/internal/api/monitor.go】
1. WS /ws/monitor - WebSocket 连接端点
   - 客户端连接后发送初始数据快照
   - 每秒推送实时数据：
     {
       "type": "realtime",
       "timestamp": 1708000000,
       "system": {"cpu": 15.2, "memory": 45.0, "network_in": 1024, "network_out": 2048},
       "rules": [{"id": 1, "upload": 512, "download": 1024, "connections": 5}],
       "total": {"upload": 10240, "download": 51200, "connections": 42}
     }
2. REST 接口：
   GET /api/v1/monitor/overview   # 系统概览（总流量、活跃规则数、在线节点数）
   GET /api/v1/monitor/traffic    # 历史流量图表数据（支持 ?period=day|week|month）

【前端仪表盘 - frontend/src/views/Dashboard.vue】
1. 顶部统计卡片：总流量上行/下行、活跃规则数、在线节点数、当前连接数
2. 实时流量折线图（最近60秒数据，用ECharts实现）
3. 系统资源使用率（CPU、内存仪表盘图）
4. 各规则流量排行榜（表格，按当日流量排序）
5. 所有数据通过 WebSocket 实时更新
6. 断线自动重连机制

完成后自检：
- [ ] WebSocket 连接正常建立
- [ ] 前端实时图表每秒更新
- [ ] CPU/内存数据准确
- [ ] 历史流量图表数据正确
- [ ] WebSocket 断线后3秒内自动重连
如有 bug 请自动修复。
```

---

## STEP 7: 前端规则管理页面

### 目标
实现完整的转发规则管理前端页面，支持所有转发模式配置。

### 发送给 AI 的 Prompt

```
基于已有的 FolstingX 项目，实现完整的转发规则管理前端页面：

【规则列表页 - frontend/src/views/Rules.vue】
1. 规则列表表格，显示：
   - 规则名称、转发模式、监听端口、目标节点/地址、协议
   - 入站类型（普通/VLESS+Reality，用图标区分）
   - 实时状态（运行中/已停止/错误）
   - 当前连接数、今日流量（上行/下行）
   - 操作按钮（编辑、启用/禁用、复制、删除）
2. 搜索和过滤功能
3. 顶部"新建规则"按钮

【规则编辑弹窗 - frontend/src/components/RuleEditor.vue】
分步表单（Stepper组件），引导用户配置规则：

第1步：基本信息
- 规则名称
- 转发模式选择：
  🌐 海外直连
  🔗 国内-海外中转
  ⚡ IX专线
  🔗 链式转发（多跳）
- 入站代理开关（默认关闭）
  - 海外直连开入站：自动锁定为 vless_reality（Xray-core），需配置伪装域名
  - 国内中转开入站：可选 shadowsocks（gost SS入站 + mwss隧道），需配置 SS 密码和加密算法

第2步：入站配置
- 选择监听节点（下拉选节点列表）
- 监听端口
- 协议（TCP/UDP/Both）
- 若选择了 vless_reality：显示伪装域名配置

第3步：出站配置
- 目标地址（IP/域名）
- 目标端口
- 若是链式转发：可添加多个中间节点（拖拽排序）
- 负载均衡配置（添加多个出站目标，选择LB策略）

第4步：高级设置
- 带宽限速（输入框，支持 MB/s 单位）
- 健康检查配置

第5步：确认 & 保存
- 显示规则配置摘要
- 保存后若是 vless_reality 模式：显示分享链接

完成后自检：
- [ ] 规则列表正确显示所有规则
- [ ] 规则状态（运行中/停止）实时更新
- [ ] 新建规则分步表单逻辑正确
- [ ] 保存规则后立即热加载生效
- [ ] 启用/禁用按钮即时生效
- [ ] 表单验证完整（端口范围、必填项等）
如有 bug 请自动修复。
```

---

## STEP 8: 用户管理 & 权限系统

### 目标
实现多用户管理功能，包括用户创建、权限控制、流量限制。

### 发送给 AI 的 Prompt

```
基于已有的 FolstingX 项目，实现用户管理和权限系统：

【后端 API - backend/internal/api/users.go】
GET    /api/v1/users              # 用户列表（super_admin/admin 可用）
POST   /api/v1/users              # 创建用户
GET    /api/v1/users/:id          # 用户详情
PUT    /api/v1/users/:id          # 更新用户
DELETE /api/v1/users/:id          # 删除用户
POST   /api/v1/users/:id/reset-traffic # 重置流量统计
GET    /api/v1/users/:id/rules    # 用户的规则列表
POST   /api/v1/users/:id/assign-rule/:rule_id # 将规则分配给用户

权限控制：
- super_admin 可以管理所有用户和规则
- admin 只能管理自己创建的普通用户
- user 只能查看/管理被分配的规则
- 流量/带宽限制在中间件层强制执行

【流量限制中间件】
1. 检查用户流量是否超出限制
2. 检查用户是否已过期
3. 超限时返回 403 并写日志

【API 密钥管理】
GET  /api/v1/auth/api-key          # 获取当前 API Key
POST /api/v1/auth/api-key/refresh  # 重新生成 API Key
支持用 API Key 代替 JWT 认证（在 Header 中传 X-API-Key）

【前端用户管理页 - frontend/src/views/Users.vue】
1. 用户列表（显示用户名、角色、流量使用、过期时间、状态）
2. 流量使用进度条
3. 创建/编辑用户表单（包含角色、带宽限制、流量限制、过期时间设置）
4. 一键重置流量按钮
5. 角色用不同颜色徽章区分

完成后自检：
- [ ] 不同角色的用户只能看到/操作权限范围内的数据
- [ ] 流量超限的用户无法继续使用转发
- [ ] 过期用户无法登录
- [ ] API Key 认证正常工作
- [ ] 前端用户列表流量显示准确
如有 bug 请自动修复。
```

---

## STEP 9: 负载均衡 & 故障转移

### 目标
实现多出口负载均衡和自动故障转移机制。

### 发送给 AI 的 Prompt

```
基于已有的 FolstingX 项目，实现负载均衡和故障转移：

【负载均衡器 - backend/pkg/forwarder/loadbalancer.go】
实现以下负载均衡策略：
1. RoundRobin - 轮询
2. WeightedRoundRobin - 加权轮询（根据 lb_targets 中的 weight 字段）
3. Random - 随机
4. LeastConn - 最少连接数优先
5. Failover - 主备模式（主节点故障切换到备节点）

每个目标配置：
type LBTarget struct {
    Address   string `json:"address"`
    Port      int    `json:"port"`
    Weight    int    `json:"weight"`
    IsBackup  bool   `json:"is_backup"`
    IsHealthy bool   // 健康检查结果（运行时字段）
}

【健康检查集成】
1. 每个 LB 目标独立健康检查（TCP 连接测试，超时3秒）
2. 连续3次失败标记为不健康，从轮询中移除
3. 每30秒检查一次不健康的目标，恢复后重新加入
4. 健康状态变化时写 SystemLog 并通过 WebSocket 通知前端

【前端更新 - RuleEditor.vue】
在出站配置步骤中：
1. 支持添加多个出站目标（IP+端口+权重）
2. 负载均衡策略下拉选择
3. 可标记某个目标为"备用节点"
4. 显示各目标健康状态

完成后自检：
- [ ] 轮询策略按顺序分配连接（可通过日志验证）
- [ ] 加权轮询按权重比例分配
- [ ] 某个目标下线后自动不再分配流量
- [ ] 目标恢复后自动重新加入轮询
- [ ] Failover 模式主节点恢复后自动切回
如有 bug 请自动修复。
```

---

## STEP 10: 日志系统 & 批量导入导出

### 目标
完善日志系统，实现规则批量导入导出功能。

### 发送给 AI 的 Prompt

```
基于已有的 FolstingX 项目，完善日志系统和批量操作功能：

【日志系统 - backend/internal/api/logs.go】
GET /api/v1/logs  # 日志查询接口
  参数：level(debug/info/warn/error), module, rule_id, start_time, end_time, page, page_size
DELETE /api/v1/logs  # 清理日志（可指定日期范围）

日志文件同时写入：
1. 数据库（近7天，供前端查询，自动清理旧日志）
2. 本地文件 logs/app-YYYY-MM-DD.log（按天轮转，保留30天）

【批量导入导出 - backend/internal/api/rules.go 补充】
GET /api/v1/rules/export  
  参数：format=json|csv, ids=1,2,3（可选，不传导出全部）
  返回：文件下载

POST /api/v1/rules/import
  body: multipart/form-data，上传 JSON 或 CSV 文件
  支持冲突处理策略：skip（跳过）/ overwrite（覆盖）/ rename（重命名）

导出格式（JSON）：
{
  "version": "1.0",
  "exported_at": "2024-02-19T00:00:00Z",
  "rules": [ {...rule对象...} ]
}

【前端日志页 - frontend/src/views/Logs.vue】
1. 日志列表（级别彩色标签、时间、模块、消息）
2. 筛选：时间范围 + 日志级别 + 关键词搜索
3. 支持实时追尾（类似 tail -f，通过WebSocket）
4. 清空日志按钮（需二次确认）

【前端批量操作】
在规则列表页添加：
1. 多选 checkbox
2. 批量导出按钮（下载 JSON 文件）
3. 导入规则按钮（上传 JSON 文件，预览后确认）
4. 导入预览弹窗（显示将导入的规则数量，冲突处理选项）

完成后自检：
- [ ] 日志接口支持多维度过滤
- [ ] 日志文件按天正确轮转
- [ ] 导出 JSON 格式正确可重新导入
- [ ] 导入时冲突处理逻辑正确
- [ ] 前端日志页实时追尾功能正常
如有 bug 请自动修复。
```

---

## STEP 11: API 文档 & 速率限制

### 目标
生成完整的 Swagger API 文档，添加 API 速率限制。

### 发送给 AI 的 Prompt

```
基于已有的 FolstingX 项目，完成 API 文档和速率限制：

【Swagger 文档 - 使用 swaggo/swag】
1. 安装 swaggo/swag 并为所有 API 接口添加注释
2. 访问 /swagger/index.html 可查看可交互的 API 文档
3. 文档中包含请求/响应示例
4. 支持 Bearer Token 认证测试

【速率限制 - backend/internal/middleware/ratelimit.go】
全局速率限制：
- 未认证请求：每IP每分钟 30 次
- 认证请求：每用户每分钟 1000 次
- 登录接口：每IP每分钟 5 次（防暴力破解）

登录失败锁定：
- 连续5次失败后锁定账号15分钟
- 写入 SystemLog

【Webhook 通知 - 可选功能】
在系统设置中可配置 Webhook URL
触发事件：
- 节点状态变化（上线/下线）
- 用户流量超限
- 规则异常

完成后自检：
- [ ] /swagger/index.html 能正常访问
- [ ] 所有接口在 Swagger 中有文档
- [ ] 超过速率限制返回 429
- [ ] 登录失败5次后返回账号锁定信息
如有 bug 请自动修复。
```

---

## STEP 12: 自动化部署 & 运维工具

### 目标
创建一键安装脚本、Docker 支持、Systemd 服务配置，完善运维文档。

### 发送给 AI 的 Prompt

```
基于已有的 FolstingX 项目，完成自动化部署支持：

【一键安装脚本 - scripts/install.sh】
功能：
1. 检测系统发行版（Ubuntu/Debian/CentOS/RHEL）
2. 安装必要依赖（Go、Node.js、Nginx、curl、git 等）
3. 从 GitHub Release 下载最新版本
4. 创建 /etc/folstingx/ 配置目录
5. 创建 /opt/folstingx/ 安装目录
6. 生成随机的 JWT Secret、数据库加密密钥
7. 配置 Systemd 服务并启动
8. 配置 Nginx 反向代理（含 HTTPS 自签名证书或 Let's Encrypt）
9. 显示安装完成信息（访问地址、默认账号密码）
10. 安全：首次安装后强制提示修改默认密码

【一键更新脚本 - scripts/update.sh】
1. 备份当前配置和数据库
2. 下载新版本
3. 停止服务 → 替换二进制 → 运行数据库迁移 → 启动服务
4. 更新失败自动回滚

【Docker 支持 - docker/】
1. Dockerfile：多阶段构建（builder阶段编译Go和前端，final阶段最小镜像）
2. docker-compose.yml：
   - folstingx-server 服务
   - nginx 服务
   - 持久化数据卷（数据库、日志、配置）
3. docker-compose.yml 中的环境变量通过 .env 文件配置

【Systemd 服务配置】
创建 /etc/systemd/system/folstingx.service：
- 自动重启策略（失败后3秒重启）
- 日志输出到 journald
- 网络就绪后启动

【配置文件 - config/config.example.yaml】
完整注释的配置文件示例，包含：
- 服务端口、调试模式
- 数据库路径/类型
- JWT Secret
- 日志配置
- 管理员初始账号密码

完成后自检：
- [ ] install.sh 在干净的 Ubuntu 22.04 上能一键安装成功
- [ ] update.sh 能正确完成更新且数据不丢失
- [ ] docker-compose up 能正常启动所有服务
- [ ] Systemd 服务开机自启动正常
- [ ] DEPLOYMENT.md 文档完整
如有 bug 请自动修复。
```

---

## 最终验收清单

完成所有 STEP 后，检查以下功能：

### 核心功能
- [ ] TCP 端口转发正常工作
- [ ] UDP 端口转发正常工作
- [ ] VLESS+Reality 加密转发正常
- [ ] 规则热更新不中断连接
- [ ] 负载均衡按策略正确分配流量
- [ ] 故障转移自动切换

### 管理功能
- [ ] 多用户权限系统正常
- [ ] 流量限制有效执行
- [ ] 实时监控数据准确
- [ ] 日志系统完整
- [ ] 批量导入/导出正常

### 运维功能
- [ ] 一键安装脚本成功
- [ ] Docker 部署成功
- [ ] 自动更新正常
- [ ] Nginx 反向代理配置正确

### 安全
- [ ] JWT 认证有效
- [ ] 登录暴力破解防护
- [ ] API 速率限制生效
- [ ] SSH 密钥加密存储
