package api

import (
	"bytes"
	"embed"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/valyala/fasthttp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"mailbox/internal/db"
	"mailbox/internal/model"
)

//go:embed web/dist/*
var testWebFS embed.FS

func setupTestDB(t *testing.T) *gorm.DB {
	dsn := "file:api_test_" + strconv.FormatInt(time.Now().UnixNano(), 10) + "?mode=memory&cache=shared&_fk=1"
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db failed: %v", err)
	}

	if err := gdb.AutoMigrate(&model.Domain{}, &model.Email{}, &model.Attachment{}, &model.Admin{}, &model.Mailbox{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	db.DB = gdb
	return gdb
}

func newTestApp() *fiber.App {
	app := fiber.New()
	api := app.Group("/api")
	api.Post("/login", userLogin)
	api.Get("/emails", getEmails)
	api.Get("/emails/stream", emailStream)
	api.Delete("/emails/page", deletePageEmails)
	api.Delete("/emails/month/all", deleteMonthEmails)
	api.Delete("/emails/all", deleteAllEmails)
	api.Get("/emails/:id", getEmailDetail)
	api.Delete("/emails/:id", deleteEmail)
	api.Get("/attachments/:id", getAttachment)

	admin := api.Group("/admin")
	admin.Post("/login", adminLogin)
	admin.Get("/domains", authMiddleware(true), getDomains)
	admin.Post("/domains", authMiddleware(true), createDomain)
	admin.Put("/domains/:id", authMiddleware(true), updateDomain)
	admin.Delete("/domains/:id", authMiddleware(true), deleteDomain)
	admin.Get("/mailboxes", authMiddleware(true), getMailboxes)
	admin.Delete("/mailboxes/:id", authMiddleware(true), deleteMailbox)
	admin.Delete("/mailboxes/domain/:domain", authMiddleware(true), deleteMailboxesByDomain)

	return app
}

func makeToken(t *testing.T, email string, isAdmin bool) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		Email:   email,
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	})
	str, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		t.Fatalf("make token failed: %v", err)
	}
	return str
}

