package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"certificate-service/internal/models"
	"certificate-service/internal/services"
	"github.com/gin-gonic/gin"
)

type CertificateHandler struct {
	service *services.CertificateService
}

func NewCertificateHandler(service *services.CertificateService) *CertificateHandler {
	return &CertificateHandler{
		service: service,
	}
}

func (h *CertificateHandler) GenerateCertificate(c *gin.Context) {
	var req models.GenerateCertificateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	certificate, err := h.service.GenerateCertificate(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := models.CertificateResponse{
		ID:        certificate.ID,
		Status:    certificate.Status,
		FilePath:  certificate.FilePath,
		EmailSent: certificate.EmailSent,
	}

	c.JSON(http.StatusAccepted, response)
}

func (h *CertificateHandler) BulkGenerate(c *gin.Context) {
	var req models.BulkGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	batch, err := h.service.BulkGenerate(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := models.BatchStatusResponse{
		ID:         batch.ID,
		TotalCount: batch.TotalCount,
		Processed:  batch.Processed,
		Failed:     batch.Failed,
		Status:     batch.Status,
		Progress:   float64(batch.Processed) / float64(batch.TotalCount) * 100,
	}

	c.JSON(http.StatusAccepted, response)
}

func (h *CertificateHandler) GetCertificate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid certificate id"})
		return
	}

	certificate, err := h.service.GetCertificate(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "certificate not found"})
		return
	}

	downloadURL := fmt.Sprintf("/api/v1/certificates/%d/download", certificate.ID)
	response := models.CertificateResponse{
		ID:          certificate.ID,
		Status:      certificate.Status,
		FilePath:    certificate.FilePath,
		EmailSent:   certificate.EmailSent,
		DownloadURL: downloadURL,
	}

	c.JSON(http.StatusOK, response)
}

func (h *CertificateHandler) GetBatchStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid batch id"})
		return
	}

	batch, err := h.service.GetBatchStatus(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "batch not found"})
		return
	}

	progress := 0.0
	if batch.TotalCount > 0 {
		progress = float64(batch.Processed) / float64(batch.TotalCount) * 100
	}

	response := models.BatchStatusResponse{
		ID:         batch.ID,
		TotalCount: batch.TotalCount,
		Processed:  batch.Processed,
		Failed:     batch.Failed,
		Status:     batch.Status,
		Progress:   progress,
	}

	c.JSON(http.StatusOK, response)
}

func (h *CertificateHandler) DownloadCertificate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid certificate id"})
		return
	}

	certificate, err := h.service.GetCertificate(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "certificate not found"})
		return
	}

	if certificate.Status != "completed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "certificate not ready"})
		return
	}

	data, err := h.service.GetStorage().Get(certificate.FilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read certificate"})
		return
	}

	c.Data(http.StatusOK, "application/pdf", data)
}
