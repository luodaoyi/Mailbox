package api

import (
	"bufio"
	"embed"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"mailbox/internal/db"
	"mailbox/internal/model"
)

const jwtSecret = "mailbox-secret-key-change-in-production"

type Claims struct {
	Email   string `json:"email"`
	IsAdmin bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

// SSE 客户端管理
type sseClient struct {
	email string
	ch    chan []byte
}

type SSEManager struct {
	clients map[string][]*sseClient
	mu      sync.RWMutex
}

var sseManager = &SSEManager{
	clients: make(map[string][]*sseClient),
}

func (m *SSEManager) register(email string, client *sseClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[email] = append(m.clients[email], client)
	fmt.Printf("[SSE] 客户端已注册: %s (总数: %d)\n", email, len(m.clients[email]))
}

func (m *SSEManager) unregister(email string, client *sseClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	clients := m.clients[email]
	for i, c := range clients {
		if c == client {
			m.clients[email] = append(clients[:i], clients[i+1:]...)
			close(client.ch)
			break
		}
	}
	if len(m.clients[email]) == 0 {
		delete(m.clients, email)
	}
	fmt.Printf("[SSE] 客户端已注销: %s (剩余: %d)\n", email, len(m.clients[email]))
}

func (m *SSEManager) NotifyNewEmail(toAddr string, email *model.Email) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	fmt.Printf("[SSE] 通知新邮件: %s (客户端数: %d)\n", toAddr, len(m.clients[toAddr]))

	data := []byte("data: {\"ID\":" + strconv.FormatUint(uint64(email.ID), 10) +
		",\"FromAddr\":\"" + email.FromAddr +
		"\",\"Subject\":\"" + email.Subject +
		"\",\"ReceivedAt\":\"" + email.ReceivedAt.Format(time.RFC3339) + "\"}\n\n")

	for i, client := range m.clients[toAddr] {
		select {
		case client.ch <- data:
			fmt.Printf("[SSE] 消息已发送到客户端 %d\n", i)
		default:
			fmt.Printf("[SSE] 客户端 %d 通道已满，跳过\n", i)
		}
	}
}

func SetupApp(webFS embed.FS) *fiber.App {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	api := app.Group("/api")
	{
		api.Post("/login", userLogin)
		api.Get("/emails", authMiddleware(false), getEmails)
		api.Get("/emails/stream", authMiddleware(false), emailStream)
		api.Get("/emails/:id", authMiddleware(false), getEmailDetail)
		api.Delete("/emails/:id", authMiddleware(false), deleteEmail)
		api.Delete("/emails/page", authMiddleware(false), deletePageEmails)
		api.Delete("/emails/month/all", authMiddleware(false), deleteMonthEmails)
		api.Delete("/emails/all", authMiddleware(false), deleteAllEmails)
		api.Get("/attachments/:id", authMiddleware(false), getAttachment)

		admin := api.Group("/admin")
		{
			admin.Post("/login", adminLogin)
			admin.Get("/domains", authMiddleware(true), getDomains)
			admin.Post("/domains", authMiddleware(true), createDomain)
			admin.Put("/domains/:id", authMiddleware(true), updateDomain)
			admin.Delete("/domains/:id", authMiddleware(true), deleteDomain)
		}
	}

	staticFS, _ := fs.Sub(webFS, "web/dist")
	app.Use(func(c *fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api") {
			return c.Next()
		}

		path := strings.TrimPrefix(c.Path(), "/")
		if path == "" {
			path = "index.html"
		}

		if data, err := fs.ReadFile(staticFS, path); err == nil {
			c.Set("Content-Type", getContentType(path))
			return c.Send(data)
		}

		if data, err := fs.ReadFile(staticFS, "index.html"); err == nil {
			c.Set("Content-Type", "text/html")
			return c.Send(data)
		}

		return c.SendStatus(404)
	})

	return app
}

func getContentType(path string) string {
	if strings.HasSuffix(path, ".js") {
		return "application/javascript"
	}
	if strings.HasSuffix(path, ".css") {
		return "text/css"
	}
	if strings.HasSuffix(path, ".html") {
		return "text/html"
	}
	if strings.HasSuffix(path, ".json") {
		return "application/json"
	}
	return "application/octet-stream"
}

