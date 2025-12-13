// internal/data/models.go
package data

import (
	"time"
	"github.com/google/uuid"
)

type Room struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`
	OwnerID   uuid.UUID `gorm:"type:uuid;not null" json:"owner_id"`
	CreatedAt time.Time `gorm:"default:now()" json:"created_at"`
}

func (Room) TableName() string {
	return "rooms"
}

type Message struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	RoomID    uuid.UUID `gorm:"type:uuid;not null;index" json:"room_id"`
	SenderID  uuid.UUID `gorm:"type:uuid;not null" json:"sender_id"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

func (Message) TableName() string {
	return "messages"
}

type RoomMember struct {
	RoomID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"room_id"`
	UserID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	JoinedAt time.Time `gorm:"default:now()" json:"joined_at"`
	Role     string    `gorm:"type:varchar(20);default:'member'" json:"role"`
}

func (RoomMember) TableName() string {
	return "room_members"
}