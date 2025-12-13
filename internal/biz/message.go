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

// MessageBiz 定义消息业务逻辑接口
type MessageBiz interface {
	SendMessage(ctx context.Context, roomID uuid.UUID, senderID uuid.UUID, content string) (*data.Message, error)
	GetMessages(ctx context.Context, roomID uuid.UUID, limit, offset int) ([]*data.Message, error)
	CountMessages(ctx context.Context, roomID uuid.UUID) (int64, error)
	GetMessageByID(ctx context.Context, messageID uuid.UUID) (*data.Message, error)
}

// MessageUsecase 实现消息业务逻辑
type MessageUsecase struct {
	messageRepo data.MessageRepo
	roomRepo    data.RoomRepo
	memberRepo  data.RoomMemberRepo
	userClient  client.UserClient
}

// NewMessageUsecase 创建新的MessageUsecase实例
func NewMessageUsecase(messageRepo data.MessageRepo, roomRepo data.RoomRepo, memberRepo data.RoomMemberRepo, userClient client.UserClient) *MessageUsecase {
	return &MessageUsecase{
		messageRepo: messageRepo,
		roomRepo:    roomRepo,
		memberRepo:  memberRepo,
		userClient:  userClient,
	}
}

// SendMessage 发送消息到房间
func (uc *MessageUsecase) SendMessage(ctx context.Context, roomID uuid.UUID, senderID uuid.UUID, content string) (*data.Message, error) {
	if roomID == uuid.Nil || senderID == uuid.Nil {
		return nil, ErrInvalidArgument
	}

	if content == "" {
		return nil, ErrInvalidMessage
	}

	// 验证房间是否存在
	if _, err := uc.roomRepo.FindByID(ctx, roomID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoomNotFound
		}
		return nil, err
	}

	// 验证发送者是否为房间成员
	isMember, _, err := uc.IsMember(ctx, roomID, senderID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrUserNotMember
	}

	// 验证用户是否存在
	if _, err := uc.userClient.GetUser(ctx, senderID.String()); err != nil {
		return nil, ErrUserNotFound
	}

	// 创建消息
	message := &data.Message{
		RoomID:    roomID,
		SenderID:  senderID,
		Content:   content,
		CreatedAt: time.Now(),
	}

	if err := uc.messageRepo.Create(ctx, message); err != nil {
		return nil, err
	}

	return message, nil
}

// GetMessages 获取房间的历史消息
func (uc *MessageUsecase) GetMessages(ctx context.Context, roomID uuid.UUID, limit, offset int) ([]*data.Message, error) {
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

	messages, err := uc.messageRepo.ListByRoom(ctx, roomID, limit, offset)
	if err != nil {
		return nil, err
	}

	// 转换为指针切片
	result := make([]*data.Message, len(messages))
	for i, msg := range messages {
		m := msg
		result[i] = &m
	}

	return result, nil
}

// CountMessages 获取房间的消息总数
func (uc *MessageUsecase) CountMessages(ctx context.Context, roomID uuid.UUID) (int64, error) {
	if roomID == uuid.Nil {
		return 0, ErrInvalidRoomID
	}

	return uc.messageRepo.CountByRoom(ctx, roomID)
}

// GetMessageByID 根据ID获取消息
func (uc *MessageUsecase) GetMessageByID(ctx context.Context, messageID uuid.UUID) (*data.Message, error) {
	if messageID == uuid.Nil {
		return nil, ErrInvalidArgument
	}

	// 这里需要扩展MessageRepo接口
	// 暂时返回nil，实际实现时需要完善
	return nil, nil
}

// IsMember 辅助方法：检查用户是否为房间成员
func (uc *MessageUsecase) IsMember(ctx context.Context, roomID, userID uuid.UUID) (bool, string, error) {
	ok, err := uc.memberRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return false, "", err
	}
	if !ok {
		return false, "", nil
	}

	role, err := uc.memberRepo.GetRole(ctx, roomID, userID)
	return ok, role, err
}