// Kuspid HLS Video Player Controller with 3D Holographic Drawer, Profile Dropdown, Collapsible Chat & High-Performance 2D Retro CRT TV Mode

let hlsPlayer = null;
let retroHlsPlayer = null;
let currentChannelID = 'kuspid-sports';
let currentQuality = '1080p';
let currentUserID = '';
let isChatOpen = true;

let isRetroMode = false;
let availableChannels = [];
let retroChannelIndex = 0;
let retroKnobAngle = 0;
let isCrtEffectOn = true;

document.addEventListener('DOMContentLoaded', async () => {
    const video = document.getElementById('hls-video');
    if (!video) return;

    await loadUserProfile();
    await loadChannels();
    playChannel(currentChannelID);
    startMetricsPolling();

    initUserDropdownEvents();
    initCustomPlayerEvents();
    init3DDrawerEvents();
    initForceLogoutEvent();
    initCollapsibleChat();
    initTelemetryToggle();
    initRetroModeEvents();
    initReactionVisibilityToggle();
});

function initReactionVisibilityToggle() {
    const btn = document.getElementById('btn-toggle-reactions-visibility');
    if (!btn) return;

    const savedState = localStorage.getItem('kuspid_show_reactions');
    if (savedState === 'false') {
        window.__showFloatingReactions = false;
        btn.innerText = '✨ OFF';
        btn.style.opacity = '0.5';
    } else {
        window.__showFloatingReactions = true;
        btn.innerText = '✨ ON';
        btn.style.opacity = '1';
    }

    btn.addEventListener('click', () => {
        window.__showFloatingReactions = !window.__showFloatingReactions;
        localStorage.setItem('kuspid_show_reactions', window.__showFloatingReactions);

        if (window.__showFloatingReactions) {
            btn.innerText = '✨ ON';
            btn.style.opacity = '1';
        } else {
            btn.innerText = '✨ OFF';
            btn.style.opacity = '0.5';
        }
    });
}

function initRetroModeEvents() {
    const btnToggle = document.getElementById('btn-retro-toggle');
    const btnExit = document.getElementById('btn-exit-retro');
    const channelKnob = document.getElementById('retro-knob-channel');
    const volumeKnob = document.getElementById('retro-knob-volume');
    const crtBtn = document.getElementById('btn-toggle-crt-effect');

    if (btnToggle) {
        btnToggle.addEventListener('click', () => {
            enterRetroMode();
        });
    }

    if (btnExit) {
        btnExit.addEventListener('click', () => {
            exitRetroMode();
        });
    }

    if (channelKnob) {
        channelKnob.addEventListener('click', () => {
            cycleRetroChannel();
        });
    }

    if (volumeKnob) {
        volumeKnob.addEventListener('click', () => {
            const retroVideo = document.getElementById('retro-hls-video');
            if (retroVideo) {
                retroVideo.muted = !retroVideo.muted;
                volumeKnob.style.transform = `rotate(${retroVideo.muted ? 180 : 0}deg)`;
            }
        });
    }

    if (crtBtn) {
        crtBtn.addEventListener('click', () => {
            isCrtEffectOn = !isCrtEffectOn;
            const retroVideo = document.getElementById('retro-hls-video');
            const scanlines = document.querySelector('.crt-scanlines-overlay');

            if (isCrtEffectOn) {
                if (retroVideo) retroVideo.style.filter = 'contrast(1.15) saturate(1.3) sepia(0.12)';
                if (scanlines) scanlines.style.display = 'block';
                crtBtn.innerText = 'CRT: ON';
                crtBtn.classList.add('active');
            } else {
                if (retroVideo) retroVideo.style.filter = 'none';
                if (scanlines) scanlines.style.display = 'none';
                crtBtn.innerText = 'CRT: OFF';
                crtBtn.classList.remove('active');
            }
        });
    }
}

function enterRetroMode() {
    isRetroMode = true;
    document.body.classList.add('retro-mode');

    const mainVideo = document.getElementById('hls-video');
    if (mainVideo) mainVideo.pause();

    if (window.triggerKuspidFlash) {
        window.triggerKuspidFlash('rgba(255,255,255,0.7)');
    }

    playRetroChannel(currentChannelID);
}

