package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Email    EmailConfig    `yaml:"email"`
	Storage  StorageConfig  `yaml:"storage"`
	Queue    QueueConfig    `yaml:"queue"`
}

type ServerConfig struct {
	Port         string `yaml:"port"`
	Host         string `yaml:"host"`
	ReadTimeout  int    `yaml:"read_timeout"`
	WriteTimeout int    `yaml:"write_timeout"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type EmailConfig struct {
	Provider     string `yaml:"provider"`
	SendGridKey  string `yaml:"sendgrid_key"`
	SMTPHost     string `yaml:"smtp_host"`
	SMTPPort     int    `yaml:"smtp_port"`
	SMTPUser     string `yaml:"smtp_user"`
	SMTPPassword string `yaml:"smtp_password"`
	FromEmail    string `yaml:"from_email"`
	FromName     string `yaml:"from_name"`
}

type StorageConfig struct {
	Type      string `yaml:"type"`
	LocalPath string `yaml:"local_path"`
	S3Bucket  string `yaml:"s3_bucket"`
	S3Region  string `yaml:"s3_region"`
}

type QueueConfig struct {
	WorkerCount int `yaml:"worker_count"`
	BatchSize   int `yaml:"batch_size"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if v := os.Getenv("PORT"); v != "" {
		config.Server.Port = v
	}
	if config.Server.Port == "" {
		config.Server.Port = "8080"
	}

	if v := os.Getenv("DB_HOST"); v != "" {
		config.Database.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &config.Database.Port)
	}
	if v := os.Getenv("DB_USER"); v != "" {
		config.Database.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		config.Database.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		config.Database.DBName = v
	}
	if config.Database.SSLMode == "" {
		config.Database.SSLMode = "disable"
	}

	if v := os.Getenv("SENDGRID_API_KEY"); v != "" {
		config.Email.SendGridKey = v
	}

	if v := os.Getenv("REDIS_HOST"); v != "" {
		config.Redis.Host = v
	}
	if v := os.Getenv("REDIS_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &config.Redis.Port)
	}

	return &config, nil
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}
