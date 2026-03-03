package notification

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"time"
)

// SMTPConfig SMTP configuration
type SMTPConfig struct {
	Host     string
	Port     int64
	Username string
	Password string
	From     string
}

// SMTPClient SMTP email client
type SMTPClient struct {
	Config SMTPConfig
}

// NewSMTPClient creates SMTP client instance
func NewSMTPClient(config SMTPConfig) *SMTPClient {
	return &SMTPClient{
		Config: config,
	}
}

// SendHTML sends HTML email via SMTP
func (s *SMTPClient) SendHTML(to, subject, htmlBody string) (string, error) {
	// Build MIME email message
	msg := "MIME-Version: 1.0\r\n"
	msg += "Content-Type: text/html; charset=\"UTF-8\"\r\n"
	msg += fmt.Sprintf("From: %s\r\n", s.Config.From)
	msg += fmt.Sprintf("To: %s\r\n", to)
	msg += fmt.Sprintf("Subject: %s\r\n", subject)
	msg += "\r\n" + htmlBody

	addr := fmt.Sprintf("%s:%d", s.Config.Host, s.Config.Port)
	auth := smtp.PlainAuth("", s.Config.Username, s.Config.Password, s.Config.Host)

	// Configure TLS
	tlsConfig := &tls.Config{
		ServerName:         s.Config.Host,
		InsecureSkipVerify: false,
	}

	// Try to connect with TLS (port 465)
	if s.Config.Port == 465 {
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return "", fmt.Errorf("failed to dial SMTP server: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, s.Config.Host)
		if err != nil {
			return "", fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer client.Close()

		if err = client.Auth(auth); err != nil {
			return "", fmt.Errorf("failed to authenticate: %w", err)
		}

		if err = client.Mail(s.Config.From); err != nil {
			return "", fmt.Errorf("failed to set sender: %w", err)
		}

		if err = client.Rcpt(to); err != nil {
			return "", fmt.Errorf("failed to set recipient: %w", err)
		}

		w, err := client.Data()
		if err != nil {
			return "", fmt.Errorf("failed to prepare data: %w", err)
		}
		defer w.Close()

		_, err = w.Write([]byte(msg))
		if err != nil {
			return "", fmt.Errorf("failed to write email content: %w", err)
		}

		client.Quit()
	} else {
		// Use STARTTLS (port 587)
		err := smtp.SendMail(addr, auth, s.Config.From, []string{to}, []byte(msg))
		if err != nil {
			return "", fmt.Errorf("failed to send email: %w", err)
		}
	}

	// Generate a simple messageID for SMTP (timestamp-based)
	// SMTP doesn't return a messageID, so we generate one for tracking
	messageID := fmt.Sprintf("smtp-%d", time.Now().UnixNano())
	return messageID, nil
}
