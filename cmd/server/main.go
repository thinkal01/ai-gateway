package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/vrviu/ai-gateway/internal/config"
	"github.com/vrviu/ai-gateway/internal/router"
	"github.com/vrviu/ai-gateway/internal/store"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化数据库
	db, err := store.NewDB(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer db.Close()

	// 创建路由
	r, err := router.New(cfg, db)
	if err != nil {
		log.Fatalf("failed to create router: %v", err)
	}

	// 启动服务器
	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// 优雅关闭
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down server...")
		srv.Close()
	}()

	log.Printf("server starting on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
