package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/folstingx/server/internal/database"
	"github.com/folstingx/server/internal/middleware"
	"github.com/folstingx/server/internal/models"
	"github.com/folstingx/server/internal/services"
	"github.com/gin-gonic/gin"
)

func RegisterTunnelRoutes(r *gin.RouterGroup) {
	tunnels := r.Group("/tunnels")
	tunnels.Use(middleware.AuthMiddleware(app.cfg))
	{
		tunnels.GET("", listTunnels)
		tunnels.POST("", createTunnel)
		tunnels.GET("/:id", getTunnel)
		tunnels.PUT("/:id", updateTunnel)
		tunnels.DELETE("/:id", deleteTunnel)
		tunnels.PUT("/:id/toggle", toggleTunnel)

		// 链路节点管理
		tunnels.POST("/:id/chain", addChainNode)
		tunnels.DELETE("/:id/chain/:chain_id", removeChainNode)
		tunnels.PUT("/:id/chain/sort", sortChainNodes)

		// 转发管理
		tunnels.POST("/:id/forwards", createForward)
		tunnels.GET("/:id/forwards", listForwards)
		tunnels.PUT("/:id/forwards/:fwd_id", updateForward)
		tunnels.DELETE("/:id/forwards/:fwd_id", deleteForward)

		// 部署操作
		tunnels.POST("/:id/deploy", deployTunnel)
		tunnels.POST("/:id/undeploy", undeployTunnel)
	}
}

// ==================== Tunnel CRUD ====================

func listTunnels(c *gin.Context) {
	var tunnels []models.Tunnel
	q := database.DB.Order("id DESC").Preload("ChainTunnels").Preload("ChainTunnels.Node")
	if typeStr := c.Query("type"); typeStr != "" {
		q = q.Where("type = ?", typeStr)
	}
	if err := q.Find(&tunnels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tunnels)
}

func createTunnel(c *gin.Context) {
	var tunnel models.Tunnel
	if err := c.ShouldBindJSON(&tunnel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if tunnel.TrafficRatio <= 0 {
		tunnel.TrafficRatio = 1.0
	}
	if err := database.DB.Create(&tunnel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, tunnel)
}

func getTunnel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var tunnel models.Tunnel
	if err := database.DB.
		Preload("ChainTunnels").
		Preload("ChainTunnels.Node").
		Preload("Forwards").
		Preload("Forwards.ForwardPorts").
		First(&tunnel, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tunnel not found"})
		return
	}
	c.JSON(http.StatusOK, tunnel)
}

func updateTunnel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var tunnel models.Tunnel
	if err := database.DB.First(&tunnel, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tunnel not found"})
		return
	}

	var input struct {
		Name         string  `json:"name"`
		Type         int     `json:"type"`
		TrafficRatio float64 `json:"traffic_ratio"`
		InboundIP    string  `json:"inbound_ip"`
		IsActive     bool    `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	tunnel.Name = input.Name
	tunnel.Type = input.Type
	tunnel.TrafficRatio = input.TrafficRatio
	tunnel.InboundIP = input.InboundIP
	tunnel.IsActive = input.IsActive

	if err := database.DB.Save(&tunnel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tunnel)
}

func deleteTunnel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	// 先取消部署
	undeploy(uint(id))
	// 删除关联
	database.DB.Where("tunnel_id = ?", id).Delete(&models.ChainTunnel{})
	database.DB.Where("tunnel_id = ?", id).Delete(&models.Forward{})
	if err := database.DB.Delete(&models.Tunnel{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func toggleTunnel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var tunnel models.Tunnel
	if err := database.DB.First(&tunnel, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tunnel not found"})
		return
	}
	tunnel.IsActive = !tunnel.IsActive
	_ = database.DB.Save(&tunnel).Error
	if !tunnel.IsActive {
		undeploy(tunnel.ID)
	}
	c.JSON(http.StatusOK, tunnel)
}

// ==================== Chain Node Management ====================

func addChainNode(c *gin.Context) {
	tunnelID, _ := strconv.Atoi(c.Param("id"))
	var chain models.ChainTunnel
	if err := c.ShouldBindJSON(&chain); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	chain.TunnelID = uint(tunnelID)
	if chain.Protocol == "" {
		chain.Protocol = "relay"
	}

	// 验证节点存在
	var node models.Node
	if err := database.DB.First(&node, chain.NodeID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node not found"})
		return
	}

	if err := database.DB.Create(&chain).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	chain.Node = node
	c.JSON(http.StatusCreated, chain)
}

func removeChainNode(c *gin.Context) {
	chainID, _ := strconv.Atoi(c.Param("chain_id"))
	if err := database.DB.Delete(&models.ChainTunnel{}, chainID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "removed"})
}

func sortChainNodes(c *gin.Context) {
	tunnelID, _ := strconv.Atoi(c.Param("id"))
	var input []struct {
		ID        uint `json:"id"`
		SortIndex int  `json:"sort_index"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	for _, item := range input {
		database.DB.Model(&models.ChainTunnel{}).Where("id = ? AND tunnel_id = ?", item.ID, tunnelID).Update("sort_index", item.SortIndex)
	}
	c.JSON(http.StatusOK, gin.H{"message": "sorted"})
}

