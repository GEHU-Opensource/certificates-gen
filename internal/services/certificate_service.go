package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"certificate-service/internal/models"
	"certificate-service/internal/queue"
	"certificate-service/internal/storage"
	"certificate-service/pkg/email"
	"certificate-service/pkg/pdf"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type CertificateService struct {
	db           *gorm.DB
	pdfGen       *pdf.HTMLGenerator
	emailService *email.Service
	storage      storage.Storage
	queue        *queue.Worker
}

func NewCertificateService(
	db *gorm.DB,
	pdfGen *pdf.HTMLGenerator,
	emailService *email.Service,
	storage storage.Storage,
	queue *queue.Worker,
) *CertificateService {
	service := &CertificateService{
		db:           db,
		pdfGen:       pdfGen,
		emailService: emailService,
		storage:      storage,
		queue:        queue,
	}

	queue.RegisterProcessor("generate_certificate", service.processCertificateJob)
	queue.RegisterProcessor("send_email", service.processEmailJob)

	return service
}

func (s *CertificateService) GenerateCertificate(ctx context.Context, req models.GenerateCertificateRequest) (*models.Certificate, error) {
	var template models.Template
	if err := s.db.Where("id = ? AND is_active = ?", req.TemplateID, true).First(&template).Error; err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	recipient := models.Recipient{
		Name:      req.Recipient.Name,
		Email:     req.Recipient.Email,
		Course:    req.Recipient.Course,
		Event:     req.Recipient.Event,
		Club:      req.Recipient.Club,
		Date:      req.Recipient.Date,
		StudentID: req.Recipient.StudentID,
	}

	if req.Recipient.Metadata != nil {
		metadataJSON, err := json.Marshal(req.Recipient.Metadata)
		if err == nil {
			recipient.Metadata = metadataJSON
		}
	}

	if err := s.db.Create(&recipient).Error; err != nil {
		return nil, fmt.Errorf("failed to create recipient: %w", err)
	}

	certificate := models.Certificate{
		TemplateID:  template.ID,
		RecipientID: recipient.ID,
		Status:      "pending",
	}

	if err := s.db.Create(&certificate).Error; err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	job := queue.Job{
		ID:        fmt.Sprintf("cert-%d", certificate.ID),
		Type:      "generate_certificate",
		CreatedAt: time.Now(),
		Data: map[string]interface{}{
			"certificate_id":    certificate.ID,
			"send_email":        req.SendEmail,
			"email_template_id": req.EmailTemplateID,
		},
	}

	if err := s.queue.Enqueue(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to enqueue job: %w", err)
	}

	return &certificate, nil
}

func (s *CertificateService) BulkGenerate(ctx context.Context, req models.BulkGenerateRequest) (*models.CertificateBatch, error) {
	var template models.Template
	if err := s.db.Where("id = ? AND is_active = ?", req.TemplateID, true).First(&template).Error; err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	batch := models.CertificateBatch{
		TemplateID: template.ID,
		TotalCount: len(req.Recipients),
		Status:     "processing",
	}

	if err := s.db.Create(&batch).Error; err != nil {
		return nil, fmt.Errorf("failed to create batch: %w", err)
	}

	var jobs []queue.Job
	for i, recipientData := range req.Recipients {
		recipient := models.Recipient{
			Name:      recipientData.Name,
			Email:     recipientData.Email,
			Course:    recipientData.Course,
			Event:     recipientData.Event,
			Club:      recipientData.Club,
			Date:      recipientData.Date,
			StudentID: recipientData.StudentID,
		}

		if recipientData.Metadata != nil {
			metadataJSON, err := json.Marshal(recipientData.Metadata)
			if err == nil {
				recipient.Metadata = metadataJSON
			}
		}

		if err := s.db.Create(&recipient).Error; err != nil {
			continue
		}

		certificate := models.Certificate{
			TemplateID:  template.ID,
			RecipientID: recipient.ID,
			Status:      "pending",
		}

		if err := s.db.Create(&certificate).Error; err != nil {
			continue
		}

		job := queue.Job{
			ID:        fmt.Sprintf("cert-%d-%d", batch.ID, i),
			Type:      "generate_certificate",
			CreatedAt: time.Now(),
			Data: map[string]interface{}{
				"certificate_id":    certificate.ID,
				"batch_id":          batch.ID,
				"send_email":        req.SendEmail,
				"email_template_id": req.EmailTemplateID,
			},
		}

		jobs = append(jobs, job)
	}

	if err := s.queue.EnqueueBatch(ctx, jobs); err != nil {
		return nil, fmt.Errorf("failed to enqueue batch: %w", err)
	}

	return &batch, nil
}

