package biz

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"github.com/google/uuid"
	"time"
	"MiniTeamChat/internal/client"
	"MiniTeamChat/internal/data"
)

// RoomBiz 定义房间业务逻辑接口
type RoomBiz interface {
	CreateRoom(ctx context.Context, name string, ownerID uuid.UUID) (*data.Room, error)
	GetRoomByID(ctx context.Context, roomID uuid.UUID) (*data.Room, error)
	ListRoomsByOwner(ctx context.Context, ownerID uuid.UUID) ([]*data.Room, error)
	ListAllRooms(ctx context.Context) ([]*data.Room, error)
	DeleteRoom(ctx context.Context, roomID uuid.UUID, userID uuid.UUID) error
}

// RoomUsecase 实现房间业务逻辑
type RoomUsecase struct {
	roomRepo    data.RoomRepo
	memberRepo  data.RoomMemberRepo
	userClient  client.UserClient
}

// NewRoomUsecase 创建新的RoomUsecase实例
func NewRoomUsecase(roomRepo data.RoomRepo, memberRepo data.RoomMemberRepo, userClient client.UserClient) *RoomUsecase {
	return &RoomUsecase{
		roomRepo:   roomRepo,
		memberRepo: memberRepo,
		userClient: userClient,
	}
}

// CreateRoom 创建新房间
func (uc *RoomUsecase) CreateRoom(ctx context.Context, name string, ownerID uuid.UUID) (*data.Room, error) {
	if name == "" {
		return nil, ErrInvalidArgument
	}

	// 验证用户是否存在
	if _, err := uc.userClient.GetUser(ctx, ownerID.String()); err != nil {
		return nil, ErrUserNotFound
	}

	// 创建房间
	room := &data.Room{
		Name:    name,
		OwnerID: ownerID,
	}

	if err := uc.roomRepo.Create(ctx, room); err != nil {
		return nil, err
	}

	// 自动将创建者加入房间并设为owner
	member := &data.RoomMember{
		RoomID:   room.ID,
		UserID:   ownerID,
		Role:     "owner",
		JoinedAt: time.Now(),
	}

	if err := uc.memberRepo.Add(ctx, member); err != nil {
		// 如果添加成员失败，尝试回滚房间创建
		uc.roomRepo.Delete(ctx, room.ID)
		return nil, err
	}

	return room, nil
}

// GetRoomByID 根据ID获取房间信息
func (uc *RoomUsecase) GetRoomByID(ctx context.Context, roomID uuid.UUID) (*data.Room, error) {
	if roomID == uuid.Nil {
		return nil, ErrInvalidRoomID
	}

	room, err := uc.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoomNotFound
		}
		return nil, err
	}
	return room, nil
}

// ListRoomsByOwner 列出用户创建的所有房间
func (uc *RoomUsecase) ListRoomsByOwner(ctx context.Context, ownerID uuid.UUID) ([]*data.Room, error) {
	if ownerID == uuid.Nil {
		return nil, ErrInvalidUserID
	}

	rooms, err := uc.roomRepo.FindByOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}

	// 转换为指针切片
	result := make([]*data.Room, len(rooms))
	for i, room := range rooms {
		r := room
		result[i] = &r
	}

	return result, nil
}

// ListAllRooms 列出所有房间（用于公开房间列表）
func (uc *RoomUsecase) ListAllRooms(ctx context.Context) ([]*data.Room, error) {
	rooms, err := uc.roomRepo.FindAllRooms(ctx)
	if err != nil {
		return nil, err
	}

	// 转换为指针切片
	result := make([]*data.Room, len(rooms))
	for i, room := range rooms {
		r := room
		result[i] = &r
	}

	return result, nil
}

// DeleteRoom 删除房间，只有房主可以删除
func (uc *RoomUsecase) DeleteRoom(ctx context.Context, roomID uuid.UUID, userID uuid.UUID) error {
	if roomID == uuid.Nil || userID == uuid.Nil {
		return ErrInvalidArgument
	}

	room, err := uc.GetRoomByID(ctx, roomID)
	if err != nil {
		return err
	}

	// 验证权限：只有房主可以删除房间
	if room.OwnerID != userID {
		return ErrUserNotAuthorized
	}

	// 删除房间（会自动处理相关数据）
	if err := uc.roomRepo.Delete(ctx, roomID); err != nil {
		return err
	}

	return nil
}