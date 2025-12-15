package server

import (
	"log"
	"net"

	v1 "MiniTeamChat/api/room/v1"
	"MiniTeamChat/internal/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type GRPCServer struct {
	server *grpc.Server
}

func NewGRPCServer(roomService *service.RoomService) *GRPCServer {
	// 1. 定义gRPC服务器选项（如拦截器、证书等）
	opts := []grpc.ServerOption{}

	// 2. 创建原生gRPC服务器实例
	server := grpc.NewServer(opts...)

	// 3. 注册房间服务到gRPC服务器
	v1.RegisterRoomServiceServer(server, roomService)

	// 4. 注册反射服务（便于grpcurl等工具调试）
	reflection.Register(server)

	// 5. 返回封装后的GRPCServer实例
	return &GRPCServer{
		server: server,
	}
}

func (s *GRPCServer) Start(addr string) error {
	// 1. 监听指定TCP地址（如 ":8080"）
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err // 监听失败（如端口被占用）返回错误
	}

	// 2. 打印启动日志
	log.Printf("gRPC server listening on %s", addr)

	// 3. 启动gRPC服务器，阻塞处理请求
	return s.server.Serve(listener)
}

// GracefulStop 优雅停止gRPC服务器
func (s *GRPCServer) GracefulStop() {
	log.Println("Gracefully stopping gRPC server...")
	s.server.GracefulStop()
	log.Println("gRPC server stopped")
}

// Stop 强制停止gRPC服务器
func (s *GRPCServer) Stop() {
	log.Println("Stopping gRPC server...")
	s.server.Stop()
	log.Println("gRPC server stopped")
}