func (s *CertificateService) processCertificateJob(ctx context.Context, job queue.Job) error {
	var certID uint
	switch v := job.Data["certificate_id"].(type) {
	case float64:
		certID = uint(v)
	case int:
		certID = uint(v)
	case uint:
		certID = v
	default:
		return fmt.Errorf("invalid certificate_id in job data: %v", job.Data["certificate_id"])
	}

	var certificate models.Certificate
	if err := s.db.Preload("Template").Preload("Recipient").First(&certificate, certID).Error; err != nil {
		return fmt.Errorf("certificate not found: %w", err)
	}

	var templateConfig map[string]interface{}
	if certificate.Template.Config != "" {
		json.Unmarshal([]byte(certificate.Template.Config), &templateConfig)
	}

	templateName := "certificate.html"
	if name, ok := templateConfig["template_name"].(string); ok && name != "" {
		templateName = name
	}

	data := map[string]string{
		"name":          certificate.Recipient.Name,
		"email":         certificate.Recipient.Email,
		"course":        certificate.Recipient.Course,
		"event":         certificate.Recipient.Event,
		"club":          certificate.Recipient.Club,
		"date":          certificate.Recipient.Date,
		"student_id":    certificate.Recipient.StudentID,
		"signer1_name":  getStringFromMetadata(certificate.Recipient.Metadata, "signer1_name", ""),
		"signer1_title": getStringFromMetadata(certificate.Recipient.Metadata, "signer1_title", "Event Coordinator"),
		"signer2_name":  getStringFromMetadata(certificate.Recipient.Metadata, "signer2_name", ""),
		"signer2_title": getStringFromMetadata(certificate.Recipient.Metadata, "signer2_title", "Head Of Department\n(CSE)"),
		"signer3_name":  getStringFromMetadata(certificate.Recipient.Metadata, "signer3_name", ""),
		"signer3_title": getStringFromMetadata(certificate.Recipient.Metadata, "signer3_title", "Director,\nBhimtal Campus"),
	}

	if sideDesign, ok := templateConfig["side_design"].(string); ok {
		data["side_design"] = sideDesign
	}
	if orgLogo, ok := templateConfig["org_logo"].(string); ok {
		data["org_logo"] = orgLogo
	}
	if clubLogo, ok := templateConfig["club_logo"].(string); ok {
		data["club_logo"] = clubLogo
	}
	if sig1, ok := templateConfig["signature1"].(string); ok {
		data["signature1"] = sig1
	}
	if sig2, ok := templateConfig["signature2"].(string); ok {
		data["signature2"] = sig2
	}
	if sig3, ok := templateConfig["signature3"].(string); ok {
		data["signature3"] = sig3
	}
	if sig4, ok := templateConfig["signature4"].(string); ok {
		data["signature4"] = sig4
	}

	pdfData, err := s.pdfGen.GenerateWithTemplate(templateName, data)
	if err != nil {
		s.db.Model(&certificate).Update("status", "failed")
		s.updateBatchOnFailure(job)
		return fmt.Errorf("failed to generate PDF: %w", err)
	}

	eventName := certificate.Recipient.Event
	if eventName == "" {
		eventName = "default"
	}

	filePath, err := s.storage.Save(pdfData, eventName, certificate.Recipient.Name, certificate.Recipient.Email)
	if err != nil {
		s.db.Model(&certificate).Update("status", "failed")
		s.updateBatchOnFailure(job)
		return fmt.Errorf("failed to save certificate: %w", err)
	}

	certificate.Status = "completed"
	certificate.FilePath = filePath
	if err := s.db.Save(&certificate).Error; err != nil {
		return fmt.Errorf("failed to update certificate: %w", err)
	}

	sendEmail, _ := job.Data["send_email"].(bool)
	if sendEmail {
		emailJob := queue.Job{
			ID:        fmt.Sprintf("email-%d", certificate.ID),
			Type:      "send_email",
			CreatedAt: time.Now(),
			Data: map[string]interface{}{
				"certificate_id":    certificate.ID,
				"email_template_id": job.Data["email_template_id"],
			},
		}
		s.queue.Enqueue(ctx, emailJob)
	}

	s.updateBatchOnSuccess(job)

	return nil
}

