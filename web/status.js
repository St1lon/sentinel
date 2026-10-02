// Публичная статус-страница: читает один эндпоинт без авторизации.
'use strict';

const slug = new URLSearchParams(location.search).get('slug');

function formatDateTime(value) {
    return new Date(value).toLocaleString('ru-RU', { dateStyle: 'short', timeStyle: 'short' });
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

const overallText = {
    operational: ['Все сервисы работают', 'up'],
    degraded: ['Часть сервисов ещё не проверена', 'pending'],
    down: ['Есть недоступные сервисы', 'down'],
    unknown: ['Данных пока нет', 'pending'],
};

function renderOverall(page) {
    const [title, tone] = overallText[page.overall] || overallText.unknown;

    document.getElementById('overall-title').textContent = title;
    document.getElementById('overall-dot').className = `dot dot--${tone}`;
    document.getElementById('overall-subtitle').textContent =
        `Период: ${formatDateTime(page.from)} — ${formatDateTime(page.to)}`;
}

function renderServices(services) {
    const container = document.getElementById('services');
    container.textContent = '';

    if (!services.length) {
        const empty = document.createElement('p');
        empty.className = 'muted center';
        empty.textContent = 'На этой странице пока нет публичных сервисов.';
        container.append(empty);
        return;
    }

    for (const service of services) {
        const card = document.createElement('section');
        card.className = 'card service';

        const head = document.createElement('div');
        head.className = 'service__head';

        const name = document.createElement('strong');
        name.textContent = service.name;

        const uptime = document.createElement('span');
        uptime.className = 'muted';
        uptime.textContent = service.buckets.length
            ? `аптайм ${(service.uptime_ratio * 100).toFixed(2)}%`
            : 'нет данных';

        const pill = document.createElement('span');
        pill.className = `pill pill--${service.status}`;
        pill.textContent = { up: 'работает', down: 'недоступен', pending: 'ожидает', paused: 'на паузе' }[service.status]
            || service.status;

        head.append(name, pill, uptime);
        card.append(head, renderBar(service.buckets));
        container.append(card);
    }
}

// Полоса из 90 дней: один прямоугольник на день, как на статус-страницах хостингов.
function renderBar(buckets) {
    const wrap = document.createElement('div');
    wrap.className = 'uptime-bar';

    if (!buckets.length) {
        return wrap;
    }

    for (const bucket of buckets) {
        const segment = document.createElement('span');
        const ratio = bucket.uptime_ratio;

        segment.className = `uptime-bar__day ${ratio >= 1 ? 'is-up' : ratio > 0 ? 'is-partial' : 'is-down'}`;
        segment.title = `${formatDateTime(bucket.bucket)}: ${(ratio * 100).toFixed(2)}% (${bucket.successful}/${bucket.total})`;

        wrap.append(segment);
    }

    return wrap;
}

function renderIncidents(incidents) {
    const card = document.getElementById('incidents-card');
    const body = document.getElementById('incidents-body');

    body.textContent = '';
    card.hidden = incidents.length === 0;

    for (const incident of incidents) {
        const row = document.createElement('tr');
        const cells = [
            incident.service,
            formatDateTime(incident.started_at),
            incident.resolved_at ? formatDuration(incident.duration_seconds) : 'идёт',
            incident.cause,
        ];

        for (const value of cells) {
            const td = document.createElement('td');
            td.textContent = value;
            row.append(td);
        }

        body.append(row);
    }
}

async function load() {
    if (!slug) {
        showError('В ссылке не указан параметр slug.');
        return;
    }

    try {
        const page = await Api.statusPage(slug);

        document.getElementById('status-error').hidden = true;
        renderOverall(page);
        renderServices(page.services);
        renderIncidents(page.incidents);
    } catch (error) {
        showError(error.status === 404 ? 'Статус-страница не найдена.' : error.message);
    }
}

function showError(message) {
    const node = document.getElementById('status-error');
    node.textContent = message;
    node.hidden = false;

    document.getElementById('overall-title').textContent = 'Статус недоступен';
    document.getElementById('overall-dot').className = 'dot dot--pending';
}

load();
setInterval(load, 60000);
