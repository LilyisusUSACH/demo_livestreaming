// Kuspid Single-Page 3D Card Flip & Auth Manager (Zero Page Reload & Zero Jump)

document.addEventListener('DOMContentLoaded', () => {
    initCardTilt();
    initModeFromURL();
});

function initCardTilt() {
    const card = document.querySelector('.kuspid-card');
    if (!card) return;

    card.addEventListener('mousemove', (e) => {
        if (card.classList.contains('flipping')) return;
        const r = card.getBoundingClientRect();
        const x = e.clientX - r.left - r.width / 2;
        const y = e.clientY - r.top - r.height / 2;
        const rx = (-y / r.height) * 8;
        const ry = (x / r.width) * 8;
        card.style.transform = `perspective(1000px) rotateX(${rx}deg) rotateY(${ry}deg)`;
    });

    card.addEventListener('mouseleave', () => {
        if (card.classList.contains('flipping')) return;
        card.style.transform = 'perspective(1000px) rotateX(0deg) rotateY(0deg)';
    });
}

function initModeFromURL() {
    if (window.location.pathname === '/register' || window.location.search.includes('mode=register')) {
        setAuthMode('register', false);
    } else {
        setAuthMode('login', false);
    }
}

// Seamless 3D Card Flip Transition (No flash, no height jump)
function toggleAuthMode(event, mode) {
    if (event) event.preventDefault();

    const card = document.querySelector('.kuspid-card');
    if (!card || card.classList.contains('flipping')) return;

    const isRegistering = mode === 'register';
    const dir = isRegistering ? 1 : -1;

    card.classList.add('flipping');
    card.style.transition = 'transform 0.22s cubic-bezier(0.4, 0, 0.2, 1), opacity 0.18s ease';
    card.style.transform = `perspective(1200px) rotateY(${-90 * dir}deg) scale(0.96)`;
    card.style.opacity = '0.3';

    setTimeout(() => {
        setAuthMode(mode, true);
        window.history.pushState({}, '', isRegistering ? '/register' : '/login');

        card.style.transform = `perspective(1200px) rotateY(${90 * dir}deg) scale(0.96)`;

        requestAnimationFrame(() => {
            requestAnimationFrame(() => {
                card.style.transition = 'transform 0.28s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.22s ease';
                card.style.transform = 'perspective(1200px) rotateY(0deg) scale(1)';
                card.style.opacity = '1';

                setTimeout(() => {
                    card.classList.remove('flipping');
                }, 280);
            });
        });
    }, 200);
}

function setAuthMode(mode, animate) {
    const isRegister = mode === 'register';

    const loginNav = document.getElementById('nav-login-link');
    const registerNav = document.getElementById('nav-register-link');
    const loginActive = document.getElementById('nav-login-active');
    const cardTag = document.getElementById('card-tag');
    const cardTitle = document.getElementById('card-title');
    const cardDesc = document.getElementById('card-desc');
    const nameGroup = document.getElementById('name-group');
    const submitBtn = document.getElementById('submit-btn');
    const footerText = document.getElementById('footer-text');
    const alertEl = document.getElementById('auth-alert');

    if (alertEl) alertEl.style.display = 'none';

    if (isRegister) {
        if (loginNav) loginNav.style.display = 'inline';
        if (loginActive) loginActive.style.display = 'none';
        if (registerNav) registerNav.style.display = 'none';

        if (cardTag) cardTag.innerText = 'NUEVO ESPECTADOR';
        if (cardTitle) cardTitle.innerText = 'Crea tu cuenta';
        if (cardDesc) cardDesc.innerText = 'Únete para sintonizar canales en vivo con reacciones 3D.';

        if (nameGroup) {
            nameGroup.classList.add('active');
        }
        if (submitBtn) submitBtn.innerText = 'Crear cuenta y entrar';

        if (footerText) {
            footerText.innerHTML = '¿Ya tienes cuenta? <a href="#" onclick="toggleAuthMode(event, \'login\')">Inicia sesión</a>';
        }

        window.__isRegisterMode = true;
    } else {
        if (loginNav) loginNav.style.display = 'none';
        if (loginActive) loginActive.style.display = 'inline';
        if (registerNav) registerNav.style.display = 'inline';

        if (cardTag) cardTag.innerText = 'ACCESO A LA SEÑAL';
        if (cardTitle) cardTitle.innerText = 'Bienvenido de nuevo';
        if (cardDesc) cardDesc.innerText = 'Entra al reproductor y sintoniza en vivo, en 3D.';

        if (nameGroup) {
            nameGroup.classList.remove('active');
        }
        if (submitBtn) submitBtn.innerText = 'Entrar a la señal';

        if (footerText) {
            footerText.innerHTML = '¿Sin cuenta? <a href="#" onclick="toggleAuthMode(event, \'register\')">Crear una</a>';
        }

        window.__isRegisterMode = false;
    }
}

// Global authenticatedFetch interceptor for X-New-Token auto-renewal
async function authenticatedFetch(url, options = {}) {
    let token = localStorage.getItem('auth_token');
    let refreshToken = localStorage.getItem('refresh_token');

    options.headers = options.headers || {};
    if (token) {
        options.headers['Authorization'] = `Bearer ${token}`;
    }
    if (refreshToken) {
        options.headers['X-Refresh-Token'] = refreshToken;
    }

    let response = await fetch(url, options);

    let newToken = response.headers.get('X-New-Token');
    if (newToken) {
        localStorage.setItem('auth_token', newToken);
    }

    if (response.status === 401 && !url.includes('/api/auth/login')) {
        localStorage.removeItem('auth_token');
        localStorage.removeItem('refresh_token');
        window.location.href = '/login';
    }

    return response;
}

async function handleAuthFormSubmit(event) {
    event.preventDefault();
    const alertEl = document.getElementById('auth-alert');
    if (alertEl) alertEl.style.display = 'none';

    const isRegister = !!window.__isRegisterMode;
    const email = document.getElementById('email').value;
    const password = document.getElementById('password').value;
    const nameInput = document.getElementById('name');
    const name = nameInput ? nameInput.value : '';

    const payload = isRegister ? { name, email, password } : { email, password };
    const endpoint = isRegister ? '/api/auth/register' : '/api/auth/login';

    try {
        const res = await fetch(endpoint, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });

        const data = await res.json();

        if (!res.ok) {
            if (alertEl) {
                alertEl.className = 'alert-message alert-error';
                alertEl.innerText = data.error || 'Ocurrió un error en la autenticación.';
                alertEl.style.display = 'block';
            }
            return;
        }

        if (data.access_token) {
            localStorage.setItem('auth_token', data.access_token);
        }
        if (data.refresh_token) {
            localStorage.setItem('refresh_token', data.refresh_token);
        }

        window.location.href = '/player';
    } catch (err) {
        if (alertEl) {
            alertEl.className = 'alert-message alert-error';
            alertEl.innerText = 'Error de conexión con el servidor.';
            alertEl.style.display = 'block';
        }
    }
}

function logout() {
    authenticatedFetch('/api/auth/logout', { method: 'POST' }).finally(() => {
        localStorage.removeItem('auth_token');
        localStorage.removeItem('refresh_token');
        window.location.href = '/login';
    });
}
