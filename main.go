package main

import (
	"embed"
	"fmt"
	"log"
	"os"

	"mailbox/internal/api"
	"mailbox/internal/config"
	"mailbox/internal/db"
	"mailbox/internal/smtp"
)

//go:embed web/dist
var webDist embed.FS

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("启动失败: 必须设置 JWT_SECRET 环境变量")
	}

	if err := db.Init(&cfg.Database, &cfg.Admin); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	go func() {
		log.Printf("SMTP 服务器启动在端口 %d", cfg.Server.SMTPPort)
		if err := smtp.Start(cfg.Server.SMTPPort); err != nil {
			log.Fatalf("SMTP 服务器启动失败: %v", err)
		}
	}()

	app := api.SetupApp(webDist, jwtSecret)
	log.Printf("HTTP 服务器启动在端口 %d", cfg.Server.HTTPPort)
	if err := app.Listen(fmt.Sprintf(":%d", cfg.Server.HTTPPort)); err != nil {
		log.Fatalf("HTTP 服务器启动失败: %v", err)
	}
}
