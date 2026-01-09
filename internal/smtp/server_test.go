package smtp

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/jhillyerd/enmime"
	"gorm.io/gorm"
	"mailbox/internal/db"
	"mailbox/internal/model"
)

func setupTestDB(t *testing.T) *gorm.DB {
	dsn := "file:smtp_test_" + time.Now().Format("20060102150405.000000000") + "?mode=memory&cache=shared&_fk=1"
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

func createDomain(t *testing.T, domain string, enabled bool) {
	if err := db.DB.Create(&model.Domain{Domain: domain}).Error; err != nil {
		t.Fatalf("create domain failed: %v", err)
	}
	if err := db.DB.Model(&model.Domain{}).Where("domain = ?", domain).Update("enabled", enabled).Error; err != nil {
		t.Fatalf("update domain failed: %v", err)
	}
}

func buildAttachmentEmail(subject, body, filename string, content []byte) []byte {
	boundary := "BOUNDARY123"
	var buf bytes.Buffer
	buf.WriteString("From: sender@example.com\r\n")
	buf.WriteString("To: test@cc.com\r\n")
	buf.WriteString("Subject: " + subject + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: multipart/mixed; boundary=\"" + boundary + "\"\r\n")
	buf.WriteString("\r\n")
	buf.WriteString("--" + boundary + "\r\n")
	buf.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n\r\n")
	buf.WriteString(body + "\r\n")
	buf.WriteString("--" + boundary + "\r\n")
	buf.WriteString("Content-Type: application/octet-stream\r\n")
	buf.WriteString("Content-Disposition: attachment; filename=\"" + filename + "\"\r\n")
	buf.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	encoded := base64.StdEncoding.EncodeToString(content)
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		buf.WriteString(encoded[i:end] + "\r\n")
	}
	buf.WriteString("--" + boundary + "--\r\n")
	return buf.Bytes()
}

func TestValidateAttachmentSizes(t *testing.T) {
	big := &enmime.Part{FileName: "big.bin", Content: make([]byte, maxAttachmentSize+1)}
	env := &enmime.Envelope{Attachments: []*enmime.Part{big}}
	if err := validateAttachmentSizes(env); err == nil {
		t.Fatalf("expected size error")
	}

	small := &enmime.Part{FileName: "small.bin", Content: make([]byte, maxAttachmentSize)}
	env = &enmime.Envelope{Attachments: []*enmime.Part{small}, Inlines: []*enmime.Part{small}}
	if err := validateAttachmentSizes(env); err != nil {
		t.Fatalf("unexpected size error: %v", err)
	}
}

func TestSessionRcptAndData(t *testing.T) {
	setupTestDB(t)
	createDomain(t, "cc.com", true)
	createDomain(t, "disabled.com", false)

	s := &Session{}
	if err := s.Rcpt("bad", nil); err == nil {
		t.Fatalf("expected invalid rcpt error")
	}
	if err := s.Rcpt("a@disabled.com", nil); err == nil {
		t.Fatalf("expected disabled domain error")
	}
	if err := s.Rcpt("test@cc.com", nil); err != nil {
		t.Fatalf("rcpt failed: %v", err)
	}

	if err := s.Mail("sender@example.com", nil); err != nil {
		t.Fatalf("mail failed: %v", err)
	}

	plain := "From: sender@example.com\r\nTo: test@cc.com\r\nSubject: hi\r\n\r\nhello"
	if err := s.Data(strings.NewReader(plain)); err != nil {
		t.Fatalf("data failed: %v", err)
	}

	var emailCount int64
	if err := db.DB.Model(&model.Email{}).Where("to_addr = ?", "test@cc.com").Count(&emailCount).Error; err != nil {
		t.Fatalf("count emails failed: %v", err)
	}
	if emailCount != 1 {
		t.Fatalf("unexpected email count: %d", emailCount)
	}

	var mailbox model.Mailbox
	if err := db.DB.Where("address = ?", "test@cc.com").First(&mailbox).Error; err != nil {
		t.Fatalf("mailbox not created: %v", err)
	}

	s.Reset()
	if s.from != "" || len(s.to) != 0 {
		t.Fatalf("reset failed")
	}
	if err := s.Logout(); err != nil {
		t.Fatalf("logout failed: %v", err)
	}
}