function exitRetroMode() {
    isRetroMode = false;
    document.body.classList.remove('retro-mode');

    if (retroHlsPlayer) {
        retroHlsPlayer.destroy();
        retroHlsPlayer = null;
    }

    const retroVideo = document.getElementById('retro-hls-video');
    if (retroVideo) {
        retroVideo.pause();
        retroVideo.removeAttribute('src');
        retroVideo.load();
    }

    const mainVideo = document.getElementById('hls-video');
    if (mainVideo && mainVideo.paused) {
        mainVideo.play().catch(e => console.warn('Resume play:', e));
    }

    if (window.triggerKuspidFlash) {
        window.triggerKuspidFlash('rgba(232,180,101,0.2)');
    }
}

function cycleRetroChannel() {
    if (!availableChannels.length) return;

    retroChannelIndex = (retroChannelIndex + 1) % availableChannels.length;
    const targetChannel = availableChannels[retroChannelIndex];
    currentChannelID = targetChannel.id;

    retroKnobAngle += 45;
    const knob = document.getElementById('retro-knob-channel');
    if (knob) knob.style.transform = `rotate(${retroKnobAngle}deg)`;

    const disp = document.getElementById('retro-channel-display');
    if (disp) disp.innerText = `CH 0${retroChannelIndex + 1}`;

    if (window.switchChannelChat) {
        window.switchChannelChat(targetChannel.id);
    }

    playRetroChannel(targetChannel.id);
}

function playRetroChannel(channelID) {
    const video = document.getElementById('retro-hls-video');
    if (!video) return;

    video.muted = false;
    video.volume = 0.85;

    const streamUrl = `/live.m3u8?channel=${channelID}`;

    if (Hls.isSupported()) {
        if (retroHlsPlayer) {
            retroHlsPlayer.destroy();
        }

        retroHlsPlayer = new Hls({
            liveSyncDurationCount: 2,
            enableWorker: true,
            lowLatencyMode: false,
            maxBufferLength: 10,
            maxMaxBufferLength: 20,
            xhrSetup: function (xhr) {
                const token = localStorage.getItem('auth_token');
                const refreshToken = localStorage.getItem('refresh_token');
                if (token) xhr.setRequestHeader('Authorization', `Bearer ${token}`);
                if (refreshToken) xhr.setRequestHeader('X-Refresh-Token', refreshToken);
            }
        });

        retroHlsPlayer.loadSource(streamUrl);
        retroHlsPlayer.attachMedia(video);

        retroHlsPlayer.on(Hls.Events.MANIFEST_PARSED, () => {
            video.play().catch(err => {
                console.warn('Unmuted playback blocked by browser, falling back to muted:', err);
                video.muted = true;
                video.play();
            });
        });
    } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
        video.src = streamUrl;
        video.play().catch(err => {
            video.muted = true;
            video.play();
        });
    }
}

function initUserDropdownEvents() {
    const badge = document.getElementById('user-profile-badge');
    const menu = document.getElementById('user-dropdown-menu');
    if (!badge || !menu) return;

    badge.addEventListener('click', (e) => {
        e.stopPropagation();
        menu.classList.toggle('open');
    });

    document.addEventListener('click', (e) => {
        if (!menu.contains(e.target) && !badge.contains(e.target)) {
            menu.classList.remove('open');
        }
    });
}

async function loadUserProfile() {
    try {
        const res = await authenticatedFetch('/api/auth/me');
        if (!res.ok) return;

        const data = await res.json();
        const user = data.user || data;

        if (user) {
            currentUserID = user.id;
            const nameEl = document.getElementById('profile-user-name');
            const dropName = document.getElementById('dropdown-user-name');
            const dropEmail = document.getElementById('dropdown-user-email');

            if (nameEl) nameEl.innerText = user.name || user.email;
            if (dropName) dropName.innerText = user.name || 'Usuario Kuspid';
            if (dropEmail) dropEmail.innerText = user.email || '';
        }

        const count = data.active_sessions_count || 1;
        const countEl = document.getElementById('profile-sessions-count');
        const dropStatus = document.getElementById('dropdown-session-status');

        if (countEl) {
            if (count > 1) {
                countEl.innerText = `⚠️ ${count} sesiones`;
                countEl.style.background = 'rgba(239, 68, 68, 0.2)';
                countEl.style.color = '#fca5a5';
                countEl.style.borderColor = 'rgba(239, 68, 68, 0.4)';
            } else {
                countEl.innerText = `🟢 1 sesión`;
                countEl.style.background = 'rgba(34, 197, 94, 0.2)';
                countEl.style.color = '#4ade80';
                countEl.style.borderColor = 'rgba(34, 197, 94, 0.4)';
            }
        }

        if (dropStatus) {
            if (count > 1) {
                dropStatus.innerText = `⚠️ ${count} sesiones activas en tu cuenta`;
                dropStatus.style.color = '#fca5a5';
            } else {
                dropStatus.innerText = `🟢 1 dispositivo conectado (Este dispositivo)`;
                dropStatus.style.color = '#4ade80';
            }
        }
    } catch (err) {
        console.error('Failed to load profile:', err);
    }
}

