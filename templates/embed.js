/**
 * SoulNexus embed widget — 桌宠式浮动助手（点击角色打开对话面板）
 *
 * Usage:
 *   <script>
 *     window.__LingEchoConfig = {
 *       apiBase: 'https://your-host/api',
 *       apiKey: 'soulnexus_...',
 *       assistantId: '123',
 *       jsSourceId: 'js_xxxx', // optional; auto-filled when loaded via /t/:jsSourceId/embed.js
 *       transport: 'text', // dialog API (default); 'websocket'|'webrtc' use voice-session
 *       title: '智能助手',
 *       primaryColor: '#165DFF',
 *       position: 'right',
 *       autoMount: true
 *     };
 *   </script>
 *   <script src="https://your-host/api/lingecho/embed/v1/embed.js" async></script>
 */
(function () {
    'use strict';

    var CFG = window.__LingEchoConfig || {};
    if (!CFG.jsSourceId && window.__LINGECHO_JS_SOURCE_ID__) {
        CFG.jsSourceId = window.__LINGECHO_JS_SOURCE_ID__;
    }
    var ROOT_ID = 'lingecho-embed-root';
    /** sprite_idle.png — 3000×3000，6 列；前 5 行满 6 帧，第 6 行仅 2 帧，共 32 帧 */
    var SPRITE_IDLE = {
        url: '',
        cols: 6,
        rows: 6,
        frames: 32,
        fps: 10,
    };

    /** sprite_hello.png — 3500×3500，7×7 网格，末格空白，共 48 帧 */
    var SPRITE_HELLO = {
        url: '',
        cols: 7,
        rows: 7,
        frames: 48,
        fps: 14,
    };

    var PET_SIZE = 128;

    function inferApiBase() {
        if (CFG.apiBase) return String(CFG.apiBase).replace(/\/$/, '');
        var scripts = document.getElementsByTagName('script');
        for (var i = scripts.length - 1; i >= 0; i--) {
            var src = scripts[i].src || '';
            var m = src.match(/^(.*)\/lingecho\/embed\/v1\/(?:t\/[^/]+\/)?embed\.js(?:\?|$)/);
            if (m) return m[1].replace(/\/$/, '');
        }
        return '/api';
    }

    function resolveTransport(cfg) {
        var t = String((cfg && cfg.transport) || 'text').toLowerCase();
        if (t === 'webrtc' || t === 'websocket') return t;
        return 'text';
    }

    function resolveJsSourceId(cfg) {
        var fromCfg = cfg && cfg.jsSourceId ? String(cfg.jsSourceId).trim() : '';
        if (fromCfg) return fromCfg;
        if (window.__LINGECHO_JS_SOURCE_ID__) {
            return String(window.__LINGECHO_JS_SOURCE_ID__).trim();
        }
        var scripts = document.getElementsByTagName('script');
        for (var i = scripts.length - 1; i >= 0; i--) {
            var src = scripts[i].src || '';
            var m = src.match(/\/lingecho\/embed\/v1\/t\/([^/?#]+)\/embed\.js(?:\?|$)/);
            if (m && m[1]) return decodeURIComponent(m[1]);
        }
        return '';
    }

    function parseWireData(data) {
        try {
            if (typeof data === 'string') return JSON.parse(data);
            if (data instanceof ArrayBuffer) {
                return JSON.parse(new TextDecoder().decode(data));
            }
            if (data && typeof data.byteLength === 'number' && data.buffer) {
                return JSON.parse(
                    new TextDecoder().decode(
                        data.buffer.slice(data.byteOffset, data.byteOffset + data.byteLength),
                    ),
                );
            }
            return JSON.parse(String(data));
        } catch (_) {
            return null;
        }
    }

    function safeParseJson(rawText) {
        if (!rawText || String(rawText).trim() === '') {
            throw new Error('服务端返回内容为空');
        }
        try {
            return JSON.parse(rawText);
        } catch (err) {
            console.error('[SoulNexus ParseError]', rawText);
            throw new Error('后端返回数据格式异常');
        }
    }

    function fetchWithTimeout(resource, options, timeout) {
        timeout = timeout || 15000;
        var controller = typeof AbortController !== 'undefined' ? new AbortController() : null;
        var timer = controller
            ? setTimeout(function () {
                controller.abort();
            }, timeout)
            : null;
        var opts = options || {};
        if (controller) opts.signal = controller.signal;

        return fetch(resource, opts)
            .then(function (resp) {
                return resp.text().then(function (raw) {
                    if (timer) clearTimeout(timer);
                    return safeParseJson(raw);
                });
            })
            .catch(function (err) {
                if (timer) clearTimeout(timer);
                if (err && err.name === 'AbortError') throw new Error('请求超时');
                throw err;
            });
    }

    function el(tag, attrs, children) {
        var node = document.createElement(tag);
        if (attrs) {
            Object.keys(attrs).forEach(function (k) {
                if (k === 'style' && typeof attrs[k] === 'object') {
                    Object.assign(node.style, attrs[k]);
                } else if (k === 'text') {
                    node.textContent = attrs[k];
                } else if (k.slice(0, 2) === 'on') {
                    node.addEventListener(k.slice(2).toLowerCase(), attrs[k]);
                } else if (k === 'className') {
                    node.className = attrs[k];
                } else {
                    node.setAttribute(k, attrs[k]);
                }
            });
        }
        (children || []).forEach(function (c) {
            if (c == null) return;
            node.appendChild(typeof c === 'string' ? document.createTextNode(c) : c);
        });
        return node;
    }

    function svgIcon(viewBox, paths, opts) {
        opts = opts || {};
        var svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
        svg.setAttribute('width', opts.size || '16');
        svg.setAttribute('height', opts.size || '16');
        svg.setAttribute('viewBox', viewBox);
        svg.setAttribute('fill', 'none');
        svg.setAttribute('stroke', opts.stroke || 'currentColor');
        svg.setAttribute('stroke-width', opts.strokeWidth || '1.75');
        svg.setAttribute('stroke-linecap', 'round');
        svg.setAttribute('stroke-linejoin', 'round');
        (paths || []).forEach(function (d) {
            var path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
            path.setAttribute('d', d);
            svg.appendChild(path);
        });
        return svg;
    }

    function assetUrl(apiBase, name) {
        return apiBase.replace(/\/$/, '') + '/lingecho/embed/v1/assets/' + name;
    }

    function preloadImage(src) {
        if (!src) return Promise.resolve(false);
        if (!document.querySelector('link[rel="preload"][href="' + src + '"]')) {
            document.head.appendChild(el('link', { rel: 'preload', as: 'image', href: src }));
        }
        return new Promise(function (resolve) {
            var img = new Image();
            img.onload = function () {
                if (img.decode) {
                    img.decode().then(function () { resolve(true); }).catch(function () { resolve(true); });
                } else {
                    resolve(true);
                }
            };
            img.onerror = function () { resolve(false); };
            img.src = src;
        });
    }

    function preloadSprites(logoUrl) {
        var tasks = [preloadImage(SPRITE_HELLO.url), preloadImage(SPRITE_IDLE.url)];
        if (logoUrl) tasks.push(preloadImage(logoUrl));
        return Promise.all(tasks);
    }

    function injectLocalCSS(primary) {
        if (document.getElementById('lingecho-embed-css')) return;
        var accent = primary || '#18181b';
        var css = [
            '#lingecho-embed-root{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",sans-serif;color:#18181b}',
            '#lingecho-embed-root *{box-sizing:border-box}',
            '#lingecho-embed-root .le-sprite{display:block;width:100%;height:100%;background-repeat:no-repeat;image-rendering:auto}',
            '#lingecho-embed-root .le-pet{width:128px;height:128px;overflow:hidden;cursor:grab;user-select:none;-webkit-user-select:none;touch-action:none;filter:drop-shadow(0 6px 14px rgba(0,0,0,.1));pointer-events:auto;opacity:0;transform:scale(1);transition:opacity .22s ease,transform .22s ease,visibility .22s ease}',
            '#lingecho-embed-root .le-pet-wrap{position:relative;width:128px;height:128px;overflow:visible;pointer-events:auto}',
            '#lingecho-embed-root .le-pet-wrap.hidden{opacity:0!important;visibility:hidden;pointer-events:none}',
            '#lingecho-embed-root .le-pet-hint{position:absolute;left:50%;bottom:calc(100% - 10px);transform:translateX(-50%);z-index:2;max-width:220px;min-width:100px;padding:8px 12px;border-radius:12px;background:#fff;border:1px solid rgba(0,0,0,.08);box-shadow:0 4px 18px rgba(0,0,0,.1);font-size:12px;line-height:1.45;color:#3f3f46;text-align:center;white-space:normal;word-break:break-word;pointer-events:none;opacity:0;visibility:hidden;transition:opacity .2s ease,transform .2s ease,visibility .2s ease}',
            '#lingecho-embed-root .le-pet-hint.show{opacity:1;visibility:visible;animation:leHintPop .28s ease both}',
            '#lingecho-embed-root .le-pet-hint::after{content:"";position:absolute;left:50%;bottom:-5px;width:10px;height:10px;background:#fff;border-right:1px solid rgba(0,0,0,.08);border-bottom:1px solid rgba(0,0,0,.08);transform:translateX(-50%) rotate(45deg)}',
            '@keyframes leHintPop{from{opacity:0;transform:translateX(-50%) translateY(6px) scale(.96)}to{opacity:1;transform:translateX(-50%) translateY(0) scale(1)}}',
            '#lingecho-embed-root .le-pet.ready{opacity:1}',
            '#lingecho-embed-root .le-pet.dragging{cursor:grabbing;opacity:.92}',
            '#lingecho-embed-root .le-pet.hidden{opacity:0!important;visibility:hidden;pointer-events:none;transform:scale(.9) translateY(4px)}',
            '#lingecho-embed-root .le-pet.le-pet-in{animation:lePetIn .24s ease both}',
            '#lingecho-embed-root .le-pet.le-pet-out{opacity:0;transform:scale(.9) translateY(4px);pointer-events:none}',
            '@keyframes lePetIn{from{opacity:0;transform:scale(.92) translateY(6px)}to{opacity:1;transform:none}}',
            '#lingecho-embed-root .le-panel{pointer-events:auto;width:360px;max-width:calc(100vw - 24px);height:480px;max-height:calc(100vh - 48px);background:#fff;border:1px solid rgba(0,0,0,.08);border-radius:10px;box-shadow:0 8px 32px rgba(0,0,0,.1);display:none;flex-direction:column;overflow:hidden;transition:opacity .18s ease,transform .18s ease}',
            '#lingecho-embed-root .le-panel-enter{animation:lePanelIn .22s ease both}',
            '#lingecho-embed-root .le-panel-out{opacity:0;transform:translateY(6px) scale(.985)}',
            '@keyframes lePanelIn{from{opacity:0;transform:translateY(8px) scale(.985)}to{opacity:1;transform:none}}',
            '#lingecho-embed-root .le-header{padding:12px 14px;display:flex;align-items:center;justify-content:space-between;gap:8px;border-bottom:1px solid rgba(0,0,0,.06);cursor:grab;user-select:none;-webkit-user-select:none;touch-action:none}',
            '#lingecho-embed-root .le-header.dragging{cursor:grabbing}',
            '#lingecho-embed-root .le-header-brand{display:flex;align-items:center;gap:8px;min-width:0;flex:1}',
            '#lingecho-embed-root .le-header-logo{width:24px;height:24px;border-radius:6px;object-fit:cover;flex-shrink:0}',
            '#lingecho-embed-root .le-title{flex:1;min-width:0}',
            '#lingecho-embed-root .le-title-main{font-weight:500;color:#18181b;font-size:14px;letter-spacing:-.01em;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}',
            '#lingecho-embed-root .le-status{display:block;margin-top:2px;font-size:11px;color:#a1a1aa;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}',
            '#lingecho-embed-root .le-header-actions{display:flex;align-items:center;gap:2px;flex-shrink:0}',
            '#lingecho-embed-root .le-new-session{width:28px;height:28px;border:0;border-radius:6px;background:transparent;color:#71717a;cursor:pointer;display:flex;align-items:center;justify-content:center;flex-shrink:0}',
            '#lingecho-embed-root .le-new-session:hover{background:#f4f4f5;color:#18181b}',
            '#lingecho-embed-root .le-new-session:disabled{opacity:.35;cursor:not-allowed}',
            '#lingecho-embed-root .le-new-session svg{display:block;pointer-events:none}',
            '#lingecho-embed-root .le-close{width:28px;height:28px;border:0;border-radius:6px;background:transparent;color:#71717a;cursor:pointer;font-size:18px;line-height:1;display:flex;align-items:center;justify-content:center;flex-shrink:0}',
            '#lingecho-embed-root .le-close:hover{background:#f4f4f5;color:#18181b}',
            '#lingecho-embed-root .le-disclaimer{padding:6px 14px;font-size:10px;line-height:1.45;color:#a1a1aa;background:#fafafa;border-bottom:1px solid rgba(0,0,0,.04)}',
            '#lingecho-embed-root .le-list{flex:1;overflow-y:auto;padding:14px 12px;background:#fff}',
            '#lingecho-embed-root .le-list::-webkit-scrollbar{width:3px}',
            '#lingecho-embed-root .le-list::-webkit-scrollbar-thumb{background:#d4d4d8;border-radius:999px}',
            '#lingecho-embed-root .le-footer{padding:12px 16px;border-top:1px solid rgba(0,0,0,.06);background:#fafafa}',
            '#lingecho-embed-root .le-file-chip{display:none;align-items:center;gap:8px;margin-bottom:8px;padding:6px 8px;border-radius:6px;background:#f4f4f5;font-size:12px;color:#3f3f46}',
            '#lingecho-embed-root .le-file-chip.show{display:flex}',
            '#lingecho-embed-root .le-file-chip-name{flex:1;min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}',
            '#lingecho-embed-root .le-file-chip-clear{border:0;background:transparent;color:#71717a;cursor:pointer;font-size:14px;line-height:1;padding:0 2px}',
            '#lingecho-embed-root .le-input-wrap{display:flex;align-items:flex-end;gap:8px}',
            '#lingecho-embed-root .le-input{flex:1;resize:none;max-height:88px;padding:9px 0;border:0;border-bottom:1px solid #e4e4e7;border-radius:0;outline:none;font-size:13px;line-height:1.5;color:#18181b;background:transparent;transition:border-color .15s ease}',
            '#lingecho-embed-root .le-input:focus{border-bottom-color:' + accent + '}',
            '#lingecho-embed-root .le-input::placeholder{color:#a1a1aa}',
            '#lingecho-embed-root .le-icon-btn{flex-shrink:0;width:32px;height:32px;border-radius:6px;border:1px solid #e4e4e7;background:#fff;color:#52525b;cursor:pointer;display:flex;align-items:center;justify-content:center;transition:background .15s ease,border-color .15s ease,color .15s ease}',
            '#lingecho-embed-root .le-icon-btn svg{display:block;pointer-events:none}',
            '#lingecho-embed-root .le-icon-btn:hover{background:#f4f4f5;border-color:#d4d4d8;color:#18181b}',
            '#lingecho-embed-root .le-icon-btn:disabled{opacity:.35;cursor:not-allowed}',
            '#lingecho-embed-root .le-send{flex-shrink:0;width:32px;height:32px;border-radius:6px;border:1px solid ' + accent + ';background:' + accent + ';color:#fff;cursor:pointer;display:flex;align-items:center;justify-content:center;transition:filter .15s ease,opacity .15s ease}',
            '#lingecho-embed-root .le-send svg{display:block;pointer-events:none;stroke:#fff}',
            '#lingecho-embed-root .le-send:not(:disabled):hover{filter:brightness(1.08)}',
            '#lingecho-embed-root .le-send:disabled{opacity:.35;cursor:not-allowed;filter:none}',
            '#lingecho-embed-root .le-file-msg{font-size:12px;color:#71717a;margin-bottom:4px}',
            '#lingecho-embed-root .le-dot{animation:leDot 1.1s ease-in-out infinite}',
            '#lingecho-embed-root .le-dot:nth-child(2){animation-delay:.12s}',
            '#lingecho-embed-root .le-dot:nth-child(3){animation-delay:.24s}',
            '@keyframes leDot{0%,80%,100%{opacity:.35}40%{opacity:1}}',
            '#lingecho-embed-root .le-msg-row{display:flex;align-items:flex-start;gap:8px;margin-bottom:14px;animation:leMsgIn .18s ease both}',
            '@keyframes leMsgIn{from{opacity:0;transform:translateY(4px)}to{opacity:1;transform:none}}',
            '#lingecho-embed-root .le-msg-row.user{flex-direction:row-reverse}',
            '#lingecho-embed-root .le-msg-row.bot{flex-direction:row}',
            '#lingecho-embed-root .le-avatar{width:28px;height:28px;border-radius:50%;flex-shrink:0;overflow:hidden;display:flex;align-items:center;justify-content:center;background:#f4f4f5}',
            '#lingecho-embed-root .le-avatar img{width:100%;height:100%;object-fit:cover;display:block}',
            '#lingecho-embed-root .le-avatar.user{background:#eef2ff;color:#4f46e5}',
            '#lingecho-embed-root .le-avatar.user svg{display:block}',
            '#lingecho-embed-root .le-bubble{max-width:calc(100% - 40px);font-size:13px;line-height:1.6;white-space:pre-wrap;word-break:break-word}',
            '#lingecho-embed-root .le-bubble.user{padding:8px 12px;border-radius:10px 10px 2px 10px;background:#f4f4f5;color:#18181b}',
            '#lingecho-embed-root .le-bubble.bot{padding:8px 12px;border-radius:10px 10px 10px 2px;background:#f8fafc;border:1px solid #eef2f7;color:#3f3f46}',
            '#lingecho-embed-root .le-typing{display:flex;gap:4px;align-items:center;padding:2px 0}',
            '#lingecho-embed-root .le-dot{width:4px;height:4px;border-radius:999px;background:#a1a1aa;display:inline-block}',
        ].join('');
        document.head.appendChild(el('style', { id: 'lingecho-embed-css', text: css }));
    }

    function SpritePlayer(viewport, spec) {
        this.viewport = viewport;
        this.sprite = viewport.querySelector('.le-sprite') || viewport;
        this.idleSpec = SPRITE_IDLE;
        this.helloSpec = SPRITE_HELLO;
        this.spec = spec || SPRITE_IDLE;
        this.frame = 0;
        this.playing = false;
        this.loop = false;
        this.timer = null;
        this.onComplete = null;
        this.setFrame(0);
    }

    SpritePlayer.prototype.setFrame = function (gridIndex) {
        var spec = this.spec;
        var cols = spec.cols;
        var rows = spec.rows;
        var col = gridIndex % cols;
        var row = Math.floor(gridIndex / cols);

        this.sprite.style.backgroundImage = 'url("' + spec.url + '")';
        // 将整张图缩放到 cols×rows 个 PET_SIZE 格子，再按行列偏移，只显示当前帧
        this.sprite.style.backgroundSize = (cols * PET_SIZE) + 'px ' + (rows * PET_SIZE) + 'px';
        this.sprite.style.backgroundPosition = (-col * PET_SIZE) + 'px ' + (-row * PET_SIZE) + 'px';
    };

    SpritePlayer.prototype.stop = function () {
        this.playing = false;
        if (this.timer) {
            clearTimeout(this.timer);
            this.timer = null;
        }
    };

    SpritePlayer.prototype.useIdle = function () {
        this.spec = this.idleSpec;
    };

    SpritePlayer.prototype.useHello = function () {
        this.spec = this.helloSpec;
    };

    SpritePlayer.prototype.playIdleLoop = function () {
        var self = this;
        self.useIdle();
        self.play({ loop: true });
    };

    SpritePlayer.prototype.playHelloThenIdle = function () {
        var self = this;
        self.useHello();
        self.play({
            loop: false,
            onComplete: function () {
                self.playIdleLoop();
            },
        });
    };

    SpritePlayer.prototype.play = function (opts) {
        var self = this;
        opts = opts || {};
        self.stop();
        self.loop = !!opts.loop;
        self.onComplete = opts.onComplete || null;
        self.frame = 0;
        self.playing = true;
        var interval = 1000 / (self.spec.fps || 12);

        function tick() {
            if (!self.playing) return;
            self.setFrame(self.frame);
            self.frame += 1;
            if (self.frame >= self.spec.frames) {
                if (self.loop) {
                    self.frame = 0;
                    self.timer = setTimeout(tick, interval);
                } else {
                    self.playing = false;
                    if (self.onComplete) self.onComplete();
                }
                return;
            }
            self.timer = setTimeout(tick, interval);
        }
        tick();
    };

    function clampViewportRect(left, top, w, h, margin) {
        margin = margin || 8;
        return {
            left: Math.max(margin, Math.min(window.innerWidth - w - margin, left)),
            top: Math.max(margin, Math.min(window.innerHeight - h - margin, top)),
        };
    }

    function layoutPanelNearPet(petLeft, petTop, pw, ph) {
        var gap = 10;
        var margin = 8;
        var petW = PET_SIZE;
        var petH = PET_SIZE;
        var candidates = [
            { left: petLeft + petW - pw, top: petTop - ph - gap },
            { left: petLeft, top: petTop - ph - gap },
            { left: petLeft + petW - pw, top: petTop + petH + gap },
            { left: petLeft, top: petTop + petH + gap },
            { left: petLeft - pw - gap, top: petTop + petH - ph },
            { left: petLeft + petW + gap, top: petTop + petH - ph },
        ];
        var best = null;
        var bestScore = -Infinity;
        for (var i = 0; i < candidates.length; i++) {
            var c = candidates[i];
            var clamped = clampViewportRect(c.left, c.top, pw, ph, margin);
            var overflow = Math.abs(clamped.left - c.left) + Math.abs(clamped.top - c.top);
            var score = 1000 - overflow - i * 2;
            if (score > bestScore) {
                bestScore = score;
                best = clamped;
            }
        }
        return best || clampViewportRect(petLeft, petTop - ph - gap, pw, ph, margin);
    }

    function resolvePetGreeting(cfg) {
        cfg = cfg || {};
        if (cfg.greetingBubble === false) return '';
        if (typeof cfg.greetingBubble === 'string' && cfg.greetingBubble.trim()) {
            return cfg.greetingBubble.trim();
        }
        if (Array.isArray(cfg.greetingBubbles) && cfg.greetingBubbles.length) {
            var list = cfg.greetingBubbles
                .map(function (s) {
                    return String(s || '').trim();
                })
                .filter(Boolean);
            if (list.length) return list[Math.floor(Math.random() * list.length)];
        }
        var defaults = ['Hi～有问题可以随时问我哦', '你好！点击我开始对话', '有什么可以帮你的？'];
        return defaults[Math.floor(Math.random() * defaults.length)];
    }

    function createPetViewport() {
        return el(
            'div',
            {
                className: 'le-pet le-sprite-viewport',
                style: { width: PET_SIZE + 'px', height: PET_SIZE + 'px' },
                title: CFG.title || '拖动移动，点击打开对话',
            },
            [
                el('div', {
                    className: 'le-sprite',
                    style: { backgroundRepeat: 'no-repeat' },
                }),
            ],
        );
    }

    function bindPetInteraction(widget) {
        var pet = widget.petEl;
        var root = widget.root;
        var dragging = false;
        var moved = false;
        var startX = 0;
        var startY = 0;
        var startLeft = 0;
        var startTop = 0;
        var DRAG_THRESHOLD = 5;

        function pointer(e) {
            if (e.touches && e.touches.length) return e.touches[0];
            if (e.changedTouches && e.changedTouches.length) return e.changedTouches[0];
            return e;
        }

        function onDown(e) {
            if (widget.open) return;
            dragging = true;
            moved = false;
            var pt = pointer(e);
            startX = pt.clientX;
            startY = pt.clientY;
            var rect = pet.getBoundingClientRect();
            startLeft = rect.left;
            startTop = rect.top;
            pet.classList.add('dragging');
            e.preventDefault();
        }

        function onMove(e) {
            if (!dragging) return;
            var pt = pointer(e);
            var dx = pt.clientX - startX;
            var dy = pt.clientY - startY;
            if (!moved && Math.abs(dx) + Math.abs(dy) < DRAG_THRESHOLD) return;
            moved = true;
            widget.hidePetHint();
            widget.petPos = { left: startLeft + dx, top: startTop + dy };
            widget.savedPetPos = widget.petPos;
            widget.layoutPetAt(widget.petPos.left, widget.petPos.top);
            e.preventDefault();
        }

        function onUp() {
            if (!dragging) return;
            dragging = false;
            pet.classList.remove('dragging');
            if (moved) {
                window.setTimeout(function () {
                    if (!widget.open) widget.showPetHint();
                }, 500);
            }
            if (!moved && !widget.open) widget.toggle(true);
            moved = false;
        }

        pet.addEventListener('mousedown', onDown);
        pet.addEventListener('touchstart', onDown, { passive: false });
        document.addEventListener('mousemove', onMove);
        document.addEventListener('touchmove', onMove, { passive: false });
        document.addEventListener('mouseup', onUp);
        document.addEventListener('touchend', onUp);
    }

    function bindPanelDrag(widget) {
        var header = widget.panel && widget.panel.querySelector('.le-header');
        if (!header) return;
        var dragging = false;
        var startX = 0;
        var startY = 0;
        var startLeft = 0;
        var startTop = 0;

        function pointer(e) {
            if (e.touches && e.touches.length) return e.touches[0];
            if (e.changedTouches && e.changedTouches.length) return e.changedTouches[0];
            return e;
        }

        function onDown(e) {
            if (!widget.open) return;
            if (e.target && e.target.closest && (e.target.closest('.le-close') || e.target.closest('.le-new-session'))) return;
            dragging = true;
            var pt = pointer(e);
            startX = pt.clientX;
            startY = pt.clientY;
            startLeft = widget.pos ? widget.pos.x : widget.root.getBoundingClientRect().left;
            startTop = widget.pos ? widget.pos.y : widget.root.getBoundingClientRect().top;
            header.classList.add('dragging');
            e.preventDefault();
        }

        function onMove(e) {
            if (!dragging || !widget.open) return;
            var pt = pointer(e);
            var left = startLeft + (pt.clientX - startX);
            var top = startTop + (pt.clientY - startY);
            var pw = widget.panel.offsetWidth || 360;
            var ph = widget.panel.offsetHeight || 480;
            widget.setRootPosition(left, top, pw, ph);
            e.preventDefault();
        }

        function onUp() {
            if (!dragging) return;
            dragging = false;
            header.classList.remove('dragging');
        }

        header.addEventListener('mousedown', onDown);
        header.addEventListener('touchstart', onDown, { passive: false });
        document.addEventListener('mousemove', onMove);
        document.addEventListener('touchmove', onMove, { passive: false });
        document.addEventListener('mouseup', onUp);
        document.addEventListener('touchend', onUp);
    }

    function bindFileUpload(widget) {
        if (!widget.fileInput) return;
        widget.fileInput.addEventListener('change', function () {
            var file = widget.fileInput.files && widget.fileInput.files[0];
            widget.setPendingFile(file || null);
            widget.fileInput.value = '';
        });
    }

    function bindInputInteraction(widget) {
        widget.composing = false;
        var input = widget.input;
        if (!input) return;

        input.addEventListener('compositionstart', function () {
            widget.composing = true;
        });
        input.addEventListener('compositionend', function () {
            widget.composing = false;
        });
        input.addEventListener('keydown', function (e) {
            if (e.key !== 'Enter' || e.shiftKey) return;
            if (e.isComposing || widget.composing) return;
            e.preventDefault();
            widget.sendText();
        });
    }

    function Widget(cfg) {
        this.cfg = cfg || {};
        this.apiBase = inferApiBase();
        this.open = false;
        this.busy = false;
        this.sessionId = null;
        this.ws = null;
        this.root = null;
        this.panel = null;
        this.list = null;
        this.status = null;
        this.input = null;
        this.sendBtn = null;
        this.typingEl = null;
        this.petPlayer = null;
        this.petEl = null;
        this.petWrapEl = null;
        this.petHintEl = null;
        this.petHintText = '';
        this.pos = null;
        this.petPos = null;
        this.savedPetPos = null;
        this.composing = false;
        this.onResize = null;
        this.pendingFile = null;
        this.fileInput = null;
        this.fileChip = null;
        this.fileChipName = null;
        this.attachBtn = null;
        this.newSessionBtn = null;
        this.logoUrl = '';
        this.pc = null;
        this.dc = null;
        this.localStream = null;
        this.remoteAudio = null;
        this.transportMode = resolveTransport(this.cfg);
        this.transcriptEls = {};
    }

    Widget.prototype.mount = function () {
        var self = this;
        var primary = this.cfg.primaryColor || '#165DFF';
        var isRight = this.cfg.position !== 'left';
        SPRITE_IDLE.url = assetUrl(this.apiBase, 'sprite_idle.png');
        SPRITE_HELLO.url = assetUrl(this.apiBase, 'sprite_hello.png');
        self.logoUrl = assetUrl(this.apiBase, 'icon-lingyu.png');

        injectLocalCSS(primary);
        return Promise.resolve().then(function () {
            if (document.getElementById(ROOT_ID)) return self;

            self.root = el('div', {
                id: ROOT_ID,
                style: {
                    all: 'initial',
                    fontFamily: '-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif',
                    zIndex: '2147483000',
                    position: 'fixed',
                    display: 'block',
                    pointerEvents: 'none',
                },
            });

            self.panel = el(
                'div',
                {
                    className: 'le-panel',
                    style: { display: 'none' },
                },
                [
                    el('div', {
                        className: 'le-header',
                    }, [
                        el('div', { className: 'le-header-brand' }, [
                            el('img', {
                                className: 'le-header-logo',
                                src: self.logoUrl,
                                alt: '',
                            }),
                            el('div', { className: 'le-title' }, [
                                el('div', {
                                    className: 'le-title-main',
                                    text: self.cfg.title || '助手',
                                }),
                                (self.status = el('span', {
                                    className: 'le-status',
                                    text: '',
                                })),
                            ]),
                        ]),
                        el('div', { className: 'le-header-actions' }, [
                            (self.newSessionBtn = el('button', {
                                type: 'button',
                                className: 'le-new-session',
                                title: '新会话',
                                onClick: function (e) {
                                    e.stopPropagation();
                                    self.resetSession();
                                },
                            }, [
                                svgIcon('0 0 24 24', [
                                    'M12 5v14',
                                    'M5 12h14',
                                ], { size: '16', strokeWidth: '2' }),
                            ])),
                            el('button', {
                                type: 'button',
                                className: 'le-close',
                                text: '×',
                                onClick: function (e) {
                                    e.stopPropagation();
                                    self.toggle(false);
                                },
                            }),
                        ]),
                    ]),
                    el('div', {
                        className: 'le-disclaimer',
                        text: self.cfg.disclaimer || '内容由 AI 生成，仅供参考，您据此所作判断及操作均由您自行承担责任。',
                    }),
                    (self.list = el('div', {
                        className: 'le-list',
                    })),
                    el('div', { className: 'le-footer' }, [
                        (self.fileChip = el('div', { className: 'le-file-chip' }, [
                            (self.fileChipName = el('span', { className: 'le-file-chip-name', text: '' })),
                            el('button', {
                                type: 'button',
                                className: 'le-file-chip-clear',
                                text: '×',
                                onClick: function () {
                                    self.setPendingFile(null);
                                },
                            }),
                        ])),
                        el('div', { className: 'le-input-wrap' }, [
                            (self.attachBtn = el('button', {
                                type: 'button',
                                className: 'le-icon-btn',
                                title: '上传文件',
                                onClick: function () {
                                    if (self.busy) return;
                                    if (self.fileInput) self.fileInput.click();
                                },
                            }, [
                                svgIcon('0 0 24 24', [
                                    'M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z',
                                    'M14 2v6h6',
                                    'M12 18v-6',
                                    'M9 15l3-3 3 3',
                                ]),
                            ])),
                            (self.input = el('textarea', {
                                rows: '1',
                                placeholder: self.cfg.placeholder || '输入消息…',
                                className: 'le-input',
                                onInput: function () {
                                    self.syncSendState();
                                    self.input.style.height = 'auto';
                                    self.input.style.height = Math.min(self.input.scrollHeight, 96) + 'px';
                                },
                            })),
                            (self.sendBtn = el('button', {
                                type: 'button',
                                className: 'le-send',
                                title: '发送',
                                onClick: function () {
                                    self.sendText();
                                },
                            }, [
                                svgIcon('0 0 24 24', ['M5 12h14', 'M13 6l6 6-6 6'], { stroke: '#fff', strokeWidth: '2' }),
                            ])),
                        ]),
                        (self.fileInput = el('input', {
                            type: 'file',
                            style: { display: 'none' },
                            accept: '.txt,.md,.pdf,.doc,.docx,.csv,.json,.xlsx,.xls,.ppt,.pptx,.html,.htm',
                        })),
                    ]),
                ],
            );

            self.petEl = createPetViewport();
            self.petHintText = resolvePetGreeting(self.cfg);
            self.petHintEl = el('div', { className: 'le-pet-hint', text: self.petHintText });
            self.petWrapEl = el('div', { className: 'le-pet-wrap' }, [self.petHintEl, self.petEl]);
            bindPetInteraction(self);
            bindPanelDrag(self);
            bindInputInteraction(self);
            bindFileUpload(self);

            self.petPlayer = new SpritePlayer(self.petEl, SPRITE_HELLO);

            self.root.appendChild(self.panel);
            self.root.appendChild(self.petWrapEl);
            document.body.appendChild(self.root);
            self.initPetPosition(isRight);
            self.syncSendState();
            preloadSprites(self.logoUrl).then(function () {
                self.petEl.classList.add('ready');
                self.petPlayer.playHelloThenIdle();
                window.setTimeout(function () {
                    self.showPetHint();
                }, 700);
            });
            self.onResize = function () {
                if (self.open) self.layoutOpenPanel();
                else if (self.petPos) self.setRootPosition(self.petPos.left, self.petPos.top, PET_SIZE, PET_SIZE);
            };
            window.addEventListener('resize', self.onResize);
            return self;
        });
    };

    Widget.prototype.showPetHint = function () {
        if (!this.petHintEl || this.open || this.cfg.greetingBubble === false) return;
        this.petHintText = resolvePetGreeting(this.cfg);
        if (!this.petHintText) return;
        this.petHintEl.textContent = this.petHintText;
        this.petHintEl.classList.add('show');
    };

    Widget.prototype.hidePetHint = function () {
        if (this.petHintEl) this.petHintEl.classList.remove('show');
    };

    Widget.prototype.initPetPosition = function (preferRight) {
        if (this.petPos) return;
        var margin = 16;
        var left = preferRight !== false
            ? window.innerWidth - PET_SIZE - margin
            : margin;
        var top = window.innerHeight - PET_SIZE - margin;
        this.petPos = { left: left, top: top };
        this.savedPetPos = this.petPos;
        this.setRootPosition(left, top, PET_SIZE, PET_SIZE);
    };

    Widget.prototype.setRootPosition = function (left, top, w, h) {
        if (!this.root) return;
        this.pos = { x: left, y: top };
        this.root.style.left = left + 'px';
        this.root.style.top = top + 'px';
        this.root.style.right = 'auto';
        this.root.style.bottom = 'auto';
        this.root.style.width = w + 'px';
        this.root.style.height = h + 'px';
    };

    Widget.prototype.capturePetPosition = function () {
        if (!this.petEl) return this.petPos;
        var rect = this.petEl.getBoundingClientRect();
        this.petPos = { left: rect.left, top: rect.top };
        this.savedPetPos = this.petPos;
        return this.petPos;
    };

    Widget.prototype.layoutPetAt = function (left, top) {
        var clamped = clampViewportRect(left, top, PET_SIZE, PET_SIZE);
        this.petPos = { left: clamped.left, top: clamped.top };
        this.savedPetPos = this.petPos;
        this.setRootPosition(clamped.left, clamped.top, PET_SIZE, PET_SIZE);
    };

    Widget.prototype.layoutOpenPanel = function () {
        if (!this.panel) return;
        var anchor = this.savedPetPos || this.petPos || this.capturePetPosition();
        this.panel.style.display = 'flex';
        this.panel.style.visibility = 'hidden';
        var pw = this.panel.offsetWidth || 360;
        var ph = this.panel.offsetHeight || 480;
        this.panel.style.visibility = '';
        var pos = layoutPanelNearPet(anchor.left, anchor.top, pw, ph);
        this.setRootPosition(pos.left, pos.top, pw, ph);
    };

    Widget.prototype.layoutClosedPet = function () {
        var pos = this.savedPetPos || this.petPos;
        if (!pos) {
            this.initPetPosition(this.cfg.position !== 'left');
            pos = this.petPos;
        }
        if (pos) this.setRootPosition(pos.left, pos.top, PET_SIZE, PET_SIZE);
    };

    Widget.prototype.isTextTransport = function () {
        return resolveTransport(this.cfg) === 'text';
    };

    Widget.prototype.cleanupVoiceConnection = function () {
        if (this.dc) {
            try {
                this.dc.close();
            } catch (_) {}
            this.dc = null;
        }
        if (this.pc) {
            try {
                this.pc.close();
            } catch (_) {}
            this.pc = null;
        }
        if (this.localStream) {
            try {
                this.localStream.getTracks().forEach(function (t) {
                    t.stop();
                });
            } catch (_) {}
            this.localStream = null;
        }
        if (this.remoteAudio) {
            this.remoteAudio.srcObject = null;
            if (this.remoteAudio.parentNode) {
                this.remoteAudio.parentNode.removeChild(this.remoteAudio);
            }
            this.remoteAudio = null;
        }
    };

    Widget.prototype.upsertTranscript = function (role, text, turnId) {
        text = String(text || '').trim();
        if (!text || !this.list) return;
        this.hideTyping();
        var isUser = role === 'user';
        var bubbleRole = isUser ? 'user' : 'bot';
        var key = turnId ? String(turnId) + ':' + role : '';
        if (key && this.transcriptEls[key]) {
            this.transcriptEls[key].textContent = text;
            this.scrollToBottom();
            return;
        }
        var row = el('div', { className: 'le-msg-row ' + bubbleRole });
        var bubble = el('div', { className: 'le-bubble ' + bubbleRole });
        bubble.textContent = text;
        row.appendChild(this.createAvatarEl(isUser ? 'user' : 'bot'));
        row.appendChild(bubble);
        this.list.appendChild(row);
        if (key) this.transcriptEls[key] = bubble;
        this.scrollToBottom();
    };

    Widget.prototype.handleWireFrame = function (fr) {
        if (!fr || !fr.type) return;
        if (fr.type === 'transcript.user' && fr.text) {
            this.upsertTranscript('user', fr.text, fr.turnId);
        } else if (fr.type === 'transcript.assistant' && fr.text) {
            this.upsertTranscript('assistant', fr.text, fr.turnId);
        } else if (fr.type === 'status' && fr.message) {
            this.setStatus(fr.message);
        } else if (fr.type === 'error') {
            this.pushMsg('bot', fr.message || '服务异常');
        }
    };

    Widget.prototype.bindDataChannel = function (dc) {
        var self = this;
        if (self.dc && self.dc !== dc) {
            try {
                self.dc.close();
            } catch (_) {}
        }
        if (self.dc === dc && dc.onmessage) return;
        dc.binaryType = 'arraybuffer';
        dc.onopen = function () {
            self.setStatus('WebRTC · 已连接');
        };
        dc.onmessage = function (ev) {
            var fr = parseWireData(ev.data);
            if (fr) self.handleWireFrame(fr);
        };
        dc.onerror = function () {
            self.pushMsg('bot', 'WebRTC 转写通道异常');
        };
        self.dc = dc;
    };

    Widget.prototype.connectWebRTC = function (sessionId) {
        var self = this;
        if (typeof RTCPeerConnection === 'undefined') {
            return Promise.reject(new Error('当前浏览器不支持 WebRTC'));
        }
        self.busy = true;
        self.syncSendState();
        self.setStatus('WebRTC · 连接中');

        var pc = new RTCPeerConnection({
            iceServers: [{ urls: 'stun:stun.l.google.com:19302' }],
        });
        self.pc = pc;

        pc.ondatachannel = function (ev) {
            if (ev.channel && ev.channel.label === 'dialog') self.bindDataChannel(ev.channel);
        };
        self.bindDataChannel(pc.createDataChannel('dialog', { ordered: true }));

        var audio = document.createElement('audio');
        audio.autoplay = true;
        audio.setAttribute('playsinline', 'true');
        audio.style.display = 'none';
        if (self.panel) self.panel.appendChild(audio);
        self.remoteAudio = audio;

        pc.ontrack = function (ev) {
            if (ev.streams && ev.streams[0]) {
                audio.srcObject = ev.streams[0];
                var p = audio.play();
                if (p && typeof p.catch === 'function') p.catch(function () {});
            }
        };
        pc.onconnectionstatechange = function () {
            var st = pc.connectionState;
            if (st === 'connected') {
                self.busy = false;
                self.syncSendState();
                self.setStatus('WebRTC · 实时语音');
            } else if (st === 'failed') {
                self.busy = false;
                self.syncSendState();
                self.pushMsg('bot', 'WebRTC 连接失败');
                self.cleanupVoiceConnection();
            }
        };

        return navigator.mediaDevices
            .getUserMedia({ audio: { echoCancellation: true, noiseSuppression: true }, video: false })
            .then(function (stream) {
                self.localStream = stream;
                stream.getTracks().forEach(function (t) {
                    pc.addTrack(t, stream);
                });
                return pc.createOffer({ offerToReceiveAudio: true });
            })
            .then(function (offer) {
                return pc.setLocalDescription(offer).then(function () {
                    return offer;
                });
            })
            .then(function (offer) {
                return new Promise(function (resolve) {
                    if (pc.iceGatheringState === 'complete') {
                        resolve();
                        return;
                    }
                    var check = function () {
                        if (pc.iceGatheringState === 'complete') {
                            pc.removeEventListener('icegatheringstatechange', check);
                            resolve();
                        }
                    };
                    pc.addEventListener('icegatheringstatechange', check);
                    window.setTimeout(resolve, 3000);
                }).then(function () {
                    return fetchWithTimeout(
                        self.apiBase + '/lingecho/voice-session/v1/webrtc/offer',
                        {
                            method: 'POST',
                            headers: self.headers(),
                            body: JSON.stringify({
                                sessionId: sessionId,
                                sdp: (pc.localDescription && pc.localDescription.sdp) || offer.sdp || '',
                                type: 'offer',
                            }),
                        },
                        30000,
                    );
                });
            })
            .then(function (body) {
                if (!body || (body.code !== 200 && body.code !== 0) || !body.data || !body.data.sdp) {
                    throw new Error((body && body.msg) || 'WebRTC 协商失败');
                }
                return pc.setRemoteDescription({ type: 'answer', sdp: body.data.sdp });
            })
            .then(function () {
                self.busy = false;
                self.syncSendState();
                self.setStatus('WebRTC · 实时语音');
            })
            .catch(function (err) {
                self.busy = false;
                self.syncSendState();
                self.cleanupVoiceConnection();
                self.setStatus('连接失败');
                self.pushMsg('bot', (err && err.message) || 'WebRTC 连接失败');
                throw err;
            });
    };

    Widget.prototype.syncSendState = function () {
        if (!this.sendBtn || !this.input) return;
        if (!this.isTextTransport()) {
            this.sendBtn.disabled = true;
            this.input.disabled = true;
            if (this.attachBtn) this.attachBtn.disabled = true;
            if (this.newSessionBtn) this.newSessionBtn.disabled = this.busy;
            return;
        }
        this.input.disabled = false;
        var hasText = (this.input.value || '').trim().length > 0;
        var hasFile = !!this.pendingFile;
        this.sendBtn.disabled = this.busy || (!hasText && !hasFile);
        if (this.attachBtn) this.attachBtn.disabled = this.busy || this.isTextTransport();
        if (this.newSessionBtn) this.newSessionBtn.disabled = this.busy;
    };

    Widget.prototype.setPendingFile = function (file) {
        this.pendingFile = file || null;
        if (this.fileChip && this.fileChipName) {
            if (this.pendingFile) {
                this.fileChipName.textContent = this.pendingFile.name;
                this.fileChip.classList.add('show');
            } else {
                this.fileChipName.textContent = '';
                this.fileChip.classList.remove('show');
            }
        }
        this.syncSendState();
    };

    Widget.prototype.authHeaders = function (forForm) {
        var h = {};
        if (!forForm) h['Content-Type'] = 'application/json';
        if (this.cfg.apiKey) h['X-API-Key'] = this.cfg.apiKey;
        if (this.cfg.token) h['Authorization'] = 'Bearer ' + this.cfg.token;
        return h;
    };

    Widget.prototype.toggle = function (force) {
        var opening = typeof force === 'boolean' ? force : !this.open;
        var self = this;
        if (opening) {
            this.hidePetHint();
            this.capturePetPosition();
            this.open = true;
            if (this.petWrapEl) {
                if (this.petEl) {
                    this.petEl.classList.remove('le-pet-in');
                    this.petEl.classList.add('le-pet-out');
                }
            }
            window.setTimeout(function () {
                if (!self.open) return;
                if (self.petWrapEl) {
                    self.petWrapEl.classList.add('hidden');
                    if (self.petEl) self.petEl.classList.remove('le-pet-out');
                }
                self.layoutOpenPanel();
                self.panel.classList.remove('le-panel-out', 'le-panel-enter');
                void self.panel.offsetWidth;
                self.panel.classList.add('le-panel-enter');
                self.ensureSession();
                window.setTimeout(function () {
                    if (self.input) self.input.focus();
                }, 120);
            }, 180);
            return;
        }

        this.open = false;
        this.panel.classList.remove('le-panel-enter');
        this.panel.classList.add('le-panel-out');
        window.setTimeout(function () {
            self.panel.style.display = 'none';
            self.panel.classList.remove('le-panel-out');
            self.hideTyping();
            if (self.petWrapEl) {
                self.layoutClosedPet();
                self.petWrapEl.classList.remove('hidden');
                if (self.petEl) {
                    self.petEl.classList.remove('hidden', 'le-pet-out', 'le-pet-in');
                    void self.petEl.offsetWidth;
                    self.petEl.classList.add('le-pet-in');
                }
                if (self.petPlayer) self.petPlayer.playIdleLoop();
                window.setTimeout(function () {
                    self.showPetHint();
                }, 280);
            }
            if (self.ws && !self.busy) {
                try {
                    self.ws.close(1000, 'panel closed');
                } catch (_) {}
                self.ws = null;
            }
            if (!self.busy) self.cleanupVoiceConnection();
            self.setStatus('');
        }, 160);
    };

    Widget.prototype.setStatus = function (s) {
        if (this.status) this.status.textContent = s;
    };

    Widget.prototype.scrollToBottom = function () {
        if (!this.list) return;
        this.list.scrollTop = this.list.scrollHeight;
    };

    Widget.prototype.createAvatarEl = function (role) {
        var isUser = role === 'user';
        var av = el('div', { className: 'le-avatar ' + (isUser ? 'user' : 'bot') });
        if (isUser) {
            av.appendChild(
                svgIcon('0 0 24 24', ['M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2', 'M12 11a4 4 0 100-8 4 4 0 000 8'], {
                    size: '15',
                    strokeWidth: '1.75',
                }),
            );
        } else {
            av.appendChild(el('img', { src: this.logoUrl, alt: '' }));
        }
        return av;
    };

    Widget.prototype.showTyping = function (show) {
        if (!this.list) return;
        if (show) {
            if (this.typingEl) return;
            this.typingEl = el('div', { className: 'le-msg-row bot' }, [
                this.createAvatarEl('bot'),
                el('div', {
                    className: 'le-bubble bot',
                }, [
                    el('div', {
                        className: 'le-typing',
                    }, [
                        el('span', { className: 'le-dot' }),
                        el('span', { className: 'le-dot' }),
                        el('span', { className: 'le-dot' }),
                    ]),
                ]),
            ]);
            this.list.appendChild(this.typingEl);
            this.scrollToBottom();
            return;
        }
        if (this.typingEl && this.typingEl.parentNode) {
            this.typingEl.parentNode.removeChild(this.typingEl);
        }
        this.typingEl = null;
    };

    Widget.prototype.pushMsg = function (role, text) {
        if (!this.list || !text) return;
        this.hideTyping();

        var isUser = role === 'user';
        var row = el('div', {
            className: 'le-msg-row ' + (isUser ? 'user' : 'bot'),
        });

        var bubble = el('div', {
            className: 'le-bubble ' + (isUser ? 'user' : 'bot'),
        });
        bubble.textContent = String(text);
        row.appendChild(this.createAvatarEl(isUser ? 'user' : 'bot'));
        row.appendChild(bubble);
        this.list.appendChild(row);
        this.scrollToBottom();
    };

    Widget.prototype.hideTyping = function () {
        this.showTyping(false);
    };

    Widget.prototype.headers = function () {
        return this.authHeaders(false);
    };

    Widget.prototype.pushFileMsg = function (fileName, text) {
        if (!this.list) return;
        this.hideTyping();
        var row = el('div', { className: 'le-msg-row user' });
        var bubble = el('div', { className: 'le-bubble user' });
        var fileLine = el('div', { className: 'le-file-msg', text: '[文件] ' + (fileName || '未命名') });
        bubble.appendChild(fileLine);
        if (text) bubble.appendChild(document.createTextNode(text));
        row.appendChild(this.createAvatarEl('user'));
        row.appendChild(bubble);
        this.list.appendChild(row);
        this.scrollToBottom();
    };

    Widget.prototype.resetSession = function () {
        var self = this;
        if (self.busy) return;
        var oldId = self.sessionId;
        self.sessionId = null;
        self.cleanupVoiceConnection();
        self.transcriptEls = {};
        if (self.ws) {
            try {
                self.ws.close(1000, 'new session');
            } catch (_) {}
            self.ws = null;
        }
        if (self.list) self.list.innerHTML = '';
        self.hideTyping();
        self.setPendingFile(null);
        if (self.input) {
            self.input.value = '';
            self.input.style.height = 'auto';
        }
        self.syncSendState();
        self.setStatus('');

        var endReq = oldId
            ? (self.isTextTransport()
                ? fetchWithTimeout(
                    self.apiBase +
                    '/lingecho/dialog/v1/conversations/' +
                    encodeURIComponent(oldId) +
                    '/end',
                    { method: 'POST', headers: self.headers(), body: '{}' },
                    15000,
                )
                : fetchWithTimeout(
                    self.apiBase +
                    '/lingecho/voice-session/v1/sessions/' +
                    encodeURIComponent(oldId),
                    { method: 'DELETE', headers: self.authHeaders(true) },
                    15000,
                ))
            : Promise.resolve();

        endReq
            .catch(function () {
                /* best-effort end */
            })
            .then(function () {
                return self.ensureSession();
            });
    };

    Widget.prototype.ensureSession = function () {
        if (this.sessionId || this.busy) return Promise.resolve(this.sessionId);
        var self = this;
        var transport = resolveTransport(this.cfg);
        self.busy = true;
        self.syncSendState();
        self.setStatus('连接中');
        self.showTyping(true);

        if (transport === 'text') {
            return fetchWithTimeout(self.apiBase + '/lingecho/dialog/v1/conversations', {
                method: 'POST',
                headers: self.headers(),
                body: JSON.stringify({
                    assistantId: String(self.cfg.assistantId || ''),
                    channel: 'api',
                }),
            })
                .then(function (body) {
                    self.hideTyping();
                    if (!body || (body.code !== 200 && body.code !== 0) || !body.data) {
                        throw new Error((body && body.msg) || '会话创建失败');
                    }
                    self.sessionId = body.data.id;
                    if (body.data.welcomeText) {
                        self.pushMsg('bot', body.data.welcomeText);
                    }
                    self.busy = false;
                    self.syncSendState();
                    self.setStatus('');
                    return self.sessionId;
                })
                .catch(function (err) {
                    self.busy = false;
                    self.syncSendState();
                    self.hideTyping();
                    self.setStatus('连接失败');
                    self.pushMsg('bot', (err && err.message) || '网络异常，请检查配置');
                });
        }

        var jsSourceId = resolveJsSourceId(self.cfg);
        var createBody = {
            transport: transport,
            assistantId: String(self.cfg.assistantId || ''),
            sampleRateHz: 16000,
        };
        if (jsSourceId) createBody.jsSourceId = jsSourceId;
        return fetchWithTimeout(self.apiBase + '/lingecho/voice-session/v1/sessions', {
            method: 'POST',
            headers: self.headers(),
            body: JSON.stringify(createBody),
        })
            .then(function (body) {
                self.hideTyping();
                if (!body || (body.code !== 200 && body.code !== 0) || !body.data) {
                    throw new Error((body && body.msg) || '会话创建失败');
                }
                self.sessionId = body.data.sessionId;
                if (transport === 'websocket') {
                    self.busy = false;
                    self.syncSendState();
                    self.setStatus('');
                    self.connectWS();
                    return self.sessionId;
                }
                if (transport === 'webrtc') {
                    return self.connectWebRTC(self.sessionId).then(function () {
                        return self.sessionId;
                    });
                }
                self.busy = false;
                self.syncSendState();
                self.setStatus('');
                return self.sessionId;
            })
            .catch(function (err) {
                self.busy = false;
                self.syncSendState();
                self.hideTyping();
                self.setStatus('连接失败');
                self.pushMsg('bot', (err && err.message) || '网络异常，请检查配置');
            });
    };

    Widget.prototype.connectWS = function () {
        var self = this;
        var base = this.apiBase.replace(/^http/, 'ws');
        var url =
            base + '/lingecho/voice-session/v1/ws?session_id=' + encodeURIComponent(this.sessionId);
        if (this.cfg.apiKey) url += '&api_key=' + encodeURIComponent(this.cfg.apiKey);
        else if (this.cfg.token) url += '&token=' + encodeURIComponent(this.cfg.token);

        this.ws = new WebSocket(url);
        this.ws.binaryType = 'arraybuffer';

        this.ws.onopen = function () {
            self.setStatus('语音通道已开启');
            try {
                self.ws.send(
                    JSON.stringify({
                        type: 'hello',
                        audio_params: {
                            format: 'pcm',
                            sample_rate: 16000,
                            channels: 1,
                            frame_duration: 60,
                            bit_depth: 16,
                        },
                    }),
                );
                self.ws.send(JSON.stringify({ type: 'listen', state: 'start', mode: 'manual' }));
            } catch (_) {}
        };

        this.ws.onmessage = function (ev) {
            if (typeof ev.data !== 'string') return;
            try {
                var fr = JSON.parse(ev.data);
                if (fr.type === 'transcript.user' && fr.text) {
                    self.upsertTranscript('user', fr.text, fr.turnId);
                } else if (fr.type === 'transcript.assistant' && fr.text) {
                    self.upsertTranscript('assistant', fr.text, fr.turnId);
                } else if (fr.type === 'error') {
                    self.pushMsg('bot', fr.message || '服务异常');
                }
            } catch (_) {}
        };

        this.ws.onclose = function () {
            self.setStatus('语音通道已关闭');
            self.ws = null;
        };
    };

    Widget.prototype.sendText = function () {
        var self = this;
        if (!this.input || this.busy || this.composing || !this.isTextTransport()) return;
        var text = (this.input.value || '').trim();
        var file = this.pendingFile;
        if (!text && !file) return;

        this.input.value = '';
        this.input.style.height = 'auto';
        this.setPendingFile(null);
        this.syncSendState();

        if (file) {
            this.pushMsg('bot', '文本模式暂不支持上传文件');
            if (!text) return;
        }
        this.pushMsg('user', text);

        requestAnimationFrame(function () {
            if (self.input && self.input.value) {
                self.input.value = '';
                self.input.style.height = 'auto';
                self.syncSendState();
            }
        });

        this.ensureSession()
            .then(function (sid) {
                if (!sid) return null;
                self.busy = true;
                self.syncSendState();
                self.setStatus('思考中');
                self.showTyping(true);
                var messagesUrl =
                    self.apiBase +
                    '/lingecho/dialog/v1/conversations/' +
                    encodeURIComponent(sid) +
                    '/messages';
                return fetchWithTimeout(
                    messagesUrl,
                    {
                        method: 'POST',
                        headers: self.headers(),
                        body: JSON.stringify({ text: text }),
                    },
                    180000,
                );
            })
            .then(function (body) {
                if (!body) return;
                self.busy = false;
                self.syncSendState();
                self.hideTyping();
                self.setStatus('');
                if ((body.code === 200 || body.code === 0) && body.data && body.data.reply) {
                    self.pushMsg('bot', body.data.reply);
                } else {
                    self.pushMsg('bot', (body && body.msg) || '消息发送失败');
                }
            })
            .catch(function (err) {
                self.busy = false;
                self.syncSendState();
                self.hideTyping();
                self.setStatus('');
                self.pushMsg('bot', (err && err.message) || '请求异常');
            });
    };

    Widget.prototype.destroy = function () {
        if (this.onResize) window.removeEventListener('resize', this.onResize);
        if (this.petPlayer) this.petPlayer.stop();
        this.cleanupVoiceConnection();
        if (this.ws) {
            try {
                this.ws.close(1000, 'destroy');
            } catch (_) {}
            this.ws = null;
        }
        if (this.root && this.root.parentNode) this.root.parentNode.removeChild(this.root);
        var css = document.getElementById('lingecho-embed-css');
        if (css) css.remove();
        delete window.LingEchoWidget;
    };

    function autoMount() {
        var widget = new Widget(CFG);
        window.LingEchoWidget = widget;
        if (CFG.autoMount !== false) {
            widget.mount().catch(function (err) {
                console.error('[SoulNexus embed]', err);
            });
        }
        return widget;
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', autoMount);
    } else {
        autoMount();
    }
})();