func authMiddleware(requireAdmin bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var tokenStr string

		// 优先从 Authorization header 获取
		auth := c.Get("Authorization")
		if auth != "" && strings.HasPrefix(auth, "Bearer ") {
			tokenStr = strings.TrimPrefix(auth, "Bearer ")
		} else {
			// 从 query parameter 获取（用于 SSE）
			tokenStr = c.Query("token")
		}

		if tokenStr == "" {
			return c.Status(401).JSON(fiber.Map{"error": "未授权"})
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(401).JSON(fiber.Map{"error": "无效的令牌"})
		}

		if requireAdmin && !claims.IsAdmin {
			return c.Status(403).JSON(fiber.Map{"error": "需要管理员权限"})
		}

		c.Locals("email", claims.Email)
		c.Locals("is_admin", claims.IsAdmin)
		return c.Next()
	}
}

func emailStream(c *fiber.Ctx) error {
	email := c.Locals("email").(string)
	fmt.Printf("[SSE] 新的 SSE 连接请求: %s\n", email)

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	client := &sseClient{
		email: email,
		ch:    make(chan []byte, 10),
	}

	sseManager.register(email, client)

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer sseManager.unregister(email, client)

		if _, err := w.WriteString(": connected\n\n"); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case data, ok := <-client.ch:
				if !ok {
					return
				}
				if _, err := w.Write(data); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			case <-ticker.C:
				if _, err := w.WriteString(": ping\n\n"); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})

	return nil
}

func userLogin(c *fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}

	if err := c.BodyParser(&req); err != nil || req.Email == "" {
		return c.Status(400).JSON(fiber.Map{"error": "无效的请求"})
	}

	parts := strings.Split(req.Email, "@")
	if len(parts) != 2 {
		return c.Status(400).JSON(fiber.Map{"error": "无效的邮箱地址"})
	}

	var domain model.Domain
	if err := db.DB.Where("domain = ? AND enabled = ?", parts[1], true).First(&domain).Error; err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "域名未启用"})
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		Email:   req.Email,
		IsAdmin: false,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	})

	tokenStr, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "生成令牌失败"})
	}

	return c.JSON(fiber.Map{"token": tokenStr})
}

func getEmails(c *fiber.Ctx) error {
	email := c.Locals("email").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit

	var total int64
	db.DB.Model(&model.Email{}).Where("to_addr = ?", email).Count(&total)

	var emails []model.Email
	if err := db.DB.Where("to_addr = ?", email).Select("id", "from_addr", "to_addr", "subject", "received_at").Order("received_at DESC").Limit(limit).Offset(offset).Find(&emails).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "查询失败"})
	}

	return c.JSON(fiber.Map{
		"emails": emails,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

func getEmailDetail(c *fiber.Ctx) error {
	email := c.Locals("email").(string)
	id := c.Params("id")

	var emailData model.Email
	if err := db.DB.Select("id", "from_addr", "to_addr", "subject", "body", "html_body", "received_at").Preload("Attachments", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "email_id", "filename", "content_type", "size")
	}).Where("id = ? AND to_addr = ?", id, email).First(&emailData).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "邮件不存在"})
	}

	type AttachmentInfo struct {
		ID          uint   `json:"id"`
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
		Size        int64  `json:"size"`
	}

	attachments := make([]AttachmentInfo, len(emailData.Attachments))
	for i, att := range emailData.Attachments {
		attachments[i] = AttachmentInfo{
			ID:          att.ID,
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Size:        att.Size,
		}
	}

	return c.JSON(fiber.Map{
		"id":          emailData.ID,
		"from_addr":   emailData.FromAddr,
		"to_addr":     emailData.ToAddr,
		"subject":     emailData.Subject,
		"body":        emailData.Body,
		"html_body":   emailData.HtmlBody,
		"received_at": emailData.ReceivedAt,
		"attachments": attachments,
	})
}

func deleteEmail(c *fiber.Ctx) error {
	email := c.Locals("email").(string)
	id := c.Params("id")

	idInt, err := strconv.Atoi(id)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "无效的ID"})
	}

	result := db.DB.Where("id = ? AND to_addr = ?", idInt, email).Delete(&model.Email{})
	if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": "删除失败"})
	}
	if result.RowsAffected == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "邮件不存在"})
	}

	return c.JSON(fiber.Map{"message": "删除成功"})
}

