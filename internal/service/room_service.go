package service

import (
	"context"
	"time"

	v1 "MiniTeamChat/api/room/v1"
	"MiniTeamChat/internal/biz"
	"MiniTeamChat/internal/client"
	"MiniTeamChat/internal/data"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type RoomService struct {
	v1.UnimplementedRoomServiceServer                   // 嵌入gRPC的未实现服务结构体，兼容接口
	roomUsecase                       biz.RoomBiz       // 房间核心业务逻辑（创建/删除/查询）
	memberUsecase                     biz.RoomMemberBiz // 房间成员业务逻辑（加入/离开/查询）
	messageUsecase                    biz.MessageBiz    // 房间消息业务逻辑（获取历史消息）
	userClient                        client.UserClient // 用户服务客户端（获取用户名等）
}

func NewRoomService(
	roomUsecase biz.RoomBiz,
	memberUsecase biz.RoomMemberBiz,
	messageUsecase biz.MessageBiz,
	userClient client.UserClient,
) *RoomService {
	return &RoomService{
		roomUsecase:    roomUsecase,
		memberUsecase:  memberUsecase,
		messageUsecase: messageUsecase,
		userClient:     userClient,
	}
}

// CreateRoom 创建新房间
func (s *RoomService) CreateRoom(ctx context.Context, req *v1.CreateRoomRequest) (*v1.CreateRoomReply, error) {
	// 从context中获取当前用户ID
	userIDStr, err := getCurrentUserID(ctx)
	if err != nil {
		return nil, err
	}

	uid, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "无效的用户ID格式")
	}

	// 创建房间
	room, err := s.roomUsecase.CreateRoom(ctx, req.Name, uid)
	if err != nil {
		return nil, handleBizError(err)
	}

	return &v1.CreateRoomReply{
		Room: s.convertRoomToProto(room),
	}, nil
}

// GetRoom 根据ID获取房间信息
func (s *RoomService) GetRoom(ctx context.Context, req *v1.GetRoomRequest) (*v1.Room, error) {
	roomID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "无效的房间ID格式")
	}

	room, err := s.roomUsecase.GetRoomByID(ctx, roomID)
	if err != nil {
		return nil, handleBizError(err)
	}

	return s.convertRoomToProto(room), nil
}

// ListRooms 列出房间（可按用户ID过滤）
func (s *RoomService) ListRooms(ctx context.Context, req *v1.ListRoomsRequest) (*v1.ListRoomsReply, error) {
	var rooms []*data.Room
	var err error

	if req.UserId != "" {
		userID, err := uuid.Parse(req.UserId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "无效的用户ID格式")
		}
		rooms, err = s.roomUsecase.ListRoomsByOwner(ctx, userID)
	} else {
		// 这里需要在biz层实现ListAllRooms方法
		return nil, status.Error(codes.Unimplemented, "列出所有房间功能未实现")
	}

	if err != nil {
		return nil, handleBizError(err)
	}

	reply := &v1.ListRoomsReply{
		Rooms: make([]*v1.Room, len(rooms)),
	}

	for i, room := range rooms {
		reply.Rooms[i] = s.convertRoomToProto(room)
	}

	return reply, nil
}

// DeleteRoom 删除房间
func (s *RoomService) DeleteRoom(ctx context.Context, req *v1.DeleteRoomRequest) (*emptypb.Empty, error) {
	userIDStr, err := getCurrentUserID(ctx)
	if err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "无效的用户ID格式")
	}

	roomID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "无效的房间ID格式")
	}

	err = s.roomUsecase.DeleteRoom(ctx, roomID, userID)
	if err != nil {
		return nil, handleBizError(err)
	}

	return &emptypb.Empty{}, nil
}

// JoinRoom 加入房间
func (s *RoomService) JoinRoom(ctx context.Context, req *v1.JoinRoomRequest) (*v1.JoinRoomReply, error) {
	userIDStr, err := getCurrentUserID(ctx)
	if err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "无效的用户ID格式")
	}

	roomID, err := uuid.Parse(req.RoomId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "无效的房间ID格式")
	}

	member, err := s.memberUsecase.JoinRoom(ctx, roomID, userID)
	if err != nil {
		return nil, handleBizError(err)
	}

	return &v1.JoinRoomReply{
		Member: s.convertMemberToProto(member),
	}, nil
}

