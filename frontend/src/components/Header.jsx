import React from 'react';
import { ShoppingBag, Activity, Zap, Server, BarChart3, User, LogOut, Shield } from 'lucide-react';

export default function Header({
  cartCount,
  onOpenCart,
  onOpenMetrics,
  onOpenAuth,
  currentUser,
  onLogout,
  backendStatus,
  pingMs,
}) {
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
        <div className="flex items-center space-x-3 sm:space-x-4">
          {/* Health Pill */}
          <div className="hidden lg:flex items-center space-x-2 px-3 py-1 rounded-full bg-slate-800/80 border border-slate-700/60 text-xs">
            <span
              className={`w-2 h-2 rounded-full ${
                backendStatus ? 'bg-emerald-400 animate-pulse' : 'bg-rose-500'
              }`}
            />
            <span className="text-slate-300 font-mono">
              {backendStatus ? `API Live (${pingMs}ms)` : 'Offline / Connecting'}
            </span>
          </div>

          {/* Quick Metrics Link / Modal */}
          <button
            onClick={onOpenMetrics}
            className="flex items-center space-x-2 px-3 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 transition text-sm font-medium"
            title="Open High-Load Simulator & Prometheus guide"
          >
            <BarChart3 className="w-4 h-4 text-cyan-400" />
            <span className="hidden md:inline">Telemetry</span>
          </button>

          {/* User Auth Pill / Button */}
          {currentUser ? (
            <div className="flex items-center space-x-2 pl-2 border-l border-slate-800">
              <div className="flex items-center space-x-2 px-2.5 py-1.5 rounded-xl bg-slate-800/90 border border-slate-700 text-xs">
                <div className="w-6 h-6 rounded-lg bg-emerald-500/20 border border-emerald-500/30 flex items-center justify-center text-emerald-400 font-bold text-[10px]">
                  {currentUser.first_name ? currentUser.first_name[0].toUpperCase() : 'U'}
                </div>
                <div className="hidden sm:block text-left">
                  <span className="font-semibold text-white block leading-tight">
                    {currentUser.first_name || currentUser.email.split('@')[0]}
                  </span>
                  <span
                    className={`text-[9px] font-mono uppercase tracking-wider block ${
                      currentUser.role === 'admin' ? 'text-cyan-400 font-bold' : 'text-slate-400'
                    }`}
                  >
                    {currentUser.role}
                  </span>
                </div>
              </div>
              <button
                onClick={onLogout}
                className="p-2 rounded-xl text-slate-400 hover:text-rose-400 hover:bg-slate-800 transition"
                title="Logout"
              >
                <LogOut className="w-4 h-4" />
              </button>
            </div>
          ) : (
            <button
              onClick={onOpenAuth}
              className="flex items-center space-x-1.5 px-3 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 transition text-xs font-semibold"
            >
              <User className="w-3.5 h-3.5 text-slate-400" />
              <span>Sign In</span>
            </button>
          )}

          {/* Cart Button */}
          <button
            onClick={onOpenCart}
            className="relative flex items-center space-x-2 px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-medium text-sm transition shadow-lg shadow-emerald-600/20"
          >
            <ShoppingBag className="w-4 h-4" />
            <span className="hidden sm:inline">Cart</span>
            {cartCount > 0 && (
              <span className="absolute -top-1.5 -right-1.5 bg-amber-500 text-slate-950 font-black text-xs w-5 h-5 rounded-full flex items-center justify-center shadow">
                {cartCount}
              </span>
            )}
          </button>
        </div>
      </div>
    </header>
  );
}
