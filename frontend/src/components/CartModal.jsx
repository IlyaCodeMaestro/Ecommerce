import React, { useState } from "react";
import {
  X,
  Trash2,
  ArrowRight,
  Zap,
  RefreshCw,
  AlertTriangle,
} from "lucide-react";
import { createOrder } from "../services/api";

export default function CartModal({
  isOpen,
  onClose,
  cart,
  onUpdateQty,
  onRemove,
  onClear,
  onOrderPlaced,
}) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  if (!isOpen) return null;

  const total = cart.reduce((sum, item) => sum + item.price * item.quantity, 0);

  const handleCheckout = async () => {
    if (cart.length === 0) return;
    setLoading(true);
    setError(null);

    try {
      const res = await createOrder(cart);
      onClear();
      onClose();
      if (onOrderPlaced) {
        onOrderPlaced(res);
      }
    } catch (err) {
      if (err.status === 429) {
        setError(
          "🛑 Rate Limit Exceeded: High-concurrency protection active (max 40 orders/min per IP). Please wait a moment.",
        );
      } else {
        setError(err.message || "Order failed");
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto bg-black/75 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-slate-900 border border-slate-800 rounded-3xl w-full max-w-lg overflow-hidden shadow-2xl animate-in fade-in zoom-in-95 duration-200">
        {/* Header */}
        <div className="p-6 border-b border-slate-800 flex items-center justify-between">
          <div className="flex items-center space-x-2.5">
            <div className="w-8 h-8 rounded-lg bg-emerald-500/10 flex items-center justify-center text-emerald-400">
              <Zap className="w-4 h-4" />
            </div>
            <h2 className="text-lg font-bold text-white tracking-tight">
              Shopping Bag
            </h2>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-xl text-slate-400 hover:text-white hover:bg-slate-800 transition"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6">
          {error && (
            <div className="mb-4 p-3.5 rounded-2xl bg-rose-500/10 border border-rose-500/20 text-rose-400 text-xs flex items-start gap-2">
              <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" />
              <span>{error}</span>
            </div>
          )}

          {cart.length === 0 ? (
            <div className="text-center py-12 text-slate-500">
              <div className="w-12 h-12 rounded-2xl bg-slate-800/50 flex items-center justify-center mx-auto mb-3 text-slate-600">
                <Zap className="w-6 h-6" />
              </div>
              <p className="text-sm font-medium text-slate-300">
                Your shopping bag is empty
              </p>
              <p className="text-xs mt-1 text-slate-500">
                Explore the catalog and add products to test instant checkout.
              </p>
            </div>
          ) : (
            <>
              <div className="space-y-3 max-h-72 overflow-y-auto pr-1">
                {cart.map((item) => (
                  <div
                    key={item.id}
                    className="flex items-center justify-between p-3.5 rounded-2xl bg-slate-950/60 border border-slate-800/80 transition hover:border-slate-700"
                  >
                    <div className="flex-1 mr-3 min-w-0">
                      <h4 className="text-sm font-semibold text-white truncate">
                        {item.name}
                      </h4>
                      <span className="text-xs text-emerald-400 font-mono">
                        ${item.price.toFixed(2)}
                      </span>
                    </div>

                    <div className="flex items-center space-x-2">
                      <button
                        onClick={() =>
                          onUpdateQty(item.id, Math.max(1, item.quantity - 1))
                        }
                        className="w-7 h-7 rounded-xl bg-slate-800 hover:bg-slate-700 text-white flex items-center justify-center text-xs transition"
                      >
                        -
                      </button>
                      <span className="font-mono text-xs w-6 text-center text-white">
                        {item.quantity}
                      </span>
                      <button
                        onClick={() => onUpdateQty(item.id, item.quantity + 1)}
                        className="w-7 h-7 rounded-xl bg-slate-800 hover:bg-slate-700 text-white flex items-center justify-center text-xs transition"
                      >
                        +
                      </button>
                      <button
                        onClick={() => onRemove(item.id)}
                        className="p-1.5 text-slate-500 hover:text-rose-400 transition ml-1"
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
                  <span className="text-slate-400 text-xs uppercase font-mono tracking-wider">
                    Subtotal
                  </span>
                  <span className="text-2xl font-black font-mono text-white">
                    ${total.toFixed(2)}
                  </span>
                </div>

                <button
                  disabled={loading}
                  onClick={handleCheckout}
                  className="w-full py-3.5 rounded-2xl bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 disabled:opacity-50 text-white font-semibold text-sm flex items-center justify-center space-x-2 transition shadow-lg shadow-emerald-600/25 active:scale-[0.99]"
                >
                  {loading ? (
                    <>
                      <RefreshCw className="w-4 h-4 animate-spin" />
                      <span>Writing to Kafka Queue...</span>
                    </>
                  ) : (
                    <>
                      <span>Place High-Speed Order (Async 202)</span>
                      <ArrowRight className="w-4 h-4" />
                    </>
                  )}
                </button>
                <p className="text-[11px] text-center text-slate-500 mt-2 font-mono">
                  Atomic stock reservation in Redis • Write-behind to Kafka
                </p>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
