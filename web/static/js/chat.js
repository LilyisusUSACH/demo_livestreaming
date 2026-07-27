// WebSocket Per-Channel Chat & 3D Reaction Client with System Welcome Banners & Redis Storage

let wsSocket = null;
let isPageUnloading = false;
let reconnectTimer = null;
let currentChatChannel = 'kuspid-sports';

// Mapa de nombres de reacción a emojis reales
const REACTION_MAP = {
    'heart':     '❤️',
    'fire':      '🔥',
    'rocket':    '🚀',
    'celebrate': '🎉',
    '❤️': '❤️',
    '🔥': '🔥',
    '🚀': '🚀',
    '🎉': '🎉'
};

function resolverEmoji(valor) {
    if (!valor) return '❤️';
    return REACTION_MAP[valor] || valor;
}

document.addEventListener('DOMContentLoaded', () => {
    initWebSocket();
    initBFCacheListeners();

    const chatForm = document.getElementById('chat-form');
    if (chatForm) {
        chatForm.addEventListener('submit', (e) => {
            e.preventDefault();
            const input = document.getElementById('chat-input');
            if (input && input.value.trim() && wsSocket && wsSocket.readyState === WebSocket.OPEN) {
                const msg = {
                    type: 'chat',
                    channel_id: currentChatChannel,
                    content: input.value.trim()
                };
                wsSocket.send(JSON.stringify(msg));
                input.value = '';
            }
        });
    }

    // Botones de reacción — resuelven el emoji real antes de enviarlo
    document.querySelectorAll('.reaction-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const raw = btn.getAttribute('data-emoji') || btn.getAttribute('data-reaction') || btn.innerText.trim();
            const emoji = resolverEmoji(raw);
            if (emoji && wsSocket && wsSocket.readyState === WebSocket.OPEN) {
                const msg = {
                    type: 'reaction',
                    channel_id: currentChatChannel,
                    emoji: emoji
                };
                wsSocket.send(JSON.stringify(msg));
            }
        });
    });
});

function switchChannelChat(newChannelID) {
    if (!newChannelID || newChannelID === currentChatChannel) return;
    currentChatChannel = newChannelID;

    const box = document.getElementById('chat-box');
    if (box) box.innerHTML = '';

    console.log(`📡 Cambiando sala de chat WebSocket a canal: ${currentChatChannel}`);
    initWebSocket();
}

window.switchChannelChat = switchChannelChat;

function initBFCacheListeners() {
    window.addEventListener('pagehide', () => {
        isPageUnloading = true;
        if (reconnectTimer) clearTimeout(reconnectTimer);
        if (wsSocket) {
            wsSocket.close(1000, 'Page hidden / BFCache');
        }
    });

    window.addEventListener('pageshow', (event) => {
        isPageUnloading = false;
        if (event.persisted || !wsSocket || wsSocket.readyState === WebSocket.CLOSED) {
            initWebSocket();
        }
    });
}

function initWebSocket() {
    if (isPageUnloading) return;

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const token = localStorage.getItem('auth_token') || '';
    const wsUrl = `${protocol}//${window.location.host}/api/ws?channel=${encodeURIComponent(currentChatChannel)}&token=${token}`;

    if (wsSocket) {
        wsSocket.onclose = null;
        wsSocket.close();
    }

    wsSocket = new WebSocket(wsUrl);

    wsSocket.onopen = () => {
        console.log(`🟢 WebSocket conectado a la sala Kuspid del canal [${currentChatChannel}]`);
    };

    wsSocket.onmessage = (event) => {
        try {
            const msg = JSON.parse(event.data);
            if (msg.type === 'system') {
                appendSystemWelcomeMessage(msg);
            } else if (msg.type === 'chat') {
                appendChatMessage(msg);
            } else if (msg.type === 'reaction') {
                if (window.trigger3DReaction && !msg.is_history) {
                    window.trigger3DReaction(msg.emoji || '❤️');
                }
                appendReactionMessage(msg);
            }
        } catch (e) {
            console.error('WS Message Parse Error:', e);
        }
    };

    wsSocket.onclose = (event) => {
        if (isPageUnloading || event.code === 1000) return;
        if (reconnectTimer) clearTimeout(reconnectTimer);
        reconnectTimer = setTimeout(initWebSocket, 3000);
    };
}

function appendSystemWelcomeMessage(msg) {
    const box = document.getElementById('chat-box');
    if (!box) return;

    // Prevent duplicate welcome banners
    const existing = box.querySelector('.chat-msg-system');
    if (existing) return;

    const div = document.createElement('div');
    div.className = 'chat-msg chat-msg-system';
    div.style.background = 'linear-gradient(135deg, rgba(234, 179, 8, 0.18), rgba(157, 107, 216, 0.18))';
    div.style.border = '1px solid rgba(234, 179, 8, 0.4)';
    div.style.borderRadius = '12px';
    div.style.padding = '0.85rem';
    div.style.marginBottom = '0.5rem';

    div.innerHTML = `
        <div style="font-family: 'Sora', sans-serif; font-weight: 800; font-size: 0.75rem; letter-spacing: 1px; color: var(--gold); margin-bottom: 0.3rem;">
            KUSPID SYSTEM WELCOME
        </div>
        <div style="font-size: 0.85rem; font-weight: 600; color: #fff;">
            ${escapeHtml(msg.content)}
        </div>
        <div style="font-size: 0.7rem; color: var(--text-muted); margin-top: 0.3rem;">
            🔒 Chat moderado en directo con persistencia en Redis.
        </div>
    `;

    box.insertBefore(div, box.firstChild);
    box.scrollTop = box.scrollHeight;
}

function appendChatMessage(msg) {
    const box = document.getElementById('chat-box');
    if (!box) return;

    const div = document.createElement('div');
    div.className = 'chat-msg';
    div.innerHTML = `
        <div class="chat-sender">${escapeHtml(msg.sender || 'Usuario')}</div>
        <div class="chat-content">${escapeHtml(msg.content || '')}</div>
    `;

    box.appendChild(div);
    box.scrollTop = box.scrollHeight;
}

function appendReactionMessage(msg) {
    const box = document.getElementById('chat-box');
    if (!box) return;

    const emoji = resolverEmoji(msg.emoji);

    const div = document.createElement('div');
    div.className = 'chat-msg chat-msg-reaction';
    div.style.background = 'rgba(234, 179, 8, 0.12)';
    div.style.border = '1px solid rgba(234, 179, 8, 0.25)';
    div.innerHTML = `
        <div class="chat-sender" style="color: var(--gold);">${escapeHtml(msg.sender || 'Usuario')}</div>
        <div class="chat-content" style="font-size: 1.25rem; display: flex; align-items: center; gap: 0.4rem;">
            <span style="opacity:0.75; font-size:0.82rem;">Reaccionó con</span>
            <span style="font-size:1.4rem; line-height:1;">${emoji}</span>
        </div>
    `;

    box.appendChild(div);
    box.scrollTop = box.scrollHeight;
}

function escapeHtml(text) {
    if (!text) return '';
    return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
