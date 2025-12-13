package biz

import (
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// 通用错误
	ErrInvalidArgument    = status.Error(codes.InvalidArgument, "无效的参数")
	ErrNotFound           = status.Error(codes.NotFound, "未找到资源")
	ErrPermissionDenied   = status.Error(codes.PermissionDenied, "权限不足")
	ErrUnauthenticated    = status.Error(codes.Unauthenticated, "未认证")
	ErrInternal           = status.Error(codes.Internal, "内部服务器错误")
	ErrAlreadyExists      = status.Error(codes.AlreadyExists, "资源已存在")
	ErrFailedPrecondition = status.Error(codes.FailedPrecondition, "前置条件失败")

	// 具体业务错误
	ErrRoomNotFound      = errors.New("房间不存在")
	ErrRoomExists        = errors.New("房间已存在")
	ErrUserNotAuthenticated = errors.New("用户未认证")
	ErrUserNotAuthorized = errors.New("用户未授权")
	ErrUserNotMember     = errors.New("用户不是房间成员")
	ErrOwnerCannotLeave  = errors.New("房主不能离开房间")
	ErrInvalidRoomID     = errors.New("无效的房间ID")
	ErrInvalidUserID     = errors.New("无效的用户ID")
	ErrInvalidMessage    = errors.New("无效的消息内容")
	ErrUserNotFound      = errors.New("用户不存在")
	ErrWebSocketClosed   = errors.New("WebSocket连接已关闭")
)

// IsNotFound 检查错误是否为NotFound类型
func IsNotFound(err error) bool {
	return err == ErrRoomNotFound || err == ErrUserNotFound
}

// IsPermissionDenied 检查错误是否为权限拒绝类型
func IsPermissionDenied(err error) bool {
	return err == ErrUserNotAuthorized || err == ErrUserNotMember
}

// ToGRPCError 将业务错误转换为gRPC状态错误
func ToGRPCError(err error) error {
	switch {
	case err == ErrRoomNotFound || err == ErrUserNotFound:
		return status.Error(codes.NotFound, err.Error())
	case err == ErrUserNotAuthorized || err == ErrUserNotMember:
		return status.Error(codes.PermissionDenied, err.Error())
	case err == ErrUserNotAuthenticated:
		return status.Error(codes.Unauthenticated, err.Error())
	case err == ErrInvalidArgument || err == ErrInvalidRoomID || err == ErrInvalidUserID || err == ErrInvalidMessage:
		return status.Error(codes.InvalidArgument, err.Error())
	case err == ErrOwnerCannotLeave:
		return status.Error(codes.FailedPrecondition, err.Error())
	case err == ErrRoomExists:
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}