// Lightweight 30 FPS TV Broadcast Standard 3D Chilean Andes Mountain Engine (Optimized for Low-End PC Resources)

function initKuspidScene(canvas, density, motion) {
    if (!canvas || window.__kuspidInit) return;
    window.__kuspidInit = true;

    var scene = new THREE.Scene();
    scene.background = new THREE.Color(0x03040a);
    scene.fog = new THREE.Fog(0x03040a, 500, 2200);

    var camera = new THREE.PerspectiveCamera(48, window.innerWidth / window.innerHeight, 1, 3000);
    camera.position.set(0, 20, 560);

    // Optimized WebGLRenderer with low-power GPU preference
    var renderer = new THREE.WebGLRenderer({
        canvas: canvas,
        antialias: false,
        alpha: true,
        powerPreference: 'low-power',
        precision: 'mediump'
    });

    var devicePixelRatio = Math.min(window.devicePixelRatio || 1, 1.25);
    renderer.setSize(window.innerWidth, window.innerHeight);
    renderer.setPixelRatio(devicePixelRatio);

    // 1. Optimized Starry Chilean Night Sky (Reduced count for maximum performance)
    var isLowPerformance = (navigator.hardwareConcurrency || 4) <= 4;
    var starCount = isLowPerformance ? 300 : 500;
    var starPos = new Float32Array(starCount * 3);
    for (var i = 0; i < starCount; i++) {
        starPos[i * 3] = (Math.random() - 0.5) * 3400;
        starPos[i * 3 + 1] = Math.random() * 820 + 140;
        starPos[i * 3 + 2] = (Math.random() - 0.5) * 2600 - 500;
    }
    var starGeo = new THREE.BufferGeometry();
    starGeo.setAttribute('position', new THREE.BufferAttribute(starPos, 3));
    var starMat = new THREE.PointsMaterial({ color: 0xcdd6f5, size: 2.0, transparent: true, opacity: 0.75 });
    var stars = new THREE.Points(starGeo, starMat);
    scene.add(stars);

    // 2. Build 3 Chilean Andes Mountain Ridges (Far, Mid, Near) with lower segment counts
    function buildRidge(seed, scaleY, zBase, snowHex, rockHex, opacity) {
        var w = 3400, d = 950, segX = isLowPerformance ? 50 : 70, segZ = 12;
        var geo = new THREE.PlaneGeometry(w, d, segX, segZ);
        geo.rotateX(-Math.PI / 2);
        var pos = geo.attributes.position;
        var colors = [];
        var snowCol = new THREE.Color(snowHex), rockCol = new THREE.Color(rockHex);

        for (var v = 0; v < pos.count; v++) {
            var x = pos.getX(v), z = pos.getZ(v);
            var ridge = Math.abs(Math.sin(x * 0.006 + seed)) * 230 +
                Math.abs(Math.sin(x * 0.021 + seed * 2)) * 95 +
                Math.abs(Math.sin(x * 0.05 + seed * 3)) * 30;
            var falloff = 1 - Math.min(1, Math.abs(z) / (d / 2));
            var h = ridge * falloff * scaleY;
            pos.setY(v, h);
            var t = Math.min(1, Math.max(0, (h - scaleY * 135) / (scaleY * 95)));
            var c = rockCol.clone().lerp(snowCol, t);
            colors.push(c.r, c.g, c.b);
        }
        geo.setAttribute('color', new THREE.Float32BufferAttribute(colors, 3));
        geo.computeVertexNormals();

        var mat = new THREE.MeshStandardMaterial({
            vertexColors: true,
            flatShading: true,
            roughness: 0.9,
            metalness: 0.04,
            transparent: opacity < 1,
            opacity: opacity
        });
        var mesh = new THREE.Mesh(geo, mat);
        mesh.position.z = zBase;
        return mesh;
    }

    var ridgeFar = buildRidge(2.4, 0.68, -1350, 0xd7deee, 0x0d0f1c, 0.85);
    var ridgeMid = buildRidge(0.8, 0.9, -950, 0xeef1fb, 0x12142a, 0.95);
    var ridgeNear = buildRidge(4.3, 1.2, -560, 0xffffff, 0x0a0b16, 1);
    scene.add(ridgeFar, ridgeMid, ridgeNear);
    [ridgeFar, ridgeMid, ridgeNear].forEach(function (r) { r.position.y -= 190; });

    // 3. Rolling Mountain Mist & Fog Layers (Shared material for zero draw overhead)
    var mistCanvas = document.createElement('canvas');
    mistCanvas.width = 128;
    mistCanvas.height = 128;
    var mctx = mistCanvas.getContext('2d');
    var mg = mctx.createRadialGradient(64, 64, 0, 64, 64, 64);
    mg.addColorStop(0, 'rgba(180,190,220,0.5)');
    mg.addColorStop(1, 'rgba(180,190,220,0)');
    mctx.fillStyle = mg;
    mctx.fillRect(0, 0, 128, 128);

    var mistTex = new THREE.CanvasTexture(mistCanvas);
    var mistMat = new THREE.SpriteMaterial({ map: mistTex, transparent: true, opacity: 0.45, depthWrite: false });
    var mistLayers = [];
    var mistSpecs = [
        { z: -520, y: -220, scale: 900 },
        { z: -900, y: -200, scale: 1100 },
        { z: -1280, y: -180, scale: 1300 }
    ];

    mistSpecs.forEach(function (spec, idx) {
        var sp = new THREE.Sprite(mistMat);
        sp.scale.set(spec.scale, spec.scale * 0.16, 1);
        sp.position.set((idx % 2 === 0 ? -1 : 1) * 260, spec.y, spec.z);
        scene.add(sp);
        mistLayers.push(sp);
    });

    // 4. Silhouetted Foreground Trees
    var groundGeo = new THREE.PlaneGeometry(3600, 1800);
    groundGeo.rotateX(-Math.PI / 2);
    var ground = new THREE.Mesh(groundGeo, new THREE.MeshBasicMaterial({ color: 0x000000 }));
    ground.position.set(0, -190, -150);
    scene.add(ground);

    function buildPine(scale) {
        var group = new THREE.Group();
        var blackMat = new THREE.MeshBasicMaterial({ color: 0x000000 });
        var trunk = new THREE.Mesh(new THREE.CylinderGeometry(2.5 * scale, 4 * scale, 18 * scale, 5), blackMat);
        trunk.position.y = 9 * scale;
        group.add(trunk);
        var tiers = [
            { y: 34 * scale, r: 26 * scale, h: 34 * scale },
            { y: 56 * scale, r: 20 * scale, h: 30 * scale },
            { y: 76 * scale, r: 14 * scale, h: 26 * scale }
        ];
        tiers.forEach(function (t) {
            var cone = new THREE.Mesh(new THREE.ConeGeometry(t.r, t.h, 6), blackMat);
            cone.position.y = t.y;
            group.add(cone);
        });
        return group;
    }

    var pine1 = buildPine(1.5); pine1.position.set(-300, -190, -180); scene.add(pine1);
    var pine2 = buildPine(1.0); pine2.position.set(-200, -190, -110); scene.add(pine2);

    // 5. Ambient & Directional Lighting
    scene.add(new THREE.AmbientLight(0x334155, 1.2));
    var moon = new THREE.DirectionalLight(0xdfe6ff, 1.0);
    moon.position.set(-350, 600, 300);
    scene.add(moon);

    // 6. 30 FPS Throttled Render Loop
    var mouseX = 0, mouseY = 0, t = 0;
    var targetFPS = 30;
    var fpsInterval = 1000 / targetFPS;
    var lastFrameTime = performance.now();

    function onMove(e) {
        mouseX = (e.clientX - window.innerWidth / 2) * 0.018;
        mouseY = (e.clientY - window.innerHeight / 2) * 0.008;
    }

    function onResize() {
        camera.aspect = window.innerWidth / window.innerHeight;
        camera.updateProjectionMatrix();
        renderer.setSize(window.innerWidth, window.innerHeight);
    }

    document.addEventListener('mousemove', onMove, { passive: true });
    window.addEventListener('resize', onResize);

    function animate(currentTime) {
        requestAnimationFrame(animate);

        var elapsed = currentTime - lastFrameTime;
        if (elapsed > fpsInterval) {
            lastFrameTime = currentTime - (elapsed % fpsInterval);
            t += 1;

            camera.position.x += (mouseX - camera.position.x) * 0.025;
            camera.position.y += (20 - mouseY - camera.position.y) * 0.025;
            camera.lookAt(0, -50, -900);

            stars.rotation.y += 0.00004;

            mistLayers.forEach(function (sp, idx) {
                sp.position.x += Math.sin(t * 0.002 + idx) * 0.12;
            });

            renderer.render(scene, camera);
        }
    }

    requestAnimationFrame(animate);
}