func doJSONRequest(t *testing.T, app *fiber.App, method, url string, body any, token string) *http.Response {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode request failed: %v", err)
		}
	}

	req := httptest.NewRequest(method, url, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

func TestUserLoginAndEmailFlow(t *testing.T) {
	setupTestDB(t)
	jwtSecret = "test-secret"
	app := newTestApp()

	if err := db.DB.Create(&model.Domain{Domain: "cc.com", Enabled: true}).Error; err != nil {
		t.Fatalf("create domain failed: %v", err)
	}

	resp := doJSONRequest(t, app, http.MethodPost, "/api/login", map[string]string{"email": "test@cc.com"}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status code: %d", resp.StatusCode)
	}

	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decode login response failed: %v", err)
	}
	if loginResp.Token == "" {
		t.Fatalf("missing token")
	}

	emails := []model.Email{
		{FromAddr: "a@x.com", ToAddr: "test@cc.com", Subject: "s1", ReceivedAt: time.Now().Add(-1 * time.Hour)},
		{FromAddr: "b@x.com", ToAddr: "test@cc.com", Subject: "s2", ReceivedAt: time.Now().Add(-2 * time.Hour)},
		{FromAddr: "c@x.com", ToAddr: "test@cc.com", Subject: "s3", ReceivedAt: time.Now().Add(-3 * time.Hour)},
	}
	if err := db.DB.Create(&emails).Error; err != nil {
		t.Fatalf("create emails failed: %v", err)
	}

	resp = doJSONRequest(t, app, http.MethodGet, "/api/emails?email=test@cc.com&page=1&limit=2", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list emails status: %d", resp.StatusCode)
	}

	var listResp struct {
		Emails []model.Email `json:"emails"`
		Total  int64         `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list response failed: %v", err)
	}
	if listResp.Total != 3 || len(listResp.Emails) != 2 {
		t.Fatalf("unexpected list result: total=%d len=%d", listResp.Total, len(listResp.Emails))
	}
}

func TestUserLoginErrors(t *testing.T) {
	setupTestDB(t)
	jwtSecret = "test-secret"
	app := newTestApp()

	resp := doJSONRequest(t, app, http.MethodPost, "/api/login", map[string]string{"email": "bad"}, "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid email status: %d", resp.StatusCode)
	}

	if err := db.DB.Create(&model.Domain{Domain: "cc.com"}).Error; err != nil {
		t.Fatalf("create domain failed: %v", err)
	}
	if err := db.DB.Model(&model.Domain{}).Where("domain = ?", "cc.com").Update("enabled", false).Error; err != nil {
		t.Fatalf("disable domain failed: %v", err)
	}
	resp = doJSONRequest(t, app, http.MethodPost, "/api/login", map[string]string{"email": "test@cc.com"}, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("disabled domain status: %d", resp.StatusCode)
	}
}

func TestEmailDetailAndAttachment(t *testing.T) {
	setupTestDB(t)
	jwtSecret = "test-secret"
	app := newTestApp()

	email := model.Email{FromAddr: "a@x.com", ToAddr: "test@cc.com", Subject: "detail", Body: "hello"}
	if err := db.DB.Create(&email).Error; err != nil {
		t.Fatalf("create email failed: %v", err)
	}
	att := model.Attachment{EmailID: email.ID, Filename: "../evil\nname.txt", ContentType: "text/plain", Size: 3, Data: []byte("abc")}
	if err := db.DB.Create(&att).Error; err != nil {
		t.Fatalf("create attachment failed: %v", err)
	}

	resp := doJSONRequest(t, app, http.MethodGet, "/api/emails/"+idToString(email.ID)+"?email=test@cc.com", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get email detail failed: %d", resp.StatusCode)
	}

	var detailResp struct {
		ID          uint `json:"id"`
		Attachments []struct {
			ID uint `json:"id"`
		} `json:"attachments"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&detailResp); err != nil {
		t.Fatalf("decode detail failed: %v", err)
	}
	if detailResp.ID != email.ID || len(detailResp.Attachments) != 1 {
		t.Fatalf("unexpected attachments: %d", len(detailResp.Attachments))
	}

	resp = doJSONRequest(t, app, http.MethodGet, "/api/attachments/"+idToString(att.ID)+"?email=test@cc.com", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get attachment failed: %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing nosniff header")
	}
	if cd := resp.Header.Get("Content-Disposition"); cd == "" || bytes.Contains([]byte(cd), []byte("\n")) {
		t.Fatalf("unsafe content disposition: %s", cd)
	}
}

