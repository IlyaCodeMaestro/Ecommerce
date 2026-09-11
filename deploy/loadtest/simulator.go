package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	BaseURL     string
	Concurrency int
	Duration    time.Duration
	RampUp      time.Duration
}

type Stats struct {
	totalRequests  int64
	successRequests int64
	failedRequests  int64

	searchRequests  int64
	cartRequests    int64
	orderRequests   int64
	cancelRequests  int64

	mu        sync.Mutex
	latencies []time.Duration
}

func main() {
	baseURL := flag.String("url", "http://localhost:8080", "Base URL of E-commerce API")
	concurrency := flag.Int("c", 50, "Number of concurrent worker goroutines")
	duration := flag.Duration("d", 15*time.Second, "Test duration")
	flag.Parse()

	cfg := Config{
		BaseURL:     *baseURL,
		Concurrency: *concurrency,
		Duration:    *duration,
	}

	fmt.Println("================================================================================")
	fmt.Printf("🚀 E-COMMERCE HIGH-LOAD & DISTRIBUTED SAGA SIMULATOR\n")
	fmt.Printf("🎯 Target: %s\n", cfg.BaseURL)
	fmt.Printf("⚡ Concurrency: %d concurrent virtual shoppers\n", cfg.Concurrency)
	fmt.Printf("⏱️ Duration: %v\n", cfg.Duration)
	fmt.Println("================================================================================")

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 500,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression: false,
		},
	}

	stats := &Stats{
		latencies: make([]time.Duration, 0, 100000),
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Duration)
	defer cancel()

	start := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			runWorker(ctx, client, cfg, stats, workerID)
		}(i)
	}

	// Live progress ticker
	progressTicker := time.NewTicker(2 * time.Second)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-progressTicker.C:
				elapsed := time.Since(start).Seconds()
				total := atomic.LoadInt64(&stats.totalRequests)
				rps := float64(total) / elapsed
				fmt.Printf("⏳ [%.1fs] Total Requests: %d | Throughput: %.0f RPS | Errors: %d\n",
					elapsed, total, rps, atomic.LoadInt64(&stats.failedRequests))
			}
		}
	}()

	wg.Wait()
	progressTicker.Stop()
	totalElapsed := time.Since(start)

	printReport(stats, totalElapsed)
}

func runWorker(ctx context.Context, client *http.Client, cfg Config, stats *Stats, workerID int) {
	sessionID := fmt.Sprintf("sim-user-%d-%d", workerID, rand.Intn(10000))
	categories := []string{"all", "laptops", "smartphones", "audio", "monitors", "accessories"}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			choice := rand.Float64()

			// 1. 60% Traffic: Browse / Search products
			if choice < 0.60 {
				cat := categories[rand.Intn(len(categories))]
				url := fmt.Sprintf("%s/api/v1/products?category=%s&limit=20", cfg.BaseURL, cat)
				t0 := time.Now()
				req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
				req.Header.Set("X-Session-ID", sessionID)
				resp, err := client.Do(req)
				dur := time.Since(t0)

				atomic.AddInt64(&stats.totalRequests, 1)
				atomic.AddInt64(&stats.searchRequests, 1)
				recordLatency(stats, dur)

				if err == nil && resp.StatusCode == http.StatusOK {
					atomic.AddInt64(&stats.successRequests, 1)
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				} else {
					atomic.AddInt64(&stats.failedRequests, 1)
					if resp != nil {
						resp.Body.Close()
					}
				}

			// 2. 20% Traffic: Add item to Redis Cart
			} else if choice < 0.80 {
				productID := rand.Intn(50) + 1
				payload, _ := json.Marshal(map[string]interface{}{
					"product_id": productID,
					"quantity":   rand.Intn(2) + 1,
				})
				url := fmt.Sprintf("%s/api/v1/cart/items", cfg.BaseURL)
				t0 := time.Now()
				req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Session-ID", sessionID)
				resp, err := client.Do(req)
				dur := time.Since(t0)

				atomic.AddInt64(&stats.totalRequests, 1)
				atomic.AddInt64(&stats.cartRequests, 1)
				recordLatency(stats, dur)

				if err == nil && (resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated) {
					atomic.AddInt64(&stats.successRequests, 1)
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				} else {
					atomic.AddInt64(&stats.failedRequests, 1)
					if resp != nil {
						resp.Body.Close()
					}
				}

			// 3. 15% Traffic: Checkout with Idempotency-Key
			} else if choice < 0.95 {
				idempotencyKey := fmt.Sprintf("idem-%d-%d", workerID, time.Now().UnixNano())
				payload, _ := json.Marshal(map[string]interface{}{
					"user_id":         sessionID,
					"idempotency_key": idempotencyKey,
					"items": []map[string]interface{}{
						{
							"product_id": rand.Intn(30) + 1,
							"quantity":   1,
						},
					},
				})
				url := fmt.Sprintf("%s/api/v1/orders", cfg.BaseURL)
				t0 := time.Now()
				req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Idempotency-Key", idempotencyKey)
				req.Header.Set("X-Session-ID", sessionID)
				resp, err := client.Do(req)
				dur := time.Since(t0)

				atomic.AddInt64(&stats.totalRequests, 1)
				atomic.AddInt64(&stats.orderRequests, 1)
				recordLatency(stats, dur)

				var orderResp struct {
					OrderID string `json:"order_id"`
				}

				if err == nil && (resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted) {
					atomic.AddInt64(&stats.successRequests, 1)
					_ = json.NewDecoder(resp.Body).Decode(&orderResp)
					resp.Body.Close()

					// If order created and randomly selected (30% chance), simulate payment or cancel order!
					if orderResp.OrderID != "" && rand.Float64() < 0.30 {
						cancelURL := fmt.Sprintf("%s/api/v1/orders/%s/cancel", cfg.BaseURL, orderResp.OrderID)
						tCancel := time.Now()
						cReq, _ := http.NewRequestWithContext(ctx, http.MethodPut, cancelURL, bytes.NewReader([]byte(`{"reason":"benchmark cancel test"}`)))
						cReq.Header.Set("Content-Type", "application/json")
						cReq.Header.Set("X-Session-ID", sessionID)
						cResp, cErr := client.Do(cReq)
						cDur := time.Since(tCancel)

						atomic.AddInt64(&stats.totalRequests, 1)
						atomic.AddInt64(&stats.cancelRequests, 1)
						recordLatency(stats, cDur)

						if cErr == nil && cResp.StatusCode == http.StatusOK {
							atomic.AddInt64(&stats.successRequests, 1)
							io.Copy(io.Discard, cResp.Body)
							cResp.Body.Close()
						} else {
							atomic.AddInt64(&stats.failedRequests, 1)
							if cResp != nil {
								cResp.Body.Close()
							}
						}
					}
				} else {
					atomic.AddInt64(&stats.failedRequests, 1)
					if resp != nil {
						resp.Body.Close()
					}
				}

			// 4. 5% Traffic: Deep health check
			} else {
				url := fmt.Sprintf("%s/healthz", cfg.BaseURL)
				t0 := time.Now()
				req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
				resp, err := client.Do(req)
				dur := time.Since(t0)

				atomic.AddInt64(&stats.totalRequests, 1)
				recordLatency(stats, dur)

				if err == nil && resp.StatusCode == http.StatusOK {
					atomic.AddInt64(&stats.successRequests, 1)
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				} else {
					atomic.AddInt64(&stats.failedRequests, 1)
					if resp != nil {
						resp.Body.Close()
					}
				}
			}

			// Micro sleep to simulate realistic human pacing
			time.Sleep(time.Duration(rand.Intn(5)+2) * time.Millisecond)
		}
	}
}

