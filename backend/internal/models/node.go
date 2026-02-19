package models

import "time"

// Node 表示一台机器，可同时承担 entry/relay/exit 任意组合角色。
type Node struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Host      string    `gorm:"size:255;not null" json:"host"`
	SSHPort   int       `gorm:"default:22" json:"ssh_port"`
	SSHUser   string    `gorm:"size:64;not null" json:"ssh_user"`
	SSHKey    string    `gorm:"type:text" json:"ssh_key,omitempty"`
	Location  string    `gorm:"size:128" json:"location"`
	Roles     JSONList  `gorm:"type:TEXT" json:"roles"` // 角色集合: entry, relay, exit
	IsActive  bool      `gorm:"default:true;index" json:"is_active"`
	LastCheck time.Time `json:"last_check"`
	LatencyMS int64     `gorm:"default:-1" json:"latency_ms"`
	CreatedAt time.Time `json:"created_at"`
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
