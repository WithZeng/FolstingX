package models

import "time"

type NodeType string

const (
	NodeTypeEntry NodeType = "entry"
	NodeTypeRelay NodeType = "relay"
	NodeTypeExit  NodeType = "exit"
)

type Node struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Host      string    `gorm:"size:255;not null" json:"host"`
	SSHPort   int       `gorm:"default:22" json:"ssh_port"`
	SSHUser   string    `gorm:"size:64;not null" json:"ssh_user"`
	SSHKey    string    `gorm:"type:text" json:"ssh_key"`
	Location  string    `gorm:"size:128" json:"location"`
	NodeType  NodeType  `gorm:"size:20;index" json:"node_type"`
	IsActive  bool      `gorm:"default:true;index" json:"is_active"`
	LastCheck time.Time `json:"last_check"`
	LatencyMS int64     `gorm:"default:-1" json:"latency_ms"`
	CreatedAt time.Time `json:"created_at"`
}

func (Node) TableName() string {
	return "nodes"
}
