import React, { useState, useEffect } from 'react';
import Header from './components/Header';
import ProductCard from './components/ProductCard';
import CartModal from './components/CartModal';
import OrderTrackingModal from './components/OrderTrackingModal';
import MetricsWidget from './components/MetricsWidget';
import { fetchProducts, fetchCategories, checkHealth } from './services/api';
import { Search, Sparkles, Filter, RefreshCw, ShoppingBag, ShieldCheck } from 'lucide-react';

const FALLBACK_PRODUCTS = [
  { id: 1, sku: 'SKU-00001', name: 'UltraBook Pro 15" M3', description: 'Flagship developer laptop with 16-core CPU, 64GB Unified RAM, 2TB NVMe SSD.', price: 2499.99, category: 'laptops', stock_quantity: 980 },
  { id: 2, sku: 'SKU-00002', name: 'CyberPhone Max 16 Pro', description: 'Next-gen titanium flagship smartphone with OLED 120Hz display and neural engine.', price: 1199.00, category: 'smartphones', stock_quantity: 1450 },
  { id: 3, sku: 'SKU-00003', name: 'Studio Headphones Wireless ANC', description: 'Planar magnetic audiophile headphones with zero-latency lossless streaming.', price: 349.50, category: 'audio', stock_quantity: 520 },
  { id: 4, sku: 'SKU-00004', name: '4K Gaming Monitor 32" 240Hz', description: 'Quantum dot OLED ultra-wide monitor with 0.03ms response time and HDR1000.', price: 899.99, category: 'monitors', stock_quantity: 310 },
  { id: 5, sku: 'SKU-00005', name: 'Mechanical Keyboard RGB Wireless', description: 'Custom CNC aluminum hot-swap mechanical keyboard with lubed linear switches.', price: 179.00, category: 'accessories', stock_quantity: 2200 },
  { id: 6, sku: 'SKU-00006', name: 'Workstation Studio 27" 5K Retina', description: 'True-tone studio display with 600 nits brightness, studio mic array and six-speaker sound.', price: 1599.00, category: 'monitors', stock_quantity: 450 },
  { id: 7, sku: 'SKU-00007', name: 'Pro Wireless Mouse 8000Hz', description: 'Ultralight carbon composite chassis, optical microswitches, sub-millisecond polling.', price: 129.99, category: 'accessories', stock_quantity: 1800 },
  { id: 8, sku: 'SKU-00008', name: 'Hi-Fi DAC Amplifier USB-C', description: 'Dual ESS Sabre DACs, balanced 4.4mm output, DSD512 native hardware decoding.', price: 219.00, category: 'audio', stock_quantity: 670 },
];

