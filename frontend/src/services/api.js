const API_BASE_URL = import.meta.env.VITE_API_URL || '';

export async function fetchProducts(options = {}) {
  const params = new URLSearchParams();

  // Support both legacy fetchProducts('laptops') and object fetchProducts({ category: 'laptops' })
  if (typeof options === 'string') {
    if (options && options !== 'all') params.append('category', options);
    params.append('limit', '50');
  } else {
    const { category = '', query = '', minPrice = null, maxPrice = null, sort = '', limit = 50, offset = 0 } = options;
    if (category && category !== 'all') params.append('category', category);
    if (query) params.append('q', query);
    if (minPrice !== null && minPrice !== undefined) params.append('min_price', minPrice.toString());
    if (maxPrice !== null && maxPrice !== undefined) params.append('max_price', maxPrice.toString());
    if (sort) params.append('sort', sort);
    params.append('limit', limit.toString());
    params.append('offset', offset.toString());
  }

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
    const error = new Error(err.error || `HTTP ${res.status}`);
    error.status = res.status;
    throw error;
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

// Real-time Server-Sent Events (SSE) subscriber
export function subscribeToOrderStatus(orderId, onMessage, onError) {
  const sseUrl = `${API_BASE_URL}/api/v1/orders/${orderId}/stream`;
  const eventSource = new EventSource(sseUrl);

  eventSource.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      onMessage(data);
      if (data.status === 'COMPLETED' || data.status === 'FAILED') {
        eventSource.close();
      }
    } catch (e) {
      console.warn('SSE message parse error:', e);
    }
  };

  eventSource.onerror = (err) => {
    if (onError) onError(err);
    eventSource.close();
  };

  return () => {
    eventSource.close();
  };
}
