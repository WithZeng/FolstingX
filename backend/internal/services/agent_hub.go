package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/folstingx/server/internal/database"
	"github.com/folstingx/server/internal/models"
	"github.com/gorilla/websocket"
)

// ===================== AES 加解密 (参照 flux-panel AES 通信) =====================

func aesEncrypt(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func aesDecrypt(key []byte, cipherBase64 string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(cipherBase64)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	plaintext, err := aesGCM.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// deriveKey 将 secret 字符串转为 32 字节 AES 密钥
func deriveKey(secret string) []byte {
	key := make([]byte, 32)
	copy(key, []byte(secret))
	return key
}

// ===================== Agent Session =====================

// AgentSession 代表一个已连接的节点 Agent
type AgentSession struct {
	NodeID   uint
	NodeName string
	Secret   string
	Conn     *websocket.Conn
	mu       sync.Mutex
}

// SendCommand 向 Agent 发送加密命令
func (s *AgentSession) SendCommand(cmd AgentCommand) error {
	payload, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	encrypted, err := aesEncrypt(deriveKey(s.Secret), string(payload))
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Conn.WriteMessage(websocket.TextMessage, []byte(encrypted))
}

// AgentCommand 面板→Agent 的命令
type AgentCommand struct {
	Action string          `json:"action"` // add_service, delete_service, add_chain, status, etc.
	ID     string          `json:"id"`     // 请求 ID (用于匹配响应)
	Data   json.RawMessage `json:"data"`   // 命令载荷
}

// AgentReport Agent→面板 的上报
type AgentReport struct {
	Type    string          `json:"type"`    // heartbeat, response, traffic, error
	ID      string          `json:"id"`      // 响应 ID (匹配请求)
	NodeID  uint            `json:"node_id"`
	Data    json.RawMessage `json:"data"`
}

// ===================== Gost Service/Chain 数据结构 =====================

// GostServiceConfig 参照 flux-panel GostUtil.AddService
type GostServiceConfig struct {
	Name     string `json:"name"`
	Addr     string `json:"addr"`     // 监听地址 :port
	Handler  string `json:"handler"`  // tcp, udp, relay, rtcp, rudp
	Listener string `json:"listener"` // tcp, udp, rtcp, rudp, ws, wss
	Forwarder *GostForwarder `json:"forwarder,omitempty"`
	Chain    string `json:"chain,omitempty"` // chain 引用名
}

// GostForwarder 目标转发配置
type GostForwarder struct {
	Nodes []GostForwarderNode `json:"nodes"`
}

// GostForwarderNode 目标节点
type GostForwarderNode struct {
	Name string `json:"name"`
	Addr string `json:"addr"`
}

// GostChainConfig 参照 flux-panel GostUtil.AddChains
type GostChainConfig struct {
	Name string          `json:"name"`
	Hops []GostChainHop  `json:"hops"`
}

// GostChainHop 一跳
type GostChainHop struct {
	Name  string          `json:"name"`
	Nodes []GostHopNode   `json:"nodes"`
}

// GostHopNode 跳节点
type GostHopNode struct {
	Name      string `json:"name"`
	Addr      string `json:"addr"`
	Connector string `json:"connector"` // relay, http, socks5
	Dialer    string `json:"dialer"`    // ws, wss, tcp
}

// ===================== Agent Hub =====================

// AgentHub 管理所有连接的 Agent 节点
type AgentHub struct {
	mu       sync.RWMutex
	sessions map[uint]*AgentSession // nodeID → session

	// 请求响应管理
	pendingMu sync.RWMutex
	pending   map[string]chan AgentReport
}

func NewAgentHub() *AgentHub {
	return &AgentHub{
		sessions: make(map[uint]*AgentSession),
		pending:  make(map[string]chan AgentReport),
	}
}

// Register 注册节点 Agent WebSocket 连接
func (h *AgentHub) Register(nodeID uint, nodeName string, secret string, conn *websocket.Conn) *AgentSession {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 关闭旧连接
	if old, ok := h.sessions[nodeID]; ok {
		_ = old.Conn.Close()
	}

	session := &AgentSession{
		NodeID:   nodeID,
		NodeName: nodeName,
		Secret:   secret,
		Conn:     conn,
	}
	h.sessions[nodeID] = session

	// 更新数据库在线状态
	_ = database.DB.Model(&models.Node{}).Where("id = ?", nodeID).Updates(map[string]interface{}{
		"is_online":  true,
		"last_check": time.Now(),
	}).Error

	WriteSystemLog("info", "agent_hub", fmt.Sprintf("node %s (%d) connected", nodeName, nodeID))
	return session
}

// Unregister 注销节点连接
func (h *AgentHub) Unregister(nodeID uint) {
	h.mu.Lock()
	if s, ok := h.sessions[nodeID]; ok {
		_ = s.Conn.Close()
		delete(h.sessions, nodeID)
	}
	h.mu.Unlock()

	_ = database.DB.Model(&models.Node{}).Where("id = ?", nodeID).Update("is_online", false).Error
	WriteSystemLog("info", "agent_hub", fmt.Sprintf("node %d disconnected", nodeID))
}

// GetSession 获取节点会话
func (h *AgentHub) GetSession(nodeID uint) (*AgentSession, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	s, ok := h.sessions[nodeID]
	return s, ok
}

// IsOnline 节点是否在线
func (h *AgentHub) IsOnline(nodeID uint) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.sessions[nodeID]
	return ok
}

// OnlineCount 在线节点数
func (h *AgentHub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.sessions)
}

