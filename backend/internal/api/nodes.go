package api

import (
  "encoding/json"
  "net/http"
  "strconv"
  "strings"

  "github.com/folstingx/server/internal/database"
  "github.com/folstingx/server/internal/middleware"
  "github.com/folstingx/server/internal/models"
  "github.com/folstingx/server/internal/services"
  "github.com/gin-gonic/gin"
)

func RegisterNodeRoutes(r *gin.RouterGroup) {
  nodes := r.Group("/nodes")
  nodes.Use(middleware.AuthMiddleware(app.cfg), middleware.RequireRoles(string(models.RoleSuperAdmin), string(models.RoleAdmin)))
  {
    nodes.GET("", listNodes)
    nodes.POST("", createNode)
    nodes.GET("/:id", getNode)
    nodes.PUT("/:id", updateNode)
    nodes.DELETE("/:id", deleteNode)
    nodes.POST("/:id/check", checkNode)
    nodes.GET("/:id/install-command", getInstallCommand)
    nodes.POST("/:id/regenerate-secret", regenerateSecret)

    nodes.POST("/import", importNodes)
    nodes.POST("/import-text", importNodesText)
    nodes.GET("/export", exportNodes)
    nodes.GET("/export-text", exportNodesText)
  }

  // 公开端点: 节点 Agent 安装脚本下载
  agent := r.Group("/node-agent")
  {
    agent.GET("/install.sh", serveInstallScript)
  }
}

func listNodes(c *gin.Context) {
  role := c.Query("role")
  var nodes []models.Node
  q := database.DB.Order("id DESC")
  if role != "" {
    q = q.Where("roles LIKE ?", "%\""+role+"\"%")
  }
  if err := q.Find(&nodes).Error; err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, nodes)
}

func createNode(c *gin.Context) {
  var node models.Node
  if err := c.ShouldBindJSON(&node); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
    return
  }
  fillNodeDefaults(&node)
  node.GenerateSecret() // 自动生成 Agent 认证密钥

  if err := database.DB.Create(&node).Error; err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusCreated, node)
}

func getNode(c *gin.Context) {
  id, _ := strconv.Atoi(c.Param("id"))
  var node models.Node
  if err := database.DB.First(&node, id).Error; err != nil {
    c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
    return
  }
  c.JSON(http.StatusOK, node)
}

func updateNode(c *gin.Context) {
  id, _ := strconv.Atoi(c.Param("id"))
  var node models.Node
  if err := database.DB.First(&node, id).Error; err != nil {
    c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
    return
  }

  if err := c.ShouldBindJSON(&node); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
    return
  }
  node.ID = uint(id)
  fillNodeDefaults(&node)

  if err := database.DB.Save(&node).Error; err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, node)
}

func deleteNode(c *gin.Context) {
  id, _ := strconv.Atoi(c.Param("id"))
  if err := database.DB.Delete(&models.Node{}, id).Error; err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func checkNode(c *gin.Context) {
  id, _ := strconv.Atoi(c.Param("id"))
  var node models.Node
  if err := database.DB.First(&node, id).Error; err != nil {
    c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
    return
  }

  latency, err := services.CheckNode(&node)
  if err != nil {
    c.JSON(http.StatusOK, gin.H{"latency_ms": latency, "online": false, "error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"latency_ms": latency, "online": latency >= 0})
}

func importNodes(c *gin.Context) {
  file, err := c.FormFile("file")
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
    return
  }
  f, err := file.Open()
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "open file failed"})
    return
  }
  defer f.Close()

  var nodes []models.Node
  if err := json.NewDecoder(f).Decode(&nodes); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json: " + err.Error()})
    return
  }

  c.JSON(http.StatusOK, gin.H{"imported": doImportNodes(nodes)})
}

func importNodesText(c *gin.Context) {
  var nodes []models.Node
  if err := c.ShouldBindJSON(&nodes); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json: " + err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"imported": doImportNodes(nodes)})
}

func doImportNodes(nodes []models.Node) int {
  imported := 0
  for _, n := range nodes {
    n.ID = 0
    fillNodeDefaults(&n)

    var exists models.Node
    if database.DB.Where("host = ?", n.Host).First(&exists).Error == nil {
      continue
    }
    if database.DB.Create(&n).Error == nil {
      imported++
    }
  }
  return imported
}

func exportNodes(c *gin.Context) {
  idsRaw := c.Query("ids")
  var nodes []models.Node
  q := database.DB
  if idsRaw != "" {
    q = q.Where("id IN ?", strings.Split(idsRaw, ","))
  }
  if err := q.Find(&nodes).Error; err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }

  for i := range nodes {
    nodes[i].SSHKey = ""
  }

  if c.DefaultQuery("format", "json") == "text" {
    var lines []string
    for _, n := range nodes {
      roles := strings.Join([]string(n.Roles), ",")
      lines = append(lines, n.Name+"|"+n.Host+"|"+strconv.Itoa(n.SSHPort)+"|"+n.SSHUser+"|"+n.Location+"|"+roles)
    }
    c.Header("Content-Disposition", "attachment; filename=nodes.txt")
    c.String(http.StatusOK, strings.Join(lines, "\n"))
    return
  }

  c.Header("Content-Disposition", "attachment; filename=nodes.json")
  c.JSON(http.StatusOK, nodes)
}

func exportNodesText(c *gin.Context) {
  var nodes []models.Node
  if err := database.DB.Find(&nodes).Error; err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }

  var lines []string
  lines = append(lines, "# name|host|ssh_port|ssh_user|location|roles(comma-separated)")
  for _, n := range nodes {
    roles := strings.Join([]string(n.Roles), ",")
    lines = append(lines, n.Name+"|"+n.Host+"|"+strconv.Itoa(n.SSHPort)+"|"+n.SSHUser+"|"+n.Location+"|"+roles)
  }
  c.String(http.StatusOK, strings.Join(lines, "\n"))
}

