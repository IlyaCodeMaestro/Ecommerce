import React, { useState, useEffect } from "react";
import {
  X,
  Package,
  Clock,
  CheckCircle2,
  AlertCircle,
  Truck,
  RotateCcw,
  Copy,
  Check,
  RefreshCw,
  ChevronRight,
  Shield,
  CreditCard,
  Ban,
  ArrowRight,
} from "lucide-react";
import {
  fetchUserOrdersApi,
  cancelOrderApi,
  adminListOrdersApi,
  adminUpdateOrderStatusApi,
} from "../services/api";

const STATUS_STEPS = [
  { key: "ACCEPTED", label: "Accepted", sub: "Order placed" },
  { key: "PAID", label: "Paid", sub: "HMAC verified" },
  { key: "PROCESSING", label: "Processing", sub: "Warehouse" },
  { key: "SHIPPED", label: "Shipped", sub: "In transit" },
  { key: "DELIVERED", label: "Delivered", sub: "Completed" },
];

function getStepIndex(status) {
  switch (status) {
    case "ACCEPTED":
    case "PENDING":
      return 0;
    case "PAID":
      return 1;
    case "PROCESSING":
      return 2;
    case "SHIPPED":
      return 3;
    case "DELIVERED":
    case "COMPLETED":
      return 4;
    default:
      return -1;
  }
}

function getStatusBadge(status) {
  switch (status) {
    case "ACCEPTED":
    case "PENDING":
      return {
        bg: "bg-amber-500/10 border-amber-500/30 text-amber-400",
        dot: "bg-amber-400 animate-pulse",
        text: "Accepted • Pending Payment",
      };
    case "PAID":
      return {
        bg: "bg-blue-500/10 border-blue-500/30 text-blue-400",
        dot: "bg-blue-400",
        text: "Paid • Verified",
      };
    case "PROCESSING":
      return {
        bg: "bg-indigo-500/10 border-indigo-500/30 text-indigo-400",
        dot: "bg-indigo-400 animate-pulse",
        text: "Processing • Warehouse",
      };
    case "SHIPPED":
      return {
        bg: "bg-purple-500/10 border-purple-500/30 text-purple-400",
        dot: "bg-purple-400",
        text: "Shipped • In Transit",
      };
    case "DELIVERED":
    case "COMPLETED":
      return {
        bg: "bg-emerald-500/10 border-emerald-500/30 text-emerald-400",
        dot: "bg-emerald-400",
        text: "Delivered • Finished",
      };
    case "CANCELLED":
      return {
        bg: "bg-rose-500/10 border-rose-500/30 text-rose-400",
        dot: "bg-rose-500",
        text: "Cancelled • Stock Restocked",
      };
    case "REFUNDED":
      return {
        bg: "bg-slate-500/10 border-slate-500/30 text-slate-400",
        dot: "bg-slate-400",
        text: "Refunded",
      };
    default:
      return {
        bg: "bg-slate-800 border-slate-700 text-slate-300",
        dot: "bg-slate-400",
        text: status,
      };
  }
}

