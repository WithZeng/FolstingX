package api

import (
	"net/http"
	"strconv"

	"github.com/folstingx/server/internal/database"
	"github.com/folstingx/server/internal/middleware"
	"github.com/folstingx/server/internal/models"
	"github.com/folstingx/server/internal/services"
	"github.com/gin-gonic/gin"
)

func RegisterNodeRoutes(r *gin.RouterGroup) {
	nodes := r.Group("/nodes")
	nodes.Use(middleware.AuthMiddleware(app.cfg), middleware.RequireRoles(string(models.RoleSuperAdmin), string(models.RoleAdmin)))
	{
		nodes.GET("", listNodes)
		nodes.POST("", createNode)
		nodes.GET("/:id", getNode)
		nodes.PUT("/:id", updateNode)
		nodes.DELETE("/:id", deleteNode)
		nodes.POST("/:id/check", checkNode)
	}
}

func listNodes(c *gin.Context) {
	var nodes []models.Node
	if err := database.DB.Order("id DESC").Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, nodes)
}

func createNode(c *gin.Context) {
	var node models.Node
	if err := c.ShouldBindJSON(&node); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if node.SSHPort == 0 {
		node.SSHPort = 22
	}
	if err := database.DB.Create(&node).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, node)
}

func getNode(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var node models.Node
	if err := database.DB.First(&node, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	c.JSON(http.StatusOK, node)
}

func updateNode(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var node models.Node
	if err := database.DB.First(&node, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}

	if err := c.ShouldBindJSON(&node); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	node.ID = uint(id)
	if err := database.DB.Save(&node).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, node)
}

func deleteNode(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := database.DB.Delete(&models.Node{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func checkNode(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var node models.Node
	if err := database.DB.First(&node, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}

	latency, err := services.CheckNode(&node)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"latency_ms": latency, "online": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"latency_ms": latency, "online": latency >= 0})
}