window.initKuspidScene = initKuspidScene;
window.initPrismaScene = initKuspidScene; // Fallback compatibility

window.triggerKuspidFlash = function (hue) {
    var el = document.createElement('div');
    el.style.position = 'fixed';
    el.style.inset = '0';
    el.style.zIndex = '9997';
    el.style.pointerEvents = 'none';
    el.style.background = hue || 'rgba(232,180,101,0.15)';
    el.style.opacity = '0';
    el.style.transition = 'opacity .12s ease';

    document.body.appendChild(el);

    requestAnimationFrame(function () {
        el.style.opacity = '1';
    });

    setTimeout(function () {
        el.style.opacity = '0';
    }, 100);

    setTimeout(function () {
        el.remove();
    }, 320);
};

window.triggerPrismaFlash = window.triggerKuspidFlash; // Fallback compatibility

// Zero-Queue Burst Floating Emojis
window.__showFloatingReactions = true;

window.trigger3DReaction = function (emoji) {
    if (window.__showFloatingReactions === false) return;

    var emojiChar = emoji || '❤️';
    var burstCount = Math.floor(Math.random() * 2) + 1; // 1 or 2 emojis for maximum performance

    for (var i = 0; i < burstCount; i++) {
        (function (index) {
            setTimeout(function () {
                spawnFloatingEmoji(emojiChar);
            }, index * 80);
        })(i);
    }
};

