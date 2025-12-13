package service

import (
	"fmt"
	"context"
	"MiniTeamChat/internal/biz"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// handleBizError 处理业务层错误，转换为gRPC状态错误
func handleBizError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case err == biz.ErrRoomNotFound || biz.IsNotFound(err):
		return status.Error(codes.NotFound, "房间不存在")
	case err == biz.ErrUserNotAuthenticated:
		return status.Error(codes.Unauthenticated, "用户未认证")
	case err == biz.ErrUserNotAuthorized || err == biz.ErrUserNotMember:
		return status.Error(codes.PermissionDenied, err.Error())
	case err == biz.ErrOwnerCannotLeave:
		return status.Error(codes.FailedPrecondition, "房主不能离开房间")
	case err == biz.ErrInvalidRoomID:
		return status.Error(codes.InvalidArgument, "无效的房间ID")
	case err == biz.ErrInvalidUserID:
		return status.Error(codes.InvalidArgument, "无效的用户ID")
	case err == biz.ErrInvalidMessage:
		return status.Error(codes.InvalidArgument, "无效的消息内容")
	case err == biz.ErrUserNotFound:
		return status.Error(codes.NotFound, "用户不存在")
	default:
		return status.Error(codes.Internal, fmt.Sprintf("内部错误: %v", err))
	}
}

// getCurrentUserID 从context中获取当前用户ID
func getCurrentUserID(ctx context.Context) (string, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		return "", status.Error(codes.Unauthenticated, "未认证的用户")
	}
	return userID, nil
}

// toGRPCError 将内部错误转换为gRPC状态错误
func toGRPCError(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if ok {
		return st.Err()
	}

	// 根据错误类型转换为适当的gRPC状态码
	switch {
	case err.Error() == "room not found":
		return status.Error(codes.NotFound, "房间不存在")
	case err.Error() == "user not authenticated":
		return status.Error(codes.Unauthenticated, "用户未认证")
	case err.Error() == "user not authorized":
		return status.Error(codes.PermissionDenied, "权限不足")
	case err.Error() == "user is not a member of this room":
		return status.Error(codes.PermissionDenied, "用户不是房间成员")
	case err.Error() == "room owner cannot leave the room":
		return status.Error(codes.FailedPrecondition, "房主不能离开房间")
	case err.Error() == "invalid room ID":
		return status.Error(codes.InvalidArgument, "无效的房间ID")
	case err.Error() == "invalid user ID":
		return status.Error(codes.InvalidArgument, "无效的用户ID")
	case err.Error() == "invalid message content":
		return status.Error(codes.InvalidArgument, "无效的消息内容")
	case err.Error() == "user not found":
		return status.Error(codes.NotFound, "用户不存在")
	default:
		return status.Error(codes.Internal, err.Error())
	}
}