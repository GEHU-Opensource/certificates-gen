.PHONY: podman-build podman-up podman-down podman-logs podman-restart migrate

podman-build:
	podman build -f Containerfile -t certificate-service:latest .

podman-up:
	podman-compose -f podman-compose.yml up -d

podman-down:
	podman-compose -f podman-compose.yml down

podman-logs:
	podman-compose -f podman-compose.yml logs -f

podman-restart:
	podman-compose -f podman-compose.yml restart

migrate:
	psql -h localhost -U postgres -d certificates -f migrations/001_init.sql
