package services

import (
  "net"
  "strconv"
  "time"

  "github.com/folstingx/server/internal/database"
  "github.com/folstingx/server/internal/models"
)

func CheckNode(node *models.Node) (int64, error) {
  address := net.JoinHostPort(node.Host, strconv.Itoa(node.SSHPort))
  start := time.Now()
  conn, err := net.DialTimeout("tcp", address, 3*time.Second)
  latency := int64(-1)
  if err == nil {
    latency = time.Since(start).Milliseconds()
    _ = conn.Close()
  }

  oldLatency := node.LatencyMS
  node.LastCheck = time.Now()
  node.LatencyMS = latency
  if saveErr := database.DB.Save(node).Error; saveErr != nil {
    return latency, saveErr
  }

  oldOnline := oldLatency >= 0
  newOnline := latency >= 0
  if oldOnline != newOnline {
    status := "offline"
    if newOnline {
      status = "online"
    }
    WriteSystemLog("warn", "node_checker", "node "+node.Name+"("+node.Host+") => "+status)
  }
  return latency, err
}

func StartNodeChecker() {
  ticker := time.NewTicker(60 * time.Second)
  go func() {
    defer ticker.Stop()
    for range ticker.C {
      var nodes []models.Node
      if err := database.DB.Where("is_active = ?", true).Find(&nodes).Error; err != nil {
        continue
      }
      for i := range nodes {
        _, _ = CheckNode(&nodes[i])
      }
    }
  }()
}
