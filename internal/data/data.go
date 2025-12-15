// internal/data/data.go
package data

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Data 代表数据层实例，包含所有仓库接口
type Data struct {
	db *gorm.DB

	// 仓库接口
	Rooms       RoomRepo       // 房间仓库
	Messages    MessageRepo    // 消息仓库
	RoomMembers RoomMemberRepo // 房间成员仓库
}

// NewData 初始化数据层，建立 PostgreSQL 连接
func NewData() (*Data, error) {
	// 构建数据库连接字符串
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai",
		getEnv("DB_HOST", "localhost"),    // 数据库主机地址，默认 localhost
		getEnv("DB_USER", "hansun"),       // 数据库用户名，默认 hansun
		getEnv("DB_PASSWORD", "w123456w"), // 数据库密码，默认 w123456w
		getEnv("DB_NAME", "roomdb"),       // 数据库名称，默认 roomdb
		getEnv("DB_PORT", "5432"),         // 数据库端口，默认 5432
	)

	// 连接数据库
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 自动迁移模型（开发环境可用；生产环境应使用迁移脚本）
	err = db.AutoMigrate(&Room{}, &Message{}, &RoomMember{})
	if err != nil {
		return nil, fmt.Errorf("自动迁移失败: %w", err)
	}

	// 创建并返回数据层实例
	return &Data{
		db:          db,
		Rooms:       &roomRepoImpl{db: db},       // 初始化房间仓库实现
		Messages:    &messageRepoImpl{db: db},    // 初始化消息仓库实现
		RoomMembers: &roomMemberRepoImpl{db: db}, // 初始化房间成员仓库实现
	}, nil
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// --- 仓库接口定义 ---

// RoomRepo 定义房间相关操作的接口
type RoomRepo interface {
	Create(ctx context.Context, room *Room) error                       // 创建房间
	FindByID(ctx context.Context, id uuid.UUID) (*Room, error)          // 根据ID查找房间
	FindByOwner(ctx context.Context, ownerID uuid.UUID) ([]Room, error) // 根据房主ID查找房间列表
	FindAllRooms(ctx context.Context) ([]Room, error)                   // 查找所有房间
	Delete(ctx context.Context, id uuid.UUID) error                     // 删除房间
}

// MessageRepo 定义消息相关操作的接口
type MessageRepo interface {
	Create(ctx context.Context, msg *Message) error                                         // 创建消息
	ListByRoom(ctx context.Context, roomID uuid.UUID, limit, offset int) ([]Message, error) // 根据房间ID分页查询消息
	CountByRoom(ctx context.Context, roomID uuid.UUID) (int64, error)                       // 获取房间消息总数
}

// RoomMemberRepo 定义房间成员相关操作的接口
type RoomMemberRepo interface {
	Add(ctx context.Context, member *RoomMember) error                             // 添加房间成员
	IsMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error)          // 检查用户是否为房间成员
	FindMembersByRoom(ctx context.Context, roomID uuid.UUID) ([]RoomMember, error) // 根据房间ID查找所有成员
	Delete(ctx context.Context, roomID, userID uuid.UUID) error                    // 删除房间成员（离开房间）
	GetRole(ctx context.Context, roomID, userID uuid.UUID) (string, error)         // 获取成员在房间中的角色
}

// --- 仓库实现（如果项目规模增大，可以拆分到单独文件）---

// roomRepoImpl 房间仓库的具体实现
type roomRepoImpl struct{ db *gorm.DB }

// messageRepoImpl 消息仓库的具体实现
type messageRepoImpl struct{ db *gorm.DB }

// roomMemberRepoImpl 房间成员仓库的具体实现
type roomMemberRepoImpl struct{ db *gorm.DB }

// Create 创建新房间
func (r *roomRepoImpl) Create(ctx context.Context, room *Room) error {
	room.ID = uuid.New()                            // 生成UUID作为房间ID
	room.CreatedAt = time.Now()                     // 设置创建时间为当前时间
	return r.db.WithContext(ctx).Create(room).Error // 保存到数据库
}

// FindByID 根据ID查找房间
func (r *roomRepoImpl) FindByID(ctx context.Context, id uuid.UUID) (*Room, error) {
	var room Room
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&room).Error
	return &room, err
}

