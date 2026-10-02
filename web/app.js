// Логика панели управления: аутентификация и CRUD над мониторами.
'use strict';

const state = {
    mode: 'login',
    user: null,
    monitors: [],
    openMonitorId: null,
    refreshTimer: null,
};

const el = (id) => document.getElementById(id);

// --- Вспомогательное ---

function toast(message, kind = 'error') {
    const node = el('toast');
    node.textContent = message;
    node.className = `toast toast--${kind}`;
    node.hidden = false;

    clearTimeout(toast.timer);
    toast.timer = setTimeout(() => {
        node.hidden = true;
    }, 4000);
}

function formatDateTime(value) {
    if (!value) {
        return '—';
    }
    return new Date(value).toLocaleString('ru-RU', { dateStyle: 'short', timeStyle: 'medium' });
}

function formatDuration(seconds) {
    if (seconds < 60) {
        return `${seconds} с`;
    }
    if (seconds < 3600) {
        return `${Math.round(seconds / 60)} мин`;
    }
    return `${(seconds / 3600).toFixed(1)} ч`;
}

function formatPercent(ratio) {
    return `${(ratio * 100).toFixed(2)}%`;
}

function statusLabel(status) {
    return { up: 'Работает', down: 'Недоступен', pending: 'Ожидает', paused: 'На паузе' }[status] || status;
}

// --- Аутентификация ---

function setAuthMode(mode) {
    state.mode = mode;
    const isLogin = mode === 'login';

    el('auth-title').textContent = isLogin ? 'Вход' : 'Регистрация';
    el('auth-subtitle').textContent = isLogin
        ? 'Войдите, чтобы управлять мониторами.'
        : 'Создайте аккаунт — публичная статус-страница появится сразу.';
    el('auth-submit').textContent = isLogin ? 'Войти' : 'Зарегистрироваться';
    el('auth-switch-text').textContent = isLogin ? 'Нет аккаунта?' : 'Уже есть аккаунт?';
    el('auth-switch').textContent = isLogin ? 'Зарегистрироваться' : 'Войти';
    el('auth-form').querySelector('[name=password]').autocomplete = isLogin
        ? 'current-password'
        : 'new-password';
}

async function handleAuthSubmit(event) {
    event.preventDefault();

    const form = new FormData(event.target);
    const email = form.get('email').trim();
    const password = form.get('password');

    try {
        const result = state.mode === 'login'
            ? await Api.login(email, password)
            : await Api.register(email, password);

        Api.setToken(result.token);
        await enterApp(result.user);
    } catch (error) {
        toast(describeError(error));
    }
}

function describeError(error) {
    const messages = {
        INVALID_CREDENTIALS: 'Неверный email или пароль.',
        EMAIL_ALREADY_USED: 'Этот email уже зарегистрирован.',
        WEAK_PASSWORD: 'Пароль должен быть не короче 8 символов.',
        INVALID_EMAIL: 'Некорректный email.',
        MONITOR_NAME_TAKEN: 'Монитор с таким названием уже есть.',
        INVALID_TARGET: 'Некорректный URL: нужен http:// или https://.',
        TARGET_NOT_ALLOWED: 'Этот адрес проверять нельзя: приватные и служебные сети запрещены.',
        INVALID_SCHEDULE: 'Таймаут должен быть меньше интервала проверки.',
        NOTHING_TO_UPDATE: 'Нечего обновлять.',
    };

    return messages[error.code] || error.message || 'Что-то пошло не так.';
}

function logout() {
    Api.setToken(null);
    state.user = null;
    state.monitors = [];
    clearInterval(state.refreshTimer);

    el('app-screen').hidden = true;
    el('session-actions').hidden = true;
    el('auth-screen').hidden = false;
    el('auth-form').reset();
}

async function enterApp(user) {
    state.user = user;

    el('auth-screen').hidden = true;
    el('app-screen').hidden = false;
    el('session-actions').hidden = false;
    el('session-email').textContent = user.email;

    const statusLink = el('status-page-link');
    statusLink.href = `status.html?slug=${encodeURIComponent(user.status_page_slug)}`;

    await loadMonitors();

    clearInterval(state.refreshTimer);
    state.refreshTimer = setInterval(loadMonitors, 15000);
}

// --- Мониторы ---

async function loadMonitors() {
    try {
        const result = await Api.listMonitors();
        state.monitors = result.items;
        renderMonitors();

        if (state.openMonitorId) {
            await openDetail(state.openMonitorId, { silent: true });
        }
    } catch (error) {
        if (error.status === 401) {
            logout();
            return;
        }
        toast(describeError(error));
    }
}

function renderMonitors() {
    const body = el('monitors-body');
    body.textContent = '';

    el('monitors-empty').hidden = state.monitors.length > 0;

    for (const monitor of state.monitors) {
        const row = document.createElement('tr');

        row.append(
            cell(statusPill(monitor)),
            cell(linkButton(monitor.name, () => openDetail(monitor.id))),
            cell(textNode(monitor.target, 'mono truncate')),
            cell(textNode(`${monitor.interval_seconds} с`)),
            cell(textNode(formatDateTime(monitor.last_checked_at))),
            cell(textNode(monitor.is_public ? 'да' : 'нет')),
            cell(rowActions(monitor)),
        );

        body.append(row);
    }
}

