import React, { useState } from 'react';
import { X, Trash2, CheckCircle2, ArrowRight, Zap, RefreshCw, Cpu, Layers } from 'lucide-react';
import { createOrder } from '../services/api';

export default function CartModal({ isOpen, onClose, cart, onUpdateQty, onRemove, onClear }) {
  const [loading, setLoading] = useState(false);
  const [orderResult, setOrderResult] = useState(null);
  const [error, setError] = useState(null);

  if (!isOpen) return null;

  const total = cart.reduce((sum, item) => sum + item.price * item.quantity, 0);

  const handleCheckout = async () => {
    if (cart.length === 0) return;
    setLoading(true);
    setError(null);
    setOrderResult(null);

    try {
      const res = await createOrder(cart);
      setOrderResult(res);
      onClear();
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-slate-900 border border-slate-800 rounded-3xl w-full max-w-lg overflow-hidden shadow-2xl">
        {/* Header */}
        <div className="p-6 border-b border-slate-800 flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <Zap className="w-5 h-5 text-emerald-400" />
            <h2 className="text-lg font-bold text-white">Your Shopping Cart</h2>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6">
          {orderResult ? (
            <div className="text-center py-6">
              <div className="w-16 h-16 bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 rounded-2xl mx-auto flex items-center justify-center mb-4">
                <CheckCircle2 className="w-8 h-8" />
              </div>
              <h3 className="text-xl font-bold text-white mb-1">Order Accepted!</h3>
              <p className="text-xs text-slate-400 mb-6">
                Dispatched asynchronously into Kafka in <span className="text-emerald-400 font-mono font-bold">{orderResult.latencyMs} ms</span>
              </p>

              {/* Architecture execution breakdown */}
              <div className="bg-slate-950/70 border border-slate-800/80 rounded-2xl p-4 text-left space-y-3 mb-6 font-mono text-xs">
                <div className="flex items-center justify-between border-b border-slate-800/60 pb-2">
                  <span className="text-slate-400">Order ID</span>
                  <span className="text-emerald-300 font-bold truncate max-w-[200px]">{orderResult.order_id}</span>
                </div>
                <div className="flex items-center justify-between border-b border-slate-800/60 pb-2">
                  <span className="text-slate-400">HTTP Status</span>
                  <span className="text-emerald-400 font-bold">202 Accepted</span>
                </div>
                <div className="flex items-center justify-between border-b border-slate-800/60 pb-2">
                  <span className="text-slate-400">1. Redis Lua Script</span>
                  <span className="text-cyan-400">Stock Decremented</span>
                </div>
                <div className="flex items-center justify-between border-b border-slate-800/60 pb-2">
                  <span className="text-slate-400">2. Kafka Message</span>
                  <span className="text-amber-400">Published (snappy)</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-slate-400">3. Postgres Worker</span>
                  <span className="text-purple-400">Batch Syncing</span>
                </div>
              </div>

              <button
                onClick={() => setOrderResult(null)}
                className="w-full py-3 rounded-xl bg-slate-800 hover:bg-slate-700 text-white font-medium text-sm transition"
              >
                Continue Shopping
              </button>
            </div>
          ) : cart.length === 0 ? (
            <div className="text-center py-12 text-slate-500">
              <p className="text-sm">Your cart is empty.</p>
              <p className="text-xs mt-1 text-slate-600">Add products from the catalog to test checkout flow.</p>
            </div>
          ) : (
            <>
              {error && (
                <div className="mb-4 p-3 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-400 text-xs">
                  {error}
                </div>
              )}

              <div className="space-y-3 max-h-72 overflow-y-auto pr-1">
                {cart.map((item) => (
                  <div
                    key={item.id}
                    className="flex items-center justify-between p-3 rounded-2xl bg-slate-950/60 border border-slate-800/80"
                  >
                    <div className="flex-1 mr-3">
                      <h4 className="text-sm font-medium text-white truncate">{item.name}</h4>
                      <span className="text-xs text-slate-400 font-mono">${item.price.toFixed(2)}</span>
                    </div>

                    <div className="flex items-center space-x-2">
                      <button
                        onClick={() => onUpdateQty(item.id, Math.max(1, item.quantity - 1))}
                        className="w-7 h-7 rounded-lg bg-slate-800 hover:bg-slate-700 text-white flex items-center justify-center text-xs"
                      >
                        -
                      </button>
                      <span className="font-mono text-sm w-6 text-center text-white">{item.quantity}</span>
                      <button
                        onClick={() => onUpdateQty(item.id, item.quantity + 1)}
                        className="w-7 h-7 rounded-lg bg-slate-800 hover:bg-slate-700 text-white flex items-center justify-center text-xs"
                      >
                        +
                      </button>
                      <button
                        onClick={() => onRemove(item.id)}
                        className="p-1.5 text-slate-500 hover:text-rose-400 transition"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                ))}
              </div>

              {/* Total & Checkout */}
              <div className="mt-6 pt-4 border-t border-slate-800">
                <div className="flex justify-between items-center mb-4">
                  <span className="text-slate-400 text-sm">Total Amount</span>
                  <span className="text-2xl font-bold font-mono text-white">${total.toFixed(2)}</span>
                </div>

                <button
                  disabled={loading}
                  onClick={handleCheckout}
                  className="w-full py-3.5 rounded-2xl bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white font-semibold flex items-center justify-center space-x-2 transition shadow-lg shadow-emerald-600/25"
                >
                  {loading ? (
                    <>
                      <RefreshCw className="w-5 h-5 animate-spin" />
                      <span>Submitting to Kafka...</span>
                    </>
                  ) : (
                    <>
                      <span>Place High-Speed Order (Async 202)</span>
                      <ArrowRight className="w-4 h-4" />
                    </>
                  )}
                </button>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
