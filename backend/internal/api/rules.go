package api

import (
  "crypto/rand"
  "errors"
  "encoding/hex"
  "encoding/json"
  "net/http"
  "strconv"
  "strings"
  "time"

  "github.com/folstingx/server/internal/database"
  "github.com/folstingx/server/internal/middleware"
  "github.com/folstingx/server/internal/models"
  "github.com/gin-gonic/gin"
)

func RegisterRuleRoutes(r *gin.RouterGroup) {
  rules := r.Group("/rules")
  rules.Use(middleware.AuthMiddleware(app.cfg))
  {
    rules.GET("", listRules)
    rules.POST("", createRule)
    rules.GET("/:id", getRule)
    rules.PUT("/:id", updateRule)
    rules.DELETE("/:id", deleteRule)
    rules.PUT("/:id/reload", reloadRule)
    rules.PUT("/:id/enable", enableRule)
    rules.PUT("/:id/disable", disableRule)
    rules.GET("/:id/stats", ruleStats)
    rules.GET("/:id/inbound", inboundPreview)

    rules.POST("/import", importRules)
    rules.POST("/import-text", importRulesText)
    rules.GET("/export", exportRules)
  }
}

func listRules(c *gin.Context) {
  var rules []models.ForwardRule
  if err := database.DB.Order("id DESC").Find(&rules).Error; err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, rules)
}

func createRule(c *gin.Context) {
  var rule models.ForwardRule
  if err := c.ShouldBindJSON(&rule); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
    return
  }
  normalizeRuleDefaults(&rule)

  if err := validateInboundRule(&rule); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }

  if err := database.DB.Create(&rule).Error; err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }
  if rule.IsActive {
    _ = app.forwarder.Start(rule)
  }
  _ = applyInbound(rule)
  c.JSON(http.StatusCreated, rule)
}

func getRule(c *gin.Context) {
  id, _ := strconv.Atoi(c.Param("id"))
  var rule models.ForwardRule
  if err := database.DB.First(&rule, id).Error; err != nil {
    c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
    return
  }
  c.JSON(http.StatusOK, rule)
}

func updateRule(c *gin.Context) {
  id, _ := strconv.Atoi(c.Param("id"))
  var existing models.ForwardRule
  if err := database.DB.First(&existing, id).Error; err != nil {
    c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
    return
  }

  if err := c.ShouldBindJSON(&existing); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
    return
  }
  existing.ID = uint(id)
  normalizeRuleDefaults(&existing)

  if err := validateInboundRule(&existing); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }

  if err := database.DB.Save(&existing).Error; err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }
  _ = app.forwarder.Reload(existing)
  _ = applyInbound(existing)
  c.JSON(http.StatusOK, existing)
}

func deleteRule(c *gin.Context) {
  id, _ := strconv.Atoi(c.Param("id"))
  _ = app.forwarder.Stop(uint(id))
  if err := database.DB.Delete(&models.ForwardRule{}, id).Error; err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func reloadRule(c *gin.Context) {
  id, _ := strconv.Atoi(c.Param("id"))
  var rule models.ForwardRule
  if err := database.DB.First(&rule, id).Error; err != nil {
    c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
    return
  }
  if err := app.forwarder.Reload(rule); err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"message": "reloaded"})
}

func enableRule(c *gin.Context) { updateRuleStatus(c, true) }
func disableRule(c *gin.Context) { updateRuleStatus(c, false) }

func updateRuleStatus(c *gin.Context, enabled bool) {
  id, _ := strconv.Atoi(c.Param("id"))
  var rule models.ForwardRule
  if err := database.DB.First(&rule, id).Error; err != nil {
    c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
    return
  }
  rule.IsActive = enabled
  _ = database.DB.Save(&rule).Error
  if enabled {
    _ = app.forwarder.Start(rule)
  } else {
    _ = app.forwarder.Stop(rule.ID)
  }
  c.JSON(http.StatusOK, rule)
}

func ruleStats(c *gin.Context) {
  id, _ := strconv.Atoi(c.Param("id"))
  stats := app.forwarder.Stats()
  if s, ok := stats[uint(id)]; ok {
    c.JSON(http.StatusOK, s)
    return
  }
  c.JSON(http.StatusOK, gin.H{"up_bytes": 0, "down_bytes": 0, "connections": 0})
}

