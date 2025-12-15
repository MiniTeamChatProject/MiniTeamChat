package server

import (
	"context"
	"log"
	"net/http"
	"time"
)

type HTTPServer struct {
	server *http.Server
}

func NewHTTPServer(grpcAddr string) (*HTTPServer, error) {
	// 创建简单的HTTP服务器，返回gRPC服务器状态
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 创建HTTP服务器
	server := &http.Server{
		Addr:         ":8080", // 默认HTTP端口
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return &HTTPServer{
		server: server,
	}, nil
}

// Start 启动HTTP服务器
func (s *HTTPServer) Start() error {
	log.Printf("HTTP server listening on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// Shutdown 优雅关闭HTTP服务器
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	log.Println("Gracefully shutting down HTTP server...")
	return s.server.Shutdown(ctx)
}
