package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	// 假设user-service的proto定义在以下路径
	// 实际项目中应该根据user-service的实际位置进行调整
	v1 "MiniTeamChat/api/user/v1"
)

// UserClient 定义user-service客户端接口
type UserClient interface {
	// VerifyToken 验证JWT token
	VerifyToken(ctx context.Context, token string) (*v1.VerifyTokenReply, error)

	// GetUser 根据ID获取用户信息
	GetUser(ctx context.Context, userID string) (*v1.GetUserReply, error)

	// HealthCheck 检查user-service健康状态
	HealthCheck(ctx context.Context) error
}

// userClientImpl 实现UserClient接口
type userClientImpl struct {
	client v1.UserServiceClient
	conn   *grpc.ClientConn
}

// NewUserClient 创建新的user-service客户端
func NewUserClient(addr string, opts ...grpc.DialOption) (UserClient, error) {
	// 设置默认连接选项
	defaultOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5 * time.Second),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(1024*1024), // 1MB
			grpc.MaxCallSendMsgSize(1024*1024), // 1MB
		),
	}

	// 合并自定义选项
	opts = append(defaultOpts, opts...)

	// 建立连接
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to user-service at %s: %w", addr, err)
	}

	// 创建gRPC客户端
	client := v1.NewUserServiceClient(conn)

	return &userClientImpl{
		client: client,
		conn:   conn,
	}, nil
}

// VerifyToken 验证JWT token
func (c *userClientImpl) VerifyToken(ctx context.Context, token string) (*v1.VerifyTokenReply, error) {
	// 设置请求超时
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &v1.VerifyTokenRequest{Token: token}
	reply, err := c.client.VerifyToken(ctx, req)
	if err != nil {
		// 转换gRPC错误为标准错误
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.Unauthenticated:
				return nil, ErrInvalidToken
			case codes.DeadlineExceeded:
				return nil, ErrRequestTimeout
			case codes.Unavailable:
				return nil, ErrServiceUnavailable
			}
		}
		return nil, fmt.Errorf("verify token failed: %w", err)
	}

	return reply, nil
}

// GetUser 根据ID获取用户信息
func (c *userClientImpl) GetUser(ctx context.Context, userID string) (*v1.GetUserReply, error) {
	// 设置请求超时
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &v1.GetUserRequest{Id: userID}
	reply, err := c.client.GetUser(ctx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.NotFound:
				return nil, ErrUserNotFound
			case codes.DeadlineExceeded:
				return nil, ErrRequestTimeout
			case codes.Unavailable:
				return nil, ErrServiceUnavailable
			}
		}
		return nil, fmt.Errorf("get user failed: %w", err)
	}

	return reply, nil
}

// HealthCheck 检查user-service健康状态
func (c *userClientImpl) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// 调用一个简单的接口来检查服务是否正常
	_, err := c.client.VerifyToken(ctx, &v1.VerifyTokenRequest{Token: "test"})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.Unavailable {
			return ErrServiceUnavailable
		}
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// Close 关闭客户端连接
func (c *userClientImpl) Close() error {
	return c.conn.Close()
}
