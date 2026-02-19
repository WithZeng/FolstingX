package models

import (
  "database/sql/driver"
  "encoding/json"
  "time"
)

type JSONList []string

func (j JSONList) Value() (driver.Value, error) {
  b, err := json.Marshal(j)
  if err != nil {
    return nil, err
  }
  return string(b), nil
}

func (j *JSONList) Scan(value interface{}) error {
  if value == nil {
    *j = JSONList{}
    return nil
  }
  switch v := value.(type) {
  case string:
    return json.Unmarshal([]byte(v), j)
  case []byte:
    return json.Unmarshal(v, j)
  default:
    *j = JSONList{}
    return nil
  }
}

type ForwardRule struct {
  ID                  uint      `gorm:"primaryKey" json:"id"`
  Name                string    `gorm:"size:100;not null" json:"name"`
  Mode                string    `gorm:"size:20;index" json:"mode"`
  ListenNodeID        uint      `json:"listen_node_id"`
  ListenPort          int       `gorm:"index" json:"listen_port"`
  Protocol            string    `gorm:"size:10" json:"protocol"`
  InboundProxyEnabled bool      `gorm:"default:false" json:"inbound_proxy_enabled"`
  InboundType         string    `gorm:"size:50" json:"inbound_type"`
  TargetAddress       string    `gorm:"size:255" json:"target_address"`
  TargetPort          int       `json:"target_port"`
  ChainNodes          JSONList  `gorm:"type:TEXT" json:"chain_nodes"`
  LBStrategy          string    `gorm:"size:30" json:"lb_strategy"`
  LBTargets           JSONList  `gorm:"type:TEXT" json:"lb_targets"`
  BandwidthLimit      int64     `gorm:"default:0" json:"bandwidth_limit"`
  IsActive            bool      `gorm:"default:true;index" json:"is_active"`
  TrafficUp           int64     `gorm:"default:0" json:"traffic_up"`
  TrafficDown         int64     `gorm:"default:0" json:"traffic_down"`
  Connections         int64     `gorm:"default:0" json:"connections"`
  OwnerID             uint      `gorm:"index" json:"owner_id"`
  CreatedAt           time.Time `json:"created_at"`
  UpdatedAt           time.Time `json:"updated_at"`
}

func (ForwardRule) TableName() string { return "forward_rules" }

type TrafficStat struct {
  ID          uint      `gorm:"primaryKey" json:"id"`
  RuleID      uint      `gorm:"index" json:"rule_id"`
  Date        string    `gorm:"size:20;index" json:"date"`
  TrafficUp   int64     `gorm:"default:0" json:"traffic_up"`
  TrafficDown int64     `gorm:"default:0" json:"traffic_down"`
  Connections int64     `gorm:"default:0" json:"connections"`
  CPUPercent  float64   `gorm:"default:0" json:"cpu_percent"`
  MemPercent  float64   `gorm:"default:0" json:"mem_percent"`
  NetIn       int64     `gorm:"default:0" json:"net_in"`
  NetOut      int64     `gorm:"default:0" json:"net_out"`
  CreatedAt   time.Time `json:"created_at"`
}

func (TrafficStat) TableName() string { return "traffic_stats" }