// OnlineNodeIDs 返回所有在线节点 ID
func (h *AgentHub) OnlineNodeIDs() []uint {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]uint, 0, len(h.sessions))
	for id := range h.sessions {
		ids = append(ids, id)
	}
	return ids
}

// SendToNode 向指定节点发送命令并等待响应
func (h *AgentHub) SendToNode(nodeID uint, cmd AgentCommand, timeout time.Duration) (*AgentReport, error) {
	session, ok := h.GetSession(nodeID)
	if !ok {
		return nil, fmt.Errorf("node %d not connected", nodeID)
	}

	// 注册 pending 响应通道
	ch := make(chan AgentReport, 1)
	h.pendingMu.Lock()
	h.pending[cmd.ID] = ch
	h.pendingMu.Unlock()

	defer func() {
		h.pendingMu.Lock()
		delete(h.pending, cmd.ID)
		h.pendingMu.Unlock()
	}()

	if err := session.SendCommand(cmd); err != nil {
		return nil, err
	}

	select {
	case report := <-ch:
		return &report, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for response from node %d", nodeID)
	}
}

// HandleReport 处理 Agent 上报 (由 WebSocket 读取循环调用)
func (h *AgentHub) HandleReport(nodeID uint, secret string, raw []byte) {
	// 解密消息
	plaintext, err := aesDecrypt(deriveKey(secret), string(raw))
	if err != nil {
		WriteSystemLog("warn", "agent_hub", fmt.Sprintf("decrypt failed from node %d: %v", nodeID, err))
		return
	}

	var report AgentReport
	if err := json.Unmarshal([]byte(plaintext), &report); err != nil {
		WriteSystemLog("warn", "agent_hub", fmt.Sprintf("unmarshal failed from node %d: %v", nodeID, err))
		return
	}
	report.NodeID = nodeID

	switch report.Type {
	case "heartbeat":
		_ = database.DB.Model(&models.Node{}).Where("id = ?", nodeID).Updates(map[string]interface{}{
			"is_online":  true,
			"last_check": time.Now(),
			"latency_ms": 0,
		}).Error

	case "response":
		h.pendingMu.RLock()
		ch, ok := h.pending[report.ID]
		h.pendingMu.RUnlock()
		if ok {
			select {
			case ch <- report:
			default:
			}
		}

	case "traffic":
		// 流量上报 (可扩展)
		WriteSystemLog("debug", "agent_hub", fmt.Sprintf("traffic report from node %d", nodeID))

	case "error":
		WriteSystemLog("error", "agent_hub", fmt.Sprintf("error from node %d: %s", nodeID, string(report.Data)))
	}
}

// ===================== Gost 操作快捷方法 =====================

// AddGostService 在节点上添加 gost 转发服务
func (h *AgentHub) AddGostService(nodeID uint, svc GostServiceConfig) error {
	data, _ := json.Marshal(svc)
	cmd := AgentCommand{
		Action: "add_service",
		ID:     generateRequestID(),
		Data:   data,
	}
	resp, err := h.SendToNode(nodeID, cmd, 10*time.Second)
	if err != nil {
		return err
	}
	if resp.Type == "error" {
		return fmt.Errorf("add_service failed: %s", string(resp.Data))
	}
	return nil
}

// DeleteGostService 在节点上删除 gost 转发服务
func (h *AgentHub) DeleteGostService(nodeID uint, serviceName string) error {
	data, _ := json.Marshal(map[string]string{"name": serviceName})
	cmd := AgentCommand{
		Action: "delete_service",
		ID:     generateRequestID(),
		Data:   data,
	}
	_, err := h.SendToNode(nodeID, cmd, 10*time.Second)
	return err
}

// AddGostChain 在节点上添加 gost chain
func (h *AgentHub) AddGostChain(nodeID uint, chain GostChainConfig) error {
	data, _ := json.Marshal(chain)
	cmd := AgentCommand{
		Action: "add_chain",
		ID:     generateRequestID(),
		Data:   data,
	}
	resp, err := h.SendToNode(nodeID, cmd, 10*time.Second)
	if err != nil {
		return err
	}
	if resp.Type == "error" {
		return fmt.Errorf("add_chain failed: %s", string(resp.Data))
	}
	return nil
}

func generateRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
