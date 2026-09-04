.PHONY: up down logs ps test bench build-front build-back k8s-up k8s-down tf-apply hpa-watch

# ==========================================
# Docker Compose Stack
# ==========================================
up:
	cd deploy && docker compose up -d --build

down:
	cd deploy && docker compose down

logs:
	cd deploy && docker compose logs -f

ps:
	cd deploy && docker compose ps

# ==========================================
# Kubernetes (KinD / Docker Desktop)
# ==========================================
k8s-cluster:
	kind create cluster --name ecommerce --config deploy/k8s/kind-cluster.yaml

k8s-apply:
	kubectl apply -f deploy/k8s/

k8s-down:
	kind delete cluster --name ecommerce

hpa-watch:
	kubectl get hpa -n ecommerce -w

# ==========================================
# Terraform (IaC)
# ==========================================
tf-init:
	cd deploy/terraform && terraform init

tf-apply:
	cd deploy/terraform && terraform apply -auto-approve

# ==========================================
# Testing & Building
# ==========================================
test:
	cd backend && go test -v -race ./...

bench:
	docker run --rm -i --network=host grafana/k6 run - < deploy/loadtest/benchmark_10k_rps.js

build-front:
	cd frontend && npm install && npm run build

build-back:
	cd backend && go build -o bin/api ./cmd/api && go build -o bin/worker ./cmd/worker
