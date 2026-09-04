# ⚡ High-Throughput E-Commerce Platform (10,000 RPS Target)

Production-grade, highly scalable e-commerce architecture designed to withstand **10,000+ requests per second (RPS)** with sub-millisecond read latency and non-blocking asynchronous order processing.

---

## 🏗️ High-Level System Architecture

```
                                  [ 10,000 RPS Ingress Traffic ]
                                                │
                                                ▼
                                    ┌───────────────────────┐
                                    │    Go API Gateway     │
                                    │      (Chi Router)     │
                                    └───────────┬───────────┘
                                                │
                 ┌──────────────────────────────┴──────────────────────────────┐
                 │ (90% Read Path - Catalog & Products)                        │ (10% Write Path - Checkout & Orders)
                 ▼                                                             ▼
       ┌──────────────────┐                                          ┌──────────────────┐
       │ L1 Cache (Memory)│ < 0.05ms Latency                         │ Redis Lua Script │ Atomic Stock Reservation
       └─────────┬────────┘                                          └─────────┬────────┘ (Zero Race Conditions)
                 │ Miss                                                        │ Success
                 ▼                                                             ▼
       ┌──────────────────┐                                          ┌──────────────────┐
       │ L2 Cache (Redis) │ < 1ms Latency                            │ Kafka Producer   │ Async Snappy-Compressed
       └─────────┬────────┘                                          └─────────┬────────┘ Event Queue (202 Accepted)
                 │ Miss                                                        │
                 ▼                                                             ▼
       ┌──────────────────┐                                          ┌──────────────────┐
       │ Singleflight     │ 1 in-flight DB query                     │ Kafka Consumer   │ Batch Worker (200 ev/batch)
       │ Stampede Guard   │ per key                                  └─────────┬────────┘
       └─────────┬────────┘                                                    │
                 │                                                             ▼
                 ▼                                                   ┌──────────────────┐
       ┌──────────────────┐                                          │ PostgreSQL 16    │ pgx.Batch inside
       │ PostgreSQL 16    │ Read Replica / Pool                      │ (ACID Store)     │ single transaction
       └──────────────────┘                                          └──────────────────┘
```

---

## 📊 Observability (Prometheus & Grafana)

The stack comes with **zero-configuration telemetry**:

- **Prometheus**: Automatically scrapes Go runtime metrics, custom HTTP histograms (`p50`, `p90`, `p95`, `p99`), cache hit/miss rates, and Kafka throughput.
- **Grafana**: Auto-provisioned with the `E-Commerce 10,000 RPS High-Load Dashboard`.

### Access URLs

| Service                   | URL                             | Credentials       | Purpose                            |
| ------------------------- | ------------------------------- | ----------------- | ---------------------------------- |
| **React Storefront**      | `http://localhost:3001`         | None              | Interactive UI with live telemetry |
| **Go API Service**        | `http://localhost:8080`         | None              | HTTP Gateway & Business Logic      |
| **Go Prometheus Metrics** | `http://localhost:8080/metrics` | None              | Raw metrics exporter               |
| **Prometheus UI**         | `http://localhost:9090`         | None              | Metric exploration & TSDB queries  |
| **Grafana Dashboard**     | `http://localhost:3000`         | `admin` / `admin` | Real-time 10k RPS visual dashboard |

---

## 🚀 Quick Start (Local Production Cluster)

### 1. Start all services in Docker Compose

```bash
make up
# or: cd deploy && docker compose up -d --build
```

### 2. Verify containers are running & healthy

```bash
docker compose -f deploy/docker-compose.yml ps
```

Services started:

- `ecommerce-postgres` (PostgreSQL 16 with initial seed of 1,000 items)
- `ecommerce-redis` (Redis 7 Alpine, in-memory tuned)
- `ecommerce-redpanda` (Kafka-compatible high-speed broker)
- `ecommerce-backend-api` (Go REST API)
- `ecommerce-backend-worker` (Go Kafka Consumer & Batch DB Persister)
- `ecommerce-prometheus` (Prometheus 2.51)
- `ecommerce-grafana` (Grafana 10.4 with preloaded dashboard)

