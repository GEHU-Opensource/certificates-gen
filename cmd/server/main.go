package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"certificate-service/internal/config"
	"certificate-service/internal/handlers"
	"certificate-service/internal/models"
	"certificate-service/internal/queue"
	"certificate-service/internal/services"
	"certificate-service/internal/storage"
	"certificate-service/pkg/email"
	"certificate-service/pkg/pdf"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}
	defer sqlDB.Close()

	db.AutoMigrate(
		&models.Certificate{},
		&models.Template{},
		&models.Recipient{},
		&models.CertificateBatch{},
		&models.EmailTemplate{},
	)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	storageService, err := storage.NewLocalStorage(cfg.Storage.LocalPath, "")
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	pdfGen, err := pdf.NewHTMLGenerator("./templates/certificates")
	if err != nil {
		log.Fatalf("Failed to initialize PDF generator: %v", err)
	}
	defer pdfGen.Close()

	emailService := email.NewService(
		cfg.Email.SendGridKey,
		cfg.Email.FromEmail,
		cfg.Email.FromName,
	)

	queueWorker := queue.NewWorker(redisClient, "certificate_queue", "worker-1")

	certService := services.NewCertificateService(
		db,
		pdfGen,
		emailService,
		storageService,
		queueWorker,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i := 0; i < cfg.Queue.WorkerCount; i++ {
		worker := queue.NewWorker(redisClient, "certificate_queue", fmt.Sprintf("worker-%d", i+1))
		worker.RegisterProcessor("generate_certificate", certService.ProcessCertificateJob)
		worker.RegisterProcessor("send_email", certService.ProcessEmailJob)
		go func(w *queue.Worker) {
			if err := w.Start(ctx); err != nil && err != context.Canceled {
				log.Printf("Worker error: %v", err)
			}
		}(worker)
	}

	if gin.Mode() == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	router.Use(gin.Logger(), gin.Recovery())

	certHandler := handlers.NewCertificateHandler(certService)
	templateHandler := handlers.NewTemplateHandler(db)

	api := router.Group("/api/v1")
	{
		api.POST("/certificates/generate", certHandler.GenerateCertificate)
		api.POST("/certificates/bulk", certHandler.BulkGenerate)
		api.GET("/certificates/:id", certHandler.GetCertificate)
		api.GET("/certificates/:id/download", certHandler.DownloadCertificate)
		api.GET("/batches/:id", certHandler.GetBatchStatus)

		api.POST("/templates", templateHandler.CreateTemplate)
		api.GET("/templates", templateHandler.GetTemplates)
		api.GET("/templates/:id", templateHandler.GetTemplate)

		api.POST("/email-templates", templateHandler.CreateEmailTemplate)
		api.GET("/email-templates", templateHandler.GetEmailTemplates)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	go func() {
		log.Printf("Server starting on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	cancel()

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
