package server

import (
	"MiniTeamChat/internal/service"
	"context"
	"log"
)

// Server 统一管理gRPC和HTTP服务器的结构体
type Server struct {
	grpcServer *GRPCServer
	httpServer *HTTPServer
	grpcAddr   string
	httpAddr   string
}

// ServerConfig 服务器配置
type ServerConfig struct {
	GRPCAddr string // gRPC服务器地址，如 ":8081"
	HTTPAddr string // HTTP服务器地址，如 ":8080"
}

// NewServer 创建新的服务器实例
func NewServer(
	roomService *service.RoomService,
	config ServerConfig,
) (*Server, error) {
	// 创建gRPC服务器
	grpcServer := NewGRPCServer(roomService)

	// 创建HTTP服务器
	httpServer, err := NewHTTPServer(config.GRPCAddr)
	if err != nil {
		return nil, err
	}

	// 设置HTTP服务器地址
	httpServer.server.Addr = config.HTTPAddr

	return &Server{
		grpcServer: grpcServer,
		httpServer: httpServer,
		grpcAddr:   config.GRPCAddr,
		httpAddr:   config.HTTPAddr,
	}, nil
}

// Start 启动服务器（非阻塞）
func (s *Server) Start() {
	// 启动gRPC服务器（在goroutine中）
	go func() {
		if err := s.grpcServer.Start(s.grpcAddr); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	// 启动HTTP服务器（在goroutine中）
	go func() {
		if err := s.httpServer.Start(); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	log.Println("Servers started successfully")
}

// Stop 优雅停止服务器
func (s *Server) Stop() {
	log.Println("Stopping servers...")

	// 停止gRPC服务器
	s.grpcServer.GracefulStop()

	// 停止HTTP服务器
	if err := s.httpServer.Shutdown(context.Background()); err != nil {
		log.Printf("Error stopping HTTP server: %v", err)
		s.grpcServer.Stop() // 强制停止gRPC服务器
	}

	log.Println("Servers stopped gracefully")
}
