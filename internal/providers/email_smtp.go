package providers

import (
	"bytes"
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

// SMTPProvider implements EmailProvider using basic SMTP auth.
type SMTPProvider struct {
	Host string
	Port int
	User string
	Pass string
	From string
}

func (p SMTPProvider) SendEmail(_ context.Context, to []string, subject string, body string) error {
	if len(to) == 0 {
		return nil
	}
	if p.Host == "" {
		return fmt.Errorf("smtp host not configured")
	}

	headers := map[string]string{
		"From":         p.From,
		"To":           strings.Join(to, ","),
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/plain; charset=\"utf-8\"",
	}

	var msg bytes.Buffer
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(body)

	addr := fmt.Sprintf("%s:%d", p.Host, p.Port)
	var auth smtp.Auth
	if p.User != "" || p.Pass != "" {
		auth = smtp.PlainAuth("", p.User, p.Pass, p.Host)
	}

	return smtp.SendMail(addr, auth, p.From, to, msg.Bytes())
}
