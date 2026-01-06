package smtp

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/emersion/go-smtp"
	"github.com/jhillyerd/enmime"
	"mailbox/internal/api"
	"mailbox/internal/db"
	"mailbox/internal/model"
)

type Backend struct{}

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

	for _, to := range s.to {
		email := &model.Email{
			FromAddr: s.from,
			ToAddr:   to,
			Subject:  env.GetHeader("Subject"),
			Body:     env.Text,
			HtmlBody: env.HTML,
			RawData:  data,
		}

		if err := db.DB.Create(email).Error; err != nil {
			return err
		}

		domain := strings.Split(to, "@")
		if len(domain) == 2 {
			var mailbox model.Mailbox
			result := db.DB.Where("address = ?", to).FirstOrCreate(&mailbox, model.Mailbox{
				Address:    to,
				Domain:     domain[1],
				EmailCount: 0,
				LastEmail:  email.ReceivedAt,
			})
			if result.Error != nil && !strings.Contains(result.Error.Error(), "Duplicate entry") {
				fmt.Printf("[Mailbox] 创建/查找邮箱失败: %v\n", result.Error)
			} else {
				fmt.Printf("[Mailbox] 邮箱记录: %s (ID: %d)\n", mailbox.Address, mailbox.ID)
			}

			var count int64
			db.DB.Model(&model.Email{}).Where("to_addr = ?", to).Count(&count)
			updateResult := db.DB.Model(&model.Mailbox{}).Where("address = ?", to).Updates(map[string]interface{}{
				"email_count": int(count),
				"last_email":  email.ReceivedAt,
			})
			if updateResult.Error != nil {
				fmt.Printf("[Mailbox] 更新邮箱统计失败: %v\n", updateResult.Error)
			} else {
				fmt.Printf("[Mailbox] 更新成功: 邮件数=%d\n", count)
			}
		}

		for _, att := range env.Attachments {
			attachment := &model.Attachment{
				EmailID:     email.ID,
				Filename:    att.FileName,
				ContentType: att.ContentType,
				Size:        int64(len(att.Content)),
				Data:        att.Content,
			}
			if err := db.DB.Create(attachment).Error; err != nil {
				return err
			}
		}

		api.GetSSEManager().NotifyNewEmail(to, email)
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
	be := &Backend{}
	s := smtp.NewServer(be)
	s.Addr = fmt.Sprintf(":%d", port)
	s.Domain = "localhost"
	s.AllowInsecureAuth = true
	return s.ListenAndServe()
}