export default function App() {
  const [products, setProducts] = useState([]);
  const [categories, setCategories] = useState(['all', 'laptops', 'smartphones', 'audio', 'monitors', 'accessories']);
  const [selectedCategory, setSelectedCategory] = useState('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [loading, setLoading] = useState(true);
  const [backendOnline, setBackendOnline] = useState(false);
  const [pingMs, setPingMs] = useState(0);

  // Cart state
  const [cart, setCart] = useState([]);
  const [cartOpen, setCartOpen] = useState(false);

  // Order tracking state
  const [placedOrder, setPlacedOrder] = useState(null);

  // Telemetry modal state
  const [metricsOpen, setMetricsOpen] = useState(false);

  // Health check polling
  useEffect(() => {
    const checkStatus = async () => {
      const start = performance.now();
      const status = await checkHealth();
      const elapsed = Math.round(performance.now() - start);
      setBackendOnline(status.healthy);
      setPingMs(elapsed);
    };

    checkStatus();
    const interval = setInterval(checkStatus, 5000);
    return () => clearInterval(interval);
  }, []);

  // Fetch products
  useEffect(() => {
    const loadCatalog = async () => {
      setLoading(true);
      try {
        const catParam = selectedCategory === 'all' ? '' : selectedCategory;
        const data = await fetchProducts(catParam, 40);
        if (data && data.products && data.products.length > 0) {
          setProducts(data.products);
        } else {
          setProducts(FALLBACK_PRODUCTS);
        }
      } catch (err) {
        setProducts(FALLBACK_PRODUCTS);
      } finally {
        setLoading(false);
      }
    };

    loadCatalog();
  }, [selectedCategory]);

  // Cart operations
  const addToCart = (product) => {
    setCart((prev) => {
      const existing = prev.find((item) => item.id === product.id);
      if (existing) {
        return prev.map((item) =>
          item.id === product.id ? { ...item, quantity: item.quantity + 1 } : item
        );
      }
      return [...prev, { ...product, quantity: 1 }];
    });
  };

  const updateCartQty = (productId, quantity) => {
    setCart((prev) =>
      prev.map((item) => (item.id === productId ? { ...item, quantity } : item))
    );
  };

  const removeFromCart = (productId) => {
    setCart((prev) => prev.filter((item) => item.id !== productId));
  };

  const clearCart = () => setCart([]);

  const filteredProducts = products.filter((p) =>
    p.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    p.description.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const cartTotalItems = cart.reduce((sum, item) => sum + item.quantity, 0);

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col selection:bg-emerald-500 selection:text-black">
      {/* Top Navbar */}
      <Header
        cartCount={cartTotalItems}
        onOpenCart={() => setCartOpen(true)}
        onOpenMetrics={() => setMetricsOpen(true)}
        backendStatus={backendOnline}
        pingMs={pingMs}
      />

      {/* Hero Header */}
      <section className="relative overflow-hidden border-b border-slate-800/80 bg-gradient-to-b from-slate-900/60 via-slate-950/40 to-slate-950 py-12 sm:py-16">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 text-center relative z-10">
          <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-xs font-mono mb-5 shadow-sm">
            <Sparkles className="w-3.5 h-3.5" />
            <span>High-Throughput E-Commerce Platform</span>
          </div>

          <h1 className="text-4xl sm:text-5xl lg:text-6xl font-black tracking-tight text-white max-w-3xl mx-auto leading-tight">
            Engineered for <span className="bg-gradient-to-r from-emerald-400 via-teal-300 to-cyan-400 bg-clip-text text-transparent">10,000 RPS</span>
          </h1>

          <p className="mt-4 text-xs sm:text-sm md:text-base text-slate-400 max-w-2xl mx-auto leading-relaxed px-4">
            Sub-millisecond L1/L2 caching, singleflight cache stampede guard, atomic Redis stock reservations, and asynchronous Kafka batch writes to PostgreSQL 16.
          </p>

          {/* Quick Metrics Bar */}
          <div className="mt-8 grid grid-cols-2 sm:grid-cols-4 gap-3 max-w-3xl mx-auto font-mono text-xs px-2">
            <div className="p-3.5 rounded-2xl bg-slate-900/80 border border-slate-800 shadow-sm">
              <div className="text-slate-400 text-[11px]">READ LATENCY</div>
              <div className="text-emerald-400 font-bold text-base mt-0.5">&lt; 0.5 ms</div>
            </div>
            <div className="p-3.5 rounded-2xl bg-slate-900/80 border border-slate-800 shadow-sm">
              <div className="text-slate-400 text-[11px]">WRITE ORDER ACK</div>
              <div className="text-cyan-400 font-bold text-base mt-0.5">&lt; 2.0 ms</div>
            </div>
            <div className="p-3.5 rounded-2xl bg-slate-900/80 border border-slate-800 shadow-sm">
              <div className="text-slate-400 text-[11px]">CACHE HIT RATIO</div>
              <div className="text-purple-400 font-bold text-base mt-0.5">&gt; 98.4%</div>
            </div>
            <div className="p-3.5 rounded-2xl bg-slate-900/80 border border-slate-800 shadow-sm">
              <div className="text-slate-400 text-[11px]">RATE LIMITER</div>
              <div className="text-amber-400 font-bold text-base mt-0.5">Active (Redis)</div>
            </div>
          </div>
        </div>
      </section>

      {/* Main Catalog */}
      <main className="flex-1 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 w-full">
        {/* Search and Filters */}
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-4 mb-8">
          {/* Category Tabs */}
          <div className="flex items-center gap-1.5 overflow-x-auto pb-2 sm:pb-0 scrollbar-none">
            {categories.map((cat) => (
              <button
                key={cat}
                onClick={() => setSelectedCategory(cat)}
                className={`px-3.5 py-1.5 rounded-xl text-xs font-medium capitalize whitespace-nowrap transition-all duration-200 ${
                  selectedCategory === cat
                    ? 'bg-emerald-500 text-slate-950 font-bold shadow-md shadow-emerald-500/20'
                    : 'bg-slate-900 text-slate-400 hover:text-white hover:bg-slate-800 border border-slate-800/90'
                }`}
              >
                {cat}
              </button>
            ))}
          </div>

          {/* Search Input */}
          <div className="relative w-full sm:w-80">
            <Search className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              placeholder="Search products or SKU..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-4 py-2 bg-slate-900 border border-slate-800 focus:border-emerald-500 rounded-xl text-xs text-white placeholder-slate-500 focus:outline-none transition shadow-inner"
            />
          </div>
        </div>

        {/* Product Grid */}
        {loading ? (
          <div className="py-24 flex flex-col items-center justify-center text-slate-500 space-y-3">
            <RefreshCw className="w-6 h-6 animate-spin text-emerald-400" />
            <span className="text-xs font-mono">Fetching catalog from high-speed cache...</span>
          </div>
        ) : filteredProducts.length === 0 ? (
          <div className="py-20 text-center text-slate-500">
            <p className="text-sm">No products found matching "{searchQuery}".</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-5">
            {filteredProducts.map((product) => (
              <ProductCard
                key={product.id}
                product={product}
                onAddToCart={addToCart}
              />
            ))}
          </div>
        )}
      </main>

      {/* Floating Mobile Cart Action Button */}
      {cartTotalItems > 0 && !cartOpen && !placedOrder && (
        <div className="fixed bottom-6 right-6 sm:hidden z-40 animate-in fade-in slide-in-from-bottom-5">
          <button
            onClick={() => setCartOpen(true)}
            className="flex items-center gap-2 px-5 py-3.5 rounded-full bg-emerald-500 text-slate-950 font-bold shadow-2xl shadow-emerald-500/40 active:scale-95 transition"
          >
            <ShoppingBag className="w-5 h-5" />
            <span>Bag ({cartTotalItems})</span>
          </button>
        </div>
      )}

      {/* Footer */}
      <footer className="border-t border-slate-900 bg-slate-950 py-8 text-center text-xs text-slate-500">
        <div className="max-w-7xl mx-auto px-4 flex flex-col sm:flex-row items-center justify-between gap-4">
          <p>Production High-Load E-Commerce Architecture • Zero-Allocation Go &amp; React</p>
          <div className="flex gap-4 font-mono text-[11px]">
            <span className="text-slate-400">Postgres 16</span>
            <span>•</span>
            <span className="text-slate-400">Redis 7</span>
            <span>•</span>
            <span className="text-slate-400">Redpanda/Kafka</span>
            <span>•</span>
            <span className="text-slate-400">Prometheus &amp; Grafana</span>
          </div>
        </div>
      </footer>

      {/* Cart Modal */}
      <CartModal
        isOpen={cartOpen}
        onClose={() => setCartOpen(false)}
        cart={cart}
        onUpdateQty={updateCartQty}
        onRemove={removeFromCart}
        onClear={clearCart}
        onOrderPlaced={(orderInfo) => setPlacedOrder(orderInfo)}
      />

      {/* Real-Time Order Stream (SSE) Modal */}
      <OrderTrackingModal
        isOpen={!!placedOrder}
        onClose={() => setPlacedOrder(null)}
        orderData={placedOrder}
      />

      {/* Telemetry & Benchmark Modal */}
      <MetricsWidget
        isOpen={metricsOpen}
        onClose={() => setMetricsOpen(false)}
      />
    </div>
  );
}
