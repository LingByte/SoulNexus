(function() {
    'use strict';

    // 全局状态
    const state = {
        x: 100,
        y: 100,
        action: 'idle',
        direction: 'right',
        isDragging: false,
        energy: 100,
        happiness: 100,
        speed: 1,
        dragOffset: { x: 0, y: 0 }
    };

    let petContainer, pet, canvas, ctx, audioContext;

    // 初始化函数
    function init() {
        // 创建样式
        injectStyles();
        
        // 创建DOM元素
        createElements();
        
        // 初始化画布
        initCanvas();
        
        // 创建粒子
        createParticles();
        
        // 启动动画
        animateParticles();
        
        // 渲染桌宠
        renderPet();
        
        // 更新位置和状态
        updatePosition();
        updateStatus();
        
        // 绑定事件
        bindEvents();
        
        // 启动自动行为
        setInterval(autoAction, 100);
        
        // 启动能量衰减
        setInterval(() => {
            state.energy = Math.max(0, state.energy - 0.5);
            state.happiness = Math.max(0, state.happiness - 0.3);
            updateStatus();
        }, 2000);
    }

    // 注入样式
    function injectStyles() {
        const style = document.createElement('style');
        style.textContent = `
            #cyber-pet-canvas {
                position: fixed;
                top: 0;
                left: 0;
                z-index: 9998;
                opacity: 0.4;
                pointer-events: none;
            }

            #cyber-pet-container {
                position: fixed;
                z-index: 9999;
                cursor: grab;
                transition: transform 0.3s;
            }

            #cyber-pet-container:active {
                cursor: grabbing;
            }

            #cyber-pet-container:hover {
                transform: scale(1.1);
            }

            .cyber-pet-pixel {
                position: absolute;
                width: 8px;
                height: 8px;
                transition: all 0.1s;
            }

            #cyber-pet-control {
                position: fixed;
                top: 20px;
                left: 20px;
                z-index: 9997;
                background: rgba(20, 25, 45, 0.9);
                backdrop-filter: blur(10px);
                border: 1px solid rgba(0, 212, 255, 0.3);
                border-radius: 10px;
                padding: 20px;
                box-shadow: 0 0 20px rgba(0, 212, 255, 0.2);
                font-family: 'Courier New', monospace;
                color: #e0e0e0;
            }

            #cyber-pet-status {
                position: fixed;
                bottom: 20px;
                right: 20px;
                z-index: 9997;
                background: rgba(20, 25, 45, 0.9);
                backdrop-filter: blur(10px);
                border: 1px solid rgba(0, 212, 255, 0.3);
                border-radius: 10px;
                padding: 20px;
                box-shadow: 0 0 20px rgba(0, 212, 255, 0.2);
                min-width: 200px;
                font-family: 'Courier New', monospace;
                color: #e0e0e0;
            }

            .cyber-pet-title {
                font-size: 14px;
                font-weight: bold;
                color: #00d4ff;
                margin-bottom: 15px;
                display: flex;
                align-items: center;
                gap: 8px;
            }

            .cyber-pet-pulse {
                width: 8px;
                height: 8px;
                background: #00d4ff;
                border-radius: 50%;
                animation: cyber-pet-pulse 2s infinite;
            }

            @keyframes cyber-pet-pulse {
                0%, 100% { opacity: 1; }
                50% { opacity: 0.3; }
            }

            .cyber-pet-btn {
                background: rgba(0, 136, 255, 0.2);
                border: 1px solid rgba(0, 136, 255, 0.5);
                color: #00d4ff;
                padding: 8px 12px;
                margin: 5px;
                border-radius: 5px;
                cursor: pointer;
                font-size: 12px;
                transition: all 0.3s;
                font-family: 'Courier New', monospace;
            }

            .cyber-pet-btn:hover {
                background: rgba(0, 136, 255, 0.4);
                box-shadow: 0 0 10px rgba(0, 212, 255, 0.5);
            }

            .cyber-pet-stat {
                margin: 10px 0;
            }

            .cyber-pet-label {
                font-size: 11px;
                color: #888;
                display: flex;
                justify-content: space-between;
                margin-bottom: 5px;
            }

            .cyber-pet-bar-container {
                width: 100%;
                height: 8px;
                background: rgba(0, 0, 0, 0.5);
                border-radius: 4px;
                overflow: hidden;
                border: 1px solid rgba(255, 255, 255, 0.1);
            }

            .cyber-pet-bar {
                height: 100%;
                transition: width 0.3s;
                border-radius: 4px;
            }

            .cyber-pet-energy {
                background: linear-gradient(90deg, #0088ff, #00d4ff);
            }

            .cyber-pet-happiness {
                background: linear-gradient(90deg, #ff0088, #ff00ff);
            }

            .cyber-pet-slider-container {
                margin: 15px 0;
            }

            .cyber-pet-slider {
                width: 100%;
                margin-top: 5px;
            }

            #cyber-pet-glow {
                position: absolute;
                width: 100%;
                height: 100%;
                border-radius: 50%;
                background: radial-gradient(circle, rgba(0, 212, 255, 0.6) 0%, transparent 70%);
                filter: blur(20px);
                opacity: 0.5;
                pointer-events: none;
            }
        `;
        document.head.appendChild(style);
    }

    // 创建DOM元素
    function createElements() {
        // 画布
        canvas = document.createElement('canvas');
        canvas.id = 'cyber-pet-canvas';
        document.body.appendChild(canvas);
        ctx = canvas.getContext('2d');

        // 控制面板
        const controlPanel = document.createElement('div');
        controlPanel.id = 'cyber-pet-control';
        controlPanel.innerHTML = `
            <div class="cyber-pet-title">
                <div class="cyber-pet-pulse"></div>
                动作控制
            </div>
            <div>
                <button class="cyber-pet-btn" data-action="idle">🧘 待机</button>
                <button class="cyber-pet-btn" data-action="walk">🚶 行走</button>
            </div>
            <div>
                <button class="cyber-pet-btn" data-action="jump">🦘 跳跃</button>
                <button class="cyber-pet-btn" data-action="attack">⚔️ 攻击</button>
            </div>
            <div>
                <button class="cyber-pet-btn" data-action="sleep">😴 睡眠</button>
            </div>
            <div class="cyber-pet-slider-container">
                <div class="cyber-pet-label">
                    <span>移动速度</span>
                    <span id="cyber-pet-speed">1x</span>
                </div>
                <input type="range" class="cyber-pet-slider" id="cyber-pet-speed-slider" 
                       min="0.5" max="3" step="0.5" value="1">
            </div>
        `;
        document.body.appendChild(controlPanel);

        // 状态面板
        const statusPanel = document.createElement('div');
        statusPanel.id = 'cyber-pet-status';
        statusPanel.innerHTML = `
            <div class="cyber-pet-title">
                <div class="cyber-pet-pulse"></div>
                CYBER PET v1.0
            </div>
            <div style="font-size: 11px; color: #888; margin-bottom: 10px;">
                状态: <span id="cyber-pet-action" style="color: #00d4ff;">待机</span>
            </div>
            <div class="cyber-pet-stat">
                <div class="cyber-pet-label">
                    <span>能量</span>
                    <span id="cyber-pet-energy-val" style="color: #00d4ff;">100%</span>
                </div>
                <div class="cyber-pet-bar-container">
                    <div class="cyber-pet-bar cyber-pet-energy" id="cyber-pet-energy-bar" style="width: 100%;"></div>
                </div>
            </div>
            <div class="cyber-pet-stat">
                <div class="cyber-pet-label">
                    <span>快乐</span>
                    <span id="cyber-pet-happiness-val" style="color: #ff00ff;">100%</span>
                </div>
                <div class="cyber-pet-bar-container">
                    <div class="cyber-pet-bar cyber-pet-happiness" id="cyber-pet-happiness-bar" style="width: 100%;"></div>
                </div>
            </div>
            <div style="font-size: 10px; color: #666; margin-top: 10px; padding-top: 10px; border-top: 1px solid rgba(255,255,255,0.1);">
                位置: <span id="cyber-pet-pos">0, 0</span>
            </div>
        `;
        document.body.appendChild(statusPanel);

        // 桌宠容器
        petContainer = document.createElement('div');
        petContainer.id = 'cyber-pet-container';
        petContainer.innerHTML = `
            <div id="cyber-pet-glow"></div>
            <div id="cyber-pet"></div>
        `;
        document.body.appendChild(petContainer);
        pet = document.getElementById('cyber-pet');
    }

    // 初始化画布
    function initCanvas() {
        canvas.width = window.innerWidth;
        canvas.height = window.innerHeight;
        window.addEventListener('resize', () => {
            canvas.width = window.innerWidth;
            canvas.height = window.innerHeight;
        });
    }

    // 粒子系统
    const particles = [];
    function createParticles() {
        const colors = ['#00d4ff', '#00ffff', '#0088ff', '#ff00ff'];
        for (let i = 0; i < 50; i++) {
            particles.push({
                x: Math.random() * canvas.width,
                y: Math.random() * canvas.height,
                vx: (Math.random() - 0.5) * 0.5,
                vy: (Math.random() - 0.5) * 0.5,
                size: Math.random() * 2 + 1,
                opacity: Math.random() * 0.5 + 0.2,
                color: colors[Math.floor(Math.random() * colors.length)]
            });
        }
    }

    function animateParticles() {
        ctx.clearRect(0, 0, canvas.width, canvas.height);

        particles.forEach((p, i) => {
            p.x += p.vx;
            p.y += p.vy;

            if (p.x < 0 || p.x > canvas.width) p.vx *= -1;
            if (p.y < 0 || p.y > canvas.height) p.vy *= -1;

            ctx.beginPath();
            ctx.arc(p.x, p.y, p.size, 0, Math.PI * 2);
            ctx.fillStyle = p.color;
            ctx.globalAlpha = p.opacity;
            ctx.fill();

            for (let j = i + 1; j < particles.length; j++) {
                const dx = particles[j].x - p.x;
                const dy = particles[j].y - p.y;
                const dist = Math.sqrt(dx * dx + dy * dy);

                if (dist < 150) {
                    ctx.beginPath();
                    ctx.moveTo(p.x, p.y);
                    ctx.lineTo(particles[j].x, particles[j].y);
                    ctx.strokeStyle = p.color;
                    ctx.globalAlpha = (1 - dist / 150) * 0.2;
                    ctx.lineWidth = 0.5;
                    ctx.stroke();
                }
            }
        });

        ctx.globalAlpha = 1;
        requestAnimationFrame(animateParticles);
    }

    // 渲染桌宠
    function renderPet() {
        const baseSize = 8;
        const pixels = [];

        if (state.action === 'idle' || state.action === 'walk') {
            pixels.push(
                [3, 2, '#00d4ff'], [4, 2, '#00d4ff'],
                [2, 3, '#00d4ff'], [3, 3, '#0088ff'], [4, 3, '#0088ff'], [5, 3, '#00d4ff'],
                [2, 4, '#00d4ff'], [3, 4, '#0088ff'], [4, 4, '#0088ff'], [5, 4, '#00d4ff'],
                [3, 5, '#00d4ff'], [4, 5, '#00d4ff'],
                [3, 3, '#00ffff'], [4, 3, '#00ffff'],
                [2, 6, '#00aaff'], [5, 6, '#00aaff'],
                [2, 7, '#0088ff'], [5, 7, '#0088ff']
            );
        } else if (state.action === 'jump') {
            pixels.push(
                [3, 1, '#00d4ff'], [4, 1, '#00d4ff'],
                [2, 2, '#00d4ff'], [3, 2, '#0088ff'], [4, 2, '#0088ff'], [5, 2, '#00d4ff'],
                [2, 3, '#00d4ff'], [3, 3, '#0088ff'], [4, 3, '#0088ff'], [5, 3, '#00d4ff'],
                [3, 4, '#00d4ff'], [4, 4, '#00d4ff'],
                [3, 2, '#00ffff'], [4, 2, '#00ffff'],
                [2, 5, '#00aaff'], [5, 5, '#00aaff']
            );
        } else if (state.action === 'attack') {
            pixels.push(
                [4, 2, '#00d4ff'], [5, 2, '#00d4ff'],
                [3, 3, '#00d4ff'], [4, 3, '#0088ff'], [5, 3, '#0088ff'], [6, 3, '#00d4ff'],
                [3, 4, '#00d4ff'], [4, 4, '#0088ff'], [5, 4, '#0088ff'], [6, 4, '#00d4ff'],
                [4, 5, '#00d4ff'], [5, 5, '#00d4ff'],
                [4, 3, '#ff0088'], [5, 3, '#ff0088'],
                [3, 6, '#00aaff'], [6, 6, '#00aaff'],
                [7, 3, '#ff00ff'], [8, 3, '#ff00ff'], [7, 4, '#ff00ff']
            );
        } else if (state.action === 'sleep') {
            pixels.push(
                [2, 4, '#00d4ff'], [3, 4, '#00d4ff'], [4, 4, '#0088ff'], [5, 4, '#0088ff'], [6, 4, '#00d4ff'],
                [2, 5, '#00d4ff'], [3, 5, '#0088ff'], [4, 5, '#0088ff'], [5, 5, '#0088ff'], [6, 5, '#00d4ff'],
                [3, 5, '#004488'], [5, 5, '#004488'],
                [7, 2, '#00ffff'], [8, 3, '#00ffff'], [9, 4, '#00ffff']
            );
        }

        pet.innerHTML = '';
        pixels.forEach(([x, y, color]) => {
            const pixel = document.createElement('div');
            pixel.className = 'cyber-pet-pixel';
            pixel.style.left = (state.direction === 'left' ? (11 - x) : x) * baseSize + 'px';
            pixel.style.top = y * baseSize + 'px';
            pixel.style.backgroundColor = color;
            pixel.style.boxShadow = `0 0 ${baseSize / 2}px ${color}`;
            pet.appendChild(pixel);
        });
    }

    // 更新位置
    function updatePosition() {
        petContainer.style.left = state.x + 'px';
        petContainer.style.top = state.y + 'px';
        petContainer.style.transform = state.action === 'jump' ? 'translateY(-20px)' : 'translateY(0)';
        document.getElementById('cyber-pet-pos').textContent = `${Math.round(state.x)}, ${Math.round(state.y)}`;
    }

    // 更新状态
    function updateStatus() {
        const actionNames = {
            idle: '待机', walk: '行走', jump: '跳跃', attack: '攻击', sleep: '睡眠'
        };
        document.getElementById('cyber-pet-action').textContent = actionNames[state.action];
        document.getElementById('cyber-pet-energy-val').textContent = Math.round(state.energy) + '%';
        document.getElementById('cyber-pet-happiness-val').textContent = Math.round(state.happiness) + '%';
        document.getElementById('cyber-pet-energy-bar').style.width = state.energy + '%';
        document.getElementById('cyber-pet-happiness-bar').style.width = state.happiness + '%';
    }

    // 触发动作
    function triggerAction(action) {
        state.action = action;
        renderPet();
        playSound(action);
        
        if (action !== 'idle' && action !== 'walk') {
            const duration = action === 'jump' ? 800 : action === 'attack' ? 600 : 3000;
            setTimeout(() => {
                state.action = 'idle';
                renderPet();
                updateStatus();
            }, duration);
        }
        
        updateStatus();
    }

    // 音效
    function playSound(action) {
        if (!audioContext) {
            audioContext = new (window.AudioContext || window.webkitAudioContext)();
        }

        const play = (freq, dur, type = 'sine') => {
            const osc = audioContext.createOscillator();
            const gain = audioContext.createGain();
            osc.connect(gain);
            gain.connect(audioContext.destination);
            osc.type = type;
            osc.frequency.value = freq;
            gain.gain.setValueAtTime(0.1, audioContext.currentTime);
            gain.gain.exponentialRampToValueAtTime(0.01, audioContext.currentTime + dur);
            osc.start(audioContext.currentTime);
            osc.stop(audioContext.currentTime + dur);
        };

        switch (action) {
            case 'jump':
                play(800, 0.2, 'square');
                setTimeout(() => play(1000, 0.1, 'square'), 100);
                break;
            case 'attack':
                play(400, 0.1, 'sawtooth');
                setTimeout(() => play(300, 0.15, 'sawtooth'), 80);
                break;
            case 'walk':
                play(200, 0.05, 'triangle');
                break;
            case 'sleep':
                play(300, 0.3, 'sine');
                setTimeout(() => play(250, 0.3, 'sine'), 300);
                break;
        }
    }

    // 自动行为
    function autoAction() {
        if (state.isDragging) return;

        if (state.energy < 20) {
            triggerAction('sleep');
            setTimeout(() => {
                state.energy = Math.min(100, state.energy + 30);
                updateStatus();
            }, 3000);
            return;
        }

        const rand = Math.random();
        if (rand < 0.02) {
            triggerAction('jump');
            state.energy = Math.max(0, state.energy - 5);
            state.happiness = Math.min(100, state.happiness + 3);
        } else if (rand < 0.05) {
            triggerAction('attack');
            state.energy = Math.max(0, state.energy - 8);
            state.happiness = Math.min(100, state.happiness + 5);
        } else if (rand < 0.1) {
            triggerAction('sleep');
            setTimeout(() => {
                state.energy = Math.min(100, state.energy + 20);
                updateStatus();
            }, 3000);
        } else if (rand < 0.3 && state.action !== 'walk') {
            state.action = 'walk';
            state.direction = Math.random() > 0.5 ? 'right' : 'left';
            renderPet();
            
            const distance = Math.random() * 200 + 50;
            const steps = Math.floor(60 / state.speed);
            const stepDist = (distance / steps) * state.speed;
            let step = 0;

            const walkInterval = setInterval(() => {
                if (step >= steps || state.action !== 'walk') {
                    clearInterval(walkInterval);
                    state.action = 'idle';
                    state.energy = Math.max(0, state.energy - 2);
                    renderPet();
                    updateStatus();
                    return;
                }

                state.x = Math.max(0, Math.min(window.innerWidth - 120,
                    state.x + (state.direction === 'right' ? stepDist : -stepDist)));
                updatePosition();
                step++;
            }, 16);
        }
    }

    // 绑定事件
    function bindEvents() {
        // 动作按钮
        document.querySelectorAll('.cyber-pet-btn[data-action]').forEach(btn => {
            btn.addEventListener('click', () => {
                triggerAction(btn.dataset.action);
            });
        });

        // 速度滑块
        document.getElementById('cyber-pet-speed-slider').addEventListener('input', (e) => {
            state.speed = parseFloat(e.target.value);
            document.getElementById('cyber-pet-speed').textContent = e.target.value + 'x';
        });

        // 拖拽
        petContainer.addEventListener('mousedown', (e) => {
            state.isDragging = true;
            state.action = 'idle';
            renderPet();
            const rect = petContainer.getBoundingClientRect();
            state.dragOffset.x = e.clientX - rect.left;
            state.dragOffset.y = e.clientY - rect.top;
        });

        document.addEventListener('mousemove', (e) => {
            if (state.isDragging) {
                state.x = Math.max(0, Math.min(window.innerWidth - 120, e.clientX - state.dragOffset.x));
                state.y = Math.max(0, Math.min(window.innerHeight - 120, e.clientY - state.dragOffset.y));
                updatePosition();
            }
        });

        document.addEventListener('mouseup', () => {
            if (state.isDragging) {
                state.isDragging = false;
                state.happiness = Math.min(100, state.happiness + 5);
                updateStatus();
            }
        });

        // 点击交互
        petContainer.addEventListener('click', (e) => {
            if (!state.isDragging && e.target.closest('#cyber-pet-container')) {
                const actions = ['jump', 'attack'];
                const action = actions[Math.floor(Math.random() * actions.length)];
                triggerAction(action);
                state.happiness = Math.min(100, state.happiness + 10);
                state.energy = Math.max(0, state.energy - 5);
            }
        });
    }

    // 页面加载完成后初始化
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

    // 暴露全局API（可选）
    window.CyberPet = {
        triggerAction: triggerAction,
        getState: () => ({ ...state })
    };

})();

