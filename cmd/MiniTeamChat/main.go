package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"MiniTeamChat/internal/biz"
	"MiniTeamChat/internal/client"
	"MiniTeamChat/internal/data"
	"MiniTeamChat/internal/server"
	"MiniTeamChat/internal/service"
)

func main() {
	// 1. 初始化数据层
	log.Println("Initializing data layer...")
	dataLayer, err := data.NewData()
	if err != nil {
		log.Fatalf("Failed to initialize data layer: %v", err)
	}
	log.Println("Data layer initialized successfully")

	// 2. 初始化业务逻辑层
	log.Println("Initializing business logic layer...")
	// 暂时注释掉用户服务客户端，使房间服务能够独立运行
	// userClient, err := client.NewUserClient("localhost:9090") // user-service地址
	// if err != nil {
	// 	log.Fatalf("Failed to initialize user service client: %v", err)
	// }
	// defer func() {
	// 	if c, ok := userClient.(interface{ Close() error }); ok {
	// 		c.Close()
	// 	}
	// }()
	// log.Println("User service client initialized successfully")

	// 使用nil作为用户服务客户端
	var userClient client.UserClient

	roomUsecase := biz.NewRoomUsecase(
		dataLayer.Rooms,
		dataLayer.RoomMembers,
		userClient,
	)

	memberUsecase := biz.NewRoomMemberUsecase(
		dataLayer.RoomMembers,
		dataLayer.Rooms,
		userClient,
	)

	messageUsecase := biz.NewMessageUsecase(
		dataLayer.Messages,
		dataLayer.Rooms,
		dataLayer.RoomMembers,
		userClient,
	)
	log.Println("Business logic layer initialized successfully")

	// 4. 初始化服务层
	log.Println("Initializing service layer...")
	roomService := service.NewRoomService(
		roomUsecase,
		memberUsecase,
		messageUsecase,
		userClient,
	)
	log.Println("Service layer initialized successfully")

	// 5. 初始化服务器
	log.Println("Initializing servers...")
	config := server.ServerConfig{
		GRPCAddr: "localhost:8081",
		HTTPAddr: "localhost:8082",
	}

	appServer, err := server.NewServer(roomService, config)
	if err != nil {
		log.Fatalf("Failed to initialize servers: %v", err)
	}
	log.Println("Servers initialized successfully")

	// 6. 启动服务器
	log.Println("Starting servers...")
	appServer.Start()

	// 7. 优雅处理退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down servers...")
	appServer.Stop()
	log.Println("Application exited gracefully")
}
