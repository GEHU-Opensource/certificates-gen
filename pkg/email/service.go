package email

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type Service struct {
	sendGridKey string
	fromEmail   string
	fromName    string
}

func NewService(sendGridKey, fromEmail, fromName string) *Service {
	return &Service{
		sendGridKey: sendGridKey,
		fromEmail:   fromEmail,
		fromName:    fromName,
	}
}

func (s *Service) SendEmail(to, subject, bodyHTML, bodyText string) error {
	from := mail.NewEmail(s.fromName, s.fromEmail)
	toEmail := mail.NewEmail("", to)
	message := mail.NewSingleEmail(from, subject, toEmail, bodyText, bodyHTML)

	client := sendgrid.NewSendClient(s.sendGridKey)
	response, err := client.Send(message)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	if response.StatusCode >= 400 {
		return fmt.Errorf("email send failed with status %d: %s", response.StatusCode, response.Body)
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
