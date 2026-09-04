import React, { useState } from 'react';
import { X, Play, Terminal, ExternalLink, Cpu, BarChart2, ShieldCheck, Activity } from 'lucide-react';
import { fetchProducts } from '../services/api';

export default function MetricsWidget({ isOpen, onClose }) {
  const [runningTest, setRunningTest] = useState(false);
  const [testStats, setTestStats] = useState(null);

  if (!isOpen) return null;

  // Browser load burst test
  const runBrowserBurst = async (requestsCount = 100) => {
    setRunningTest(true);
    setTestStats(null);

    const start = performance.now();
    let success = 0;
    let failed = 0;
    const latencies = [];

    const promises = Array.from({ length: requestsCount }, async (_, i) => {
      const reqStart = performance.now();
      try {
        const id = (i % 500) + 1;
        await fetchProducts('', 1, id);
        success++;
        latencies.push(performance.now() - reqStart);
      } catch (e) {
        failed++;
      }
    });

    await Promise.all(promises);
    const totalDuration = (performance.now() - start) / 1000;
    const rps = Math.round(requestsCount / totalDuration);

    latencies.sort((a, b) => a - b);
    const p50 = latencies[Math.floor(latencies.length * 0.5)] || 0;
    const p95 = latencies[Math.floor(latencies.length * 0.95)] || 0;

    setTestStats({
      total: requestsCount,
      success,
      failed,
      durationSec: totalDuration.toFixed(2),
      browserRPS: rps,
      p50: p50.toFixed(1),
      p95: p95.toFixed(1),
    });
    setRunningTest(false);
  };

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto bg-black/75 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-slate-900 border border-slate-800 rounded-3xl w-full max-w-2xl overflow-hidden shadow-2xl">
        {/* Header */}
        <div className="p-6 border-b border-slate-800 flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <BarChart2 className="w-5 h-5 text-cyan-400" />
            <h2 className="text-lg font-bold text-white">Observability & 10k RPS Stress Testing</h2>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-6 space-y-6">
          {/* Quick links to Dashboards */}
          <div>
            <h3 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-3">
              Live Monitoring Consoles
            </h3>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              <a
                href="http://localhost:3000"
                target="_blank"
                rel="noreferrer"
                className="flex items-center justify-between p-3 rounded-xl bg-slate-950/70 border border-slate-800 hover:border-emerald-500/50 transition group"
              >
                <div>
                  <div className="text-sm font-semibold text-white group-hover:text-emerald-400">Grafana</div>
                  <div className="text-xs text-slate-400">:3000 (auto-login)</div>
                </div>
                <ExternalLink className="w-4 h-4 text-slate-500 group-hover:text-emerald-400" />
              </a>

              <a
                href="http://localhost:9090"
                target="_blank"
                rel="noreferrer"
                className="flex items-center justify-between p-3 rounded-xl bg-slate-950/70 border border-slate-800 hover:border-cyan-500/50 transition group"
              >
                <div>
                  <div className="text-sm font-semibold text-white group-hover:text-cyan-400">Prometheus</div>
                  <div className="text-xs text-slate-400">:9090 (raw metrics)</div>
                </div>
                <ExternalLink className="w-4 h-4 text-slate-500 group-hover:text-cyan-400" />
              </a>

              <a
                href="http://localhost:8080/metrics"
                target="_blank"
                rel="noreferrer"
                className="flex items-center justify-between p-3 rounded-xl bg-slate-950/70 border border-slate-800 hover:border-purple-500/50 transition group"
              >
                <div>
                  <div className="text-sm font-semibold text-white group-hover:text-purple-400">Go Exporter</div>
                  <div className="text-xs text-slate-400">:8080/metrics</div>
                </div>
                <ExternalLink className="w-4 h-4 text-slate-500 group-hover:text-purple-400" />
              </a>
            </div>
          </div>

          {/* How to run 10k RPS test */}
          <div>
            <h3 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2 flex items-center gap-1.5">
              <Terminal className="w-3.5 h-3.5 text-emerald-400" />
              How to trigger 10,000 RPS load test (k6)
            </h3>
            <div className="bg-slate-950 rounded-2xl p-3 border border-slate-800/80 font-mono text-xs text-slate-300 space-y-2">
              <div className="text-slate-500"># Run high-load benchmark using Docker container:</div>
              <div className="text-emerald-400 select-all">
                docker run --rm -i --network=host grafana/k6 run - &lt; deploy/loadtest/benchmark_10k_rps.js
              </div>
              <div className="text-slate-500"># Or run natively if k6 is installed:</div>
              <div className="text-cyan-300 select-all">
                k6 run deploy/loadtest/benchmark_10k_rps.js
              </div>
            </div>
          </div>

          {/* Mini Browser Burst Test */}
          <div className="bg-slate-950/50 border border-slate-800 rounded-2xl p-4">
            <div className="flex items-center justify-between mb-3">
              <div>
                <h4 className="text-sm font-semibold text-white">Browser Concurrency Burst Test</h4>
                <p className="text-xs text-slate-400">Send concurrent parallel requests to verify L1/L2 caching</p>
              </div>
              <div className="flex gap-2">
                <button
                  disabled={runningTest}
                  onClick={() => runBrowserBurst(50)}
                  className="px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-xs font-mono text-white transition disabled:opacity-50"
                >
                  50 reqs
                </button>
                <button
                  disabled={runningTest}
                  onClick={() => runBrowserBurst(150)}
                  className="px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-xs font-mono text-white font-medium transition disabled:opacity-50 flex items-center gap-1"
                >
                  <Play className="w-3 h-3" />
                  150 reqs
                </button>
              </div>
            </div>

            {testStats && (
              <div className="grid grid-cols-4 gap-2 pt-3 border-t border-slate-800/80 font-mono text-center">
                <div className="p-2 rounded-xl bg-slate-900 border border-slate-800">
                  <div className="text-slate-400 text-[10px]">SUCCESS</div>
                  <div className="text-emerald-400 font-bold text-sm">{testStats.success}/{testStats.total}</div>
                </div>
                <div className="p-2 rounded-xl bg-slate-900 border border-slate-800">
                  <div className="text-slate-400 text-[10px]">CLIENT RPS</div>
                  <div className="text-cyan-400 font-bold text-sm">{testStats.browserRPS}</div>
                </div>
                <div className="p-2 rounded-xl bg-slate-900 border border-slate-800">
                  <div className="text-slate-400 text-[10px]">p50 LATENCY</div>
                  <div className="text-purple-400 font-bold text-sm">{testStats.p50}ms</div>
                </div>
                <div className="p-2 rounded-xl bg-slate-900 border border-slate-800">
                  <div className="text-slate-400 text-[10px]">p95 LATENCY</div>
                  <div className="text-amber-400 font-bold text-sm">{testStats.p95}ms</div>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
