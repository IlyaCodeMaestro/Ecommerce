const API_BASE_URL = import.meta.env.VITE_API_URL || '';

// Token and Auth State Management (localStorage)
const TOKEN_KEY = 'highload_access_token';
const REFRESH_KEY = 'highload_refresh_token';
const USER_KEY = 'highload_user';

export function getAccessToken() {
  return localStorage.getItem(TOKEN_KEY) || null;
}

export function getRefreshToken() {
  return localStorage.getItem(REFRESH_KEY) || null;
}

export function getStoredUser() {
  try {
    const raw = localStorage.getItem(USER_KEY);
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

export function setAuthData(authResponse) {
  if (authResponse.access_token) {
    localStorage.setItem(TOKEN_KEY, authResponse.access_token);
  }
  if (authResponse.refresh_token) {
    localStorage.setItem(REFRESH_KEY, authResponse.refresh_token);
  }
  if (authResponse.user) {
    localStorage.setItem(USER_KEY, JSON.stringify(authResponse.user));
  }
}

export function clearAuthData() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(REFRESH_KEY);
  localStorage.removeItem(USER_KEY);
}

// Authentication API calls
export async function loginUser(email, password) {
  const res = await fetch(`${API_BASE_URL}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Login failed' }));
    throw new Error(err.error || `Login failed (HTTP ${res.status})`);
  }

  const data = await res.json();
  setAuthData(data);
  return data;
}

export async function registerUser({ email, password, firstName, lastName }) {
  const res = await fetch(`${API_BASE_URL}/api/v1/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      email,
      password,
      first_name: firstName,
      last_name: lastName,
    }),
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Registration failed' }));
    throw new Error(err.error || `Registration failed (HTTP ${res.status})`);
  }

  const data = await res.json();
  setAuthData(data);
  return data;
}

export async function refreshAuthToken() {
  const currentRefreshToken = getRefreshToken();
  if (!currentRefreshToken) {
    clearAuthData();
    return null;
  }

  try {
    const res = await fetch(`${API_BASE_URL}/api/v1/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: currentRefreshToken }),
    });

    if (!res.ok) {
      clearAuthData();
      return null;
    }

    const data = await res.json();
    setAuthData(data);
    return data;
  } catch (err) {
    clearAuthData();
    return null;
  }
}

export async function logoutUser() {
  const currentRefreshToken = getRefreshToken();
  try {
    if (currentRefreshToken) {
      await fetch(`${API_BASE_URL}/api/v1/auth/logout`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: currentRefreshToken }),
      });
    }
  } catch (err) {
    console.warn('Logout request failed:', err);
  } finally {
    clearAuthData();
  }
}

export async function fetchCurrentUser() {
  const token = getAccessToken();
  if (!token) return null;

  const res = await fetch(`${API_BASE_URL}/api/v1/auth/me`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    if (res.status === 401) {
      // Try refresh
      const refreshed = await refreshAuthToken();
      if (refreshed && refreshed.access_token) {
        return fetchCurrentUser();
      }
      clearAuthData();
      return null;
    }
    return null;
  }

  const user = await res.json();
  localStorage.setItem(USER_KEY, JSON.stringify(user));
  return user;
}

// Products & Categories
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

// Orders & Checkout
export async function createOrder(items, userId = 'user-frontend', customIdempotencyKey = null) {
  const idempotencyKey =
    customIdempotencyKey ||
    (typeof crypto !== 'undefined' && crypto.randomUUID
      ? crypto.randomUUID()
      : `idemp-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`);

  const payload = {
    user_id: userId,
    idempotency_key: idempotencyKey,
    items: items.map((item) => ({
      product_id: item.id,
      quantity: item.quantity,
    })),
  };

  const headers = {
    'Content-Type': 'application/json',
    'Idempotency-Key': idempotencyKey,
  };

  const token = getAccessToken();
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const startTime = performance.now();
  const res = await fetch(`${API_BASE_URL}/api/v1/orders`, {
    method: 'POST',
    headers,
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
  return { ...data, latencyMs, idempotencyKey };
}

// Payment simulation
export async function simulatePayment(orderId, amount = 0) {
  const res = await fetch(`${API_BASE_URL}/api/v1/payments/simulate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ order_id: orderId, amount }),
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Payment simulation failed' }));
    throw new Error(err.error || `HTTP ${res.status}`);
  }

  return res.json();
}

// Health Check
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