// ==================== Forward Management ====================

func createForward(c *gin.Context) {
	tunnelID, _ := strconv.Atoi(c.Param("id"))
	var fwd models.Forward
	if err := c.ShouldBindJSON(&fwd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	fwd.TunnelID = uint(tunnelID)
	if fwd.Protocol == "" {
		fwd.Protocol = "tcp"
	}

	if err := database.DB.Create(&fwd).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 如果开启入站代理，生成配置
	if fwd.InboundEnabled {
		fwd.InboundConfig = generateInboundConfig(fwd.InboundType, fwd.ListenPort)
		_ = database.DB.Save(&fwd).Error
	}

	c.JSON(http.StatusCreated, fwd)
}

func listForwards(c *gin.Context) {
	tunnelID, _ := strconv.Atoi(c.Param("id"))
	var forwards []models.Forward
	if err := database.DB.Where("tunnel_id = ?", tunnelID).Preload("ForwardPorts").Find(&forwards).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, forwards)
}

func updateForward(c *gin.Context) {
	fwdID, _ := strconv.Atoi(c.Param("fwd_id"))
	var fwd models.Forward
	if err := database.DB.First(&fwd, fwdID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "forward not found"})
		return
	}
	if err := c.ShouldBindJSON(&fwd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	fwd.ID = uint(fwdID)

	if fwd.InboundEnabled && fwd.InboundConfig == "" {
		fwd.InboundConfig = generateInboundConfig(fwd.InboundType, fwd.ListenPort)
	}

	if err := database.DB.Save(&fwd).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, fwd)
}