function cell(child) {
    const td = document.createElement('td');
    td.append(child);
    return td;
}

function textNode(text, className) {
    const span = document.createElement('span');
    span.textContent = text;
    if (className) {
        span.className = className;
    }
    return span;
}

function statusPill(monitor) {
    const pill = document.createElement('span');
    pill.className = `pill pill--${monitor.status}`;
    pill.textContent = statusLabel(monitor.status);
    return pill;
}

function linkButton(text, onClick) {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'link';
    button.textContent = text;
    button.addEventListener('click', onClick);
    return button;
}

function rowActions(monitor) {
    const wrap = document.createElement('div');
    wrap.className = 'row-actions';

    const toggle = document.createElement('button');
    toggle.type = 'button';
    toggle.className = 'btn btn--small';
    toggle.textContent = monitor.paused ? 'Включить' : 'Пауза';
    toggle.addEventListener('click', () => togglePause(monitor));

    const edit = document.createElement('button');
    edit.type = 'button';
    edit.className = 'btn btn--small';
    edit.textContent = 'Изменить';
    edit.addEventListener('click', () => openForm(monitor));

    const remove = document.createElement('button');
    remove.type = 'button';
    remove.className = 'btn btn--small btn--danger';
    remove.textContent = 'Удалить';
    remove.addEventListener('click', () => removeMonitor(monitor));

    wrap.append(toggle, edit, remove);
    return wrap;
}

async function togglePause(monitor) {
    try {
        await Api.updateMonitor(monitor.id, { paused: !monitor.paused });
        await loadMonitors();
    } catch (error) {
        toast(describeError(error));
    }
}

async function removeMonitor(monitor) {
    if (!confirm(`Удалить монитор «${monitor.name}» вместе с историей проверок?`)) {
        return;
    }

    try {
        await Api.deleteMonitor(monitor.id);

        if (state.openMonitorId === monitor.id) {
            closeDetail();
        }

        await loadMonitors();
        toast('Монитор удалён.', 'ok');
    } catch (error) {
        toast(describeError(error));
    }
}

// --- Форма создания и редактирования ---

function openForm(monitor) {
    const form = el('monitor-form');
    form.reset();

    el('monitor-form-card').hidden = false;
    el('monitor-form-title').textContent = monitor ? 'Изменить монитор' : 'Новый монитор';
    el('monitor-submit').textContent = monitor ? 'Сохранить' : 'Создать';

    if (monitor) {
        form.elements.id.value = monitor.id;
        form.elements.name.value = monitor.name;
        form.elements.target.value = monitor.target;
        form.elements.method.value = monitor.method;
        form.elements.interval_seconds.value = monitor.interval_seconds;
        form.elements.timeout_seconds.value = monitor.timeout_seconds;
        form.elements.expected_status.value = monitor.expected_status;
        form.elements.failure_threshold.value = monitor.failure_threshold;
        form.elements.is_public.checked = monitor.is_public;
    }

    form.elements.name.focus();
}

function closeForm() {
    el('monitor-form-card').hidden = true;
    el('monitor-form').reset();
}

async function handleMonitorSubmit(event) {
    event.preventDefault();

    const form = event.target.elements;
    const payload = {
        name: form.name.value.trim(),
        target: form.target.value.trim(),
        method: form.method.value,
        interval_seconds: Number(form.interval_seconds.value),
        timeout_seconds: Number(form.timeout_seconds.value),
        expected_status: Number(form.expected_status.value),
        failure_threshold: Number(form.failure_threshold.value),
        is_public: form.is_public.checked,
    };

    try {
        if (form.id.value) {
            await Api.updateMonitor(form.id.value, payload);
            toast('Монитор обновлён.', 'ok');
        } else {
            await Api.createMonitor(payload);
            toast('Монитор создан, первая проверка пойдёт в ближайшие секунды.', 'ok');
        }

        closeForm();
        await loadMonitors();
    } catch (error) {
        toast(describeError(error));
    }
}

// --- Детали монитора ---

async function openDetail(monitorId, options = {}) {
    state.openMonitorId = monitorId;

    const monitor = state.monitors.find((item) => item.id === monitorId);
    if (!monitor) {
        closeDetail();
        return;
    }

    const to = new Date();
    const from = new Date(to.getTime() - 24 * 3600 * 1000);

    try {
        const [stats, checks, incidents] = await Promise.all([
            Api.monitorStats(monitorId, from.toISOString(), to.toISOString(), 'hour'),
            Api.monitorChecks(monitorId, from.toISOString(), to.toISOString(), 50),
            Api.monitorIncidents(monitorId),
        ]);

        el('detail-card').hidden = false;
        el('detail-title').textContent = monitor.name;

        renderStats(stats, monitor);
        renderChart(stats.buckets);
        renderIncidents(incidents.items);
        renderChecks(checks.items);

        if (!options.silent) {
            el('detail-card').scrollIntoView({ behavior: 'smooth', block: 'nearest' });
        }
    } catch (error) {
        toast(describeError(error));
    }
}

