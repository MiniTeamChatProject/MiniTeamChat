package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Room 定义了聊天室的结构
// 对应数据库表: rooms
type Room struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name      string    `gorm:"size:255;not null" json:"name"`
	OwnerID   uuid.UUID `gorm:"type:uuid;not null" json:"owner_id"` // 谁创建的房间
	CreatedAt time.Time `json:"created_at"`
}

// Message 定义了聊天消息的结构 (Day 3 核心)
// 对应数据库表: messages
type Message struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	RoomID    uuid.UUID `gorm:"type:uuid;index" json:"room_id"`      // 属于哪个房间
	SenderID  uuid.UUID `gorm:"type:uuid;not null" json:"sender_id"` // 谁发的
	Content   string    `gorm:"type:text" json:"content"`            // 发了什么
	CreatedAt time.Time `gorm:"index" json:"created_at"`             // 什么时候发的
}

// BeforeCreate 钩子：在创建 Room 前自动生成 UUID
func (r *Room) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return
}

// BeforeCreate 钩子：在创建 Message 前自动生成 UUID
func (m *Message) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return
}
