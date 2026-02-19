package main

import (
  "net/http"
  "strconv"

  "github.com/gin-gonic/gin"
  "github.com/folstingx/server/config"
  "github.com/folstingx/server/internal/api"
  "github.com/folstingx/server/internal/database"
  "github.com/folstingx/server/internal/middleware"
  "github.com/folstingx/server/internal/services"
)

func main() {
  cfg, err := config.LoadConfig("../config/config.yaml")
  if err != nil {
    cfg = config.DefaultConfig()
  }

  if err := database.Init(cfg); err != nil {
    panic(err)
  }

  fm := services.NewForwardManager()
  collector := services.NewTrafficCollector(fm)
  xrayMgr := services.NewXrayManager("./bin/xray")
  gostMgr := services.NewGostManager("./bin/gost")

  api.Init(cfg, fm, collector, xrayMgr, gostMgr)
  services.StartNodeChecker()
  services.StartLogRetentionJobs()
  _ = fm.StartAll()
  fm.StartPersistLoop()
  collector.Start()

  r := gin.New()
  r.Use(gin.Logger(), gin.Recovery(), corsMiddleware(), middleware.APIKeyMiddleware(), middleware.RateLimitMiddleware(), middleware.QuotaMiddleware())

  // 健康检查接口。
  r.GET("/health", func(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"status": "ok", "version": "1.0.1"})
  })
  r.GET("/swagger/index.html", func(c *gin.Context) {
    c.Header("Content-Type", "text/html; charset=utf-8")
    c.String(http.StatusOK, "<html><body><h2>FolstingX Swagger</h2><p>请在后续接入 swaggo 自动文档生成。</p></body></html>")
  })

  apiGroup := r.Group("/api/v1")
  {
    api.RegisterAuthRoutes(apiGroup, cfg)
    api.RegisterNodeRoutes(apiGroup)
    api.RegisterRuleRoutes(apiGroup)
    api.RegisterMonitorRoutes(apiGroup)
    api.RegisterUserRoutes(apiGroup)
    api.RegisterLogRoutes(apiGroup)
    apiGroup.GET("/ping", func(c *gin.Context) {
      c.JSON(http.StatusOK, gin.H{"message": "pong"})
    })
  }

  r.GET("/ws/monitor", api.MonitorWSHandler)

  addr := cfg.Server.Host + ":" + strconv.Itoa(cfg.Server.Port)
  _ = r.Run(addr)
}

func corsMiddleware() gin.HandlerFunc {
  return func(c *gin.Context) {
    c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
    c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
    c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
    c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
    if c.Request.Method == http.MethodOptions {
      c.AbortWithStatus(http.StatusNoContent)
      return
    }
    c.Next()
  }
}
