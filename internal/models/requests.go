package models

type GenerateCertificateRequest struct {
	TemplateID      uint          `json:"template_id" binding:"required"`
	Recipient       RecipientData `json:"recipient" binding:"required"`
	SendEmail       bool          `json:"send_email"`
	EmailTemplateID *uint         `json:"email_template_id"`
}

type RecipientData struct {
	Name      string                 `json:"name" binding:"required"`
	Email     string                 `json:"email" binding:"required,email"`
	Course    string                 `json:"course"`
	Event     string                 `json:"event"`
	Club      string                 `json:"club"`
	Date      string                 `json:"date"`
	StudentID string                 `json:"student_id"`
	Metadata  map[string]interface{} `json:"metadata"`
}

type BulkGenerateRequest struct {
	TemplateID      uint            `json:"template_id" binding:"required"`
	Recipients      []RecipientData `json:"recipients" binding:"required,min=1"`
	SendEmail       bool            `json:"send_email"`
	EmailTemplateID *uint           `json:"email_template_id"`
}

type CreateTemplateRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config" binding:"required"`
}

type CreateEmailTemplateRequest struct {
	Name     string `json:"name" binding:"required"`
	Subject  string `json:"subject" binding:"required"`
	BodyHTML string `json:"body_html" binding:"required"`
	BodyText string `json:"body_text"`
}

type CertificateResponse struct {
	ID          uint   `json:"id"`
	Status      string `json:"status"`
	FilePath    string `json:"file_path"`
	EmailSent   bool   `json:"email_sent"`
	DownloadURL string `json:"download_url,omitempty"`
}

type BatchStatusResponse struct {
	ID         uint    `json:"id"`
	TotalCount int     `json:"total_count"`
	Processed  int     `json:"processed"`
	Failed     int     `json:"failed"`
	Status     string  `json:"status"`
	Progress   float64 `json:"progress"`
}