func TestDeleteEmailAndMailboxCleanup(t *testing.T) {
	setupTestDB(t)
	jwtSecret = "test-secret"
	app := newTestApp()

	email := model.Email{FromAddr: "a@x.com", ToAddr: "test@cc.com", Subject: "del"}
	if err := db.DB.Create(&email).Error; err != nil {
		t.Fatalf("create email failed: %v", err)
	}
	mailbox := model.Mailbox{Address: "test@cc.com", Domain: "cc.com", EmailCount: 1, LastEmail: time.Now()}
	if err := db.DB.Create(&mailbox).Error; err != nil {
		t.Fatalf("create mailbox failed: %v", err)
	}

	resp := doJSONRequest(t, app, http.MethodDelete, "/api/emails/"+idToString(email.ID)+"?email=test@cc.com", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete email failed: %d", resp.StatusCode)
	}

	var count int64
	if err := db.DB.Model(&model.Email{}).Where("to_addr = ?", "test@cc.com").Count(&count).Error; err != nil {
		t.Fatalf("count emails failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("email not deleted")
	}
	if err := db.DB.Where("address = ?", "test@cc.com").First(&model.Mailbox{}).Error; err == nil {
		t.Fatalf("mailbox not cleaned")
	}
}

func TestDeletePageMonthAll(t *testing.T) {
	setupTestDB(t)
	jwtSecret = "test-secret"
	app := newTestApp()

	emails := []model.Email{
		{FromAddr: "a@x.com", ToAddr: "test@cc.com", Subject: "p1", ReceivedAt: time.Now().Add(-1 * time.Hour)},
		{FromAddr: "b@x.com", ToAddr: "test@cc.com", Subject: "p2", ReceivedAt: time.Now().Add(-2 * time.Hour)},
		{FromAddr: "c@x.com", ToAddr: "test@cc.com", Subject: "p3", ReceivedAt: time.Now().AddDate(0, -1, 0)},
	}
	if err := db.DB.Create(&emails).Error; err != nil {
		t.Fatalf("create emails failed: %v", err)
	}

	resp := doJSONRequest(t, app, http.MethodDelete, "/api/emails/page?email=test@cc.com&page=1&limit=2", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete page failed: %d", resp.StatusCode)
	}

	resp = doJSONRequest(t, app, http.MethodDelete, "/api/emails/month/all?email=test@cc.com", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete month failed: %d", resp.StatusCode)
	}

	resp = doJSONRequest(t, app, http.MethodDelete, "/api/emails/all?email=test@cc.com", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete all failed: %d", resp.StatusCode)
	}
}

func TestAdminFlows(t *testing.T) {
	setupTestDB(t)
	jwtSecret = "test-secret"
	app := newTestApp()

	pass, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate password failed: %v", err)
	}
	if err := db.DB.Create(&model.Admin{Username: "admin", Password: string(pass)}).Error; err != nil {
		t.Fatalf("create admin failed: %v", err)
	}

	resp := doJSONRequest(t, app, http.MethodPost, "/api/admin/login", map[string]string{"username": "admin", "password": "admin123"}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin login failed: %d", resp.StatusCode)
	}

	resp = doJSONRequest(t, app, http.MethodPost, "/api/admin/login", map[string]string{"username": "admin", "password": "bad"}, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("admin bad password status: %d", resp.StatusCode)
	}

	token := makeToken(t, "admin", true)
	resp = doJSONRequest(t, app, http.MethodPost, "/api/admin/domains", map[string]any{"domain": "cc.com", "enabled": true}, token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create domain failed: %d", resp.StatusCode)
	}

	resp = doJSONRequest(t, app, http.MethodGet, "/api/admin/domains", nil, token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get domains failed: %d", resp.StatusCode)
	}

	resp = doJSONRequest(t, app, http.MethodPut, "/api/admin/domains/1", map[string]any{"enabled": false}, token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update domain failed: %d", resp.StatusCode)
	}

	if err := db.DB.Create(&model.Mailbox{Address: "a@cc.com", Domain: "cc.com"}).Error; err != nil {
		t.Fatalf("create mailbox failed: %v", err)
	}
	resp = doJSONRequest(t, app, http.MethodDelete, "/api/admin/domains/1", nil, token)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("delete domain should fail: %d", resp.StatusCode)
	}

	resp = doJSONRequest(t, app, http.MethodGet, "/api/admin/mailboxes?domain=cc.com&page=1&limit=10", nil, token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get mailboxes failed: %d", resp.StatusCode)
	}

	resp = doJSONRequest(t, app, http.MethodDelete, "/api/admin/mailboxes/1", nil, token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete mailbox failed: %d", resp.StatusCode)
	}

	resp = doJSONRequest(t, app, http.MethodDelete, "/api/admin/domains/1", nil, token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete domain failed: %d", resp.StatusCode)
	}
}

func TestDeleteMailboxesByDomain(t *testing.T) {
	setupTestDB(t)
	jwtSecret = "test-secret"
	app := newTestApp()

	if err := db.DB.Create(&model.Domain{Domain: "cc.com", Enabled: true}).Error; err != nil {
		t.Fatalf("create domain failed: %v", err)
	}
	if err := db.DB.Create(&model.Mailbox{Address: "a@cc.com", Domain: "cc.com"}).Error; err != nil {
		t.Fatalf("create mailbox failed: %v", err)
	}
	if err := db.DB.Create(&model.Email{FromAddr: "x@x.com", ToAddr: "a@cc.com", Subject: "s"}).Error; err != nil {
		t.Fatalf("create email failed: %v", err)
	}

	token := makeToken(t, "admin", true)
	resp := doJSONRequest(t, app, http.MethodDelete, "/api/admin/mailboxes/domain/cc.com", nil, token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete domain mailboxes failed: %d", resp.StatusCode)
	}

	resp = doJSONRequest(t, app, http.MethodDelete, "/api/admin/mailboxes/domain/cc.com", nil, token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete empty domain mailboxes failed: %d", resp.StatusCode)
	}
}

