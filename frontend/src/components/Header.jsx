import React from 'react';
import { ShoppingBag, Activity, Zap, Server, BarChart3 } from 'lucide-react';

export default function Header({ cartCount, onOpenCart, onOpenMetrics, backendStatus, pingMs }) {
  return (
    <header className="sticky top-0 z-40 bg-slate-900/80 backdrop-blur-md border-b border-slate-800">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
        {/* Brand */}
        <div className="flex items-center space-x-3">
          <div className="w-10 h-10 rounded-xl bg-gradient-to-tr from-emerald-500 to-cyan-500 flex items-center justify-center shadow-lg shadow-emerald-500/20">
            <Zap className="w-5 h-5 text-white" />
          </div>
          <div>
            <div className="flex items-center space-x-2">
              <span className="font-bold text-lg tracking-tight text-white">HyperScale</span>
              <span className="text-xs px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 font-mono border border-emerald-500/20">
                10k RPS
              </span>
            </div>
            <p className="text-xs text-slate-400">Go • Redis • Kafka • Postgres</p>
          </div>
        </div>

        {/* Status & Actions */}
        <div className="flex items-center space-x-4">
          {/* Health Pill */}
          <div className="hidden sm:flex items-center space-x-2 px-3 py-1 rounded-full bg-slate-800/80 border border-slate-700/60 text-xs">
            <span className={`w-2 h-2 rounded-full ${backendStatus ? 'bg-emerald-400 animate-pulse' : 'bg-rose-500'}`} />
            <span className="text-slate-300 font-mono">
              {backendStatus ? `API Live (${pingMs}ms)` : 'Offline / Connecting'}
            </span>
          </div>

          {/* Quick Metrics Link / Modal */}
          <button
            onClick={onOpenMetrics}
            className="flex items-center space-x-2 px-3 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 transition text-sm font-medium"
            title="Open High-Load Simulator & Prometheus guide"
          >
            <BarChart3 className="w-4 h-4 text-cyan-400" />
            <span className="hidden md:inline">Telemetry & Benchmarks</span>
          </button>

          {/* Cart Button */}
          <button
            onClick={onOpenCart}
            className="relative flex items-center space-x-2 px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-medium text-sm transition shadow-lg shadow-emerald-600/20"
          >
            <ShoppingBag className="w-4 h-4" />
            <span>Cart</span>
            {cartCount > 0 && (
              <span className="absolute -top-2 -right-2 bg-amber-500 text-slate-950 font-black text-xs w-5 h-5 rounded-full flex items-center justify-center shadow">
                {cartCount}
              </span>
            )}
          </button>
        </div>
      </div>
    </header>
  );
}
