package api

import (
	"net/http"
	"time"

	"github.com/folstingx/server/config"
	"github.com/folstingx/server/internal/database"
	"github.com/folstingx/server/internal/middleware"
	"github.com/folstingx/server/internal/models"
	"github.com/folstingx/server/pkg/utils"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	cfg *config.Config
}

func RegisterAuthRoutes(r *gin.RouterGroup, cfg *config.Config) {
	h := &AuthHandler{cfg: cfg}
	auth := r.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.Refresh)
		auth.GET("/profile", middleware.AuthMiddleware(cfg), h.Profile)
		auth.PUT("/password", middleware.AuthMiddleware(cfg), h.ChangePassword)
	}
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	var user models.User
	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if !user.IsActive || (user.ExpireAt.Before(time.Now()) && !user.ExpireAt.IsZero()) {
		c.JSON(http.StatusForbidden, gin.H{"error": "user is inactive or expired"})
		return
	}
	if !utils.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	accessToken, err := utils.GenerateToken(h.cfg.Auth.JWTSecret, user.ID, user.Username, string(user.Role), utils.TokenAccess)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generate access token failed"})
		return
	}
	refreshToken, err := utils.GenerateToken(h.cfg.Auth.JWTSecret, user.ID, user.Username, string(user.Role), utils.TokenRefresh)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generate refresh token failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	claims, err := utils.ParseToken(h.cfg.Auth.JWTSecret, req.RefreshToken)
	if err != nil || claims.Type != utils.TokenRefresh {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	accessToken, err := utils.GenerateToken(h.cfg.Auth.JWTSecret, claims.UserID, claims.Username, claims.Role, utils.TokenAccess)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generate access token failed"})
		return
	}

	newRefreshToken, err := utils.GenerateToken(h.cfg.Auth.JWTSecret, claims.UserID, claims.Username, claims.Role, utils.TokenRefresh)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generate refresh token failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": accessToken, "refresh_token": newRefreshToken})
}

func (h *AuthHandler) Profile(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID, _ := userIDAny.(uint)

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":              user.ID,
		"username":        user.Username,
		"role":            user.Role,
		"api_key":         user.APIKey,
		"bandwidth_limit": user.BandwidthLimit,
		"traffic_limit":   user.TrafficLimit,
		"traffic_used":    user.TrafficUsed,
		"is_active":       user.IsActive,
		"expire_at":       user.ExpireAt,
	})
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID, _ := userIDAny.(uint)

	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if !utils.CheckPassword(req.OldPassword, user.PasswordHash) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "old password is incorrect"})
		return
	}

	hashed, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash password failed"})
		return
	}

	if err := database.DB.Model(&user).Update("password_hash", hashed).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update password failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password updated"})
}
