package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/llm"
	"backend/internal/router"
	"backend/internal/service"
)

func main() {
	configPath := os.Getenv("CONFIG_FILE")
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	if cfg.Server.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	if _, err := db.Init(cfg.DB); err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	if err := llm.Init(cfg.LLM); err != nil {
		log.Fatalf("LLM 初始化失败: %v", err)
	}

	if err := service.InitOSS(cfg.OSS); err != nil {
		log.Fatalf("OSS init failed: %v", err)
	}

	r := router.NewRouter(cfg)

	if err := r.Run(cfg.Server.Addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
