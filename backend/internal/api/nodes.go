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
    nodes.POST("/import-text", importNodesText)
    nodes.GET("/export", exportNodes)
    nodes.GET("/export-text", exportNodesText)
  }
}

func listNodes(c *gin.Context) {
  role := c.Query("role")
  var nodes []models.Node
  q := database.DB.Order("id DESC")
  if role != "" {
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
  fillNodeDefaults(&node)

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
  fillNodeDefaults(&node)

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

  c.JSON(http.StatusOK, gin.H{"imported": doImportNodes(nodes)})
}

func importNodesText(c *gin.Context) {
  var nodes []models.Node
  if err := c.ShouldBindJSON(&nodes); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json: " + err.Error()})
    return
  }
  c.JSON(http.StatusOK, gin.H{"imported": doImportNodes(nodes)})
}

func doImportNodes(nodes []models.Node) int {
  imported := 0
  for _, n := range nodes {
    n.ID = 0
    fillNodeDefaults(&n)

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

func exportNodes(c *gin.Context) {
  idsRaw := c.Query("ids")
  var nodes []models.Node
  q := database.DB
  if idsRaw != "" {
    q = q.Where("id IN ?", strings.Split(idsRaw, ","))
  }
  if err := q.Find(&nodes).Error; err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }

  for i := range nodes {
    nodes[i].SSHKey = ""
  }

  if c.DefaultQuery("format", "json") == "text" {
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

func exportNodesText(c *gin.Context) {
  var nodes []models.Node
  if err := database.DB.Find(&nodes).Error; err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }

  var lines []string
  lines = append(lines, "# name|host|ssh_port|ssh_user|location|roles(comma-separated)")
  for _, n := range nodes {
    roles := strings.Join([]string(n.Roles), ",")
    lines = append(lines, n.Name+"|"+n.Host+"|"+strconv.Itoa(n.SSHPort)+"|"+n.SSHUser+"|"+n.Location+"|"+roles)
  }
  c.String(http.StatusOK, strings.Join(lines, "\n"))
}

func fillNodeDefaults(node *models.Node) {
  if node.SSHPort == 0 {
    node.SSHPort = 22
  }
  if node.SSHUser == "" {
    node.SSHUser = "folstingx"
  }
  if len(node.Roles) == 0 {
    node.Roles = models.JSONList{"entry", "relay", "exit"}
  }
}
