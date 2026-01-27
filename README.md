# Certificate Service

Production-ready certificate generation service built with Go. Generates PDF certificates using HTML templates.

## Features

- HTML-based PDF generation using headless browser
- Customizable certificate templates
- Bulk certificate generation
- Email delivery with templates
- Redis-backed job queue
- RESTful API
- PostgreSQL storage

## Prerequisites

- Podman
- Chrome/Chromium (included in container)

## Quick Start

### Start Services

```bash
podman-compose down 
podman-compose build app
podman-compose up -d
```

Services:
- API: http://localhost:8080
- PostgreSQL: localhost:5432
- Redis: localhost:6379

### Stop Services

```bash
make podman-down
```

### View Logs

```bash
make podman-logs
```

## API Documentation

Import `postman_collection.json` into Postman for complete API documentation.

### Endpoints

**Health Check**
```
GET /health
```

**Certificates**
```
POST /api/v1/certificates/generate
POST /api/v1/certificates/bulk
GET  /api/v1/certificates/:id
GET  /api/v1/certificates/:id/download
```

**Batches**
```
GET /api/v1/batches/:id
```

**Templates**
```
POST /api/v1/templates
GET  /api/v1/templates
GET  /api/v1/templates/:id
```

**Email Templates**
```
POST /api/v1/email-templates
GET  /api/v1/email-templates
```

## Configuration

Edit `config.yaml` or set environment variables:

- `PORT` - Server port
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` - Database config
- `REDIS_HOST`, `REDIS_PORT` - Redis config
- `SENDGRID_API_KEY` - Email service key

## Project Structure

```
certificate-service/
├── cmd/server/          # Application entry point
├── internal/
│   ├── config/          # Configuration
│   ├── handlers/        # HTTP handlers
│   ├── models/          # Data models
│   ├── queue/           # Job queue
│   ├── services/         # Business logic
│   └── storage/         # Storage abstraction
├── pkg/
│   ├── email/           # Email service
│   └── pdf/             # PDF generation
├── templates/            # Certificate templates
├── migrations/          # Database migrations
├── config.yaml          # Configuration
├── Containerfile        # Container definition
└── podman-compose.yml   # Service orchestration
```

## Make Commands

- `make podman-build` - Build container
- `make podman-up` - Start services
- `make podman-down` - Stop services
- `make podman-logs` - View logs
- `make podman-restart` - Restart services
- `make migrate` - Run database migrations
