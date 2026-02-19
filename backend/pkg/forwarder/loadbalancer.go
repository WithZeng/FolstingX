package forwarder

import (
  "math/rand"
  "net"
  "strconv"
  "sync"
  "time"
)

type LBTarget struct {
  Address    string `json:"address"`
  Port       int    `json:"port"`
  Weight     int    `json:"weight"`
  IsBackup   bool   `json:"is_backup"`
  IsHealthy  bool   `json:"is_healthy"`
  failCount  int
  activeConn int
}

type LoadBalancer struct {
  mu       sync.Mutex
  targets  []*LBTarget
  strategy string
  rrIndex  int
}

func NewLoadBalancer(strategy string, targets []*LBTarget) *LoadBalancer {
  lb := &LoadBalancer{strategy: strategy, targets: targets}
  for _, t := range lb.targets {
    t.IsHealthy = true
    if t.Weight <= 0 {
      t.Weight = 1
    }
  }
  return lb
}

func (lb *LoadBalancer) Select() *LBTarget {
  lb.mu.Lock()
  defer lb.mu.Unlock()

  healthy := make([]*LBTarget, 0)
  for _, t := range lb.targets {
    if t.IsHealthy && !t.IsBackup {
      healthy = append(healthy, t)
    }
  }
  if len(healthy) == 0 {
    for _, t := range lb.targets {
      if t.IsHealthy {
        healthy = append(healthy, t)
      }
    }
  }
  if len(healthy) == 0 {
    return nil
  }

  switch lb.strategy {
  case "random":
    return healthy[rand.Intn(len(healthy))]
  case "least_conn":
    best := healthy[0]
    for _, t := range healthy[1:] {
      if t.activeConn < best.activeConn {
        best = t
      }
    }
    best.activeConn++
    return best
  case "weighted_round_robin":
    weighted := make([]*LBTarget, 0)
    for _, t := range healthy {
      for i := 0; i < t.Weight; i++ {
        weighted = append(weighted, t)
      }
    }
    if len(weighted) == 0 {
      return healthy[0]
    }
    lb.rrIndex = (lb.rrIndex + 1) % len(weighted)
    return weighted[lb.rrIndex]
  case "failover":
    return healthy[0]
  default:
    lb.rrIndex = (lb.rrIndex + 1) % len(healthy)
    return healthy[lb.rrIndex]
  }
}

func (lb *LoadBalancer) ReportResult(target *LBTarget, ok bool) {
  lb.mu.Lock()
  defer lb.mu.Unlock()
  if target == nil {
    return
  }
  if target.activeConn > 0 {
    target.activeConn--
  }
  if ok {
    target.failCount = 0
    target.IsHealthy = true
    return
  }
  target.failCount++
  if target.failCount >= 3 {
    target.IsHealthy = false
  }
}

func (lb *LoadBalancer) StartHealthCheck() {
  go func() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    for range ticker.C {
      lb.mu.Lock()
      targets := append([]*LBTarget{}, lb.targets...)
      lb.mu.Unlock()
      for _, t := range targets {
        addr := net.JoinHostPort(t.Address, strconv.Itoa(t.Port))
        conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
        lb.mu.Lock()
        if err != nil {
          t.failCount++
          if t.failCount >= 3 {
            t.IsHealthy = false
          }
        } else {
          _ = conn.Close()
          t.failCount = 0
          t.IsHealthy = true
        }
        lb.mu.Unlock()
      }
    }
  }()
}
