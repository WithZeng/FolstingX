package services

import (
  "fmt"
  "os"
  "path/filepath"
  "time"

  "github.com/folstingx/server/internal/database"
  "github.com/folstingx/server/internal/models"
)

func WriteSystemLog(level string, module string, message string) {
  _ = database.DB.Create(&models.SystemLog{Level: level, Module: module, Message: message}).Error

  // 文件日志按天分片，便于后续轮转清理。
  _ = os.MkdirAll("logs", 0o755)
  file := filepath.Join("logs", fmt.Sprintf("app-%s.log", time.Now().Format("2006-01-02")))
  line := fmt.Sprintf("%s [%s] [%s] %s\n", time.Now().Format(time.RFC3339), level, module, message)
  f, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
  if err == nil {
    _, _ = f.WriteString(line)
    _ = f.Close()
  }
}

func StartLogRetentionJobs() {
  go func() {
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()
    for range ticker.C {
      // DB 保留最近 7 天。
      _ = database.DB.Where("created_at < ?", time.Now().AddDate(0, 0, -7)).Delete(&models.SystemLog{}).Error

      // 文件保留最近 30 天。
      entries, err := os.ReadDir("logs")
      if err != nil {
        continue
      }
      cutoff := time.Now().AddDate(0, 0, -30)
      for _, e := range entries {
        if e.IsDir() {
          continue
        }
        info, err := e.Info()
        if err != nil {
          continue
        }
        if info.ModTime().Before(cutoff) {
          _ = os.Remove(filepath.Join("logs", e.Name()))
        }
      }
    }
  }()
}
