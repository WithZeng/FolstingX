package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/folstingx/server/internal/database"
	"github.com/folstingx/server/internal/middleware"
	"github.com/folstingx/server/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type MonitorHub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]struct{}
}

func NewMonitorHub() *MonitorHub {
	return &MonitorHub{clients: map[*websocket.Conn]struct{}{}}
}

func (h *MonitorHub) Start() {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			h.broadcast(app.collector.Latest())
		}
	}()
}

func (h *MonitorHub) add(conn *websocket.Conn) {
	h.mu.Lock()
	h.clients[conn] = struct{}{}
	h.mu.Unlock()
}

func (h *MonitorHub) remove(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
	_ = conn.Close()
}

func (h *MonitorHub) broadcast(v interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		_ = c.SetWriteDeadline(time.Now().Add(3 * time.Second))
		if err := c.WriteJSON(v); err != nil {
			go h.remove(c)
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func MonitorWSHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	app.hub.add(conn)
	defer app.hub.remove(conn)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func RegisterMonitorRoutes(r *gin.RouterGroup) {
	monitor := r.Group("/monitor")
	monitor.Use(middleware.AuthMiddleware(app.cfg))
	{
		monitor.GET("/overview", monitorOverview)
		monitor.GET("/traffic", monitorTraffic)
	}
}

func monitorOverview(c *gin.Context) {
	snap := app.collector.Latest()
	c.JSON(http.StatusOK, gin.H{
		"total_up":       snap.TotalUp,
		"total_down":     snap.TotalDown,
		"active_rules":   snap.ActiveRules,
		"online_nodes":   snap.OnlineNodes,
		"connections":    snap.TotalConn,
		"cpu_percent":    snap.CPUPercent,
		"memory_percent": snap.MemPercent,
	})
}

func monitorTraffic(c *gin.Context) {
	period := c.DefaultQuery("period", "day")
	var from time.Time
	switch period {
	case "week":
		from = time.Now().AddDate(0, 0, -7)
	case "month":
		from = time.Now().AddDate(0, -1, 0)
	default:
		from = time.Now().AddDate(0, 0, -1)
	}

	var stats []models.TrafficStat
	_ = database.DB.Where("created_at >= ?", from).Order("created_at ASC").Find(&stats).Error
	c.JSON(http.StatusOK, stats)
}
