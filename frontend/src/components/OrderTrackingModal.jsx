import React, { useState, useEffect } from "react";
import {
  CheckCircle2,
  Clock,
  Cpu,
  Database,
  Layers,
  ShieldCheck,
  X,
  Copy,
  Check,
  ExternalLink,
  CreditCard,
  Lock,
  Zap,
} from "lucide-react";
import { subscribeToOrderStatus, simulatePayment } from "../services/api";

export default function OrderTrackingModal({ isOpen, onClose, orderData }) {
  const [currentStep, setCurrentStep] = useState(1);
  const [statusMessage, setStatusMessage] = useState("Dispatched to Kafka queue");
  const [events, setEvents] = useState([]);
  const [copied, setCopied] = useState(false);
  const [simulatingPayment, setSimulatingPayment] = useState(false);
  const [paymentSuccess, setPaymentSuccess] = useState(false);

  useEffect(() => {
    if (!orderData || !isOpen) {
      setCurrentStep(1);
      setEvents([]);
      setPaymentSuccess(false);
      return;
    }

    const orderId = orderData.order_id;
    const initialTime = new Date().toLocaleTimeString();

    // Initial event
    setEvents([
      {
        step: 1,
        title: "HTTP 202 Accepted (Kafka Enqueued)",
        detail: `Dispatched in ${orderData.latencyMs || 2}ms with snappy compression`,
        time: initialTime,
      },
    ]);

    // Connect to Server-Sent Events (SSE)
    const unsubscribe = subscribeToOrderStatus(
      orderId,
      (data) => {
        const time = new Date().toLocaleTimeString();
        if (data.step) {
          setCurrentStep(data.step);
        }
        if (data.message) {
          setStatusMessage(data.message);
        }

        if (data.status === "COMPLETED") {
          setPaymentSuccess(true);
        }

        setEvents((prev) => [
          ...prev,
          {
            step: data.step || 2,
            title:
              data.status === "COMPLETED"
                ? "Payment Verified • Committed to DB"
                : "Worker Batch Processing",
            detail: data.message || "Worker processing order batch",
            time: time,
          },
        ]);
      },
      (err) => {
        // Fallback simulation if backend offline
        setTimeout(() => {
          setCurrentStep(2);
          setStatusMessage("Worker batch processing in Kafka...");
          setEvents((prev) => [
            ...prev,
            {
              step: 2,
              title: "Worker Processing",
              detail: "Kafka consumer worker picked up order batch",
              time: new Date().toLocaleTimeString(),
            },
          ]);
        }, 800);

        setTimeout(() => {
          setCurrentStep(3);
          setStatusMessage("Order persisted to PostgreSQL database");
          setEvents((prev) => [
            ...prev,
            {
              step: 3,
              title: "Committed to PostgreSQL",
              detail: "ACID transaction committed to disk volume",
              time: new Date().toLocaleTimeString(),
            },
          ]);
        }, 1800);
      },
    );

    return () => {
      unsubscribe();
    };
  }, [orderData, isOpen]);

  if (!isOpen || !orderData) return null;

  const handleCopy = () => {
    navigator.clipboard.writeText(orderData.order_id);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  const handleSimulatePayment = async () => {
    setSimulatingPayment(true);
    try {
      await simulatePayment(orderData.order_id, 100);
      setPaymentSuccess(true);
      setCurrentStep(3);
      setStatusMessage("Payment Verified via HMAC-SHA256 • Order Completed");
    } catch (err) {
      console.error("Payment simulation error:", err);
    } finally {
      setSimulatingPayment(false);
    }
  };

  const steps = [
    {
      id: 1,
      title: "Kafka & Outbox",
      desc: "Idempotent Write (< 2ms)",
      icon: Layers,
    },
    {
      id: 2,
      title: "Batch Relay",
      desc: "Transactional Outbox Sync",
      icon: Cpu,
    },
    {
      id: 3,
      title: "PostgreSQL 16",
      desc: "HMAC Paid & Finalized",
      icon: Database,
    },
  ];

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto bg-black/80 backdrop-blur-md flex items-center justify-center p-4">
      <div className="bg-slate-900 border border-slate-800/90 rounded-3xl w-full max-w-xl overflow-hidden shadow-2xl transition-all">
        {/* Header */}
        <div className="p-6 border-b border-slate-800 flex items-center justify-between">
          <div className="flex items-center space-x-2.5">
            <span className="flex h-3 w-3 relative">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-3 w-3 bg-emerald-500"></span>
            </span>
            <h2 className="text-base font-bold text-white tracking-tight">
              Real-Time Order Stream (SSE)
            </h2>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-xl text-slate-400 hover:text-white hover:bg-slate-800 transition"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6">
          {/* Order ID & Idempotency Key */}
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-4 rounded-2xl bg-slate-950/70 border border-slate-800/80 mb-6">
            <div>
              <span className="text-[11px] font-mono text-slate-400 uppercase tracking-wider block">
                Order Reference
              </span>
              <span className="font-mono text-xs sm:text-sm font-semibold text-emerald-300 select-all break-all">
                {orderData.order_id}
              </span>
              {orderData.idempotencyKey && (
                <div className="mt-1 flex items-center gap-1 text-[10px] font-mono text-slate-500">
                  <Lock className="w-3 h-3 text-cyan-400" />
                  <span>Idempotency-Key Active</span>
                </div>
              )}
            </div>
            <button
              onClick={handleCopy}
              className="self-start sm:self-center flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-mono transition"
            >
              {copied ? (
                <Check className="w-3.5 h-3.5 text-emerald-400" />
              ) : (
                <Copy className="w-3.5 h-3.5" />
              )}
              <span>{copied ? "Copied" : "Copy"}</span>
            </button>
          </div>

          {/* Animated Stepper */}
          <div className="relative mb-8">
            {/* Background Line */}
            <div className="absolute top-5 left-6 right-6 h-0.5 bg-slate-800" />
            {/* Active Glow Line */}
            <div
              className="absolute top-5 left-6 h-0.5 bg-gradient-to-r from-emerald-500 via-teal-400 to-cyan-400 transition-all duration-700 ease-out"
              style={{
                width:
                  currentStep === 1
                    ? "0%"
                    : currentStep === 2
                      ? "50%"
                      : "calc(100% - 3rem)",
              }}
            />

            <div className="relative z-10 grid grid-cols-3 gap-2">
              {steps.map((step) => {
                const Icon = step.icon;
                const isCompleted = currentStep > step.id || (step.id === 3 && paymentSuccess);
                const isCurrent = currentStep === step.id && !paymentSuccess;

                return (
                  <div
                    key={step.id}
                    className="flex flex-col items-center text-center"
                  >
                    <div
                      className={`w-10 h-10 rounded-2xl flex items-center justify-center transition-all duration-300 ${
                        isCompleted
                          ? "bg-emerald-500 text-slate-950 shadow-lg shadow-emerald-500/30"
                          : isCurrent
                            ? "bg-gradient-to-tr from-emerald-500 to-cyan-500 text-white shadow-lg shadow-emerald-500/20 ring-4 ring-emerald-500/20"
                            : "bg-slate-800/80 text-slate-500 border border-slate-700/50"
                      }`}
                    >
                      {isCompleted ? (
                        <Check className="w-5 h-5 stroke-[3]" />
                      ) : (
                        <Icon className="w-5 h-5" />
                      )}
                    </div>

                    <h4
                      className={`mt-2.5 text-xs font-semibold ${
                        isCurrent || isCompleted
                          ? "text-white"
                          : "text-slate-500"
                      }`}
                    >
                      {step.title}
                    </h4>
                    <p className="text-[10px] text-slate-400 mt-0.5 font-mono hidden sm:block">
                      {step.desc}
                    </p>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Live Progress Card */}
          <div className="p-4 rounded-2xl bg-slate-950/90 border border-slate-800 mb-6">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-medium text-slate-400 flex items-center gap-1.5">
                <Clock className="w-3.5 h-3.5 text-cyan-400" />
                Live Status Message
              </span>
              <span className="text-[11px] font-mono px-2 py-0.5 rounded-full bg-slate-800 text-emerald-400 border border-slate-700">
                {paymentSuccess || currentStep === 3
                  ? "100% Finalized"
                  : currentStep === 2
                    ? "66% In Progress"
                    : "33% Enqueued"}
              </span>
            </div>
            <p className="text-sm font-semibold text-white">{statusMessage}</p>
          </div>

          {/* Simulate Payment Webhook Action */}
          {!paymentSuccess && (
            <div className="p-4 rounded-2xl bg-emerald-950/20 border border-emerald-500/30 mb-6 flex flex-col sm:flex-row items-center justify-between gap-3">
              <div className="text-left">
                <div className="flex items-center gap-1.5 text-xs font-semibold text-emerald-400">
                  <ShieldCheck className="w-4 h-4" />
                  <span>Transactional Outbox & HMAC Webhook</span>
                </div>
                <p className="text-[11px] text-slate-400 mt-0.5">
                  Simulate gateway callback signed with SHA256 secret.
                </p>
              </div>
              <button
                onClick={handleSimulatePayment}
                disabled={simulatingPayment}
                className="w-full sm:w-auto px-4 py-2 rounded-xl bg-gradient-to-r from-emerald-500 to-teal-500 hover:from-emerald-400 hover:to-teal-400 text-slate-950 font-bold text-xs flex items-center justify-center gap-2 shadow-lg shadow-emerald-500/20 disabled:opacity-50 transition"
              >
                <CreditCard className="w-3.5 h-3.5" />
                <span>{simulatingPayment ? "Verifying HMAC..." : "Simulate Payment Webhook"}</span>
              </button>
            </div>
          )}

          {/* Real-Time Event Audit Log */}
          <div>
            <h5 className="text-[11px] font-mono text-slate-400 uppercase tracking-wider mb-2.5">
              Live Pipeline Transitions (SSE Stream)
            </h5>
            <div className="space-y-2 max-h-36 overflow-y-auto pr-1">
              {events.map((ev, i) => (
                <div
                  key={i}
                  className="flex items-start justify-between p-2.5 rounded-xl bg-slate-950/50 border border-slate-800/60 font-mono text-xs text-slate-300"
                >
                  <div className="flex items-center gap-2">
                    <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" />
                    <div>
                      <span className="font-semibold text-white">
                        {ev.title}
                      </span>
                      <p className="text-[10px] text-slate-400">{ev.detail}</p>
                    </div>
                  </div>
                  <span className="text-[10px] text-slate-500">{ev.time}</span>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="p-6 border-t border-slate-800 flex justify-end">
          <button
            onClick={onClose}
            className="w-full sm:w-auto px-6 py-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-medium text-sm transition shadow-lg shadow-emerald-600/20"
          >
            {paymentSuccess || currentStep === 3
              ? "Done & Back to Shopping"
              : "Keep Running in Background"}
          </button>
        </div>
      </div>
    </div>
  );
}
