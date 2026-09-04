import React, { useState } from 'react';
import { Plus, Check, Box, Laptop, Smartphone, Headphones, Monitor, Sparkles } from 'lucide-react';

export default function ProductCard({ product, onAddToCart }) {
  const [added, setAdded] = useState(false);

  const handleAdd = () => {
    onAddToCart(product);
    setAdded(true);
    setTimeout(() => setAdded(false), 1200);
  };

  const getCategoryIcon = (cat) => {
    switch (cat) {
      case 'laptops': return <Laptop className="w-3.5 h-3.5" />;
      case 'smartphones': return <Smartphone className="w-3.5 h-3.5" />;
      case 'audio': return <Headphones className="w-3.5 h-3.5" />;
      case 'monitors': return <Monitor className="w-3.5 h-3.5" />;
      default: return <Sparkles className="w-3.5 h-3.5" />;
    }
  };

  const categoryColors = {
    laptops: 'bg-blue-500/10 text-blue-400 border-blue-500/20',
    smartphones: 'bg-purple-500/10 text-purple-400 border-purple-500/20',
    audio: 'bg-amber-500/10 text-amber-400 border-amber-500/20',
    monitors: 'bg-cyan-500/10 text-cyan-400 border-cyan-500/20',
    accessories: 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20',
  };

  const badgeClass = categoryColors[product.category] || 'bg-slate-800 text-slate-300 border-slate-700';

  return (
    <div className="group relative bg-slate-900/90 hover:bg-slate-900 border border-slate-800/80 hover:border-slate-700 rounded-3xl p-5 flex flex-col justify-between transition-all duration-300 hover:shadow-2xl hover:shadow-emerald-500/5 hover:-translate-y-1">
      {/* Glow on hover */}
      <div className="absolute inset-0 rounded-3xl bg-gradient-to-b from-emerald-500/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none" />

      <div>
        {/* Top Badges */}
        <div className="flex items-center justify-between mb-3.5">
          <span className={`inline-flex items-center gap-1.5 text-[11px] px-2.5 py-1 rounded-full border font-mono uppercase tracking-wider ${badgeClass}`}>
            {getCategoryIcon(product.category)}
            {product.category}
          </span>
          <span className="text-[11px] text-slate-400 flex items-center gap-1 font-mono">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
            {product.stock_quantity.toLocaleString()} in stock
          </span>
        </div>

        {/* Product SKU */}
        <span className="text-[10px] font-mono text-slate-500 block mb-1">
          {product.sku}
        </span>

        {/* Product Title */}
        <h3 className="font-bold text-white text-base tracking-tight group-hover:text-emerald-400 transition-colors duration-200 mb-2 line-clamp-1">
          {product.name}
        </h3>

        {/* Description */}
        <p className="text-xs text-slate-400 line-clamp-2 mb-4 leading-relaxed">
          {product.description}
        </p>
      </div>

      {/* Footer / Price & CTA */}
      <div className="pt-4 border-t border-slate-800/60 flex items-center justify-between">
        <div>
          <span className="text-[10px] uppercase font-mono text-slate-500 block">Price</span>
          <div className="flex items-baseline gap-0.5">
            <span className="text-xs font-mono text-emerald-400 font-bold">$</span>
            <span className="text-xl font-black text-white font-mono tracking-tight">
              {product.price.toFixed(2)}
            </span>
          </div>
        </div>

        <button
          onClick={handleAdd}
          className={`flex items-center space-x-1.5 px-3.5 py-2 rounded-2xl text-xs font-semibold transition-all duration-200 active:scale-95 ${
            added
              ? 'bg-emerald-500 text-slate-950 font-bold shadow-lg shadow-emerald-500/20'
              : 'bg-slate-800/90 hover:bg-emerald-600 hover:text-white text-slate-200 border border-slate-700/80 hover:border-emerald-600'
          }`}
        >
          {added ? <Check className="w-3.5 h-3.5 stroke-[3]" /> : <Plus className="w-3.5 h-3.5" />}
          <span>{added ? 'Added' : 'Add to Bag'}</span>
        </button>
      </div>
    </div>
  );
}
