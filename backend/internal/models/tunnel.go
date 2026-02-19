package models

import "time"

// TunnelType 隧道类型
const (
	TunnelTypePortForward = 1 // 端口转发 (直连)
	TunnelTypeChainRelay  = 2 // 链式中转 (多跳)
)

// ChainType 链路节点角色
const (
	ChainTypeEntry = 1 // 入口节点
	ChainTypeRelay = 2 // 中继节点
	ChainTypeExit  = 3 // 出口节点
)

// Tunnel 隧道 —— 参照 flux-panel 架构，将转发链路抽象为隧道。
// Type=1 时只需一个入口节点和出口地址；Type=2 时需要完整链路(entry→relay→exit)。
type Tunnel struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"size:200;not null" json:"name"`
	Type         int       `gorm:"default:1;index" json:"type"`                   // 1=端口转发, 2=链式中转
	TrafficRatio float64   `gorm:"default:1.0" json:"traffic_ratio"`              // 流量倍率
	InboundIP    string    `gorm:"size:64" json:"inbound_ip"`                     // 入口IP限制(空=不限)
	IsActive     bool      `gorm:"default:true;index" json:"is_active"`           // 启用状态
	FlowIn       int64     `gorm:"default:0" json:"flow_in"`                      // 累计入站流量
	FlowOut      int64     `gorm:"default:0" json:"flow_out"`                     // 累计出站流量
	OwnerID      uint      `gorm:"index" json:"owner_id"`                         // 所属用户
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// 关联
	ChainTunnels []ChainTunnel `gorm:"foreignKey:TunnelID" json:"chain_tunnels,omitempty"`
	Forwards     []Forward     `gorm:"foreignKey:TunnelID" json:"forwards,omitempty"`
}

func (Tunnel) TableName() string { return "tunnels" }

// ChainTunnel 链路节点 —— 定义隧道中每一跳的配置。
// 参照 flux-panel ChainTunnel：通过 chain_type 确定节点在隧道中的角色。
type ChainTunnel struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TunnelID  uint      `gorm:"index;not null" json:"tunnel_id"`
	ChainType int       `gorm:"not null" json:"chain_type"`  // 1=entry, 2=relay, 3=exit
	NodeID    uint      `gorm:"index;not null" json:"node_id"`
	Port      int       `gorm:"default:0" json:"port"`       // 该节点上分配的端口
	Protocol  string    `gorm:"size:20;default:'relay'" json:"protocol"` // relay, wss, ws, tcp, udp, mws, mwss
	Strategy  string    `gorm:"size:30" json:"strategy"`     // LB策略(仅对多出口有意义)
	SortIndex int       `gorm:"default:0" json:"sort_index"` // 排序(inx)，决定链路顺序
	CreatedAt time.Time `json:"created_at"`

	// 关联
	Node Node `gorm:"foreignKey:NodeID" json:"node,omitempty"`
}

func (ChainTunnel) TableName() string { return "chain_tunnels" }

// Forward 转发规则 —— 用户在隧道上创建的具体转发端口映射。
// 参照 flux-panel Forward：每条 Forward 属于一个 Tunnel，绑定远程地址。
type Forward struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	TunnelID      uint      `gorm:"index;not null" json:"tunnel_id"`
	OwnerID       uint      `gorm:"index" json:"owner_id"`
	Name          string    `gorm:"size:200" json:"name"`
	RemoteAddress string    `gorm:"size:255" json:"remote_address"` // 远端地址 host:port
	Protocol      string    `gorm:"size:10;default:'tcp'" json:"protocol"` // tcp, udp, both
	Strategy      string    `gorm:"size:30" json:"strategy"`               // 负载均衡策略
	ListenPort    int       `gorm:"default:0" json:"listen_port"`          // 入口监听端口
	IsActive      bool      `gorm:"default:true" json:"is_active"`
	FlowIn        int64     `gorm:"default:0" json:"flow_in"`
	FlowOut       int64     `gorm:"default:0" json:"flow_out"`
	Connections   int64     `gorm:"default:0" json:"connections"`

	// 入站代理配置 (FolstingX 特有，flux-panel 无此功能)
	InboundEnabled bool   `gorm:"default:false" json:"inbound_enabled"`
	InboundType    string `gorm:"size:50" json:"inbound_type"`   // vless_reality, shadowsocks, trojan
	InboundConfig  string `gorm:"type:TEXT" json:"inbound_config"` // JSON: uuid, password, sni 等

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联
	Tunnel      Tunnel        `gorm:"foreignKey:TunnelID" json:"tunnel,omitempty"`
	ForwardPorts []ForwardPort `gorm:"foreignKey:ForwardID" json:"forward_ports,omitempty"`
}

func (Forward) TableName() string { return "forwards" }

// ForwardPort 端口分配 —— 记录每个转发在各节点上占用的端口。
// 参照 flux-panel ForwardPort。
type ForwardPort struct {
	ID        uint `gorm:"primaryKey" json:"id"`
	ForwardID uint `gorm:"index;not null" json:"forward_id"`
	NodeID    uint `gorm:"index;not null" json:"node_id"`
	Port      int  `gorm:"not null" json:"port"`

	Node Node `gorm:"foreignKey:NodeID" json:"node,omitempty"`
}

func (ForwardPort) TableName() string { return "forward_ports" }

// ===================== Helper Functions =====================

// ChainTypeName 返回链路类型的中文名
func ChainTypeName(ct int) string {
	switch ct {
	case ChainTypeEntry:
		return "入口"
	case ChainTypeRelay:
		return "中继"
	case ChainTypeExit:
		return "出口"
	default:
		return "未知"
	}
}

// TunnelTypeName 返回隧道类型名
func TunnelTypeName(tt int) string {
	switch tt {
	case TunnelTypePortForward:
		return "端口转发"
	case TunnelTypeChainRelay:
		return "链式中转"
	default:
		return "未知"
	}
}
