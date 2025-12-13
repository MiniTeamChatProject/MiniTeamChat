package client

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// ErrUserNotFound 用户不存在错误
	ErrUserNotFound = status.Error(codes.NotFound, "用户不存在")
	
	// ErrInvalidToken 无效的token错误
	ErrInvalidToken = status.Error(codes.Unauthenticated, "无效的token")
	
	// ErrServiceUnavailable 服务不可用错误
	ErrServiceUnavailable = status.Error(codes.Unavailable, "服务不可用错误")
	
	// ErrRequestTimeout 请求超时错误
	ErrRequestTimeout = status.Error(codes.DeadlineExceeded, "请求超时错误")
)

// IsNotFound 检查错误是否为NotFound类型
func IsNotFound(err error) bool {
	st, ok := status.FromError(err)
	return ok && st.Code() == codes.NotFound
}

// IsUnauthenticated 检查错误是否为未认证类型
func IsUnauthenticated(err error) bool {
	st, ok := status.FromError(err)
	return ok && st.Code() == codes.Unauthenticated
}

// IsUnavailable 检查错误是否为服务不可用类型
func IsUnavailable(err error) bool {
	st, ok := status.FromError(err)
	return ok && st.Code() == codes.Unavailable
}