func (s *CertificateService) updateBatchOnSuccess(job queue.Job) {
	var batchID uint
	switch v := job.Data["batch_id"].(type) {
	case float64:
		batchID = uint(v)
	case int:
		batchID = uint(v)
	case uint:
		batchID = v
	default:
		return
	}

	if batchID == 0 {
		return
	}

	var batch models.CertificateBatch
	if err := s.db.First(&batch, batchID).Error; err == nil {
		batch.Processed++
		if batch.Processed+batch.Failed >= batch.TotalCount {
			batch.Status = "completed"
		}
		s.db.Save(&batch)
	}
}

func (s *CertificateService) updateBatchOnFailure(job queue.Job) {
	var batchID uint
	switch v := job.Data["batch_id"].(type) {
	case float64:
		batchID = uint(v)
	case int:
		batchID = uint(v)
	case uint:
		batchID = v
	default:
		return
	}

	if batchID == 0 {
		return
	}

	var batch models.CertificateBatch
	if err := s.db.First(&batch, batchID).Error; err == nil {
		batch.Failed++
		if batch.Processed+batch.Failed >= batch.TotalCount {
			if batch.Failed == batch.TotalCount {
				batch.Status = "failed"
			} else {
				batch.Status = "completed"
			}
		}
		s.db.Save(&batch)
	}
}

func (s *CertificateService) processEmailJob(ctx context.Context, job queue.Job) error {
	var certID uint
	switch v := job.Data["certificate_id"].(type) {
	case float64:
		certID = uint(v)
	case int:
		certID = uint(v)
	case uint:
		certID = v
	default:
		return fmt.Errorf("invalid certificate_id in job data: %v", job.Data["certificate_id"])
	}

	var certificate models.Certificate
	if err := s.db.Preload("Recipient").First(&certificate, certID).Error; err != nil {
		return fmt.Errorf("certificate not found: %w", err)
	}

	if certificate.FilePath == "" {
		return fmt.Errorf("certificate file not generated yet")
	}

	var emailTemplate models.EmailTemplate
	var emailTemplateID uint

	if templateIDRaw, ok := job.Data["email_template_id"]; ok && templateIDRaw != nil {
		switch v := templateIDRaw.(type) {
		case float64:
			emailTemplateID = uint(v)
		case int:
			emailTemplateID = uint(v)
		case uint:
			emailTemplateID = v
		}
	}

	if emailTemplateID > 0 {
		if err := s.db.Where("id = ? AND is_active = ?", emailTemplateID, true).First(&emailTemplate).Error; err != nil {
			return fmt.Errorf("email template not found: %w", err)
		}
	} else {
		if err := s.db.Where("name = ? AND is_active = ?", "default", true).First(&emailTemplate).Error; err != nil {
			return fmt.Errorf("default email template not found: %w", err)
		}
	}

	downloadURL := fmt.Sprintf("/api/v1/certificates/%d/download", certificate.ID)

	data := map[string]interface{}{
		"name":         certificate.Recipient.Name,
		"email":        certificate.Recipient.Email,
		"course":       certificate.Recipient.Course,
		"event":        certificate.Recipient.Event,
		"club":         certificate.Recipient.Club,
		"date":         certificate.Recipient.Date,
		"download_url": downloadURL,
	}

	if err := s.emailService.SendWithTemplate(
		certificate.Recipient.Email,
		emailTemplate.Subject,
		emailTemplate.BodyHTML,
		data,
	); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	now := time.Now()
	certificate.EmailSent = true
	certificate.EmailSentAt = &now
	if err := s.db.Save(&certificate).Error; err != nil {
		return fmt.Errorf("failed to update certificate: %w", err)
	}

	return nil
}

func getStringFromMetadata(metadataJSON datatypes.JSON, key, defaultValue string) string {
	if len(metadataJSON) == 0 {
		return defaultValue
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		return defaultValue
	}

	if val, ok := metadata[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}

	return defaultValue
}

func (s *CertificateService) GetCertificate(id uint) (*models.Certificate, error) {
	var certificate models.Certificate
	if err := s.db.Preload("Template").Preload("Recipient").First(&certificate, id).Error; err != nil {
		return nil, err
	}
	return &certificate, nil
}

func (s *CertificateService) GetBatchStatus(id uint) (*models.CertificateBatch, error) {
	var batch models.CertificateBatch
	if err := s.db.Preload("Template").First(&batch, id).Error; err != nil {
		return nil, err
	}
	return &batch, nil
}

func (s *CertificateService) GetStorage() storage.Storage {
	return s.storage
}

func (s *CertificateService) ProcessCertificateJob(ctx context.Context, job queue.Job) error {
	return s.processCertificateJob(ctx, job)
}

func (s *CertificateService) ProcessEmailJob(ctx context.Context, job queue.Job) error {
	return s.processEmailJob(ctx, job)
}
