// Тонкая обёртка над fetch: один слой, который знает про базовый путь API,
// Bearer-токен и формат ошибок {code, description}.
'use strict';

const API_BASE = '/api/v1';
const TOKEN_KEY = 'sentinel.token';

const Api = {
    token() {
        try {
            return localStorage.getItem(TOKEN_KEY);
        } catch (_) {
            return null;
        }
    },

    setToken(token) {
        try {
            if (token) {
                localStorage.setItem(TOKEN_KEY, token);
            } else {
                localStorage.removeItem(TOKEN_KEY);
            }
        } catch (_) {
            // Приватный режим браузера — работаем в рамках одной сессии.
        }
    },

    async request(method, path, body) {
        const headers = {};
        const token = this.token();

        if (token) {
            headers['Authorization'] = `Bearer ${token}`;
        }

        if (body !== undefined) {
            headers['Content-Type'] = 'application/json';
        }

        const response = await fetch(`${API_BASE}${path}`, {
            method,
            headers,
            body: body === undefined ? undefined : JSON.stringify(body),
        });

        if (response.status === 204) {
            return null;
        }

        const text = await response.text();
        const payload = text ? JSON.parse(text) : null;

        if (!response.ok) {
            const error = new Error(payload?.description || `HTTP ${response.status}`);
            error.code = payload?.code || 'HTTP_ERROR';
            error.status = response.status;
            throw error;
        }

        return payload;
    },

    get(path) {
        return this.request('GET', path);
    },
    post(path, body) {
        return this.request('POST', path, body);
    },
    patch(path, body) {
        return this.request('PATCH', path, body);
    },
    delete(path) {
        return this.request('DELETE', path);
    },

    // --- Эндпоинты ---
    register(email, password) {
        return this.post('/auth/register', { email, password });
    },
    login(email, password) {
        return this.post('/auth/login', { email, password });
    },
    me() {
        return this.get('/me');
    },
    listMonitors() {
        return this.get('/monitors?limit=200');
    },
    createMonitor(payload) {
        return this.post('/monitors', payload);
    },
    updateMonitor(id, payload) {
        return this.patch(`/monitors/${id}`, payload);
    },
    deleteMonitor(id) {
        return this.delete(`/monitors/${id}`);
    },
    monitorStats(id, from, to, bucket) {
        const query = new URLSearchParams({ from, to, bucket });
        return this.get(`/monitors/${id}/stats?${query}`);
    },
    monitorChecks(id, from, to, limit) {
        const query = new URLSearchParams({ from, to, limit: String(limit) });
        return this.get(`/monitors/${id}/checks?${query}`);
    },
    monitorIncidents(id) {
        return this.get(`/monitors/${id}/incidents?limit=20`);
    },
    statusPage(slug) {
        return this.get(`/public/status/${encodeURIComponent(slug)}`);
    },
};
