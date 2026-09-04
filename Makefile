.PHONY: up down logs ps test bench build-front build-back

# Start complete infrastructure stack (Postgres, Redis, Kafka, Prometheus, Grafana, API, Worker)
up:
	cd deploy && docker compose up -d --build

# Stop and remove containers
down:
	cd deploy && docker compose down

# Follow logs from all services
logs:
	cd deploy && docker compose logs -f

# Check container health & status
ps:
	cd deploy && docker compose ps

# Run Go backend tests
test:
	cd backend && go test -v -race ./...

# Run 10,000 RPS k6 benchmark
bench:
	docker run --rm -i --network=host grafana/k6 run - < deploy/loadtest/benchmark_10k_rps.js

# Build React frontend
build-front:
	cd frontend && npm install && npm run build

# Build Go backend binaries
build-back:
	cd backend && go build -o bin/api ./cmd/api && go build -o bin/worker ./cmd/worker