### 3. Start the React Storefront (Frontend)

```bash
cd frontend
npm install
npm run dev
```

Open `http://localhost:3001` in your browser.

---

## ⚡ Simulating 10,000 RPS Load Test

We provide an automated **k6** stress scenario (`deploy/loadtest/benchmark_10k_rps.js`) that ramps from 1,000 RPS to 10,000 RPS:

```bash
# Run with Docker (no local k6 installation needed):
docker run --rm -i --network=host grafana/k6 run - < deploy/loadtest/benchmark_10k_rps.js

# Or run natively if k6 is installed on your machine:
k6 run deploy/loadtest/benchmark_10k_rps.js
```

### What to watch in Grafana during the test

1. Open `http://localhost:3000` (Grafana).
2. Watch the **Total Requests Per Second (RPS)** climb to **10,000 req/s**.
3. Observe **p95 Latency** staying **under 20–35 ms**.
4. Check **L1 & L2 Cache Hit Rate** absorbing over 90% of traffic.
5. Watch **Kafka Orders Produced** and **Worker Consumed** processing batches in lockstep.

---

## 💰 Free Tier & Production Feasibility Guide

### Can this be hosted 100% free?

**Yes!** Here is how mature production setups balance zero cost with extreme load:

1. **Frontend (Vercel)**:
   - 100% Free forever on **Vercel Hobby**.
   - Push to `main` on GitHub triggers instant zero-config deployments.
   - Global Edge CDN handles unlimited static assets.
2. **Repository & CI/CD (GitHub)**:
   - 100% Free on **GitHub**.
   - GitHub Actions provides 2,000 free runner minutes each month for linting, testing, and builds.
3. **High-Load Simulation (Local Docker Cluster)**:
   - **Why local?** No free cloud provider in the world (Supabase, Neon, Upstash) allows 10,000 RPS. Upstash Redis free tier caps at 10,000 commands _per day_ (exhausted in 1 second at 10k RPS).
   - Running the containerized stack locally lets you stress test at 10,000–30,000 RPS on your own CPU cores without paying a single dollar or hitting arbitrary cloud rate limits.
4. **Cloud Demo Backend**:
   - For a public live portfolio demo, you can deploy the backend to **Oracle Cloud Always Free** (up to 4 ARM vCPUs and 24 GB RAM free forever) or **Fly.io**.

---

## 📦 Project Structure

```
ecommerce-app/
├── .github/
│   └── workflows/
│       └── ci.yml                 # Production CI pipeline (Go test, vet, Vite build)
├── backend/
│   ├── cmd/
│   │   ├── api/                   # HTTP Gateway entrypoint
│   │   └── worker/                # Kafka consumer batch worker
│   ├── internal/
│   │   ├── domain/                # Product, Order, Inventory models
│   │   ├── queue/kafka/           # Snappy async producer & batch reader
│   │   ├── repository/
│   │   │   ├── postgres/          # pgxpool & batched transactions
│   │   │   └── redis/             # Redis pool & atomic Lua script
│   │   ├── service/               # L1 memory cache, Singleflight, Order logic
│   │   └── transport/http/        # Chi router, Prometheus middlewares, Handlers
│   ├── pkg/metrics/               # Prometheus metric vectors
│   ├── migrations/                # Schema init + 1,000 product seed
│   ├── Dockerfile                 # Multi-stage Alpine container
│   └── go.mod
├── frontend/
│   ├── src/                       # React 18 + Tailwind CSS Storefront
│   ├── vercel.json                # Zero-config Vercel deployment spec
│   └── package.json
├── deploy/
│   ├── docker-compose.yml         # Complete 7-service production stack
│   ├── prometheus/                # Scrape configuration
│   ├── grafana/                   # Auto-provisioning & 10k RPS dashboard
│   └── loadtest/                  # k6 stress test scenario
├── Makefile                       # Developer shortcuts (make up, make bench)
└── README.md
```