// LeaveRoom 离开房间
func (s *RoomService) LeaveRoom(ctx context.Context, req *v1.LeaveRoomRequest) (*emptypb.Empty, error) {
	userIDStr, err := getCurrentUserID(ctx)
	if err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "无效的用户ID格式")
	}

	roomID, err := uuid.Parse(req.RoomId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "无效的房间ID格式")
	}

	err = s.memberUsecase.LeaveRoom(ctx, roomID, userID)
	if err != nil {
		return nil, handleBizError(err)
	}

	return &emptypb.Empty{}, nil
}

// ListRoomMembers 获取房间成员列表
func (s *RoomService) ListRoomMembers(ctx context.Context, req *v1.ListRoomMembersRequest) (*v1.ListRoomMembersReply, error) {
	roomID, err := uuid.Parse(req.RoomId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "无效的房间ID格式")
	}

	members, err := s.memberUsecase.ListMembers(ctx, roomID)
	if err != nil {
		return nil, handleBizError(err)
	}

	reply := &v1.ListRoomMembersReply{
		Members: make([]*v1.RoomMember, len(members)),
	}

	for i, member := range members {
		reply.Members[i] = s.convertMemberToProto(member)
	}

	return reply, nil
}

// GetMessages 获取房间消息历史
func (s *RoomService) GetMessages(ctx context.Context, req *v1.GetMessagesRequest) (*v1.GetMessagesReply, error) {
	roomID, err := uuid.Parse(req.RoomId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "无效的房间ID格式")
	}

	limit := int(req.Limit)
	offset := int(req.Offset)

	if limit <= 0 {
		limit = 20 // 默认分页大小
	}
	if offset < 0 {
		offset = 0
	}

	messages, err := s.messageUsecase.GetMessages(ctx, roomID, limit, offset)
	if err != nil {
		return nil, handleBizError(err)
	}

	reply := &v1.GetMessagesReply{
		Messages: make([]*v1.Message, len(messages)),
	}

	for i, msg := range messages {
		// 获取发送者用户名
		senderName := "用户"
		user, err := s.userClient.GetUser(ctx, msg.SenderID.String())
		if err == nil && user != nil {
			senderName = user.Username
		}

		reply.Messages[i] = s.convertMessageToProto(msg, senderName)
	}

	return reply, nil
}

// IsMember 检查用户是否为房间成员
func (s *RoomService) IsMember(ctx context.Context, req *v1.IsMemberRequest) (*v1.IsMemberReply, error) {
	userIDStr, err := getCurrentUserID(ctx)
	if err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "无效的用户ID格式")
	}

	roomID, err := uuid.Parse(req.RoomId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "无效的房间ID格式")
	}

	isMember, role, err := s.memberUsecase.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, handleBizError(err)
	}

	return &v1.IsMemberReply{
		IsMember: isMember,
		Role:     role,
	}, nil
}

// convertRoomToProto 将Room模型转换为Proto消息
func (s *RoomService) convertRoomToProto(room *data.Room) *v1.Room {
	return &v1.Room{
		Id:        room.ID.String(),
		Name:      room.Name,
		OwnerId:   room.OwnerID.String(),
		CreatedAt: room.CreatedAt.Format(time.RFC3339),
	}
}

// convertMemberToProto 将RoomMember模型转换为Proto消息
func (s *RoomService) convertMemberToProto(member *data.RoomMember) *v1.RoomMember {
	return &v1.RoomMember{
		RoomId:   member.RoomID.String(),
		UserId:   member.UserID.String(),
		JoinedAt: member.JoinedAt.Format(time.RFC3339),
		Role:     member.Role,
	}
}

// convertMessageToProto 将Message模型转换为Proto消息
func (s *RoomService) convertMessageToProto(msg *data.Message, senderName string) *v1.Message {
	return &v1.Message{
		Id:         msg.ID.String(),
		RoomId:     msg.RoomID.String(),
		SenderId:   msg.SenderID.String(),
		SenderName: senderName,
		Content:    msg.Content,
		CreatedAt:  msg.CreatedAt.Format(time.RFC3339),
	}
}

