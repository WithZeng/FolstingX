package api

import (
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
		rules.POST("/import", importRules)
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
	if rule.Protocol == "" {
		rule.Protocol = "tcp"
	}
	if err := database.DB.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rule.IsActive {
		_ = app.forwarder.Start(rule)
	}
	applyInbound(rule)
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
	var rule models.ForwardRule
	if err := database.DB.First(&rule, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	rule.ID = uint(id)
	if err := database.DB.Save(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = app.forwarder.Reload(rule)
	applyInbound(rule)
	c.JSON(http.StatusOK, rule)
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
		idParts := strings.Split(idsRaw, ",")
		q = q.Where("id IN ?", idParts)
	}
	if err := q.Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Disposition", "attachment; filename=rules.json")
	c.JSON(http.StatusOK, rules)
}

func applyInbound(rule models.ForwardRule) {
	if !rule.InboundProxyEnabled {
		return
	}
	if rule.Mode == "direct" && rule.InboundType == "vless_reality" {
		_ = app.xray.EnsureBinary()
		_ = app.xray.Reload()
		return
	}
	if rule.InboundType == "shadowsocks" {
		_ = app.gost.EnsureBinary()
		_ = app.gost.Reload()
	}
}
