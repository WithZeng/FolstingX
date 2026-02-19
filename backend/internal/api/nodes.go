package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

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
		nodes.POST("/import", importNodes)
		nodes.GET("/export", exportNodes)
		nodes.POST("/import-text", importNodesText) // 纯文本批量导入
		nodes.GET("/export-text", exportNodesText)   // 纯文本批量导出
	}
}

func listNodes(c *gin.Context) {
	// 支持按角色过滤节点, 如 /nodes?role=entry
	role := c.Query("role")
	var nodes []models.Node
	q := database.DB.Order("id DESC")
	if role != "" {
		// SQLite JSON 搜索: roles 字段含有该角色
		q = q.Where("roles LIKE ?", "%\""+role+"\"%")
	}
	if err := q.Find(&nodes).Error; err != nil {
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
	if node.SSHUser == "" {
		node.SSHUser = "folstingx"
	}
	// 默认为三种角色都可以
	if len(node.Roles) == 0 {
		node.Roles = models.JSONList{"entry", "relay", "exit"}
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

// =====================
// 节点批量导入(JSON 文件 / JSON 文本) + 批量导出
// =====================

// importNodes 文件上传导入 (JSON)
func importNodes(c *gin.Context) {
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

	var nodes []models.Node
	if err := json.NewDecoder(f).Decode(&nodes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json: " + err.Error()})
		return
	}
	imported := doImportNodes(nodes)
	c.JSON(http.StatusOK, gin.H{"imported": imported})
}

// importNodesText 纯文本 JSON 导入 (POST body 就是 JSON 数组)
func importNodesText(c *gin.Context) {
	var nodes []models.Node
	if err := c.ShouldBindJSON(&nodes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json: " + err.Error()})
		return
	}
	imported := doImportNodes(nodes)
	c.JSON(http.StatusOK, gin.H{"imported": imported})
}

func doImportNodes(nodes []models.Node) int {
	imported := 0
	for _, n := range nodes {
		n.ID = 0
		if n.SSHPort == 0 {
			n.SSHPort = 22
		}
		if n.SSHUser == "" {
			n.SSHUser = "folstingx"
		}
		if len(n.Roles) == 0 {
			n.Roles = models.JSONList{"entry", "relay", "exit"}
		}
		// 跳过重复 host
		var exists models.Node
		if database.DB.Where("host = ?", n.Host).First(&exists).Error == nil {
			continue
		}
		if database.DB.Create(&n).Error == nil {
			imported++
		}
	}
	return imported
}

// exportNodes JSON 文件导出
func exportNodes(c *gin.Context) {
	idsRaw := c.Query("ids")
	var nodes []models.Node
	q := database.DB
	if idsRaw != "" {
		idParts := strings.Split(idsRaw, ",")
		q = q.Where("id IN ?", idParts)
	}
	if err := q.Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 清除敏感信息
	for i := range nodes {
		nodes[i].SSHKey = ""
	}
	format := c.DefaultQuery("format", "json")
	if format == "text" {
		// 纯文本格式: name|host|ssh_port|ssh_user|location|roles
		var lines []string
		for _, n := range nodes {
			roles := strings.Join([]string(n.Roles), ",")
			lines = append(lines, n.Name+"|"+n.Host+"|"+strconv.Itoa(n.SSHPort)+"|"+n.SSHUser+"|"+n.Location+"|"+roles)
		}
		c.Header("Content-Disposition", "attachment; filename=nodes.txt")
		c.String(http.StatusOK, strings.Join(lines, "\n"))
		return
	}
	c.Header("Content-Disposition", "attachment; filename=nodes.json")
	c.JSON(http.StatusOK, nodes)
}

// exportNodesText 纯文本格式导出 (适合复制粘贴)
func exportNodesText(c *gin.Context) {
	var nodes []models.Node
	if err := database.DB.Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var lines []string
	lines = append(lines, "# name|host|ssh_port|ssh_user|location|roles(逗号分隔)")
	for _, n := range nodes {
		roles := strings.Join([]string(n.Roles), ",")
		lines = append(lines, n.Name+"|"+n.Host+"|"+strconv.Itoa(n.SSHPort)+"|"+n.SSHUser+"|"+n.Location+"|"+roles)
	}
	c.String(http.StatusOK, strings.Join(lines, "\n"))
}