func TestSSEManager(t *testing.T) {
	mgr := &SSEManager{clients: make(map[string][]*sseClient)}
	client := &sseClient{email: "test@cc.com", ch: make(chan []byte, 1)}
	mgr.register("test@cc.com", client)

	email := &model.Email{ID: 1, FromAddr: "a@x.com", Subject: "s", ReceivedAt: time.Now()}
	mgr.NotifyNewEmail("test@cc.com", email)
	select {
	case <-client.ch:
	default:
		t.Fatalf("no sse message")
	}

	mgr.unregister("test@cc.com", client)
}

func TestSSEManagerConcurrent(t *testing.T) {
	mgr := &SSEManager{clients: make(map[string][]*sseClient)}
	const workers = 20
	errCh := make(chan error, workers)

	for i := 0; i < workers; i++ {
		go func(idx int) {
			email := "user" + strconv.Itoa(idx%5) + "@cc.com"
			client := &sseClient{email: email, ch: make(chan []byte, 1)}
			mgr.register(email, client)
			mgr.NotifyNewEmail(email, &model.Email{ID: uint(idx + 1), FromAddr: "a@x.com", Subject: "s", ReceivedAt: time.Now()})
			mgr.unregister(email, client)
			errCh <- nil
		}(i)
	}

	for i := 0; i < workers; i++ {
		if err := <-errCh; err != nil {
			t.Fatalf("concurrent sse failed: %v", err)
		}
	}
}

func TestSetupAppStatic(t *testing.T) {
	jwtSecret = "test-secret"
	app := SetupApp(testWebFS, jwtSecret)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte("test index")) {
		t.Fatalf("unexpected index content")
	}

	req = httptest.NewRequest(http.MethodGet, "/not-found", nil)
	resp, err = app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected fallback status: %d", resp.StatusCode)
	}
}

func TestGetContentType(t *testing.T) {
	cases := map[string]string{
		"app.js":    "application/javascript",
		"app.css":   "text/css",
		"index":     "application/octet-stream",
		"page.html": "text/html",
		"data.json": "application/json",
	}
	for path, expected := range cases {
		if got := getContentType(path); got != expected {
			t.Fatalf("content type mismatch for %s: %s", path, got)
		}
	}
}

func TestAuthMiddleware(t *testing.T) {
	jwtSecret = "test-secret"
	app := fiber.New()
	app.Get("/user", authMiddleware(false), func(c *fiber.Ctx) error {
		return c.SendString(c.Locals("email").(string))
	})
	app.Get("/admin", authMiddleware(true), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/user", nil)
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("missing token status: %d", resp.StatusCode)
	}

	req = httptest.NewRequest(http.MethodGet, "/user?token=bad", nil)
	resp, _ = app.Test(req, -1)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("invalid token status: %d", resp.StatusCode)
	}

	userToken := makeToken(t, "user@cc.com", false)
	req = httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	resp, _ = app.Test(req, -1)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("non-admin status: %d", resp.StatusCode)
	}

	adminToken := makeToken(t, "admin", true)
	req = httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, _ = app.Test(req, -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin status: %d", resp.StatusCode)
	}
}

