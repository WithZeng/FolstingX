package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/folstingx/server/internal/database"
	"github.com/folstingx/server/internal/middleware"
	"github.com/folstingx/server/internal/models"
	"github.com/folstingx/server/pkg/utils"
	"github.com/gin-gonic/gin"
)

func RegisterUserRoutes(r *gin.RouterGroup) {
	users := r.Group("/users")
	users.Use(middleware.AuthMiddleware(app.cfg))
	{
		users.GET("", listUsers)
		users.POST("", createUser)
		users.PUT("/:id", updateUser)
		users.DELETE("/:id", deleteUser)
		users.POST("/:id/reset-traffic", resetTraffic)
	}
}

func listUsers(c *gin.Context) {
	var users []models.User
	role := c.GetString("role")
	userID := c.GetUint("user_id")

	q := database.DB
	if role == string(models.RoleAdmin) {
		q = q.Where("role = ?", models.RoleUser)
	}
	if role == string(models.RoleUser) {
		q = q.Where("id = ?", userID)
	}

	if err := q.Order("id DESC").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

func createUser(c *gin.Context) {
	role := c.GetString("role")
	if role != string(models.RoleSuperAdmin) && role != string(models.RoleAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	var req struct {
		Username       string    `json:"username"`
		Password       string    `json:"password"`
		Role           string    `json:"role"`
		BandwidthLimit int64     `json:"bandwidth_limit"`
		TrafficLimit   int64     `json:"traffic_limit"`
		ExpireAt       time.Time `json:"expire_at"`
		IsActive       bool      `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	hash, _ := utils.HashPassword(req.Password)
	user := models.User{
		Username:       req.Username,
		PasswordHash:   hash,
		Role:           models.UserRole(req.Role),
		BandwidthLimit: req.BandwidthLimit,
		TrafficLimit:   req.TrafficLimit,
		ExpireAt:       req.ExpireAt,
		IsActive:       req.IsActive,
	}
	if user.Role == "" {
		user.Role = models.RoleUser
	}
	user.APIKey = newAPIKey()
	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, user)
}

func updateUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	user.ID = uint(id)
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}

func deleteUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := database.DB.Delete(&models.User{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func resetTraffic(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := database.DB.Model(&models.User{}).Where("id = ?", id).Update("traffic_used", 0).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "traffic reset"})
}

func newAPIKey() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return "fx_" + hex.EncodeToString(b)
}