function initForceLogoutEvent() {
    const btnForce = document.getElementById('btn-force-logout-other');
    if (!btnForce) return;

    btnForce.addEventListener('click', async (e) => {
        e.stopPropagation();
        const menu = document.getElementById('user-dropdown-menu');
        if (menu) menu.classList.remove('open');

        if (!confirm('¿Deseas cerrar inmediatamente todas las otras sesiones activas conectadas a tu cuenta?')) {
            return;
        }

        try {
            const res = await authenticatedFetch('/api/admin/sessions/revoke', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ user_id: currentUserID })
            });

            if (res.ok) {
                if (window.triggerKuspidFlash) window.triggerKuspidFlash('rgba(239,68,68,0.2)');
                alert('Todas las otras sesiones en otros dispositivos han sido revocadas exitosamente en Redis.');
                await loadUserProfile();
            } else {
                alert('No se pudo revocar las sesiones.');
            }
        } catch (err) {
            console.error('Error al forzar salida:', err);
        }
    });
}

function initCollapsibleChat() {
    const btnToggle = document.getElementById('btn-toggle-chat');
    const btnOpen = document.getElementById('btn-open-chat');
    const chatSidebar = document.getElementById('chat-sidebar');
    const playerLayout = document.getElementById('player-layout');

    const toggleChat = () => {
        isChatOpen = !isChatOpen;
        if (isChatOpen) {
            if (chatSidebar) chatSidebar.style.display = 'flex';
            if (btnOpen) btnOpen.style.display = 'none';
            if (playerLayout) playerLayout.classList.remove('chat-collapsed');
        } else {
            if (chatSidebar) chatSidebar.style.display = 'none';
            if (btnOpen) btnOpen.style.display = 'flex';
            if (playerLayout) playerLayout.classList.add('chat-collapsed');
        }
    };

    if (btnToggle) btnToggle.addEventListener('click', toggleChat);
    if (btnOpen) btnOpen.addEventListener('click', toggleChat);
}

function initTelemetryToggle() {
    const btn = document.getElementById('btn-toggle-telemetry');
    const grid = document.getElementById('telemetry-grid');
    const chevron = document.getElementById('telemetry-chevron');

    if (!btn || !grid) return;

    btn.addEventListener('click', () => {
        const isHidden = grid.style.display === 'none';
        grid.style.display = isHidden ? 'grid' : 'none';
        if (chevron) chevron.innerText = isHidden ? '▲' : '▼';
    });
}

function init3DDrawerEvents() {
    const btnToggle = document.getElementById('btn-channels-toggle');
    const btnClose = document.getElementById('btn-close-drawer');
    const drawer = document.getElementById('channels-drawer');
    const backdrop = document.getElementById('drawer-backdrop');

    const openDrawer = () => {
        if (drawer) drawer.classList.add('open');
        if (backdrop) backdrop.classList.add('open');
        document.body.classList.add('drawer-open');
    };

    const closeDrawer = () => {
        if (drawer) drawer.classList.remove('open');
        if (backdrop) backdrop.classList.remove('open');
        document.body.classList.remove('drawer-open');
    };

    if (btnToggle) btnToggle.addEventListener('click', openDrawer);
    if (btnClose) btnClose.addEventListener('click', closeDrawer);
    if (backdrop) backdrop.addEventListener('click', closeDrawer);
}