function spawnFloatingEmoji(emojiChar) {
    var el = document.createElement('div');
    el.textContent = emojiChar;
    el.className = 'floating-3d-emoji';

    var startX = Math.random() * 60 + 20;
    var rotation = (Math.random() - 0.5) * 40;
    var scale = (Math.random() * 0.3 + 1.1).toFixed(2);
    var duration = (Math.random() * 0.4 + 1.5).toFixed(2);

    el.style.position = 'fixed';
    el.style.left = startX + 'vw';
    el.style.bottom = '8vh';
    el.style.fontSize = '2.2rem';
    el.style.zIndex = '9999';
    el.style.pointerEvents = 'none';
    el.style.willChange = 'transform, opacity';
    el.style.transition = 'transform ' + duration + 's cubic-bezier(0.1, 0.8, 0.3, 1), opacity ' + duration + 's ease';
    el.style.filter = 'drop-shadow(0 0 10px rgba(234, 179, 8, 0.5))';
    el.style.opacity = '1';

    document.body.appendChild(el);

    requestAnimationFrame(function () {
        requestAnimationFrame(function () {
            el.style.transform = 'translate3d(' + ((Math.random() - 0.5) * 60) + 'px, -65vh, 0) scale(' + scale + ') rotate(' + rotation + 'deg)';
            el.style.opacity = '0';
        });
    });

    setTimeout(function () {
        el.remove();
    }, duration * 1000 + 100);
}

// Deferred non-blocking startup (yields main thread for video player)
document.addEventListener('DOMContentLoaded', function () {
    var canvas = document.getElementById('webgl-canvas');
    if (canvas) {
        if ('requestIdleCallback' in window) {
            requestIdleCallback(function () {
                initKuspidScene(canvas, 26, 'parallax');
            });
        } else {
            setTimeout(function () {
                initKuspidScene(canvas, 26, 'parallax');
            }, 60);
        }
    }
});