function closeDetail() {
    state.openMonitorId = null;
    el('detail-card').hidden = true;
}

function renderStats(stats, monitor) {
    const items = [
        ['Статус', statusLabel(monitor.status)],
        ['Аптайм за 24 ч', stats.total_checks ? formatPercent(stats.uptime_ratio) : '—'],
        ['Проверок', String(stats.total_checks)],
        ['Средняя задержка', stats.total_checks ? `${Math.round(stats.avg_latency_ms)} мс` : '—'],
        ['p95 задержки', stats.total_checks ? `${stats.p95_latency_ms} мс` : '—'],
        ['Цель', monitor.target],
    ];

    const container = el('detail-stats');
    container.textContent = '';

    for (const [label, value] of items) {
        const box = document.createElement('div');
        box.className = 'stat';

        const title = document.createElement('span');
        title.className = 'stat__label';
        title.textContent = label;

        const content = document.createElement('span');
        content.className = 'stat__value';
        content.textContent = value;

        box.append(title, content);
        container.append(box);
    }
}

// Простая столбиковая диаграмма на SVG: без библиотек, по данным корзин,
// которые БД уже свернула за нас.
function renderChart(buckets) {
    const container = el('detail-chart');
    container.textContent = '';

    if (!buckets.length) {
        container.append(textNode('Нет данных за выбранный период.', 'muted'));
        return;
    }

    const width = 100;
    const height = 30;
    const gap = 0.4;
    const barWidth = width / buckets.length;

    const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    svg.setAttribute('viewBox', `0 0 ${width} ${height}`);
    svg.setAttribute('preserveAspectRatio', 'none');
    svg.classList.add('chart__svg');

    buckets.forEach((bucket, index) => {
        const ratio = bucket.uptime_ratio;
        const barHeight = Math.max(height * ratio, 0.8);

        const rect = document.createElementNS('http://www.w3.org/2000/svg', 'rect');
        rect.setAttribute('x', String(index * barWidth + gap / 2));
        rect.setAttribute('y', String(height - barHeight));
        rect.setAttribute('width', String(Math.max(barWidth - gap, 0.4)));
        rect.setAttribute('height', String(barHeight));
        rect.setAttribute('class', ratio >= 1 ? 'bar bar--up' : ratio > 0 ? 'bar bar--partial' : 'bar bar--down');

        const title = document.createElementNS('http://www.w3.org/2000/svg', 'title');
        title.textContent = `${formatDateTime(bucket.bucket)}: ${formatPercent(ratio)} (${bucket.successful}/${bucket.total})`;
        rect.append(title);

        svg.append(rect);
    });

    container.append(svg);
}

function renderIncidents(incidents) {
    const body = el('detail-incidents');
    body.textContent = '';

    el('detail-incidents-empty').hidden = incidents.length > 0;

    for (const incident of incidents) {
        const row = document.createElement('tr');
        const duration = incident.is_open
            ? `${formatDuration(incident.duration_seconds)} (идёт)`
            : formatDuration(incident.duration_seconds);

        row.append(
            cell(textNode(formatDateTime(incident.started_at))),
            cell(textNode(duration)),
            cell(textNode(incident.cause, 'mono')),
        );

        body.append(row);
    }
}

function renderChecks(checks) {
    const body = el('detail-checks');
    body.textContent = '';

    for (const check of checks) {
        const row = document.createElement('tr');
        const result = document.createElement('span');
        result.className = `pill pill--${check.up ? 'up' : 'down'}`;
        result.textContent = check.up ? 'ок' : 'сбой';

        row.append(
            cell(textNode(formatDateTime(check.checked_at))),
            cell(result),
            cell(textNode(check.status_code ?? '—')),
            cell(textNode(`${check.latency_ms} мс`)),
            cell(textNode(check.error ?? '', 'mono truncate')),
        );

        body.append(row);
    }
}

// --- Инициализация ---

function bindEvents() {
    el('auth-form').addEventListener('submit', handleAuthSubmit);
    el('auth-switch').addEventListener('click', () => setAuthMode(state.mode === 'login' ? 'register' : 'login'));
    el('logout-btn').addEventListener('click', logout);

    el('new-monitor-btn').addEventListener('click', () => openForm(null));
    el('monitor-cancel').addEventListener('click', closeForm);
    el('monitor-form').addEventListener('submit', handleMonitorSubmit);
    el('refresh-btn').addEventListener('click', loadMonitors);
    el('detail-close').addEventListener('click', closeDetail);
}

async function init() {
    bindEvents();
    setAuthMode('login');

    if (!Api.token()) {
        return;
    }

    // Токен есть — проверяем, жив ли он, одним запросом /me.
    try {
        const user = await Api.me();
        await enterApp(user);
    } catch (_) {
        logout();
    }
}

init();