function initCustomPlayerEvents() {
    const video = document.getElementById('hls-video');
    const container = document.getElementById('video-container');
    const centerOverlay = document.getElementById('center-overlay');
    const btnPlayPause = document.getElementById('btn-play-pause');
    const centerPlayBtn = document.getElementById('center-play-btn');
    const btnMute = document.getElementById('btn-mute');
    const volumeSlider = document.getElementById('volume-slider');
    const btnFullscreen = document.getElementById('btn-fullscreen');
    const liveBadge = document.getElementById('live-badge');

    if (container) {
        container.addEventListener('click', (e) => {
            if (e.target.closest('.custom-controls') || e.target.closest('.live-badge-top')) {
                return;
            }
            if (video.paused) {
                video.play();
            } else {
                video.pause();
            }
        });
    }

    if (btnPlayPause) {
        btnPlayPause.addEventListener('click', (e) => {
            e.stopPropagation();
            if (video.paused) {
                video.play();
            } else {
                video.pause();
            }
        });
    }

    if (centerPlayBtn) {
        centerPlayBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            if (video.paused) {
                video.play();
            } else {
                video.pause();
            }
        });
    }

    video.addEventListener('play', () => {
        if (centerOverlay) centerOverlay.classList.remove('visible');
        if (container) container.classList.remove('paused');
        if (btnPlayPause) btnPlayPause.innerHTML = '<span class="material-symbols-rounded" aria-hidden="true">pause</span>';
    });

    video.addEventListener('pause', () => {
        if (centerOverlay) centerOverlay.classList.add('visible');
        if (container) container.classList.add('paused');
        if (btnPlayPause) btnPlayPause.innerHTML = '<span class="material-symbols-rounded" aria-hidden="true">play_arrow</span>';
    });

    if (btnMute && volumeSlider) {
        btnMute.addEventListener('click', (e) => {
            e.stopPropagation();
            video.muted = !video.muted;
            updateVolumeUI();
        });

        volumeSlider.addEventListener('input', (e) => {
            e.stopPropagation();
            video.volume = parseFloat(e.target.value);
            video.muted = video.volume === 0;
            updateVolumeUI();
        });

        const updateVolumeUI = () => {
            if (video.muted || video.volume === 0) {
                btnMute.innerHTML = '<span class="material-symbols-rounded" aria-hidden="true">volume_off</span>';
                volumeSlider.value = 0;
            } else {
                btnMute.innerHTML = `<span class="material-symbols-rounded" aria-hidden="true">${video.volume > 0.5 ? 'volume_up' : 'volume_down'}</span>`;
                volumeSlider.value = video.volume;
            }
        };

        updateVolumeUI();
    }

    if (btnFullscreen && container) {
        btnFullscreen.addEventListener('click', (e) => {
            e.stopPropagation();
            if (!document.fullscreenElement) {
                if (container.requestFullscreen) {
                    container.requestFullscreen();
                } else if (container.webkitRequestFullscreen) {
                    container.webkitRequestFullscreen();
                }
            } else {
                if (document.exitFullscreen) {
                    document.exitFullscreen();
                }
            }
        });
    }

    if (liveBadge) {
        liveBadge.addEventListener('click', (e) => {
            e.stopPropagation();
            jumpToLiveEdge();
        });
    }

    const qualitySelect = document.getElementById('quality-select');
    if (qualitySelect) {
        qualitySelect.addEventListener('change', (e) => {
            currentQuality = e.target.value;
            const bitrateEl = document.getElementById('hud-bitrate');
            if (bitrateEl) {
                const mbps = currentQuality === '1080p' ? '5.0' : (currentQuality === '720p' ? '2.5' : '1.0');
                bitrateEl.innerText = `${mbps} Mbps`;
            }
            playChannel(currentChannelID);
        });
    }
}

async function loadChannels() {
    try {
        const res = await authenticatedFetch('/api/stream/channels');
        if (!res.ok) return;

        const channels = await res.json();
        availableChannels = channels;

        const drawerList = document.getElementById('kuspid-drawer-list');
        if (!drawerList || !channels.length) return;

        drawerList.innerHTML = '';
        channels.forEach((ch, idx) => {
            const card = document.createElement('div');
            card.className = `drawer-channel-card ${ch.id === currentChannelID ? 'active' : ''}`;
            card.id = `drawer-card-${ch.id}`;
            const codeNum = String(idx + 1).padStart(2, '0');

            card.innerHTML = `
                <div class="drawer-channel-thumb">SEÑAL ${codeNum}</div>
                <div style="flex: 1; min-width: 0;">
                    <div class="ch-info-name">${ch.name}</div>
                    <div class="ch-info-cat">${ch.category} · ${ch.description || 'En vivo'}</div>
                </div>
            `;
            card.addEventListener('click', () => {
                switchChannel(ch);

                if (window.triggerKuspidFlash) {
                    const hue = idx % 2 === 0 ? 'rgba(232,180,101,0.15)' : 'rgba(157,107,216,0.15)';
                    window.triggerKuspidFlash(hue);
                }

                const drawer = document.getElementById('channels-drawer');
                const backdrop = document.getElementById('drawer-backdrop');
                if (drawer) drawer.classList.remove('open');
                if (backdrop) backdrop.classList.remove('open');
                document.body.classList.remove('drawer-open');
            });
            drawerList.appendChild(card);
        });
    } catch (err) {
        console.error('Failed to load channels:', err);
    }
}

