export const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080';

export async function apiFetch(path, options = {}, retry = true) {
  const headers = new Headers(options.headers || {});
  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
    credentials: 'include'
  });

  if (response.status === 401 && retry && path !== '/v1/auth/refresh') {
    const refreshed = await fetch(`${API_BASE}/v1/auth/refresh`, {
      method: 'POST',
      credentials: 'include'
    });
    if (refreshed.ok) {
      return apiFetch(path, options, false);
    }
  }
  return response;
}

export async function readJSON(response) {
  const text = await response.text();
  if (!text) return {};
  try { return JSON.parse(text); }
  catch { return {error: text}; }
}

export async function apiJSON(path, options = {}) {
  const response = await apiFetch(path, options);
  const data = await readJSON(response);
  return {response, data};
}

export async function currentUser() {
  const {response, data} = await apiJSON('/v1/auth/me');
  if (!response.ok) return null;
  return data.user || null;
}

export async function logout() {
  await apiFetch('/v1/auth/logout', {method: 'POST'}, false);
}
