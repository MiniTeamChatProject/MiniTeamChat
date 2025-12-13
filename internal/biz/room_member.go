package biz

import (
	"context"
	"errors"
	"time"
	"gorm.io/gorm"
	"github.com/google/uuid"
	"MiniTeamChat/internal/client"
	"MiniTeamChat/internal/data"
)

// RoomMemberBiz 定义房间成员业务逻辑接口
type RoomMemberBiz interface {
	JoinRoom(ctx context.Context, roomID uuid.UUID, userID uuid.UUID) (*data.RoomMember, error)
	LeaveRoom(ctx context.Context, roomID uuid.UUID, userID uuid.UUID) error
	IsMember(ctx context.Context, roomID uuid.UUID, userID uuid.UUID) (bool, string, error)
	ListMembers(ctx context.Context, roomID uuid.UUID) ([]*data.RoomMember, error)
	GetMemberRole(ctx context.Context, roomID uuid.UUID, userID uuid.UUID) (string, error)
}

// RoomMemberUsecase 实现房间成员业务逻辑
type RoomMemberUsecase struct {
	memberRepo data.RoomMemberRepo
	roomRepo   data.RoomRepo
	userClient client.UserClient
}

// NewRoomMemberUsecase 创建新的RoomMemberUsecase实例
func NewRoomMemberUsecase(memberRepo data.RoomMemberRepo, roomRepo data.RoomRepo, userClient client.UserClient) *RoomMemberUsecase {
	return &RoomMemberUsecase{
		memberRepo: memberRepo,
		roomRepo:   roomRepo,
		userClient: userClient,
	}
}

// JoinRoom 加入房间
func (uc *RoomMemberUsecase) JoinRoom(ctx context.Context, roomID uuid.UUID, userID uuid.UUID) (*data.RoomMember, error) {
	if roomID == uuid.Nil || userID == uuid.Nil {
		return nil, ErrInvalidArgument
	}

	// 验证房间是否存在
	if _, err := uc.roomRepo.FindByID(ctx, roomID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoomNotFound
		}
		return nil, err
	}

	// 验证用户是否存在
	if _, err := uc.userClient.GetUser(ctx, userID.String()); err != nil {
		return nil, ErrUserNotFound
	}

	// 检查是否已经是成员
	isMember, _, err := uc.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if isMember {
		return nil, ErrAlreadyExists
	}

	// 添加新成员
	member := &data.RoomMember{
		RoomID:   roomID,
		UserID:   userID,
		Role:     "member",
		JoinedAt: time.Now(),
	}

	if err := uc.memberRepo.Add(ctx, member); err != nil {
		return nil, err
	}

	return member, nil
}

// LeaveRoom 离开房间
func (uc *RoomMemberUsecase) LeaveRoom(ctx context.Context, roomID uuid.UUID, userID uuid.UUID) error {
	if roomID == uuid.Nil || userID == uuid.Nil {
		return ErrInvalidArgument
	}

	// 检查是否是成员
	isMember, role, err := uc.IsMember(ctx, roomID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrUserNotMember
	}

	// 检查是否是房主
	if role == "owner" {
		return ErrOwnerCannotLeave
	}

	// 删除成员关系
	if err := uc.memberRepo.Delete(ctx, roomID, userID); err != nil {
		return err
	}

	return nil
}

// IsMember 检查用户是否为房间成员，并返回角色
func (uc *RoomMemberUsecase) IsMember(ctx context.Context, roomID uuid.UUID, userID uuid.UUID) (bool, string, error) {
	if roomID == uuid.Nil || userID == uuid.Nil {
		return false, "", ErrInvalidArgument
	}

	ok, err := uc.memberRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return false, "", err
	}
	if !ok {
		return false, "", nil
	}

	// 获取成员角色
	role, err := uc.memberRepo.GetRole(ctx, roomID, userID)
	if err != nil {
		return false, "", err
	}

	return true, role, nil
}

// ListMembers 获取房间的所有成员
func (uc *RoomMemberUsecase) ListMembers(ctx context.Context, roomID uuid.UUID) ([]*data.RoomMember, error) {
	if roomID == uuid.Nil {
		return nil, ErrInvalidRoomID
	}

	// 验证房间是否存在
	if _, err := uc.roomRepo.FindByID(ctx, roomID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoomNotFound
		}
		return nil, err
	}

	members, err := uc.memberRepo.FindMembersByRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}

	// 转换为指针切片
	result := make([]*data.RoomMember, len(members))
	for i, member := range members {
		m := member
		result[i] = &m
	}

	return result, nil
}

// GetMemberRole 获取成员在房间中的角色
func (uc *RoomMemberUsecase) GetMemberRole(ctx context.Context, roomID uuid.UUID, userID uuid.UUID) (string, error) {
	if roomID == uuid.Nil || userID == uuid.Nil {
		return "", ErrInvalidArgument
	}

	_, role, err := uc.IsMember(ctx, roomID, userID)
	if err != nil {
		return "", err
	}
	if role == "" {
		return "", ErrUserNotMember
	}
	return role, nil
}