func TestEmailStreamMissingEmail(t *testing.T) {
	jwtSecret = "test-secret"
	app := fiber.New()
	app.Get("/api/emails/stream", emailStream)
	req := httptest.NewRequest(http.MethodGet, "/api/emails/stream", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestEmailStreamFlow(t *testing.T) {
	jwtSecret = "test-secret"
	app := fiber.New()
	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod(http.MethodGet)
	reqCtx.Request.SetRequestURI("/api/emails/stream?email=test@cc.com")
	ctx := app.AcquireCtx(reqCtx)
	defer app.ReleaseCtx(ctx)

	if err := emailStream(ctx); err != nil {
		t.Fatalf("email stream failed: %v", err)
	}

	sseManager.mu.RLock()
	client := sseManager.clients["test@cc.com"][0]
	sseManager.mu.RUnlock()

	stream := ctx.Context().Response.BodyStream()
	var closer io.Closer
	if c, ok := stream.(io.Closer); ok {
		closer = c
	}
	go func() {
		client.ch <- []byte("data: {\"ID\":1}\n\n")
	}()

	buf := make([]byte, 128)
	if _, err := stream.Read(buf); err != nil && err != io.EOF {
		t.Fatalf("read stream failed: %v", err)
	}
	if closer != nil {
		_ = closer.Close()
	}
}

func TestCleanupMailboxTx(t *testing.T) {
	setupTestDB(t)
	mailbox := model.Mailbox{Address: "test@cc.com", Domain: "cc.com", EmailCount: 1, LastEmail: time.Now().Add(-2 * time.Hour)}
	if err := db.DB.Create(&mailbox).Error; err != nil {
		t.Fatalf("create mailbox failed: %v", err)
	}
	emails := []model.Email{
		{FromAddr: "a@x.com", ToAddr: "test@cc.com", Subject: "s1", ReceivedAt: time.Now().Add(-1 * time.Hour)},
		{FromAddr: "b@x.com", ToAddr: "test@cc.com", Subject: "s2", ReceivedAt: time.Now()},
	}
	if err := db.DB.Create(&emails).Error; err != nil {
		t.Fatalf("create emails failed: %v", err)
	}

	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		return cleanupMailboxTx(tx, "test@cc.com")
	}); err != nil {
		t.Fatalf("cleanup mailbox failed: %v", err)
	}

	var updated model.Mailbox
	if err := db.DB.Where("address = ?", "test@cc.com").First(&updated).Error; err != nil {
		t.Fatalf("fetch mailbox failed: %v", err)
	}
	if updated.EmailCount != 2 {
		t.Fatalf("unexpected email count: %d", updated.EmailCount)
	}

	cleanupMailbox("test@cc.com")
}

func TestInvalidRequests(t *testing.T) {
	setupTestDB(t)
	jwtSecret = "test-secret"
	app := newTestApp()

	resp := doJSONRequest(t, app, http.MethodGet, "/api/emails", nil, "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing email status: %d", resp.StatusCode)
	}

	resp = doJSONRequest(t, app, http.MethodGet, "/api/emails/1", nil, "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing email detail status: %d", resp.StatusCode)
	}

	resp = doJSONRequest(t, app, http.MethodDelete, "/api/emails/notint?email=test@cc.com", nil, "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid id status: %d", resp.StatusCode)
	}

	token := makeToken(t, "admin", true)
	resp = doJSONRequest(t, app, http.MethodPost, "/api/admin/domains", map[string]any{}, token)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("create domain invalid status: %d", resp.StatusCode)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/admin/domains/1", strings.NewReader("invalid"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, _ = app.Test(req, -1)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("update domain invalid status: %d", resp.StatusCode)
	}
}

func TestDeleteErrorsAndGetSSEManager(t *testing.T) {
	setupTestDB(t)
	jwtSecret = "test-secret"
	app := newTestApp()

	resp := doJSONRequest(t, app, http.MethodDelete, "/api/emails/999?email=test@cc.com", nil, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("delete missing email status: %d", resp.StatusCode)
	}

	token := makeToken(t, "admin", true)
	resp = doJSONRequest(t, app, http.MethodDelete, "/api/admin/domains/999", nil, token)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("delete missing domain status: %d", resp.StatusCode)
	}

	resp = doJSONRequest(t, app, http.MethodDelete, "/api/admin/mailboxes/999", nil, token)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("delete missing mailbox status: %d", resp.StatusCode)
	}

	if GetSSEManager() == nil {
		t.Fatalf("GetSSEManager returned nil")
	}
}

func idToString(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}
