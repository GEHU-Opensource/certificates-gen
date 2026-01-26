package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Certificate struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	TemplateID  uint           `gorm:"not null" json:"template_id"`
	RecipientID uint           `gorm:"not null" json:"recipient_id"`
	Status      string         `gorm:"not null;default:'pending'" json:"status"`
	FilePath    string         `json:"file_path"`
	EmailSent   bool           `gorm:"default:false" json:"email_sent"`
	EmailSentAt *time.Time     `json:"email_sent_at"`
	Metadata    datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Template  Template  `gorm:"foreignKey:TemplateID" json:"template,omitempty"`
	Recipient Recipient `gorm:"foreignKey:RecipientID" json:"recipient,omitempty"`
}

type Template struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"not null;unique" json:"name"`
	Description string    `json:"description"`
	Config      string    `gorm:"type:jsonb" json:"config"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Recipient struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"not null" json:"name"`
	Email     string         `gorm:"not null" json:"email"`
	Course    string         `json:"course"`
	Event     string         `json:"event"`
	Club      string         `json:"club"`
	Date      string         `json:"date"`
	StudentID string         `json:"student_id"`
	Metadata  datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type CertificateBatch struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	TemplateID uint           `gorm:"not null" json:"template_id"`
	TotalCount int            `gorm:"not null" json:"total_count"`
	Processed  int            `gorm:"default:0" json:"processed"`
	Failed     int            `gorm:"default:0" json:"failed"`
	Status     string         `gorm:"not null;default:'processing'" json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	Metadata   datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	Template   Template       `gorm:"foreignKey:TemplateID" json:"template,omitempty"`
}

type EmailTemplate struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null;unique" json:"name"`
	Subject   string    `gorm:"not null" json:"subject"`
	BodyHTML  string    `gorm:"type:text" json:"body_html"`
	BodyText  string    `gorm:"type:text" json:"body_text"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Certificate) TableName() string {
	return "certificates"
}

func (Template) TableName() string {
	return "templates"
}

func (Recipient) TableName() string {
	return "recipients"
}

func (CertificateBatch) TableName() string {
	return "certificate_batches"
}

func (EmailTemplate) TableName() string {
	return "email_templates"
}
