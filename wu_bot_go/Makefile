.PHONY: build test docker-build docker-run docker-up docker-down clean

# Build Go binary (requires local Go)
build:
	go build -o wubot ./cmd/wubot

# Run tests
test:
	go test ./...

# Build Docker image
docker-build:
	docker build -t wubot .

# Run in Docker (headless)
docker-run: docker-build
	docker run --rm -v $(PWD)/config.yaml:/app/config.yaml wubot

# Run with docker compose (headless, detached)
docker-up:
	docker compose up -d --build

# Stop docker compose
docker-down:
	docker compose down

# Run with TUI via docker compose
docker-tui:
	docker compose run --rm wubot ./wubot --config /app/config.yaml

# Clean build artifacts
clean:
	rm -f wubot
	docker rmi wubot 2>/dev/null || true
