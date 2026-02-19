package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Node 表示一台机器，可同时承担 entry/relay/exit 任意组合角色。
// 参照 flux-panel 架构：节点通过 go-gost Agent 以 WebSocket 连接到面板。
type Node struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Host      string    `gorm:"size:255;not null" json:"host"`
	SSHPort   int       `gorm:"default:22" json:"ssh_port"`
	SSHUser   string    `gorm:"size:64;not null" json:"ssh_user"`
	SSHKey    string    `gorm:"type:text" json:"ssh_key,omitempty"`
	Location  string    `gorm:"size:128" json:"location"`
	Roles     JSONList  `gorm:"type:TEXT" json:"roles"` // 角色集合: entry, relay, exit

	// Agent 相关字段 (参照 flux-panel Node)
	Secret    string    `gorm:"size:64" json:"-"`             // Agent 认证密钥 (不暴露给前端)
	AgentPort int       `gorm:"default:8443" json:"agent_port"` // Agent 监听端口
	AgentVer  string    `gorm:"size:32" json:"agent_ver"`     // Agent 版本
	IsOnline  bool      `gorm:"default:false" json:"is_online"` // WebSocket 在线状态

	IsActive  bool      `gorm:"default:true;index" json:"is_active"`
	LastCheck time.Time `json:"last_check"`
	LatencyMS int64     `gorm:"default:-1" json:"latency_ms"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Node) TableName() string {
	return "nodes"
}

func (n *Node) HasRole(role string) bool {
	for _, r := range n.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// GenerateSecret 生成随机通信密钥，用于 Agent 认证
func (n *Node) GenerateSecret() {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	n.Secret = hex.EncodeToString(b)
}

// GetInstallCommand 生成该节点的安装命令（类似 flux-panel 的 getInstallCommand）
func (n *Node) GetInstallCommand(panelAddr string) string {
	return "curl -fsSL " + panelAddr + "/api/v1/node-agent/install.sh -o install.sh && chmod +x install.sh && bash install.sh -a " + panelAddr + " -s " + n.Secret
}