export default function OrderHistoryModal({
  isOpen,
  onClose,
  currentUser,
  onOpenAuth,
}) {
  const [orders, setOrders] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [actionLoadingId, setActionLoadingId] = useState(null);
  const [copiedId, setCopiedId] = useState(null);
  const [adminTab, setAdminTab] = useState("my"); // 'my' or 'admin'
  const [cancellingId, setCancellingId] = useState(null);
  const [cancelReason, setCancelReason] = useState("");

  const isAdmin = currentUser?.role === "admin";

  const loadOrders = async () => {
    if (!currentUser) return;
    setLoading(true);
    setError(null);
    try {
      if (adminTab === "admin" && isAdmin) {
        const data = await adminListOrdersApi("", 30, 0);
        setOrders(data.orders || []);
      } else {
        const data = await fetchUserOrdersApi(30, 0);
        setOrders(data.orders || []);
      }
    } catch (err) {
      console.error(err);
      setError(err.message || "Failed to load order history");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (isOpen) {
      loadOrders();
    } else {
      setCancellingId(null);
      setCancelReason("");
    }
  }, [isOpen, adminTab, currentUser]);

  const handleCopy = (id) => {
    navigator.clipboard.writeText(id);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 1500);
  };

  const handleCancelOrder = async (orderId) => {
    setActionLoadingId(orderId);
    try {
      await cancelOrderApi(orderId, cancelReason || "Customer cancelled via dashboard");
      setCancellingId(null);
      setCancelReason("");
      await loadOrders();
    } catch (err) {
      alert(`Cancellation failed: ${err.message}`);
    } finally {
      setActionLoadingId(null);
    }
  };

  const handleAdminStatusChange = async (orderId, newStatus) => {
    setActionLoadingId(orderId);
    try {
      await adminUpdateOrderStatusApi(orderId, newStatus);
      await loadOrders();
    } catch (err) {
      alert(`Status update failed: ${err.message}`);
    } finally {
      setActionLoadingId(null);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-md animate-fadeIn">
      <div className="relative w-full max-w-4xl bg-slate-900/95 border border-slate-800 rounded-3xl shadow-2xl overflow-hidden flex flex-col max-h-[90vh]">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-5 border-b border-slate-800/80">
          <div className="flex items-center space-x-3">
            <div className="w-10 h-10 rounded-2xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400">
              <Package className="w-5 h-5" />
            </div>
            <div>
              <div className="flex items-center space-x-2">
                <h2 className="text-lg font-bold text-white tracking-tight">
                  Order History & Fulfillment Saga
                </h2>
                <span className="text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-mono font-semibold">
                  FSM Active
                </span>
              </div>
              <p className="text-xs text-slate-400">
                Finite State Machine • Atomic Restock • Server-Sent Events
              </p>
            </div>
          </div>

          <div className="flex items-center space-x-2">
            <button
              onClick={loadOrders}
              disabled={loading}
              className="p-2 rounded-xl text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition"
              title="Refresh orders"
            >
              <RefreshCw
                className={`w-4 h-4 ${loading ? "animate-spin text-emerald-400" : ""}`}
              />
            </button>
            <button
              onClick={onClose}
              className="p-2 rounded-xl text-slate-400 hover:text-white hover:bg-slate-800 transition"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Admin Tabs */}
        {isAdmin && (
          <div className="flex items-center space-x-2 px-6 pt-3 pb-1 border-b border-slate-800/50 bg-slate-950/40">
            <button
              onClick={() => setAdminTab("my")}
              className={`px-3 py-1.5 rounded-xl text-xs font-semibold transition ${
                adminTab === "my"
                  ? "bg-slate-800 text-white border border-slate-700"
                  : "text-slate-400 hover:text-slate-200"
              }`}
            >
              My Orders
            </button>
            <button
              onClick={() => setAdminTab("admin")}
              className={`flex items-center space-x-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold transition ${
                adminTab === "admin"
                  ? "bg-cyan-500/20 text-cyan-300 border border-cyan-500/30"
                  : "text-slate-400 hover:text-slate-200"
              }`}
            >
              <Shield className="w-3.5 h-3.5" />
              <span>All Store Orders (Admin Saga Control)</span>
            </button>
          </div>
        )}

        {/* Content Body */}
        <div className="flex-1 overflow-y-auto p-6 space-y-4">
          {!currentUser ? (
            <div className="py-16 text-center space-y-4">
              <div className="w-14 h-14 mx-auto rounded-3xl bg-slate-800/60 border border-slate-700 flex items-center justify-center text-slate-400">
                <Package className="w-7 h-7" />
              </div>
              <div>
                <h3 className="text-base font-semibold text-white">
                  Sign in to view orders
                </h3>
                <p className="text-xs text-slate-400 max-w-sm mx-auto mt-1">
                  Access your order fulfillment lifecycle, real-time live tracking, and one-click restock cancellation.
                </p>
              </div>
              <button
                onClick={() => {
                  onClose();
                  onOpenAuth();
                }}
                className="px-5 py-2.5 rounded-2xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold transition shadow-lg shadow-emerald-600/20"
              >
                Sign In or Register
              </button>
            </div>
          ) : loading && orders.length === 0 ? (
            <div className="py-16 text-center space-y-3">
              <RefreshCw className="w-8 h-8 text-emerald-400 animate-spin mx-auto" />
              <p className="text-xs text-slate-400 font-mono">
                Querying PostgreSQL order transactions...
              </p>
            </div>
          ) : error ? (
            <div className="p-4 rounded-2xl bg-rose-500/10 border border-rose-500/20 text-rose-300 text-xs flex items-center space-x-3">
              <AlertCircle className="w-5 h-5 text-rose-400 shrink-0" />
              <span>{error}</span>
            </div>
          ) : orders.length === 0 ? (
            <div className="py-16 text-center space-y-3">
              <div className="w-14 h-14 mx-auto rounded-3xl bg-slate-800/40 border border-slate-700/60 flex items-center justify-center text-slate-400">
                <Package className="w-7 h-7" />
              </div>
              <h3 className="text-base font-semibold text-white">
                No orders found
              </h3>
              <p className="text-xs text-slate-400 max-w-sm mx-auto">
                Place an order from the store catalog to watch the asynchronous Kafka ingestion, outbox relay, and FSM lifecycle in action.
              </p>
            </div>
          ) : (
            orders.map((order) => {
              const badge = getStatusBadge(order.status);
              const stepIdx = getStepIndex(order.status);
              const isCancelled = order.status === "CANCELLED" || order.status === "FAILED";
              const canCancel =
                !isCancelled &&
                order.status !== "DELIVERED" &&
                order.status !== "COMPLETED" &&
                order.status !== "REFUNDED";

              return (
                <div
                  key={order.id}
                  className="p-5 rounded-2xl bg-slate-800/40 border border-slate-800 hover:border-slate-700/80 transition space-y-4"
                >
                  {/* Top row: ID, Date, Status */}
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <div className="flex items-center space-x-2.5">
                      <span className="font-mono text-xs text-slate-300 font-bold">
                        #{order.id.substring(0, 13)}...
                      </span>
                      <button
                        onClick={() => handleCopy(order.id)}
                        className="p-1 rounded-lg text-slate-500 hover:text-slate-300 hover:bg-slate-700/50 transition"
                        title="Copy full Order ID"
                      >
                        {copiedId === order.id ? (
                          <Check className="w-3.5 h-3.5 text-emerald-400" />
                        ) : (
                          <Copy className="w-3.5 h-3.5" />
                        )}
                      </button>
                      <span className="text-[11px] text-slate-500 font-mono">
                        {new Date(order.created_at).toLocaleString()}
                      </span>
                    </div>

                    <div className="flex items-center space-x-3">
                      <span
                        className={`inline-flex items-center space-x-1.5 px-3 py-1 rounded-full text-xs font-semibold border ${badge.bg}`}
                      >
                        <span className={`w-1.5 h-1.5 rounded-full ${badge.dot}`} />
                        <span>{badge.text}</span>
                      </span>

                      <span className="text-sm font-black text-white font-mono">
                        ${Number(order.total_amount).toFixed(2)}
                      </span>
                    </div>
                  </div>

                  {/* Items summary */}
                  {order.items && order.items.length > 0 && (
                    <div className="flex flex-wrap gap-2 pt-1">
                      {order.items.map((item, idx) => (
                        <div
                          key={idx}
                          className="px-2.5 py-1 rounded-lg bg-slate-900/60 border border-slate-800 text-[11px] text-slate-300 font-mono flex items-center space-x-2"
                        >
                          <span>Item #{item.product_id}</span>
                          <span className="text-emerald-400 font-bold">×{item.quantity}</span>
                          <span className="text-slate-500">${Number(item.price).toFixed(2)}</span>
                        </div>
                      ))}
                    </div>
                  )}

                  {/* FSM Progress Timeline */}
                  {!isCancelled ? (
                    <div className="py-2">
                      <div className="grid grid-cols-5 gap-1.5">
                        {STATUS_STEPS.map((step, idx) => {
                          const isDone = stepIdx >= idx;
                          const isCurrent = stepIdx === idx;
                          return (
                            <div key={step.key} className="space-y-1">
                              <div
                                className={`h-1.5 rounded-full transition-all ${
                                  isDone
                                    ? isCurrent
                                      ? "bg-emerald-400 shadow-sm shadow-emerald-400/50"
                                      : "bg-emerald-600"
                                    : "bg-slate-700/50"
                                }`}
                              />
                              <div className="text-left">
                                <span
                                  className={`text-[10px] font-semibold block leading-none ${
                                    isCurrent
                                      ? "text-emerald-400"
                                      : isDone
                                      ? "text-slate-300"
                                      : "text-slate-600"
                                  }`}
                                >
                                  {step.label}
                                </span>
                                <span className="text-[9px] text-slate-500 hidden sm:block">
                                  {step.sub}
                                </span>
                              </div>
                            </div>
                          );
                        })}
                      </div>
                    </div>
                  ) : (
                    <div className="p-3 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-300 text-xs flex items-center space-x-2">
                      <Ban className="w-4 h-4 text-rose-400 shrink-0" />
                      <span>
                        Order cancelled. Compensating transaction atomically restored reserved stock in PostgreSQL & Redis RAM.
                      </span>
                    </div>
                  )}

                  {/* Actions Bar */}
                  <div className="pt-2 border-t border-slate-800/60 flex flex-wrap items-center justify-between gap-3">
                    {/* Cancellation Trigger */}
                    {canCancel && cancellingId !== order.id && (
                      <button
                        onClick={() => setCancellingId(order.id)}
                        disabled={actionLoadingId === order.id}
                        className="px-3 py-1.5 rounded-xl bg-slate-800 hover:bg-rose-500/10 text-slate-400 hover:text-rose-400 border border-slate-700 hover:border-rose-500/30 text-xs font-semibold transition flex items-center space-x-1.5"
                      >
                        <RotateCcw className="w-3.5 h-3.5" />
                        <span>Cancel Order (Restock)</span>
                      </button>
                    )}

                    {/* Inline Cancel Confirmation */}
                    {cancellingId === order.id && (
                      <div className="flex flex-wrap items-center gap-2 bg-slate-900/80 p-2 rounded-xl border border-rose-500/30 w-full sm:w-auto">
                        <input
                          type="text"
                          placeholder="Reason (optional)"
                          value={cancelReason}
                          onChange={(e) => setCancelReason(e.target.value)}
                          className="px-2.5 py-1 text-xs bg-slate-800 border border-slate-700 rounded-lg text-white focus:outline-none focus:border-rose-500 placeholder-slate-500"
                        />
                        <button
                          onClick={() => handleCancelOrder(order.id)}
                          disabled={actionLoadingId === order.id}
                          className="px-3 py-1 bg-rose-600 hover:bg-rose-500 text-white rounded-lg text-xs font-bold transition flex items-center space-x-1"
                        >
                          {actionLoadingId === order.id ? (
                            <RefreshCw className="w-3 h-3 animate-spin" />
                          ) : (
                            <Check className="w-3 h-3" />
                          )}
                          <span>Confirm Cancel</span>
                        </button>
                        <button
                          onClick={() => setCancellingId(null)}
                          className="px-2 py-1 text-slate-400 hover:text-white text-xs"
                        >
                          Keep Order
                        </button>
                      </div>
                    )}

                    {/* Admin Status Controls */}
                    {isAdmin && (
                      <div className="flex items-center space-x-1 ml-auto">
                        <span className="text-[10px] font-mono text-slate-500 mr-1 hidden sm:inline">
                          Admin FSM:
                        </span>
                        {["PAID", "PROCESSING", "SHIPPED", "DELIVERED"].map((st) => (
                          <button
                            key={st}
                            onClick={() => handleAdminStatusChange(order.id, st)}
                            disabled={actionLoadingId === order.id || order.status === st}
                            className={`px-2 py-1 rounded-lg text-[10px] font-mono font-bold transition ${
                              order.status === st
                                ? "bg-slate-700 text-slate-400 cursor-not-allowed"
                                : "bg-slate-800 hover:bg-cyan-500/20 text-slate-300 hover:text-cyan-300 border border-slate-700"
                            }`}
                          >
                            {st}
                          </button>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              );
            })
          )}
        </div>

        {/* Footer info pill */}
        <div className="px-6 py-3 border-t border-slate-800/80 bg-slate-950/60 flex items-center justify-between text-xs text-slate-400">
          <div className="flex items-center space-x-2">
            <Shield className="w-4 h-4 text-emerald-400" />
            <span className="font-mono text-[11px]">
              Transactional Outbox • Distributed Saga • Zero Phantom Stock
            </span>
          </div>
          <span className="text-[11px] font-mono text-slate-500">
            Auto-Sweeper Active (15m TTL)
          </span>
        </div>
      </div>
    </div>
  );
}