function switchChannel(channelObj) {
    if (channelObj.id === currentChannelID) return;

    document.querySelectorAll('.drawer-channel-card').forEach(card => card.classList.remove('active'));
    const targetCard = document.getElementById(`drawer-card-${channelObj.id}`);
    if (targetCard) targetCard.classList.add('active');

    currentChannelID = channelObj.id;

    const titleEl = document.getElementById('stream-title');
    if (titleEl) titleEl.innerText = `${channelObj.name}`;

    if (window.switchChannelChat) {
        window.switchChannelChat(channelObj.id);
    }

    playChannel(channelObj.id);
}

function playChannel(channelID) {
    const video = document.getElementById('hls-video');
    const streamUrl = `/live.m3u8?channel=${channelID}`;

    if (Hls.isSupported()) {
        if (hlsPlayer) {
            hlsPlayer.destroy();
        }

        hlsPlayer = new Hls({
            liveSyncDurationCount: 2,
            liveMaxLatencyDurationCount: 4,
            enableWorker: true,
            lowLatencyMode: false,
            maxBufferLength: 12,
            maxMaxBufferLength: 24,
            xhrSetup: function (xhr) {
                const token = localStorage.getItem('auth_token');
                const refreshToken = localStorage.getItem('refresh_token');
                if (token) xhr.setRequestHeader('Authorization', `Bearer ${token}`);
                if (refreshToken) xhr.setRequestHeader('X-Refresh-Token', refreshToken);
            }
        });

        hlsPlayer.loadSource(streamUrl);
        hlsPlayer.attachMedia(video);

        hlsPlayer.on(Hls.Events.MANIFEST_PARSED, () => {
            video.play().catch(err => {
                console.warn('Autoplay with sound blocked by browser, attempting muted playback:', err);
                video.muted = true;
                video.play().catch(e => console.error('Play failed:', e));
            });
        });

        hlsPlayer.on(Hls.Events.LEVEL_UPDATED, (event, data) => {
            const seqEl = document.getElementById('hud-sequence');
            if (seqEl && data && data.details) {
                const seqNum = (data.details.startSN !== undefined) ? data.details.startSN : (data.details.mediaSequence || 0);
                seqEl.innerText = `#${seqNum}`;
            }
        });

        hlsPlayer.on(Hls.Events.BUFFER_APPENDED, () => {
            const bufEl = document.getElementById('hud-buffer');
            if (bufEl && video.buffered.length > 0) {
                const bufLen = (video.buffered.end(video.buffered.length - 1) - video.currentTime).toFixed(1);
                bufEl.innerText = `${bufLen}s`;
            }
        });

        hlsPlayer.on(Hls.Events.ERROR, (event, data) => {
            if (data.fatal) {
                switch (data.type) {
                    case Hls.ErrorTypes.NETWORK_ERROR:
                        console.log('Network error encountered, recovering load...');
                        hlsPlayer.startLoad();
                        break;
                    case Hls.ErrorTypes.MEDIA_ERROR:
                        console.log('Media error encountered, recovering media...');
                        hlsPlayer.recoverMediaError();
                        break;
                    default:
                        hlsPlayer.destroy();
                        break;
                }
            }
        });
    } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
        video.src = streamUrl;
        video.addEventListener('loadedmetadata', () => {
            video.play().catch(e => {
                video.muted = true;
                video.play();
            });
        });
    }
}

function jumpToLiveEdge() {
    const video = document.getElementById('hls-video');
    if (hlsPlayer && hlsPlayer.liveSyncPosition) {
        video.currentTime = hlsPlayer.liveSyncPosition;
    } else if (video.duration) {
        video.currentTime = video.duration - 0.5;
    }
}

function startMetricsPolling() {
    const poll = async () => {
        try {
            const res = await authenticatedFetch('/api/metrics');
            if (res.ok) {
                const data = await res.json();
                const memEl = document.getElementById('hud-ram');
                const goEl = document.getElementById('hud-goroutines');
                const connEl = document.getElementById('hud-conns');
                const viewersEl = document.getElementById('live-viewers-count');

                if (memEl) memEl.innerText = `${data.memory_alloc_mb} MB`;
                if (goEl) goEl.innerText = `${data.goroutines}`;

                const activeConns = data.active_ws_connections || 1;
                if (connEl) connEl.innerText = `${activeConns}`;
                if (viewersEl) viewersEl.innerText = `${activeConns} ${activeConns === 1 ? 'espectador' : 'espectadores'} en vivo`;
            }
        } catch (e) {
            console.error('Metrics poll error:', e);
        }
    };

    poll();
    setInterval(poll, 4000);
}
