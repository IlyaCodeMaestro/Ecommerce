const API_BASE_URL = import.meta.env.VITE_API_URL || '';

export async function fetchProducts(category = '', limit = 50, offset = 0) {
  const params = new URLSearchParams();
  if (category) params.append('category', category);
  params.append('limit', limit.toString());
  params.append('offset', offset.toString());

  const res = await fetch(`${API_BASE_URL}/api/v1/products?${params.toString()}`);
  if (!res.ok) throw new Error(`Failed to load products: ${res.statusText}`);
  return res.json();
}

export async function fetchCategories() {
  const res = await fetch(`${API_BASE_URL}/api/v1/categories`);
  if (!res.ok) throw new Error(`Failed to load categories: ${res.statusText}`);
  return res.json();
}

export async function createOrder(items, userId = 'user-frontend') {
  const payload = {
    user_id: userId,
    items: items.map(item => ({
      product_id: item.id,
      quantity: item.quantity,
    })),
  };

  const startTime = performance.now();
  const res = await fetch(`${API_BASE_URL}/api/v1/orders`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  const latencyMs = Math.round(performance.now() - startTime);

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Order failed' }));
    throw new Error(err.error || `HTTP ${res.status}`);
  }

  const data = await res.json();
  return { ...data, latencyMs };
}

export async function checkHealth() {
  try {
    const res = await fetch(`${API_BASE_URL}/healthz`);
    if (!res.ok) return { healthy: false };
    const data = await res.json();
    return { healthy: true, ...data };
  } catch (err) {
    return { healthy: false, error: err.message };
  }
}