func deleteForward(c *gin.Context) {
	fwdID, _ := strconv.Atoi(c.Param("fwd_id"))
	database.DB.Where("forward_id = ?", fwdID).Delete(&models.ForwardPort{})
	if err := database.DB.Delete(&models.Forward{}, fwdID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ==================== Deploy / Undeploy ====================

// deployTunnel 将隧道配置下发到所有链路节点的 gost Agent
func deployTunnel(c *gin.Context) {
	tunnelID, _ := strconv.Atoi(c.Param("id"))
	var tunnel models.Tunnel
	if err := database.DB.
		Preload("ChainTunnels").
		Preload("ChainTunnels.Node").
		Preload("Forwards").
		First(&tunnel, tunnelID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tunnel not found"})
		return
	}

	errs := deployTunnelToNodes(tunnel)
	if len(errs) > 0 {
		errStrs := make([]string, len(errs))
		for i, e := range errs {
			errStrs[i] = e.Error()
		}
		c.JSON(http.StatusOK, gin.H{"message": "deployed with errors", "errors": errStrs})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deployed successfully"})
}

func undeployTunnel(c *gin.Context) {
	tunnelID, _ := strconv.Atoi(c.Param("id"))
	undeploy(uint(tunnelID))
	c.JSON(http.StatusOK, gin.H{"message": "undeployed"})
}

// ==================== 内部实现 ====================

func deployTunnelToNodes(tunnel models.Tunnel) []error {
	var errs []error
	chains := tunnel.ChainTunnels

	if tunnel.Type == models.TunnelTypePortForward {
		// 端口转发: 在入口节点添加 gost tcp/udp 服务直达目标
		for _, fwd := range tunnel.Forwards {
			entryNode := findChainByType(chains, models.ChainTypeEntry)
			if entryNode == nil {
				errs = append(errs, fmt.Errorf("no entry node for tunnel %d", tunnel.ID))
				continue
			}
			svcName := fmt.Sprintf("fwd_%d_%d", tunnel.ID, fwd.ID)
			svc := services.GostServiceConfig{
				Name:     svcName,
				Addr:     fmt.Sprintf(":%d", fwd.ListenPort),
				Handler:  "tcp",
				Listener: "tcp",
				Forwarder: &services.GostForwarder{
					Nodes: []services.GostForwarderNode{
						{Name: "target", Addr: fwd.RemoteAddress},
					},
				},
			}
			if err := app.agentHub.AddGostService(entryNode.NodeID, svc); err != nil {
				errs = append(errs, fmt.Errorf("deploy fwd %d to node %d: %v", fwd.ID, entryNode.NodeID, err))
			}
		}
	} else if tunnel.Type == models.TunnelTypeChainRelay {
		// 链式中转: 参照 flux-panel TunnelServiceImpl
		// entry → relay1 → relay2 → ... → exit
		for _, fwd := range tunnel.Forwards {
			entryNode := findChainByType(chains, models.ChainTypeEntry)
			exitNode := findChainByType(chains, models.ChainTypeExit)
			relayNodes := findChainsByType(chains, models.ChainTypeRelay)

			if entryNode == nil || exitNode == nil {
				errs = append(errs, fmt.Errorf("tunnel %d needs entry and exit nodes", tunnel.ID))
				continue
			}

			// 1. Exit 节点: 添加 relay 服务，监听端口转发到最终目标
			exitSvcName := fmt.Sprintf("chain_%d_%d_exit", tunnel.ID, fwd.ID)
			exitSvc := services.GostServiceConfig{
				Name:     exitSvcName,
				Addr:     fmt.Sprintf(":%d", exitNode.Port),
				Handler:  "relay",
				Listener: protocolToListener(exitNode.Protocol),
				Forwarder: &services.GostForwarder{
					Nodes: []services.GostForwarderNode{
						{Name: "target", Addr: fwd.RemoteAddress},
					},
				},
			}
			if err := app.agentHub.AddGostService(exitNode.NodeID, exitSvc); err != nil {
				errs = append(errs, err)
			}

			// 2. Relay 节点: 中继转发
			prevAddr := exitNode.Node.Host + ":" + strconv.Itoa(exitNode.Port)
			for i := len(relayNodes) - 1; i >= 0; i-- {
				relay := relayNodes[i]
				relaySvcName := fmt.Sprintf("chain_%d_%d_relay_%d", tunnel.ID, fwd.ID, relay.ID)
				relaySvc := services.GostServiceConfig{
					Name:     relaySvcName,
					Addr:     fmt.Sprintf(":%d", relay.Port),
					Handler:  "relay",
					Listener: protocolToListener(relay.Protocol),
					Forwarder: &services.GostForwarder{
						Nodes: []services.GostForwarderNode{
							{Name: "next", Addr: prevAddr},
						},
					},
				}
				if err := app.agentHub.AddGostService(relay.NodeID, relaySvc); err != nil {
					errs = append(errs, err)
				}
				prevAddr = relay.Node.Host + ":" + strconv.Itoa(relay.Port)
			}

			// 3. Entry 节点: 添加服务，chain 到下一跳
			entrySvcName := fmt.Sprintf("chain_%d_%d_entry", tunnel.ID, fwd.ID)
			chainName := fmt.Sprintf("chain_%d_%d", tunnel.ID, fwd.ID)

			// 添加 chain
			chain := services.GostChainConfig{
				Name: chainName,
				Hops: []services.GostChainHop{
					{
						Name: "hop0",
						Nodes: []services.GostHopNode{
							{
								Name:      "next",
								Addr:      prevAddr,
								Connector: "relay",
								Dialer:    protocolToDialer(entryNode.Protocol),
							},
						},
					},
				},
			}
			if err := app.agentHub.AddGostChain(entryNode.NodeID, chain); err != nil {
				errs = append(errs, err)
			}

			entrySvc := services.GostServiceConfig{
				Name:     entrySvcName,
				Addr:     fmt.Sprintf(":%d", fwd.ListenPort),
				Handler:  "tcp",
				Listener: "tcp",
				Chain:    chainName,
			}
			if err := app.agentHub.AddGostService(entryNode.NodeID, entrySvc); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errs
}

func undeploy(tunnelID uint) {
	var chains []models.ChainTunnel
	database.DB.Where("tunnel_id = ?", tunnelID).Preload("Node").Find(&chains)
	var forwards []models.Forward
	database.DB.Where("tunnel_id = ?", tunnelID).Find(&forwards)

	for _, chain := range chains {
		for _, fwd := range forwards {
			svcNames := []string{
				fmt.Sprintf("fwd_%d_%d", tunnelID, fwd.ID),
				fmt.Sprintf("chain_%d_%d_entry", tunnelID, fwd.ID),
				fmt.Sprintf("chain_%d_%d_exit", tunnelID, fwd.ID),
				fmt.Sprintf("chain_%d_%d_relay_%d", tunnelID, fwd.ID, chain.ID),
			}
			for _, name := range svcNames {
				_ = app.agentHub.DeleteGostService(chain.NodeID, name)
			}
		}
	}
}

func findChainByType(chains []models.ChainTunnel, ct int) *models.ChainTunnel {
	for i := range chains {
		if chains[i].ChainType == ct {
			return &chains[i]
		}
	}
	return nil
}

func findChainsByType(chains []models.ChainTunnel, ct int) []models.ChainTunnel {
	var result []models.ChainTunnel
	for _, c := range chains {
		if c.ChainType == ct {
			result = append(result, c)
		}
	}
	return result
}

func protocolToListener(proto string) string {
	switch proto {
	case "wss":
		return "wss"
	case "ws":
		return "ws"
	case "mwss":
		return "mwss"
	case "mws":
		return "mws"
	default:
		return "tcp"
	}
}

func protocolToDialer(proto string) string {
	return protocolToListener(proto) // 对称
}

func generateInboundConfig(inboundType string, listenPort int) string {
	hex := func(n int) string { return randomHex(n) }
	switch inboundType {
	case "vless_reality":
		cfg := map[string]interface{}{
			"uuid":        hex(16),
			"public_key":  hex(16),
			"short_id":    hex(4),
			"server_name": "www.cloudflare.com",
			"listen_port": listenPort,
		}
		b, _ := json.Marshal(cfg)
		return string(b)
	case "shadowsocks":
		cfg := map[string]interface{}{
			"method":      "aes-256-gcm",
			"password":    hex(8),
			"listen_port": listenPort,
		}
		b, _ := json.Marshal(cfg)
		return string(b)
	case "trojan":
		cfg := map[string]interface{}{
			"password":    hex(12),
			"listen_port": listenPort,
		}
		b, _ := json.Marshal(cfg)
		return string(b)
	default:
		return "{}"
	}
}
