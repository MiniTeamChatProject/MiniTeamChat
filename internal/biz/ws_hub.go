package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WebSocket消息类型
const (
	MsgTypeAuth    = "auth"    // 认证消息
	MsgTypeMessage = "message" // 聊天消息
	MsgTypeError   = "error"   // 错误消息
	MsgTypeSystem  = "system"  // 系统消息（如心跳、认证成功）
)

// WSHub 管理所有WebSocket连接
type WSHub struct {
	rooms      map[string]*RoomHub
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	messageBiz MessageBiz
}

// RoomHub 管理单个房间的WebSocket连接
type RoomHub struct {
	roomID     string
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	done       chan struct{}
	hub        *WSHub
}

// Client 代表一个WebSocket客户端连接
type Client struct {
	hub    *RoomHub
	conn   *websocket.Conn
	send   chan []byte
	userID uuid.UUID
	roomID uuid.UUID
	ctx    context.Context
	cancel context.CancelFunc
}

// Message 代表WebSocket传输的消息
type Message struct {
	Type       string    `json:"type"`
	RoomID     string    `json:"room_id,omitempty"`
	SenderID   string    `json:"sender_id,omitempty"`
	SenderName string    `json:"sender_name,omitempty"`
	Content    string    `json:"content,omitempty"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
	Error      string    `json:"error,omitempty"`
}

// NewWSHub 创建新的WSHub实例
func NewWSHub(messageBiz MessageBiz) *WSHub {
	return &WSHub{
		rooms:      make(map[string]*RoomHub),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		messageBiz: messageBiz,
	}
}

// GetRoomHub 获取或创建房间的hub
func (h *WSHub) GetRoomHub(roomID uuid.UUID) *RoomHub {
	h.mu.RLock()
	roomHub, exists := h.rooms[roomID.String()]
	h.mu.RUnlock()

	if exists {
		return roomHub
	}

	// 如果不存在，创建新的RoomHub
	h.mu.Lock()
	defer h.mu.Unlock()

	// 双重检查
	if roomHub, exists = h.rooms[roomID.String()]; exists {
		return roomHub
	}

	roomHub = &RoomHub{
		roomID:     roomID.String(),
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		done:       make(chan struct{}),
		hub:        h,
	}

	h.rooms[roomID.String()] = roomHub
	go roomHub.run()

	return roomHub
}

// RemoveRoomHub 移除房间hub
func (h *WSHub) RemoveRoomHub(roomID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if roomHub, exists := h.rooms[roomID.String()]; exists {
		close(roomHub.done)
		delete(h.rooms, roomID.String())
	}
}

// BroadcastMessage 广播消息到指定房间
func (h *WSHub) BroadcastMessage(roomID uuid.UUID, message *Message) error {
	roomHub := h.GetRoomHub(roomID)
	if roomHub == nil {
		return errors.New("房间hub不存在")
	}

	msgBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	roomHub.broadcast <- msgBytes
	return nil
}

// run 启动WSHub的主循环
func (h *WSHub) run() {
	for {
		select {
		case client := <-h.register:
			// 新客户端注册
			h.handleRegister(client)
		case client := <-h.unregister:
			// 客户端注销
			h.handleUnregister(client)
		}
	}
}

// handleRegister 处理客户端注册
func (h *WSHub) handleRegister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	roomHub, exists := h.rooms[client.roomID.String()]
	if !exists {
		// 创建新的RoomHub
		roomHub = &RoomHub{
			roomID:     client.roomID.String(),
			clients:    make(map[*Client]bool),
			broadcast:  make(chan []byte, 256),
			register:   make(chan *Client),
			unregister: make(chan *Client),
			done:       make(chan struct{}),
			hub:        h,
		}
		h.rooms[client.roomID.String()] = roomHub
		go roomHub.run()
	}

	roomHub.register <- client
}

// handleUnregister 处理客户端注销
func (h *WSHub) handleUnregister(client *Client) {
	h.mu.RLock()
	roomHub, exists := h.rooms[client.roomID.String()]
	h.mu.RUnlock()

	if exists {
		roomHub.unregister <- client
	}
}

// run 启动RoomHub的主循环
func (r *RoomHub) run() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		r.cleanup()
	}()

	for {
		select {
		case <-r.done:
			return
		case client := <-r.register:
			r.clients[client] = true
			log.Printf("Client registered for room %s, total clients: %d", r.roomID, len(r.clients))
		case client := <-r.unregister:
			if _, ok := r.clients[client]; ok {
				delete(r.clients, client)
				close(client.send)
				log.Printf("Client unregistered for room %s, total clients: %d", r.roomID, len(r.clients))
			}
		case message := <-r.broadcast:
			r.broadcastToClients(message)
		case <-ticker.C:
			r.pingClients()
		}
	}
}

// broadcastToClients 广播消息到所有客户端
func (r *RoomHub) broadcastToClients(message []byte) {
	for client := range r.clients {
		select {
		case client.send <- message:
		default:
			// 客户端发送缓冲区满，移除客户端
			close(client.send)
			delete(r.clients, client)
		}
	}
}

// pingClients 向所有客户端发送ping消息
func (r *RoomHub) pingClients() {
	msg := Message{
		Type:      MsgTypeSystem,
		Content:   "ping",
		CreatedAt: time.Now(),
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling ping message: %v", err)
		return
	}

	r.broadcastToClients(msgBytes)
}

// cleanup 清理RoomHub资源
func (r *RoomHub) cleanup() {
	for client := range r.clients {
		close(client.send)
		client.conn.Close()
	}
	r.clients = make(map[*Client]bool)
	close(r.broadcast)
	close(r.register)
	close(r.unregister)
}

// NewClient 创建新的客户端连接
func NewClient(conn *websocket.Conn, userID uuid.UUID, roomID uuid.UUID) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
		roomID: roomID,
		ctx:    ctx,
		cancel: cancel,
	}
}

// readPump 读取WebSocket消息
func (c *Client) readPump(hub *WSHub) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		// 处理收到的消息
		go c.handleMessage(hub, message)
	}
}

// handleMessage 处理客户端消息
func (c *Client) handleMessage(hub *WSHub, message []byte) {
	var msg Message
	if err := json.Unmarshal(message, &msg); err != nil {
		c.sendError(fmt.Sprintf("无效的消息格式: %v", err))
		return
	}

	switch msg.Type {
	case MsgTypeAuth:
		// 认证消息已在握手阶段处理
		c.sendMessage(MsgTypeSystem, "已认证", nil)
	case MsgTypeMessage:
		c.handleChatMessage(hub, &msg)
	default:
		c.sendError(fmt.Sprintf("不支持的消息类型: %s", msg.Type))
	}
}

// handleChatMessage 处理聊天消息
func (c *Client) handleChatMessage(hub *WSHub, msg *Message) {
	// 保存消息到数据库
	message, err := hub.messageBiz.SendMessage(c.ctx, c.roomID, c.userID, msg.Content)
	if err != nil {
		c.sendError(fmt.Sprintf("发送消息失败: %v", err))
		return
	}

	// 构建广播消息
	broadcastMsg := Message{
		Type:       MsgTypeMessage,
		RoomID:     c.roomID.String(),
		SenderID:   c.userID.String(),
		SenderName: "用户", // 实际项目中需要从user-service获取用户名
		Content:    message.Content,
		CreatedAt:  message.CreatedAt,
	}

	// 广播消息
	if err := hub.BroadcastMessage(c.roomID, &broadcastMsg); err != nil {
		log.Printf("广播消息失败: %v", err)
	}
}

// sendMessage 发送消息到客户端
func (c *Client) sendMessage(msgType, content string, data interface{}) {
	msg := Message{
		Type:      msgType,
		Content:   content,
		CreatedAt: time.Now(),
	}

	if data != nil {
		// 根据需要扩展消息内容
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("序列化消息失败: %v", err)
		return
	}

	select {
	case c.send <- msgBytes:
	case <-time.After(5 * time.Second):
		log.Println("发送消息超时，关闭连接")
		c.conn.Close()
	}
}

// sendError 发送错误消息
func (c *Client) sendError(error string) {
	c.sendMessage(MsgTypeError, error, nil)
}

// writePump 写入WebSocket消息
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// 通道已关闭
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 将缓冲区中的消息也写入
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte("\n"))
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.ctx.Done():
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
	}
}
