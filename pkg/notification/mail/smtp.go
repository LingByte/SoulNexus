// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package mail

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"time"
)

// SMTPConfig holds SMTP connection settings.
type SMTPConfig struct {
	Host     string
	Port     int64
	Username string
	Password string
	From     string
	FromName string
}

// SMTPClient implements MailProvider over SMTP.
type SMTPClient struct {
	Config SMTPConfig
	sender ParsedSender
}

// NewSMTPClient builds an SMTP mail provider.
func NewSMTPClient(config SMTPConfig) (*SMTPClient, error) {
	p, err := ParseMailSender(config.From, config.FromName)
	if err != nil {
		return nil, err
	}
	return &SMTPClient{Config: config, sender: p}, nil
}

// Kind implements MailProvider.
func (s *SMTPClient) Kind() string {
	return ProviderSMTP
}

// SendHTMLWith sends HTML mail with variable substitution.
func (s *SMTPClient) SendHTMLWith(to, subject, htmlBody string, vars map[string]any) (string, error) {
	return s.sendMail(to, subject, htmlBody, vars, "text/html; charset=\"UTF-8\"")
}

// SendTextWith sends plain text mail with variable substitution.
func (s *SMTPClient) SendTextWith(to, subject, textBody string, vars map[string]any) (string, error) {
	return s.sendMail(to, subject, textBody, vars, "text/plain; charset=\"UTF-8\"")
}

// sendMail is the shared implementation for SendHTMLWith and SendTextWith.
func (s *SMTPClient) sendMail(to, subject, body string, vars map[string]any, contentType string) (string, error) {
	msg := "MIME-Version: 1.0\r\n"
	msg += fmt.Sprintf("Content-Type: %s\r\n", contentType)
	msg += fmt.Sprintf("From: %s\r\n", s.sender.HeaderFrom)
	msg += fmt.Sprintf("To: %s\r\n", to)
	msg += fmt.Sprintf("Subject: %s\r\n", ReplacePlaceholders(subject, vars))
	msg += "\r\n" + ReplacePlaceholders(body, vars)

	addr := fmt.Sprintf("%s:%d", s.Config.Host, s.Config.Port)
	auth := smtp.PlainAuth("", s.Config.Username, s.Config.Password, s.Config.Host)
	env := s.sender.Envelope

	tlsConfig := &tls.Config{
		ServerName:         s.Config.Host,
		InsecureSkipVerify: false,
	}

	if s.Config.Port == 465 {
		return s.sendWithTLS(addr, auth, env, to, []byte(msg), tlsConfig)
	}
	return s.sendWithoutTLS(addr, auth, env, to, []byte(msg))
}

// sendWithTLS handles SMTP over TLS (port 465).
func (s *SMTPClient) sendWithTLS(addr string, auth smtp.Auth, from, to string, msg []byte, tlsConfig *tls.Config) (string, error) {
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return "", fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.Config.Host)
	if err != nil {
		return "", fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if err = client.Auth(auth); err != nil {
		return "", fmt.Errorf("smtp auth: %w", err)
	}
	if err = client.Mail(from); err != nil {
		return "", fmt.Errorf("smtp mail from: %w", err)
	}
	if err = client.Rcpt(to); err != nil {
		return "", fmt.Errorf("smtp rcpt: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return "", fmt.Errorf("smtp data: %w", err)
	}
	if _, err = w.Write(msg); err != nil {
		_ = w.Close()
		return "", fmt.Errorf("smtp write: %w", err)
	}
	if err = w.Close(); err != nil {
		return "", fmt.Errorf("smtp close writer: %w", err)
	}
	_ = client.Quit()
	return fmt.Sprintf("smtp-%d", time.Now().UnixNano()), nil
}

// sendWithoutTLS handles SMTP without TLS or with STARTTLS.
func (s *SMTPClient) sendWithoutTLS(addr string, auth smtp.Auth, from, to string, msg []byte) (string, error) {
	if err := smtp.SendMail(addr, auth, from, []string{to}, msg); err != nil {
		return "", fmt.Errorf("smtp send: %w", err)
	}
	return fmt.Sprintf("smtp-%d", time.Now().UnixNano()), nil
}
