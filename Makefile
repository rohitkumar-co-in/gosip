# Leadomi SIP supported development commands (GNU Make).
.PHONY: all build build-backend build-frontend run dev dev-frontend deps test test-frontend vet check docker docker-up docker-down docker-logs source-backup clean help
all: build
build: build-frontend build-backend
build-backend:
	CGO_ENABLED=1 go build -o bin/gosip ./cmd/gosip
build-frontend:
	cd frontend && pnpm install --frozen-lockfile && pnpm build
run: build
	./bin/gosip
dev:
	go run ./cmd/gosip
dev-frontend:
	cd frontend && pnpm dev
deps:
	go mod download
	cd frontend && pnpm install --frozen-lockfile
test:
	go test -race ./...
test-frontend:
	cd frontend && pnpm test
vet:
	go vet ./...
check: vet test test-frontend build-frontend
docker:
	docker build -t leadomi-sip:local .
# Fresh direct Compose installs; existing Coolify apps retain their configured compose file.
docker-up:
	docker compose -f docker-compose.production.yml up -d --build
docker-down:
	docker compose -f docker-compose.production.yml down
docker-logs:
	docker compose -f docker-compose.production.yml logs -f
source-backup:
	python3 scripts/source-backup.py
clean:
	python3 scripts/clean-generated.py
help:
	@echo "Build: build, build-backend, build-frontend, docker"
	@echo "Develop: deps, dev, dev-frontend (separate terminals)"
	@echo "Verify: check, vet, test, test-frontend"
	@echo "Direct Compose: docker-up, docker-down, docker-logs"
	@echo "Source archive: source-backup; remove generated files: clean"
