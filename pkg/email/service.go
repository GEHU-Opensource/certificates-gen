package email

import (
	"bytes"
	"fmt"
	"html/template"
	"net/mail"
	"net/smtp"
	"strings"
)

type Service struct {
	smtpHost     string
	smtpPort     int
	smtpUser     string
	smtpPassword string
	fromEmail    string
	fromName     string
}

func NewService(smtpHost string, smtpPort int, smtpUser, smtpPassword, fromEmail, fromName string) *Service {
	return &Service{
		smtpHost:     smtpHost,
		smtpPort:     smtpPort,
		smtpUser:     smtpUser,
		smtpPassword: smtpPassword,
		fromEmail:    fromEmail,
		fromName:     fromName,
	}
}

func (s *Service) SendEmail(to, subject, bodyHTML, bodyText string) error {
	// Validate email addresses
	fromAddr, err := mail.ParseAddress(s.fromEmail)
	if err != nil {
		return fmt.Errorf("invalid from email address: %w", err)
	}

	toAddr, err := mail.ParseAddress(to)
	if err != nil {
		return fmt.Errorf("invalid to email address: %w", err)
	}

	// Set from name if provided
	from := fromAddr.Address
	if s.fromName != "" {
		from = fmt.Sprintf("%s <%s>", s.fromName, fromAddr.Address)
	}

	// Build email message
	msg := bytes.Buffer{}
	msg.WriteString(fmt.Sprintf("From: %s\r\n", from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", toAddr.Address))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(bodyHTML)

	// SMTP server address
	addr := fmt.Sprintf("%s:%d", s.smtpHost, s.smtpPort)

	// Authentication
	auth := smtp.PlainAuth("", s.smtpUser, s.smtpPassword, s.smtpHost)

	// Send email
	err = smtp.SendMail(addr, auth, fromAddr.Address, []string{toAddr.Address}, msg.Bytes())
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (s *Service) SendWithTemplate(to, subject, templateHTML string, data map[string]interface{}) error {
	tmpl, err := template.New("email").Parse(templateHTML)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	bodyHTML := buf.String()
	bodyText := s.htmlToText(bodyHTML)

	return s.SendEmail(to, subject, bodyHTML, bodyText)
}

func (s *Service) htmlToText(html string) string {
	text := html
	text = strings.ReplaceAll(text, "<br>", "\n")
	text = strings.ReplaceAll(text, "<br/>", "\n")
	text = strings.ReplaceAll(text, "<p>", "")
	text = strings.ReplaceAll(text, "</p>", "\n\n")
	text = strings.ReplaceAll(text, "<div>", "")
	text = strings.ReplaceAll(text, "</div>", "\n")
	text = strings.ReplaceAll(text, "<strong>", "")
	text = strings.ReplaceAll(text, "</strong>", "")
	text = strings.ReplaceAll(text, "<em>", "")
	text = strings.ReplaceAll(text, "</em>", "")
	return text
}
