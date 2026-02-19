package services

import (
  "encoding/json"
  "fmt"
  "sync"
  "time"

  "github.com/folstingx/server/internal/database"
  "github.com/folstingx/server/internal/models"
  "github.com/folstingx/server/pkg/forwarder"
)

type ForwardManager struct {
  mu         sync.RWMutex
  forwarders map[uint]forwarder.Forwarder
  statsCache map[uint]forwarder.Stats
}

func NewForwardManager() *ForwardManager {
  return &ForwardManager{
    forwarders: make(map[uint]forwarder.Forwarder),
    statsCache: make(map[uint]forwarder.Stats),
  }
}

func (m *ForwardManager) buildForwarder(rule models.ForwardRule) (forwarder.Forwarder, error) {
  targetHost := rule.TargetAddress
  targetPort := rule.TargetPort

  if len(rule.LBTargets) > 0 {
    var lbTargets []*forwarder.LBTarget
    for _, item := range rule.LBTargets {
      var t forwarder.LBTarget
      if err := json.Unmarshal([]byte(item), &t); err == nil {
        lbTargets = append(lbTargets, &t)
      }
    }
    if len(lbTargets) > 0 {
      lb := forwarder.NewLoadBalancer(rule.LBStrategy, lbTargets)
      lb.StartHealthCheck()
      selected := lb.Select()
      if selected != nil {
        targetHost = selected.Address
        targetPort = selected.Port
      }
    }
  }

  switch rule.Protocol {
  case "udp":
    return forwarder.NewUDPForwarder("0.0.0.0", rule.ListenPort, targetHost, targetPort, rule.BandwidthLimit)
  case "both":
    return forwarder.NewTCPForwarder("0.0.0.0", rule.ListenPort, targetHost, targetPort, rule.BandwidthLimit), nil
  default:
    return forwarder.NewTCPForwarder("0.0.0.0", rule.ListenPort, targetHost, targetPort, rule.BandwidthLimit), nil
  }
}

func (m *ForwardManager) Start(rule models.ForwardRule) error {
  m.mu.Lock()
  defer m.mu.Unlock()
  if _, ok := m.forwarders[rule.ID]; ok {
    return fmt.Errorf("rule already started")
  }
  f, err := m.buildForwarder(rule)
  if err != nil {
    return err
  }
  if err := f.Start(); err != nil {
    return err
  }
  m.forwarders[rule.ID] = f
  return nil
}

func (m *ForwardManager) Stop(ruleID uint) error {
  m.mu.Lock()
  defer m.mu.Unlock()
  f, ok := m.forwarders[ruleID]
  if !ok {
    return nil
  }
  if err := f.Stop(); err != nil {
    return err
  }
  delete(m.forwarders, ruleID)
  return nil
}

func (m *ForwardManager) Reload(rule models.ForwardRule) error {
  _ = m.Stop(rule.ID)
  if !rule.IsActive {
    return nil
  }
  return m.Start(rule)
}

func (m *ForwardManager) StartAll() error {
  var rules []models.ForwardRule
  if err := database.DB.Where("is_active = ?", true).Find(&rules).Error; err != nil {
    return err
  }
  for _, r := range rules {
    _ = m.Start(r)
  }
  return nil
}

func (m *ForwardManager) Stats() map[uint]forwarder.Stats {
  m.mu.RLock()
  defer m.mu.RUnlock()
  out := make(map[uint]forwarder.Stats, len(m.forwarders))
  for id, f := range m.forwarders {
    out[id] = f.Stats()
  }
  return out
}

func (m *ForwardManager) StartPersistLoop() {
  ticker := time.NewTicker(5 * time.Second)
  go func() {
    defer ticker.Stop()
    for range ticker.C {
      stats := m.Stats()
      for ruleID, s := range stats {
        _ = database.DB.Model(&models.ForwardRule{}).Where("id = ?", ruleID).Updates(map[string]interface{}{
          "traffic_up":   s.UpBytes,
          "traffic_down": s.DownBytes,
          "connections":  s.Connections,
          "updated_at":   time.Now(),
        }).Error
      }
    }
  }()
}