func inboundPreview(c *gin.Context) {
  id, _ := strconv.Atoi(c.Param("id"))
  var rule models.ForwardRule
  if err := database.DB.First(&rule, id).Error; err != nil {
    c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
    return
  }
  if !rule.InboundProxyEnabled {
    c.JSON(http.StatusOK, gin.H{"enabled": false})
    return
  }

  host := "0.0.0.0"
  if rule.ListenNodeID > 0 {
    var node models.Node
    if err := database.DB.First(&node, rule.ListenNodeID).Error; err == nil {
      host = node.Host
    }
  }

  if rule.InboundType == "vless_reality" {
    uuid := randomHex(16)
    shortID := randomHex(4)
    serverName := "www.cloudflare.com"
    link := "vless://" + uuid + "@" + host + ":" + strconv.Itoa(rule.ListenPort) +
      "?encryption=none&security=reality&sni=" + serverName + "&fp=chrome&pbk=" + randomHex(16) + "&sid=" + shortID + "#FolstingX"
    c.JSON(http.StatusOK, gin.H{
      "enabled":      true,
      "type":         "vless_reality",
      "share_link":   link,
      "listen_host":  host,
      "listen_port":  rule.ListenPort,
      "server_name":  serverName,
      "short_id":     shortID,
      "warning":      "preview only, use generated credentials from real xray runtime in production",
    })
    return
  }

  // shadowsocks preview
  password := randomHex(8)
  method := "aes-256-gcm"
  c.JSON(http.StatusOK, gin.H{
    "enabled":     true,
    "type":        "shadowsocks",
    "method":      method,
    "password":    password,
    "listen_host": host,
    "listen_port": rule.ListenPort,
    "uri":         "ss://" + method + ":" + password + "@" + host + ":" + strconv.Itoa(rule.ListenPort),
  })
}

func importRules(c *gin.Context) {
  file, err := c.FormFile("file")
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
    return
  }
  f, err := file.Open()
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "open file failed"})
    return
  }
  defer f.Close()

  var rules []models.ForwardRule
  if err := json.NewDecoder(f).Decode(&rules); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
    return
  }
  strategy := c.DefaultPostForm("conflict", "skip")
  imported := 0
  for _, r := range rules {
    normalizeRuleDefaults(&r)
    var exists models.ForwardRule
    err := database.DB.Where("name = ?", r.Name).First(&exists).Error
    if err == nil {
      switch strategy {
      case "overwrite":
        r.ID = exists.ID
        _ = database.DB.Save(&r).Error
      case "rename":
        r.Name = r.Name + "-" + time.Now().Format("150405")
        _ = database.DB.Create(&r).Error
      default:
        continue
      }
      imported++
      continue
    }
    _ = database.DB.Create(&r).Error
    imported++
  }
  c.JSON(http.StatusOK, gin.H{"imported": imported})
}

func exportRules(c *gin.Context) {
  idsRaw := c.Query("ids")
  var rules []models.ForwardRule
  q := database.DB
  if idsRaw != "" {
    q = q.Where("id IN ?", strings.Split(idsRaw, ","))
  }
  if err := q.Find(&rules).Error; err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }
  c.Header("Content-Disposition", "attachment; filename=rules.json")
  c.JSON(http.StatusOK, rules)
}

func importRulesText(c *gin.Context) {
  var rules []models.ForwardRule
  if err := c.ShouldBindJSON(&rules); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json: " + err.Error()})
    return
  }
  imported := 0
  for _, r := range rules {
    r.ID = 0
    normalizeRuleDefaults(&r)
    if database.DB.Create(&r).Error == nil {
      imported++
    }
  }
  c.JSON(http.StatusOK, gin.H{"imported": imported})
}

func normalizeRuleDefaults(rule *models.ForwardRule) {
  if rule.Protocol == "" {
    rule.Protocol = "tcp"
  }
  if rule.InboundProxyEnabled && rule.InboundType == "" {
    if rule.Mode == "direct" {
      rule.InboundType = "vless_reality"
    } else {
      rule.InboundType = "shadowsocks"
    }
  }
}

func validateInboundRule(rule *models.ForwardRule) error {
  if !rule.InboundProxyEnabled {
    return nil
  }

  if rule.Mode == "direct" && rule.InboundType != "vless_reality" {
    return errors.New("direct mode inbound must be vless_reality")
  }
  if rule.Mode != "direct" && rule.InboundType != "shadowsocks" {
    return errors.New("relay/ix/chain inbound must be shadowsocks")
  }

  if rule.ListenNodeID > 0 {
    var node models.Node
    if err := database.DB.First(&node, rule.ListenNodeID).Error; err != nil {
      return err
    }
    if !node.HasRole("entry") {
      return errors.New("listen node must have entry role")
    }
  }
  return nil
}

func applyInbound(rule models.ForwardRule) error {
  if !rule.InboundProxyEnabled {
    return nil
  }

  if rule.InboundType == "vless_reality" {
    if err := app.xray.EnsureBinary(); err != nil {
      return err
    }
    return app.xray.Reload()
  }

  if rule.InboundType == "shadowsocks" {
    if err := app.gost.EnsureBinary(); err != nil {
      return err
    }
    return app.gost.Reload()
  }
  return nil
}

func randomHex(n int) string {
  b := make([]byte, n)
  _, _ = rand.Read(b)
  return hex.EncodeToString(b)
}