func deletePageEmails(c *fiber.Ctx) error {
	email := c.Locals("email").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit

	var emailIDs []uint
	db.DB.Model(&model.Email{}).Where("to_addr = ?", email).Order("received_at DESC").Limit(limit).Offset(offset).Pluck("id", &emailIDs)

	if len(emailIDs) == 0 {
		return c.JSON(fiber.Map{"message": "没有邮件可删除", "count": 0})
	}

	result := db.DB.Where("id IN ? AND to_addr = ?", emailIDs, email).Delete(&model.Email{})
	if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": "删除失败"})
	}

	return c.JSON(fiber.Map{"message": "删除成功", "count": result.RowsAffected})
}

func deleteMonthEmails(c *fiber.Ctx) error {
	email := c.Locals("email").(string)

	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	result := db.DB.Where("to_addr = ? AND received_at >= ? AND received_at < ?", email, startOfMonth, endOfMonth).Delete(&model.Email{})
	if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": "删除失败"})
	}

	return c.JSON(fiber.Map{"message": "删除成功", "count": result.RowsAffected})
}

func deleteAllEmails(c *fiber.Ctx) error {
	email := c.Locals("email").(string)

	result := db.DB.Where("to_addr = ?", email).Delete(&model.Email{})
	if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": "删除失败"})
	}

	return c.JSON(fiber.Map{"message": "删除成功", "count": result.RowsAffected})
}

func getAttachment(c *fiber.Ctx) error {
	email := c.Locals("email").(string)
	id := c.Params("id")

	var att model.Attachment
	if err := db.DB.Joins("JOIN emails ON emails.id = attachments.email_id").
		Where("attachments.id = ? AND emails.to_addr = ?", id, email).
		First(&att).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "附件不存在"})
	}

	c.Set("Content-Disposition", "attachment; filename="+att.Filename)
	c.Set("Content-Type", att.ContentType)
	return c.Send(att.Data)
}

func adminLogin(c *fiber.Ctx) error {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&req); err != nil || req.Username == "" || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{"error": "无效的请求"})
	}

	var admin model.Admin
	if err := db.DB.Where("username = ?", req.Username).First(&admin).Error; err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "用户名或密码错误"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(req.Password)); err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "用户名或密码错误"})
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		Email:   admin.Username,
		IsAdmin: true,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	})

	tokenStr, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "生成令牌失败"})
	}

	return c.JSON(fiber.Map{"token": tokenStr})
}

func getDomains(c *fiber.Ctx) error {
	var domains []model.Domain
	if err := db.DB.Order("created_at DESC").Find(&domains).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "查询失败"})
	}

	return c.JSON(domains)
}

func createDomain(c *fiber.Ctx) error {
	var req struct {
		Domain  string `json:"domain"`
		Enabled bool   `json:"enabled"`
	}

	if err := c.BodyParser(&req); err != nil || req.Domain == "" {
		return c.Status(400).JSON(fiber.Map{"error": "无效的请求"})
	}

	domain := model.Domain{
		Domain:  req.Domain,
		Enabled: req.Enabled,
	}

	if err := db.DB.Create(&domain).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "创建失败"})
	}

	return c.JSON(domain)
}

func updateDomain(c *fiber.Ctx) error {
	id := c.Params("id")

	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "无效的请求"})
	}

	if err := db.DB.Model(&model.Domain{}).Where("id = ?", id).Update("enabled", req.Enabled).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "更新失败"})
	}

	return c.JSON(fiber.Map{"message": "更新成功"})
}

func deleteDomain(c *fiber.Ctx) error {
	id := c.Params("id")

	idInt, err := strconv.Atoi(id)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "无效的ID"})
	}

	if err := db.DB.Delete(&model.Domain{}, idInt).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "删除失败"})
	}

	return c.JSON(fiber.Map{"message": "删除成功"})
}

// GetSSEManager 返回全局 SSE 管理器，供 SMTP 服务器调用
func GetSSEManager() *SSEManager {
	return sseManager
}