// FindByOwner 根据房主ID查找该用户创建的所有房间
func (r *roomRepoImpl) FindByOwner(ctx context.Context, ownerID uuid.UUID) ([]Room, error) {
	var rooms []Room
	err := r.db.WithContext(ctx).Where("owner_id = ?", ownerID).Find(&rooms).Error
	return rooms, err
}

// Delete 删除房间及其所有相关数据
func (r *roomRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
	// 使用事务保证数据一致性
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 1. 删除房间中的所有消息
	if err := tx.Where("room_id = ?", id).Delete(&Message{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 2. 删除房间的所有成员关系
	if err := tx.Where("room_id = ?", id).Delete(&RoomMember{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 3. 删除房间本身
	if err := tx.Where("id = ?", id).Delete(&Room{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// FindAllRooms 查询所有房间
func (r *roomRepoImpl) FindAllRooms(ctx context.Context) ([]Room, error) {
	var rooms []Room
	err := r.db.WithContext(ctx).Find(&rooms).Error
	return rooms, err
}

// Create 创建新消息
func (m *messageRepoImpl) Create(ctx context.Context, msg *Message) error {
	msg.ID = uuid.New()                            // 生成UUID作为消息ID
	msg.CreatedAt = time.Now()                     // 设置创建时间为当前时间
	return m.db.WithContext(ctx).Create(msg).Error // 保存到数据库
}

// ListByRoom 根据房间ID分页查询消息，按创建时间升序排列
func (m *messageRepoImpl) ListByRoom(ctx context.Context, roomID uuid.UUID, limit, offset int) ([]Message, error) {
	var msgs []Message
	err := m.db.WithContext(ctx).
		Where("room_id = ?", roomID). // 过滤指定房间的消息
		Order("created_at ASC").      // 按创建时间升序排列（最早的在前）
		Limit(limit).                 // 限制返回数量
		Offset(offset).               // 设置分页偏移量
		Find(&msgs).Error
	return msgs, err
}

// CountByRoom 获取指定房间的消息总数
func (m *messageRepoImpl) CountByRoom(ctx context.Context, roomID uuid.UUID) (int64, error) {
	var count int64
	err := m.db.WithContext(ctx).
		Model(&Message{}).
		Where("room_id = ?", roomID).
		Count(&count).Error
	return count, err
}

// Add 添加房间成员
func (rm *roomMemberRepoImpl) Add(ctx context.Context, member *RoomMember) error {
	member.JoinedAt = time.Now()                       // 设置加入时间为当前时间
	return rm.db.WithContext(ctx).Create(member).Error // 保存到数据库
}

// IsMember 检查用户是否为指定房间的成员
func (rm *roomMemberRepoImpl) IsMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	var count int64
	// 查询满足条件的记录数量
	err := rm.db.WithContext(ctx).Model(&RoomMember{}).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Count(&count).Error
	return count > 0, err // 如果数量大于0，则用户是成员
}

// FindMembersByRoom 根据房间ID查找所有成员
func (rm *roomMemberRepoImpl) FindMembersByRoom(ctx context.Context, roomID uuid.UUID) ([]RoomMember, error) {
	var members []RoomMember
	err := rm.db.WithContext(ctx).Where("room_id = ?", roomID).Find(&members).Error
	return members, err
}

// Delete 删除房间成员（离开房间）
func (rm *roomMemberRepoImpl) Delete(ctx context.Context, roomID, userID uuid.UUID) error {
	// 检查是否为房主，房主不能直接离开房间
	var room Room
	if err := rm.db.WithContext(ctx).Where("id = ? AND owner_id = ?", roomID, userID).First(&room).Error; err == nil {
		return fmt.Errorf("房主不能离开房间，请先删除房间或转让房主")
	}

	// 删除成员关系
	result := rm.db.WithContext(ctx).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Delete(&RoomMember{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("用户不是房间成员")
	}

	return nil
}

// GetRole 获取成员在房间中的角色
func (rm *roomMemberRepoImpl) GetRole(ctx context.Context, roomID, userID uuid.UUID) (string, error) {
	var member RoomMember
	err := rm.db.WithContext(ctx).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		First(&member).Error

	if err != nil {
		return "", err
	}

	return member.Role, nil
}
