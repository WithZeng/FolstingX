package services

import (
  "sync"
  "time"

  "github.com/folstingx/server/internal/database"
  "github.com/folstingx/server/internal/models"
  "github.com/shirou/gopsutil/v3/cpu"
  "github.com/shirou/gopsutil/v3/mem"
  "github.com/shirou/gopsutil/v3/net"
)

type MonitorSnapshot struct {
  Timestamp      int64                  `json:"timestamp"`
  CPUPercent     float64                `json:"cpu_percent"`
  MemPercent     float64                `json:"mem_percent"`
  NetIn          int64                  `json:"net_in"`
  NetOut         int64                  `json:"net_out"`
  TotalUp        int64                  `json:"total_up"`
  TotalDown      int64                  `json:"total_down"`
  TotalConn      int64                  `json:"total_conn"`
  RuleStats      map[uint]RuleLiveStats `json:"rule_stats"`
  ActiveRules    int                    `json:"active_rules"`
  OnlineNodes    int64                  `json:"online_nodes"`
}

type RuleLiveStats struct {
  UpBytes     int64 `json:"up_bytes"`
  DownBytes   int64 `json:"down_bytes"`
  Connections int64 `json:"connections"`
}

type TrafficCollector struct {
  fm      *ForwardManager
  mu      sync.RWMutex
  history []MonitorSnapshot
  latest  MonitorSnapshot
}

func NewTrafficCollector(fm *ForwardManager) *TrafficCollector {
  return &TrafficCollector{fm: fm, history: make([]MonitorSnapshot, 0, 60)}
}

func (t *TrafficCollector) Start() {
  go t.collectLoop()
}

func (t *TrafficCollector) collectLoop() {
  tick := time.NewTicker(1 * time.Second)
  flush := time.NewTicker(5 * time.Second)
  defer tick.Stop()
  defer flush.Stop()

  var prevIn, prevOut uint64
  for {
    select {
    case <-tick.C:
      stats := t.fm.Stats()
      snap := MonitorSnapshot{Timestamp: time.Now().Unix(), RuleStats: map[uint]RuleLiveStats{}}
      for id, s := range stats {
        snap.RuleStats[id] = RuleLiveStats{UpBytes: s.UpBytes, DownBytes: s.DownBytes, Connections: s.Connections}
        snap.TotalUp += s.UpBytes
        snap.TotalDown += s.DownBytes
        snap.TotalConn += s.Connections
      }
      snap.ActiveRules = len(stats)

      cpuP, _ := cpu.Percent(0, false)
      if len(cpuP) > 0 {
        snap.CPUPercent = cpuP[0]
      }
      vm, _ := mem.VirtualMemory()
      snap.MemPercent = vm.UsedPercent
      if ioStats, err := net.IOCounters(false); err == nil && len(ioStats) > 0 {
        if prevIn == 0 {
          prevIn = ioStats[0].BytesRecv
          prevOut = ioStats[0].BytesSent
        }
        snap.NetIn = int64(ioStats[0].BytesRecv - prevIn)
        snap.NetOut = int64(ioStats[0].BytesSent - prevOut)
        prevIn = ioStats[0].BytesRecv
        prevOut = ioStats[0].BytesSent
      }
      var online int64
      _ = database.DB.Model(&models.Node{}).Where("is_active = ? AND latency_ms >= 0", true).Count(&online).Error
      snap.OnlineNodes = online

      t.mu.Lock()
      t.latest = snap
      t.history = append(t.history, snap)
      if len(t.history) > 60 {
        t.history = t.history[len(t.history)-60:]
      }
      t.mu.Unlock()
    case <-flush.C:
      t.persist()
    }
  }
}

func (t *TrafficCollector) persist() {
  t.mu.RLock()
  snap := t.latest
  t.mu.RUnlock()
  if snap.Timestamp == 0 {
    return
  }
  date := time.Now().Format("2006-01-02")
  _ = database.DB.Create(&models.TrafficStat{
    RuleID:      0,
    Date:        date,
    TrafficUp:   snap.TotalUp,
    TrafficDown: snap.TotalDown,
    Connections: snap.TotalConn,
    CPUPercent:  snap.CPUPercent,
    MemPercent:  snap.MemPercent,
    NetIn:       snap.NetIn,
    NetOut:      snap.NetOut,
  }).Error
}

func (t *TrafficCollector) Latest() MonitorSnapshot {
  t.mu.RLock()
  defer t.mu.RUnlock()
  return t.latest
}

func (t *TrafficCollector) History() []MonitorSnapshot {
  t.mu.RLock()
  defer t.mu.RUnlock()
  cp := make([]MonitorSnapshot, len(t.history))
  copy(cp, t.history)
  return cp
}
