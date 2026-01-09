package smtp

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/jhillyerd/enmime"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"mailbox/internal/api"
	"mailbox/internal/db"
	"mailbox/internal/model"
)

type Backend struct{}

const maxAttachmentSize = 10 * 1024 * 1024

func validateAttachmentSizes(env *enmime.Envelope) error {
	for _, att := range env.Attachments {
		if len(att.Content) > maxAttachmentSize {
			return fmt.Errorf("attachment too large: %s", att.FileName)
		}
	}
	for _, att := range env.Inlines {
		if len(att.Content) > maxAttachmentSize {
			return fmt.Errorf("inline attachment too large: %s", att.FileName)
		}
	}
	return nil
}

func (b *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &Session{}, nil
}

type Session struct {
	from string
	to   []string
}

func (s *Session) AuthPlain(username, password string) error {
	return nil
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	s.from = from
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	domain := strings.Split(to, "@")
	if len(domain) != 2 {
		return fmt.Errorf("invalid email address")
	}

	var d model.Domain
	if err := db.DB.Where("domain = ? AND enabled = ?", domain[1], true).First(&d).Error; err != nil {
		return fmt.Errorf("domain not accepted")
	}

	s.to = append(s.to, to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	env, err := enmime.ReadEnvelope(bytes.NewReader(data))
	if err != nil {
		return err
	}

	if err := validateAttachmentSizes(env); err != nil {
		return err
	}

	type notifyItem struct {
		to    string
		email *model.Email
	}
	var notifyList []notifyItem

	for _, to := range s.to {
		var createdEmail *model.Email
		if err := db.DB.Transaction(func(tx *gorm.DB) error {
			email := &model.Email{
				FromAddr: s.from,
				ToAddr:   to,
				Subject:  env.GetHeader("Subject"),
				Body:     env.Text,
				HtmlBody: env.HTML,
				RawData:  data,
			}

			if err := tx.Create(email).Error; err != nil {
				return err
			}

			if err := upsertMailboxTx(tx, to, email.ReceivedAt); err != nil {
				return err
			}

			for _, att := range env.Attachments {
				attachment := &model.Attachment{
					EmailID:     email.ID,
					Filename:    att.FileName,
					ContentType: att.ContentType,
					Size:        int64(len(att.Content)),
					Data:        att.Content,
				}
				if err := tx.Create(attachment).Error; err != nil {
					return err
				}
			}

			createdEmail = email
			return nil
		}); err != nil {
			return err
		}

		notifyList = append(notifyList, notifyItem{to: to, email: createdEmail})
	}

	for _, item := range notifyList {
		api.GetSSEManager().NotifyNewEmail(item.to, item.email)
	}

	return nil
}

func upsertMailboxTx(tx *gorm.DB, address string, receivedAt time.Time) error {
	parts := strings.Split(address, "@")
	if len(parts) != 2 {
		return nil
	}

	mailbox := model.Mailbox{
		Address:    address,
		Domain:     parts[1],
		EmailCount: 0,
		LastEmail:  receivedAt,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "address"}},
		DoNothing: true,
	}).Create(&mailbox).Error; err != nil {
		return err
	}

	var count int64
	if err := tx.Model(&model.Email{}).Where("to_addr = ?", address).Count(&count).Error; err != nil {
		return err
	}

	if err := tx.Model(&model.Mailbox{}).Where("address = ?", address).Updates(map[string]interface{}{
		"email_count": int(count),
		"last_email":  receivedAt,
	}).Error; err != nil {
		return err
	}

	return nil
}

func (s *Session) Reset() {
	s.from = ""
	s.to = nil
}

func (s *Session) Logout() error {
	return nil
}

func Start(port int) error {
	return newServer(port).ListenAndServe()
}

func newServer(port int) *smtp.Server {
	be := &Backend{}
	s := smtp.NewServer(be)
	s.Addr = fmt.Sprintf(":%d", port)
	s.Domain = "localhost"
	s.AllowInsecureAuth = true
	return s
}
