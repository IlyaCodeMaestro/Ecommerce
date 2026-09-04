import React from 'react';
import { Plus, Check, Box } from 'lucide-react';

export default function ProductCard({ product, onAddToCart }) {
  const [added, setAdded] = React.useState(false);

  const handleAdd = () => {
    onAddToCart(product);
    setAdded(true);
    setTimeout(() => setAdded(false), 1200);
  };

  // Category badge colors
  const categoryColors = {
    laptops: 'bg-blue-500/10 text-blue-400 border-blue-500/20',
    smartphones: 'bg-purple-500/10 text-purple-400 border-purple-500/20',
    audio: 'bg-amber-500/10 text-amber-400 border-amber-500/20',
    monitors: 'bg-cyan-500/10 text-cyan-400 border-cyan-500/20',
    accessories: 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20',
  };

  const badgeClass = categoryColors[product.category] || 'bg-slate-800 text-slate-300 border-slate-700';

  return (
    <div className="group relative bg-slate-900 border border-slate-800 hover:border-slate-700 rounded-2xl p-5 flex flex-col justify-between transition-all duration-200 hover:shadow-xl hover:shadow-emerald-500/5">
      <div>
        <div className="flex items-center justify-between mb-3">
          <span className={`text-xs px-2.5 py-1 rounded-full border font-medium uppercase tracking-wider ${badgeClass}`}>
            {product.category}
          </span>
          <span className="text-xs text-slate-400 flex items-center gap-1 font-mono">
            <Box className="w-3.5 h-3.5 text-slate-500" />
            Stock: {product.stock_quantity.toLocaleString()}
          </span>
        </div>

        <h3 className="font-semibold text-white text-base group-hover:text-emerald-400 transition mb-2">
          {product.name}
        </h3>

        <p className="text-xs text-slate-400 line-clamp-2 mb-4 leading-relaxed">
          {product.description}
        </p>
      </div>

      <div className="pt-4 border-t border-slate-800/80 flex items-center justify-between">
        <div>
          <span className="text-xs text-slate-500 block">Price</span>
          <span className="text-xl font-bold text-white font-mono">
            ${product.price.toFixed(2)}
          </span>
        </div>

        <button
          onClick={handleAdd}
          className={`flex items-center space-x-1.5 px-3.5 py-2 rounded-xl text-sm font-medium transition ${
            added
              ? 'bg-emerald-500 text-white'
              : 'bg-slate-800 hover:bg-emerald-600 hover:text-white text-slate-200 border border-slate-700'
          }`}
        >
          {added ? <Check className="w-4 h-4" /> : <Plus className="w-4 h-4" />}
          <span>{added ? 'Added' : 'Add'}</span>
        </button>
      </div>
    </div>
  );
}