func TestSessionDataWithAttachment(t *testing.T) {
	setupTestDB(t)
	createDomain(t, "cc.com", true)

	s := &Session{}
	if err := s.Mail("sender@example.com", nil); err != nil {
		t.Fatalf("mail failed: %v", err)
	}
	if err := s.Rcpt("test@cc.com", nil); err != nil {
		t.Fatalf("rcpt failed: %v", err)
	}

	data := buildAttachmentEmail("sub", "body", "test.txt", []byte("hello"))
	if err := s.Data(bytes.NewReader(data)); err != nil {
		t.Fatalf("data failed: %v", err)
	}

	var attCount int64
	if err := db.DB.Model(&model.Attachment{}).Count(&attCount).Error; err != nil {
		t.Fatalf("count attachments failed: %v", err)
	}
	if attCount != 1 {
		t.Fatalf("unexpected attachment count: %d", attCount)
	}
}

func TestSessionDataConcurrent(t *testing.T) {
	setupTestDB(t)
	createDomain(t, "cc.com", true)

	const workers = 20
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		go func(idx int) {
			s := &Session{}
			if err := s.Mail("sender@example.com", nil); err != nil {
				errCh <- err
				return
			}
			if err := s.Rcpt("test@cc.com", nil); err != nil {
				errCh <- err
				return
			}
			msg := "From: sender@example.com\r\nTo: test@cc.com\r\nSubject: hi\r\n\r\nhello"
			errCh <- s.Data(strings.NewReader(msg))
		}(i)
	}

	for i := 0; i < workers; i++ {
		if err := <-errCh; err != nil {
			t.Fatalf("concurrent data failed: %v", err)
		}
	}

	var count int64
	if err := db.DB.Model(&model.Email{}).Where("to_addr = ?", "test@cc.com").Count(&count).Error; err != nil {
		t.Fatalf("count emails failed: %v", err)
	}
	if count != workers {
		t.Fatalf("unexpected email count: %d", count)
	}
}

func TestBackendNewSessionAndAuth(t *testing.T) {
	be := &Backend{}
	session, err := be.NewSession(nil)
	if err != nil {
		t.Fatalf("new session failed: %v", err)
	}
	s, ok := session.(*Session)
	if !ok {
		t.Fatalf("unexpected session type")
	}
	if err := s.AuthPlain("user", "pass"); err != nil {
		t.Fatalf("auth plain failed: %v", err)
	}
}

func TestNewServerConfig(t *testing.T) {
	server := newServer(2525)
	if server.Addr != ":2525" {
		t.Fatalf("unexpected addr: %s", server.Addr)
	}
	if server.Domain != "localhost" {
		t.Fatalf("unexpected domain: %s", server.Domain)
	}
	if !server.AllowInsecureAuth {
		t.Fatalf("allow insecure auth should be true")
	}
}

func TestUpsertMailboxTx(t *testing.T) {
	setupTestDB(t)
	now := time.Now()
	if err := db.DB.Create(&model.Email{FromAddr: "a@x.com", ToAddr: "test@cc.com", Subject: "s", ReceivedAt: now}).Error; err != nil {
		t.Fatalf("create email failed: %v", err)
	}

	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		return upsertMailboxTx(tx, "test@cc.com", now)
	}); err != nil {
		t.Fatalf("upsert mailbox failed: %v", err)
	}

	var mailbox model.Mailbox
	if err := db.DB.Where("address = ?", "test@cc.com").First(&mailbox).Error; err != nil {
		t.Fatalf("mailbox not found: %v", err)
	}
	if mailbox.EmailCount != 1 {
		t.Fatalf("unexpected email count: %d", mailbox.EmailCount)
	}

	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		return upsertMailboxTx(tx, "invalid", now)
	}); err != nil {
		t.Fatalf("unexpected error for invalid address: %v", err)
	}
}

func TestUpsertMailboxTxConcurrent(t *testing.T) {
	setupTestDB(t)
	now := time.Now()
	if err := db.DB.Create(&model.Email{FromAddr: "a@x.com", ToAddr: "test@cc.com", Subject: "s", ReceivedAt: now}).Error; err != nil {
		t.Fatalf("create email failed: %v", err)
	}

	const workers = 20
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		go func() {
			errCh <- db.DB.Transaction(func(tx *gorm.DB) error {
				return upsertMailboxTx(tx, "test@cc.com", now)
			})
		}()
	}

	for i := 0; i < workers; i++ {
		if err := <-errCh; err != nil {
			t.Fatalf("concurrent upsert failed: %v", err)
		}
	}

	var count int64
	if err := db.DB.Model(&model.Mailbox{}).Where("address = ?", "test@cc.com").Count(&count).Error; err != nil {
		t.Fatalf("count mailbox failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("unexpected mailbox count: %d", count)
	}
}
