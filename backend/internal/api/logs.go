package api

import (
	"net/http"
	"strconv"

	"github.com/folstingx/server/internal/database"
	"github.com/folstingx/server/internal/middleware"
	"github.com/folstingx/server/internal/models"
	"github.com/gin-gonic/gin"
)

func RegisterLogRoutes(r *gin.RouterGroup) {
	logs := r.Group("/logs")
	logs.Use(middleware.AuthMiddleware(app.cfg))
	{
		logs.GET("", getLogs)
		logs.DELETE("", clearLogs)
	}
}

func getLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}

	q := database.DB.Model(&models.SystemLog{})
	if level := c.Query("level"); level != "" {
		q = q.Where("level = ?", level)
	}
	if module := c.Query("module"); module != "" {
		q = q.Where("module = ?", module)
	}
	if start := c.Query("start"); start != "" {
		q = q.Where("created_at >= ?", start)
	}
	if end := c.Query("end"); end != "" {
		q = q.Where("created_at <= ?", end)
	}

	var total int64
	_ = q.Count(&total).Error
	var rows []models.SystemLog
	_ = q.Order("id DESC").Offset((page-1)*pageSize).Limit(pageSize).Find(&rows).Error
	c.JSON(http.StatusOK, gin.H{"items": rows, "total": total})
}

func clearLogs(c *gin.Context) {
	_ = database.DB.Where("1=1").Delete(&models.SystemLog{}).Error
	c.JSON(http.StatusOK, gin.H{"message": "cleared"})
}
