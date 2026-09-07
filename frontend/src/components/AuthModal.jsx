import React, { useState } from 'react';
import {
  X,
  Mail,
  Lock,
  User,
  Shield,
  ArrowRight,
  AlertTriangle,
  CheckCircle2,
  Sparkles,
  Zap,
} from 'lucide-react';
import { loginUser, registerUser } from '../services/api';

export default function AuthModal({ isOpen, onClose, onAuthSuccess }) {
  const [isLogin, setIsLogin] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [successMsg, setSuccessMsg] = useState(null);

  // Form fields
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [firstName, setFirstName] = useState('');
  const [lastName, setLastName] = useState('');

  if (!isOpen) return null;

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError(null);
    setSuccessMsg(null);
    setLoading(true);

    try {
      if (isLogin) {
        const resp = await loginUser(email, password);
        setSuccessMsg(`Welcome back, ${resp.user.first_name || resp.user.email}!`);
        setTimeout(() => {
          onAuthSuccess(resp.user);
          onClose();
        }, 700);
      } else {
        const resp = await registerUser({
          email,
          password,
          firstName,
          lastName,
        });
        setSuccessMsg('Account created successfully! Logged in.');
        setTimeout(() => {
          onAuthSuccess(resp.user);
          onClose();
        }, 700);
      }
    } catch (err) {
      setError(err.message || 'Authentication failed');
    } finally {
      setLoading(false);
    }
  };

  const handleQuickLogin = (demoEmail, demoPassword) => {
    setEmail(demoEmail);
    setPassword(demoPassword);
    setError(null);
  };

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto bg-black/80 backdrop-blur-md flex items-center justify-center p-4">
      <div className="bg-slate-900 border border-slate-800/90 rounded-3xl w-full max-w-md overflow-hidden shadow-2xl animate-in fade-in zoom-in-95 duration-200">
        {/* Header */}
        <div className="p-6 border-b border-slate-800 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-2xl bg-gradient-to-tr from-emerald-500 to-cyan-500 flex items-center justify-center shadow-lg shadow-emerald-500/20">
              <Shield className="w-5 h-5 text-white" />
            </div>
            <div>
              <h3 className="text-lg font-bold text-white tracking-tight">
                {isLogin ? 'Sign In' : 'Create Account'}
              </h3>
              <p className="text-xs text-slate-400 font-mono">
                Stateless JWT • HMAC-SHA256 • Redis Token Rotation
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-xl text-slate-400 hover:text-white hover:bg-slate-800 transition"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Tab Switcher */}
        <div className="p-6 pb-2">
          <div className="grid grid-cols-2 p-1 bg-slate-950 rounded-2xl border border-slate-800 mb-5">
            <button
              type="button"
              onClick={() => {
                setIsLogin(true);
                setError(null);
                setSuccessMsg(null);
              }}
              className={`py-2 text-xs font-semibold rounded-xl transition ${
                isLogin
                  ? 'bg-slate-800 text-white shadow-sm'
                  : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              Sign In
            </button>
            <button
              type="button"
              onClick={() => {
                setIsLogin(false);
                setError(null);
                setSuccessMsg(null);
              }}
              className={`py-2 text-xs font-semibold rounded-xl transition ${
                !isLogin
                  ? 'bg-slate-800 text-white shadow-sm'
                  : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              Register
            </button>
          </div>

          {error && (
            <div className="mb-4 p-3 rounded-2xl bg-rose-500/10 border border-rose-500/20 text-rose-400 text-xs flex items-start gap-2">
              <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" />
              <span>{error}</span>
            </div>
          )}

          {successMsg && (
            <div className="mb-4 p-3 rounded-2xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-xs flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 shrink-0" />
              <span>{successMsg}</span>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            {!isLogin && (
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-[11px] font-mono text-slate-400 mb-1">
                    First Name
                  </label>
                  <div className="relative">
                    <User className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
                    <input
                      type="text"
                      required
                      value={firstName}
                      onChange={(e) => setFirstName(e.target.value)}
                      placeholder="Jane"
                      className="w-full bg-slate-950/80 border border-slate-800 rounded-xl pl-9 pr-3 py-2 text-xs text-white placeholder:text-slate-600 focus:outline-none focus:border-emerald-500 transition"
                    />
                  </div>
                </div>
                <div>
                  <label className="block text-[11px] font-mono text-slate-400 mb-1">
                    Last Name
                  </label>
                  <input
                    type="text"
                    required
                    value={lastName}
                    onChange={(e) => setLastName(e.target.value)}
                    placeholder="Doe"
                    className="w-full bg-slate-950/80 border border-slate-800 rounded-xl px-3 py-2 text-xs text-white placeholder:text-slate-600 focus:outline-none focus:border-emerald-500 transition"
                  />
                </div>
              </div>
            )}

            <div>
              <label className="block text-[11px] font-mono text-slate-400 mb-1">
                Email Address
              </label>
              <div className="relative">
                <Mail className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
                <input
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="developer@example.com"
                  className="w-full bg-slate-950/80 border border-slate-800 rounded-xl pl-9 pr-3 py-2.5 text-xs text-white placeholder:text-slate-600 focus:outline-none focus:border-emerald-500 transition"
                />
              </div>
            </div>

            <div>
              <label className="block text-[11px] font-mono text-slate-400 mb-1">
                Password
              </label>
              <div className="relative">
                <Lock className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
                <input
                  type="password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••"
                  className="w-full bg-slate-950/80 border border-slate-800 rounded-xl pl-9 pr-3 py-2.5 text-xs text-white placeholder:text-slate-600 focus:outline-none focus:border-emerald-500 transition"
                />
              </div>
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full py-2.5 mt-2 rounded-xl bg-gradient-to-r from-emerald-500 to-teal-500 hover:from-emerald-400 hover:to-teal-400 text-slate-950 font-bold text-xs flex items-center justify-center gap-2 shadow-lg shadow-emerald-500/20 disabled:opacity-50 transition"
            >
              {loading ? (
                <span>Authenticating...</span>
              ) : (
                <>
                  <span>{isLogin ? 'Sign In to Account' : 'Register Account'}</span>
                  <ArrowRight className="w-3.5 h-3.5" />
                </>
              )}
            </button>
          </form>

          {/* Seeded Demo Accounts Helper */}
          <div className="mt-6 pt-5 border-t border-slate-800/80">
            <span className="text-[10px] font-mono text-slate-500 uppercase tracking-wider block mb-2.5">
              Instant Demo Accounts (Seeded in PostgreSQL)
            </span>
            <div className="grid grid-cols-2 gap-2">
              <button
                type="button"
                onClick={() => handleQuickLogin('test@example.com', 'secret123')}
                className="p-2.5 rounded-xl bg-slate-950/70 border border-slate-800 hover:border-slate-700 text-left transition group"
              >
                <div className="flex items-center justify-between">
                  <span className="text-xs font-semibold text-white group-hover:text-emerald-400">
                    Customer
                  </span>
                  <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                    User
                  </span>
                </div>
                <p className="text-[10px] font-mono text-slate-500 mt-1 truncate">
                  test@example.com
                </p>
              </button>

              <button
                type="button"
                onClick={() => handleQuickLogin('admin@ecommerce.internal', 'admin123')}
                className="p-2.5 rounded-xl bg-slate-950/70 border border-slate-800 hover:border-slate-700 text-left transition group"
              >
                <div className="flex items-center justify-between">
                  <span className="text-xs font-semibold text-white group-hover:text-cyan-400">
                    Admin
                  </span>
                  <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-cyan-500/10 text-cyan-400 border border-cyan-500/20">
                    RBAC
                  </span>
                </div>
                <p className="text-[10px] font-mono text-slate-500 mt-1 truncate">
                  admin@ecommerce.internal
                </p>
              </button>
            </div>
          </div>
        </div>

        {/* Footer info */}
        <div className="px-6 py-4 bg-slate-950/40 border-t border-slate-800/60 flex items-center justify-between text-[11px] text-slate-500">
          <span className="flex items-center gap-1.5">
            <Sparkles className="w-3.5 h-3.5 text-amber-400" />
            15m Access Token • 7d Refresh
          </span>
          <span className="font-mono text-[10px] text-slate-600">
            BCrypt Cost 10
          </span>
        </div>
      </div>
    </div>
  );
}