func fillNodeDefaults(node *models.Node) {
  if node.SSHPort == 0 {
    node.SSHPort = 22
  }
  if node.SSHUser == "" {
    node.SSHUser = "folstingx"
  }
  if len(node.Roles) == 0 {
    node.Roles = models.JSONList{"entry", "relay", "exit"}
  }
  if node.AgentPort == 0 {
    node.AgentPort = 8443
  }
}

// ==================== 安装命令 & Agent ====================

// getInstallCommand 生成节点安装命令 (参照 flux-panel getInstallCommand)
func getInstallCommand(c *gin.Context) {
  id, _ := strconv.Atoi(c.Param("id"))
  var node models.Node
  if err := database.DB.First(&node, id).Error; err != nil {
    c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
    return
  }

  if node.Secret == "" {
    node.GenerateSecret()
    _ = database.DB.Save(&node).Error
  }

  // 面板地址: 从请求 Header 推断或用配置
  panelAddr := c.Request.Header.Get("X-Panel-Addr")
  if panelAddr == "" {
    scheme := "http"
    if c.Request.TLS != nil {
      scheme = "https"
    }
    panelAddr = scheme + "://" + c.Request.Host
  }

  installCmd := node.GetInstallCommand(panelAddr)

  c.JSON(http.StatusOK, gin.H{
    "install_command": installCmd,
    "panel_addr":      panelAddr,
    "secret":          node.Secret,
    "node_id":         node.ID,
  })
}

// regenerateSecret 重新生成节点密钥
func regenerateSecret(c *gin.Context) {
  id, _ := strconv.Atoi(c.Param("id"))
  var node models.Node
  if err := database.DB.First(&node, id).Error; err != nil {
    c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
    return
  }
  node.GenerateSecret()
  if err := database.DB.Save(&node).Error; err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"message": "secret regenerated", "secret": node.Secret})
}

// serveInstallScript 提供节点安装脚本下载
func serveInstallScript(c *gin.Context) {
  c.Header("Content-Type", "text/plain; charset=utf-8")
  c.Header("Content-Disposition", "attachment; filename=install.sh")
  c.String(200, nodeInstallScript)
}

// AgentWSHandler 节点 Agent WebSocket 连接入口
func AgentWSHandler(c *gin.Context) {
  secret := c.Query("secret")
  if secret == "" {
    c.JSON(401, gin.H{"error": "missing secret"})
    return
  }

  // 通过 secret 查找节点
  var node models.Node
  if err := database.DB.Where("secret = ?", secret).First(&node).Error; err != nil {
    c.JSON(401, gin.H{"error": "invalid secret"})
    return
  }

  conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
  if err != nil {
    return
  }

  session := app.agentHub.Register(node.ID, node.Name, node.Secret, conn)
  defer app.agentHub.Unregister(node.ID)

  for {
    _, msg, err := conn.ReadMessage()
    if err != nil {
      break
    }
    app.agentHub.HandleReport(session.NodeID, session.Secret, msg)
  }
}

// nodeInstallScript 节点 Agent 安装脚本 (参照 flux-panel install.sh)
const nodeInstallScript = `#!/usr/bin/env bash
set -euo pipefail

# FolstingX Node Agent 安装脚本
# 用法: bash install.sh -a <panel_addr> -s <secret>

PANEL_ADDR=""
SECRET=""
INSTALL_DIR="/etc/folstingx_agent"
SERVICE_NAME="folstingx-agent"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -a|--addr) PANEL_ADDR="$2"; shift 2 ;;
    -s|--secret) SECRET="$2"; shift 2 ;;
    -d|--dir) INSTALL_DIR="$2"; shift 2 ;;
    *) echo "未知参数: $1"; exit 1 ;;
  esac
done

if [[ -z "${PANEL_ADDR}" ]] || [[ -z "${SECRET}" ]]; then
  echo "用法: bash install.sh -a <panel_addr> -s <secret>"
  exit 1
fi

echo "=== FolstingX Node Agent 安装 ==="
echo "面板地址: ${PANEL_ADDR}"
echo "安装目录: ${INSTALL_DIR}"

# 创建目录
mkdir -p "${INSTALL_DIR}"

# 检测架构
ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64) ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  *) echo "不支持架构: ${ARCH}"; exit 1 ;;
esac

# 下载 gost (使用 go-gost v3)
GOST_URL="https://github.com/go-gost/gost/releases/latest/download/gost_linux_${ARCH}"
echo "下载 gost..."
curl -fsSL -o "${INSTALL_DIR}/gost" "${GOST_URL}" 2>/dev/null || {
  echo "gost 下载失败，创建占位文件"
  echo '#!/bin/sh' > "${INSTALL_DIR}/gost"
  echo 'echo "gost placeholder"' >> "${INSTALL_DIR}/gost"
}
chmod +x "${INSTALL_DIR}/gost"

# 写入配置
cat > "${INSTALL_DIR}/config.json" <<EOF
{
  "addr": "${PANEL_ADDR}",
  "secret": "${SECRET}"
}
EOF

# 空 gost 配置
echo '{}' > "${INSTALL_DIR}/gost.json"

# 创建 systemd 服务
cat > /etc/systemd/system/${SERVICE_NAME}.service <<EOF
[Unit]
Description=FolstingX Node Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/gost -C ${INSTALL_DIR}/gost.json
Restart=always
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable ${SERVICE_NAME}
systemctl restart ${SERVICE_NAME}

echo ""
echo "=== 安装完成 ==="
echo "服务状态: systemctl status ${SERVICE_NAME}"
echo "查看日志: journalctl -u ${SERVICE_NAME} -f"
`