func recordLatency(stats *Stats, d time.Duration) {
	stats.mu.Lock()
	stats.latencies = append(stats.latencies, d)
	stats.mu.Unlock()
}

func printReport(stats *Stats, elapsed time.Duration) {
	total := atomic.LoadInt64(&stats.totalRequests)
	success := atomic.LoadInt64(&stats.successRequests)
	failed := atomic.LoadInt64(&stats.failedRequests)

	rps := float64(total) / elapsed.Seconds()
	successRate := 0.0
	if total > 0 {
		successRate = (float64(success) / float64(total)) * 100.0
	}

	stats.mu.Lock()
	latencies := make([]time.Duration, len(stats.latencies))
	copy(latencies, stats.latencies)
	stats.mu.Unlock()

	var p50, p90, p95, p99, maxLat time.Duration
	if len(latencies) > 0 {
		sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
		p50 = latencies[int(float64(len(latencies))*0.50)]
		p90 = latencies[int(float64(len(latencies))*0.90)]
		p95 = latencies[int(float64(len(latencies))*0.95)]
		p99 = latencies[int(float64(len(latencies))*0.99)]
		maxLat = latencies[len(latencies)-1]
	}

	fmt.Println("\n================================================================================")
	fmt.Printf("📊 BENCHMARK & LOAD TEST RESULTS (Duration: %v)\n", elapsed.Round(time.Millisecond))
	fmt.Println("================================================================================")
	fmt.Printf("📌 Total Requests Handled : %d\n", total)
	fmt.Printf("🚀 Average Throughput      : %.1f RPS\n", rps)
	fmt.Printf("✅ Success Rate           : %.2f%% (%d OK / %d Errors)\n", successRate, success, failed)
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("⏱️ LATENCY PERCENTILES (SLA Target: p99 < 50ms):")
	fmt.Printf("   • p50 (Median) : %v\n", p50)
	fmt.Printf("   • p90          : %v\n", p90)
	fmt.Printf("   • p95          : %v\n", p95)
	fmt.Printf("   • p99          : %v\n", p99)
	fmt.Printf("   • Max Latency  : %v\n", maxLat)
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("📦 TRAFFIC COMPOSITION BREAKDOWN:")
	fmt.Printf("   • Catalog Search / PDP : %d\n", atomic.LoadInt64(&stats.searchRequests))
	fmt.Printf("   • Redis Cart Mutations : %d\n", atomic.LoadInt64(&stats.cartRequests))
	fmt.Printf("   • Idempotent Checkouts : %d\n", atomic.LoadInt64(&stats.orderRequests))
	fmt.Printf("   • Compensating Cancels : %d (Restocked into DB & Redis RAM)\n", atomic.LoadInt64(&stats.cancelRequests))
	fmt.Println("================================================================================")
}
