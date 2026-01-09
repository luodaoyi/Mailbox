package main

import (
	"embed"
	"errors"
	"fmt"
	"log"
	"os"

	"mailbox/internal/api"
	"mailbox/internal/config"
	"mailbox/internal/db"
	"mailbox/internal/smtp"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

//go:embed web/dist
var webDist embed.FS

func main() {
	if err := runFunc("config.yaml", webDist, func(app *fiber.App, port int) error {
		return app.Listen(fmt.Sprintf(":%d", port))
	}); err != nil {
		log.Fatal(err)
	}
}

var (
	initDB    = db.Init
	startSMTP = smtp.Start
	setupApp  = api.SetupApp
	fatalf    = log.Fatalf
	runFunc   = run
	loadEnv   = godotenv.Load
)

func run(cfgPath string, webFS embed.FS, startHTTP func(app *fiber.App, port int) error) error {
	_ = loadEnv()

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return errors.New("启动失败: 必须设置 JWT_SECRET 环境变量")
	}

	if err := initDB(&cfg.Database, &cfg.Admin); err != nil {
		return fmt.Errorf("初始化数据库失败: %w", err)
	}

	go func() {
		log.Printf("SMTP 服务器启动在端口 %d", cfg.Server.SMTPPort)
		if err := startSMTP(cfg.Server.SMTPPort); err != nil {
			fatalf("SMTP 服务器启动失败: %v", err)
		}
	}()

	app := setupApp(webFS, jwtSecret)
	log.Printf("HTTP 服务器启动在端口 %d", cfg.Server.HTTPPort)
	if err := startHTTP(app, cfg.Server.HTTPPort); err != nil {
		return fmt.Errorf("HTTP 服务器启动失败: %w", err)
	}

	return nil
}
