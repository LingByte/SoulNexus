/**
 * SoulNexus — Mao 桌宠（mao.js）
 * Cubism 4 · 官方样本 Mao
 * 每形象单独 JS 模板；助手配置默认与懒懒一致。
 */
(function () {
    'use strict';

    /** 与 desktop-pet / 懒懒默认一致，embed 注入时可被 __LingEchoConfig / __LanlanConfig 覆盖 */
    var DEFAULTS = {
        apiBase: 'https://soulmy.top/api',
        apiKey: 'soulnexus_user_PI2mRsBxioqpTkAS3K3yG3Z_YzY2smCsidEWJRuamMI',
        assistantId: '8859281265343332864',
        transport: 'websocket',
        size: 240,
        position: 'right',
        name: 'Mao',
        title: 'Mao',
        autoMount: true,
        persist: true,
        voiceHotkey: 'Alt+Shift+V',
        talkHotkey: 'Alt+Shift+T',
    };

    var CFG = Object.assign({}, DEFAULTS, window.__LingEchoConfig || {}, window.__LanlanConfig || {});
    var ROOT_ID = 'lingecho-embed-root';
    var CSS_ID = 'lingecho-cubism-mao-css';
    var STORAGE_KEY = 'cubism-pet-mao-v1';

    var PET = {
        name: 'Mao',
        modelUrl: 'https://cdn.lingecho.com/live2d/mao/Mao.model3.json',
        idleMotion: 'Idle',
        tapMotion: 'TapBody',
    };

    function resolvePetSize(cfg) {
        return Math.max(96, Math.min(384, Number(cfg && cfg.size) || DEFAULTS.size));
    }

    /** Live2D 模型在画布内的额外缩放 */
    function resolveModelScale(cfg) {
        var n = Number(cfg && cfg.live2dModelScale);
        return n > 0 ? n : 1.05;
    }

    /** 全身立绘需要比 width 更高的 canvas，避免裁头（默认约为 size 的 1.5 倍高） */
    function resolveCanvasHeight(cfg, petSize) {
        var ratio = Number(cfg && cfg.live2dCanvasRatio);
        if (ratio > 0.8 && ratio < 2.5) return Math.round(petSize * ratio);
        return Math.round(petSize * 1.5);
    }

    var ASSET_CDN = 'https://cdn.lingecho.com';
    var BUILTIN_CUBISM_CORE = ASSET_CDN + '/live2d/sdk/live2dcubismcore.min.js';
    var BUILTIN_LIVE2D_LEGACY = ASSET_CDN + '/live2d/sdk/live2d.min.js';
    var BUILTIN_PIXI = ASSET_CDN + '/live2d/sdk/pixi.min.js';
    var BUILTIN_LIVE2D_SDK = 'https://cdn.lingecho.com/live2d/sdk/cubism4.min.js';

    var CUBISM_CORE_CDN = String(CFG.live2dCubismCoreCdn || BUILTIN_CUBISM_CORE);
    var LIVE2D_LEGACY_CDN = String(CFG.live2dLegacyCdn || BUILTIN_LIVE2D_LEGACY);
    var PIXI_CDN = String(CFG.live2dPixiCdn || BUILTIN_PIXI);
    var LIVE2D_CDN = String(CFG.live2dSdkCdn || BUILTIN_LIVE2D_SDK);

    var LIVE2D_RUNTIME = 'cubism4';

    function live2dRuntimeReady() {
        if (!window.PIXI || !window.PIXI.live2d || !window.PIXI.live2d.Live2DModel) {
            return false;
        }
        return window.__SoulNexusLive2dRuntime === LIVE2D_RUNTIME;
    }

    var LINES = {
        greet: ['你好呀～', '我在呢', '点我可以互动'],
        thinking: ['让我想想…', '嗯…'],
        noLlm: ['还没接上助手（检查 apiBase / assistantId）', '现在只能做动作～'],
        error: ['加载模型失败了', '网络或模型地址有问题'],
    };

    function el(tag, attrs, children) {
        var node = document.createElement(tag);
        if (attrs) {
            Object.keys(attrs).forEach(function (k) {
                if (k === 'style' && typeof attrs[k] === 'object') Object.assign(node.style, attrs[k]);
                else if (k === 'text') node.textContent = attrs[k];
                else if (k.slice(0, 2) === 'on') node.addEventListener(k.slice(2).toLowerCase(), attrs[k]);
                else if (k === 'className') node.className = attrs[k];
                else node.setAttribute(k, attrs[k]);
            });
        }
        (children || []).forEach(function (c) {
            if (c == null) return;
            node.appendChild(typeof c === 'string' ? document.createTextNode(c) : c);
        });
        return node;
    }

    function pick(arr) {
        return arr[Math.floor(Math.random() * arr.length)];
    }

    function now() {
        return Date.now();
    }

    function inferApiBase(cfg) {
        cfg = cfg || {};
        if (cfg.apiBase) return String(cfg.apiBase).replace(/\/$/, '');
        var scripts = document.getElementsByTagName('script');
        for (var i = scripts.length - 1; i >= 0; i--) {
            var src = scripts[i].src || '';
            var m = src.match(/^(.*)\/lingecho\/embed\/v1\/(?:t\/[^/]+\/)?embed\.js(?:\?|$)/);
            if (m) return m[1].replace(/\/$/, '');
        }
        return String(DEFAULTS.apiBase || '/api').replace(/\/$/, '');
    }

    function parseHotkey(spec) {
        var parts = String(spec || '')
            .split('+')
            .map(function (s) {
                return s.trim().toLowerCase();
            })
            .filter(Boolean);
        var need = { alt: false, ctrl: false, shift: false, meta: false, key: '' };
        parts.forEach(function (p) {
            if (p === 'alt' || p === 'option') need.alt = true;
            else if (p === 'ctrl' || p === 'control' || p === 'controlorcommand' || p === 'commandorcontrol') {
                need.ctrl = true;
            } else if (p === 'shift') need.shift = true;
            else if (p === 'meta' || p === 'cmd' || p === 'command') need.meta = true;
            else need.key = p;
        });
        return need;
    }

    function matchHotkey(e, need) {
        if (!need || !need.key) return false;
        var key = String(e.key || '').toLowerCase();
        var code = String(e.code || '').toLowerCase();
        if (key === ' ') key = 'space';
        var want = need.key;
        var keyOk = false;
        if (want === 'space') keyOk = key === 'space' || code === 'space';
        else if (want.length === 1) keyOk = key === want || code === 'key' + want;
        else keyOk = key === want || code === want;
        if (!keyOk) return false;
        if (!!e.altKey !== need.alt) return false;
        if (!!e.shiftKey !== need.shift) return false;
        var mod = !!e.ctrlKey || !!e.metaKey;
        if (need.ctrl || need.meta) {
            if (!mod) return false;
        } else if (mod) return false;
        return true;
    }

    function ensureProcessShim() {
        var g = typeof globalThis !== 'undefined' ? globalThis : window;
        if (typeof g.process === 'undefined') {
            g.process = { env: { NODE_ENV: 'production' } };
        }
    }

    function loadScript(src) {
        return new Promise(function (resolve, reject) {
            var existing = document.querySelector('script[data-soulnexus-src="' + src + '"]');
            if (existing) {
                if (existing.getAttribute('data-loaded') === '1') {
                    resolve();
                    return;
                }
                existing.addEventListener('load', function () { resolve(); });
                existing.addEventListener('error', function () {
                    reject(new Error('script load failed: ' + src));
                });
                return;
            }
            var s = document.createElement('script');
            s.src = src;
            s.async = false;
            s.setAttribute('data-soulnexus-src', src);
            s.onload = function () {
                s.setAttribute('data-loaded', '1');
                resolve();
            };
            s.onerror = function () {
                reject(new Error('script load failed: ' + src));
            };
            document.head.appendChild(s);
        });
    }

    function assertLive2dRuntime() {
        if (!window.PIXI || !window.PIXI.live2d || !window.PIXI.live2d.Live2DModel) {
            throw new Error('Live2D 运行时未就绪');
        }


        if (typeof window.Live2DCubismCore === 'undefined') {
            throw new Error('Live2DCubismCore 未加载');
        }
    }

    function ensureLive2dRuntime() {
        ensureProcessShim();
        if (live2dRuntimeReady()) {
            return Promise.resolve();
        }
        return loadScript(CUBISM_CORE_CDN)
            .then(function () {
                return loadScript(PIXI_CDN);
            })
            .then(function () {
                return loadScript(LIVE2D_CDN);
            })
            .then(function () {
                window.__SoulNexusLive2dRuntime = LIVE2D_RUNTIME;
                assertLive2dRuntime();
            });
    }

    function safeParseJson(raw) {
        if (!raw || !String(raw).trim()) throw new Error('empty response');
        return JSON.parse(raw);
    }

    function fetchJSON(url, options, timeout) {
        timeout = timeout || 60000;
        var controller = typeof AbortController !== 'undefined' ? new AbortController() : null;
        var timer = controller ? setTimeout(function () { controller.abort(); }, timeout) : null;
        var opts = options || {};
        if (controller) opts.signal = controller.signal;
        return fetch(url, opts)
            .then(function (resp) {
                return resp.text().then(function (raw) {
                    if (timer) clearTimeout(timer);
                    var body = safeParseJson(raw);
                    if (!resp.ok && body && body.code == null) body.code = resp.status;
                    return body;
                });
            })
            .catch(function (err) {
                if (timer) clearTimeout(timer);
                if (err && err.name === 'AbortError') throw new Error('timeout');
                throw err;
            });
    }

    function injectCSS(width, height) {
        if (document.getElementById(CSS_ID)) return;
        var css = [
            '#' + ROOT_ID + '{position:fixed;z-index:2147483000;pointer-events:none;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC",sans-serif}',
            '#' + ROOT_ID + ' *{box-sizing:border-box}',
            '#' + ROOT_ID + ' .ll-wrap{position:relative;width:' + width + 'px;height:' + height + 'px;pointer-events:auto;overflow:visible}',
            '#' + ROOT_ID + ' .ll-pet{width:' + width + 'px;height:' + height + 'px;cursor:grab;user-select:none;overflow:visible;opacity:0;transition:opacity .35s ease}',
            '#' + ROOT_ID + ' .ll-pet.ready{opacity:1}',
            '#' + ROOT_ID + ' .ll-pet.dragging{cursor:grabbing}',
            '#' + ROOT_ID + ' .ll-live2d-host{width:100%;height:100%;display:block;overflow:visible}',
            '#' + ROOT_ID + ' .ll-live2d-host canvas{display:block;width:100%!important;height:100%!important;overflow:visible}',
            '#' + ROOT_ID + ' .ll-hint{position:absolute;left:50%;bottom:calc(100% - 4px);transform:translateX(-50%);z-index:3;max-width:240px;padding:8px 12px;border-radius:14px;background:rgba(255,255,255,.96);border:1px solid rgba(0,0,0,.08);box-shadow:0 6px 22px rgba(0,0,0,.12);font-size:12px;line-height:1.45;color:#3f3f46;text-align:center;pointer-events:none;opacity:0;visibility:hidden;transition:opacity .2s ease,visibility .2s ease}',
            '#' + ROOT_ID + ' .ll-hint.show{opacity:1;visibility:visible}',
            '#' + ROOT_ID + ' .ll-talk{position:absolute;left:50%;bottom:calc(100% + 8px);transform:translateX(-50%);z-index:5;display:none;width:min(260px,70vw);padding:8px;border-radius:14px;background:rgba(255,255,255,.98);border:1px solid rgba(0,0,0,.08);box-shadow:0 10px 28px rgba(0,0,0,.14);pointer-events:auto;flex-direction:column;gap:6px}',
            '#' + ROOT_ID + ' .ll-talk.open{display:flex}',
            '#' + ROOT_ID + ' .ll-talk input{width:100%;border:0;border-bottom:1px solid #e4e4e7;outline:none;font-size:13px;padding:8px 2px;background:transparent}',
            '#' + ROOT_ID + ' .ll-talk-actions{display:flex;justify-content:flex-end;gap:6px}',
            '#' + ROOT_ID + ' .ll-talk button{border:0;border-radius:8px;padding:6px 10px;font-size:12px;cursor:pointer;background:#27272a;color:#fff}',
        ].join('');
        document.head.appendChild(el('style', { id: CSS_ID, text: css }));
    }

    function parseReplyTags(text) {
        var motion = null;
        var expression = null;
        var clean = String(text || '');
        var motionRe = /\[motion:([^\]\s]+)\]/gi;
        var exprRe = /\[expression:([^\]\s]+)\]/gi;
        var m;
        while ((m = motionRe.exec(clean))) motion = m[1];
        while ((m = exprRe.exec(clean))) expression = m[1];
        clean = clean.replace(motionRe, '').replace(exprRe, '').trim();
        return { text: clean, motion: motion, expression: expression };
    }

    function Live2dPet(cfg) {
        this.cfg = cfg || {};
        this.petSize = resolvePetSize(this.cfg);
        this.canvasHeight = resolveCanvasHeight(this.cfg, this.petSize);
        this.modelScale = resolveModelScale(this.cfg);
        this.name = String(cfg.name || cfg.title || PET.name).trim() || PET.name;
        this.apiBase = inferApiBase(this.cfg);
        this.talkHotkey = parseHotkey(cfg.talkHotkey || DEFAULTS.talkHotkey);
        this.voiceHotkey = parseHotkey(cfg.voiceHotkey || DEFAULTS.voiceHotkey);
        this._unsubs = [];
        this.pos = { left: null, top: null };
        this.root = null;
        this.wrap = null;
        this.petEl = null;
        this.hintEl = null;
        this.talkEl = null;
        this.talkInput = null;
        this.talkOpen = false;
        this.app = null;
        this.model = null;
        this.busy = false;
        this.destroyed = false;
        this.textSessionId = null;
        this.modelUrl = String(cfg.live2dModelUrl || cfg.modelUrl || PET.modelUrl).trim();
        this.idleMotion = String(cfg.live2dIdleMotion || PET.idleMotion).trim() || PET.idleMotion;
        this.tapMotion = String(cfg.live2dTapMotion || PET.tapMotion).trim() || PET.tapMotion;
    }

    Live2dPet.prototype.hasLlm = function () {
        var id = String(this.cfg.assistantId || '').trim();
        return !!(this.apiBase && id && id !== 'YOUR_ASSISTANT_ID');
    };

    Live2dPet.prototype.authHeaders = function () {
        var h = { 'Content-Type': 'application/json' };
        if (this.cfg.apiKey) h['X-API-Key'] = this.cfg.apiKey;
        if (this.cfg.token) h['Authorization'] = 'Bearer ' + this.cfg.token;
        return h;
    };

    Live2dPet.prototype.say = function (text, ms) {
        var self = this;
        if (!self.hintEl) return;
        self.hintEl.textContent = String(text || '');
        self.hintEl.classList.add('show');
        clearTimeout(self._hintTimer);
        self._hintTimer = setTimeout(function () {
            self.hintEl.classList.remove('show');
        }, ms || 2600);
    };

    Live2dPet.prototype.playMotion = function (name, priority) {
        var self = this;
        if (!self.model || !name) return false;
        try {
            return self.model.motion(String(name), priority == null ? 2 : priority);
        } catch (e) {
            console.warn('[Live2D] motion failed', name, e);
            return false;
        }
    };

    Live2dPet.prototype.loadLive2dModel = function () {
        var self = this;
        if (!self.app) return Promise.reject(new Error('pixi not ready'));
        return PIXI.live2d.Live2DModel.from(self.modelUrl, { autoHitTest: false, autoFocus: false }).then(function (model) {
            if (self.destroyed) {
                model.destroy();
                return;
            }
            self.model = model;
            self.app.stage.addChild(model);
            self.fitModel();
            self.petEl.classList.add('ready');
            self.playMotion(self.idleMotion);
        });
    };

    Live2dPet.prototype.setExpression = function (name) {
        if (!this.model || !name) return;
        try {
            this.model.expression(String(name));
        } catch (e) {
            console.warn('[Cubism] expression failed', name, e);
        }
    };

    Live2dPet.prototype.fitModel = function () {
        var self = this;
        if (!self.model) return;
        var wBox = self.petSize;
        var hBox = self.canvasHeight;
        var bounds = self.model.getLocalBounds();
        var w = bounds.width || 1;
        var h = bounds.height || 1;
        var padX = 0.94;
        var padY = 0.92;
        var scale = Math.min((wBox * padX) / w, (hBox * padY) / h) * self.modelScale;
        self.model.scale.set(scale);
        self.model.anchor.set(0.5, 1);
        self.model.x = wBox / 2;
        self.model.y = hBox - 6;
    };

    Live2dPet.prototype.bindDrag = function () {
        var self = this;
        var dragging = false;
        var startX = 0;
        var startY = 0;
        var origLeft = 0;
        var origTop = 0;

        self.petEl.addEventListener('mousedown', function (e) {
            if (e.button !== 0) return;
            dragging = true;
            self.petEl.classList.add('dragging');
            startX = e.clientX;
            startY = e.clientY;
            var rect = self.root.getBoundingClientRect();
            origLeft = rect.left;
            origTop = rect.top;
            e.preventDefault();
        });

        window.addEventListener('mousemove', function (e) {
            if (!dragging) return;
            var dx = e.clientX - startX;
            var dy = e.clientY - startY;
            self.pos.left = origLeft + dx;
            self.pos.top = origTop + dy;
            self.applyPos();
        });

        window.addEventListener('mouseup', function () {
            if (!dragging) return;
            dragging = false;
            self.petEl.classList.remove('dragging');
            self.savePos();
        });
    };

    Live2dPet.prototype.applyPos = function () {
        if (this.pos.left == null || this.pos.top == null) return;
        this.root.style.left = Math.round(this.pos.left) + 'px';
        this.root.style.top = Math.round(this.pos.top) + 'px';
        this.root.style.right = 'auto';
        this.root.style.bottom = 'auto';
    };

    Live2dPet.prototype.restorePos = function () {
        try {
            var raw = localStorage.getItem(STORAGE_KEY);
            if (!raw) return;
            var data = JSON.parse(raw);
            if (data && data.left != null && data.top != null) {
                this.pos.left = data.left;
                this.pos.top = data.top;
                this.applyPos();
            }
        } catch (_) {}
    };

    Live2dPet.prototype.savePos = function () {
        try {
            localStorage.setItem(
                STORAGE_KEY,
                JSON.stringify({ left: this.pos.left, top: this.pos.top, savedAt: now() }),
            );
        } catch (_) {}
    };

    Live2dPet.prototype.defaultPosition = function () {
        var side = String(this.cfg.position || DEFAULTS.position || 'right').toLowerCase();
        var margin = 24;
        this.root.style.bottom = margin + 'px';
        this.root.style.top = 'auto';
        if (side === 'left') {
            this.root.style.left = margin + 'px';
            this.root.style.right = 'auto';
        } else {
            this.root.style.right = margin + 'px';
            this.root.style.left = 'auto';
        }
    };

    Live2dPet.prototype.bindHotkeys = function () {
        var self = this;
        if (window.__LINGECHO_EMBED_MODE__ === 'desktop') return;
        function onKey(e) {
            if (e.repeat) return;
            var tag = (e.target && e.target.tagName) || '';
            if (self.talkOpen && tag === 'INPUT') return;
            if (tag === 'INPUT' || tag === 'TEXTAREA' || (e.target && e.target.isContentEditable)) return;
            if (matchHotkey(e, self.talkHotkey)) {
                e.preventDefault();
                self.toggleTalk();
            }
        }
        document.addEventListener('keydown', onKey, true);
        self._unsubs.push(function () {
            document.removeEventListener('keydown', onKey, true);
        });
    };

    Live2dPet.prototype.buildTalkBox = function () {
        var self = this;
        var input = el('input', { type: 'text', placeholder: '说点什么…' });
        var send = el('button', { text: '发送', className: 'primary' });
        var close = el('button', { text: '关闭' });
        var actions = el('div', { className: 'll-talk-actions' }, [close, send]);
        var box = el('div', { className: 'll-talk' }, [input, actions]);
        send.addEventListener('click', function () {
            var t = String(input.value || '').trim();
            if (!t) return;
            input.value = '';
            self.ask(t);
        });
        input.addEventListener('keydown', function (e) {
            if (e.key === 'Enter') send.click();
        });
        close.addEventListener('click', function () {
            self.closeTalk();
        });
        self.talkInput = input;
        return box;
    };

    Live2dPet.prototype.openTalk = function () {
        this.talkOpen = true;
        if (this.talkEl) this.talkEl.classList.add('open');
        if (this.talkInput) this.talkInput.focus();
    };

    Live2dPet.prototype.closeTalk = function () {
        this.talkOpen = false;
        if (this.talkEl) this.talkEl.classList.remove('open');
    };

    Live2dPet.prototype.toggleTalk = function (force) {
        if (typeof force === 'boolean') {
            if (force) this.openTalk();
            else this.closeTalk();
            return;
        }
        if (this.talkOpen) this.closeTalk();
        else this.openTalk();
    };

    Live2dPet.prototype.ensureTextSession = function () {
        var self = this;
        if (!self.hasLlm()) return Promise.reject(new Error('no llm'));
        if (self.textSessionId) return Promise.resolve(self.textSessionId);
        return fetchJSON(
            self.apiBase + '/lingecho/dialog/v1/conversations',
            {
                method: 'POST',
                headers: self.authHeaders(),
                body: JSON.stringify({
                    assistantId: String(self.cfg.assistantId || ''),
                    channel: 'api',
                }),
            },
            30000,
        ).then(function (body) {
            if (!body || (body.code !== 200 && body.code !== 0) || !body.data) {
                throw new Error((body && body.msg) || 'session failed');
            }
            self.textSessionId = body.data.id;
            return self.textSessionId;
        });
    };

    Live2dPet.prototype.ask = function (userText) {
        var self = this;
        var text = String(userText || '').trim();
        if (!text) return Promise.resolve(null);
        if (!self.hasLlm()) {
            self.say(pick(LINES.noLlm));
            self.playMotion(self.idleMotion);
            return Promise.resolve(null);
        }
        if (self.busy) {
            self.say('等我一下…', 1200);
            return Promise.resolve(null);
        }
        self.busy = true;
        self.say(pick(LINES.thinking), 8000);
        self.playMotion(self.idleMotion, 1);

        var prefix =
            '【' +
            self.name +
            '桌宠】你是「' +
            self.name +
            '」。回复 1～2 句中文。可选标签：[motion:TapBody]、[expression:f01]。';

        return self
            .ensureTextSession()
            .then(function (sid) {
                return fetchJSON(
                    self.apiBase +
                        '/lingecho/dialog/v1/conversations/' +
                        encodeURIComponent(sid) +
                        '/messages',
                    {
                        method: 'POST',
                        headers: self.authHeaders(),
                        body: JSON.stringify({ text: prefix + '\n主人说：' + text }),
                    },
                    120000,
                );
            })
            .then(function (body) {
                self.busy = false;
                var reply = body && body.data && body.data.reply;
                if ((body.code === 200 || body.code === 0) && reply != null && String(reply).trim()) {
                    var parsed = parseReplyTags(String(reply));
                    self.say(parsed.text || String(reply));
                    if (parsed.expression) self.setExpression(parsed.expression);
                    if (parsed.motion) self.playMotion(parsed.motion);
                    else self.playMotion(self.idleMotion);
                    return String(reply);
                }
                self.say((body && body.msg) || '没听清…', 2400);
                self.playMotion(self.idleMotion);
                return null;
            })
            .catch(function (err) {
                self.busy = false;
                self.say((err && err.message) || '请求失败', 2800);
                self.playMotion(self.idleMotion);
                return null;
            });
    };

    Live2dPet.prototype.initPixi = function () {
        var self = this;
        var host = self.petEl.querySelector('.ll-live2d-host');
        if (!host) return Promise.reject(new Error('no host'));

        return ensureLive2dRuntime().then(function () {
            if (self.destroyed) return;
            if (PIXI.settings) PIXI.settings.CROSS_ORIGIN = 'anonymous';
            self.app = new PIXI.Application({
                width: self.petSize,
                height: self.canvasHeight,
                backgroundAlpha: 0,
                antialias: true,
                autoDensity: true,
                resolution: window.devicePixelRatio || 1,
            });
            host.appendChild(self.app.view);

            return self.loadLive2dModel().then(function () {
                if (self.destroyed || !self.model) return;
                self.say(pick(LINES.greet));
                self.model.on('hit', function (hitAreas) {
                    if (!hitAreas || !hitAreas.length) return;
                    self.playMotion(self.tapMotion);
                    self.say('碰到 ' + hitAreas[0], 1600);
                });
                self.petEl.addEventListener('click', function (e) {
                    if (e.button !== 0) return;
                    self.playMotion(self.tapMotion);
                });
            });
        });
    };

    Live2dPet.prototype.mount = function () {
        var self = this;
        if (document.getElementById(ROOT_ID)) {
            console.warn('[Cubism] already mounted');
            return self;
        }
        injectCSS(this.petSize, this.canvasHeight);
        self.root = el('div', { id: ROOT_ID });
        self.hintEl = el('div', { className: 'll-hint' });
        self.talkEl = self.buildTalkBox();
        self.petEl = el(
            'div',
            {
                className: 'll-pet le-pet',
                title:
                    self.name +
                    ' · 拖动 · 双击对话 · ' +
                    (self.cfg.talkHotkey || DEFAULTS.talkHotkey) +
                    ' 文字',
            },
            [el('div', { className: 'll-live2d-host' })],
        );
        self.wrap = el('div', { className: 'll-wrap le-pet-wrap' }, [self.hintEl, self.talkEl, self.petEl]);
        self.root.appendChild(self.wrap);
        document.body.appendChild(self.root);

        self.restorePos();
        if (self.pos.left == null) self.defaultPosition();

        self.bindDrag();
        self.bindHotkeys();
        self.petEl.addEventListener('dblclick', function (e) {
            e.preventDefault();
            self.toggleTalk(true);
        });
        self.petEl.addEventListener('contextmenu', function (e) {
            e.preventDefault();
            self.toggleTalk(true);
        });

        self.initPixi().catch(function (err) {
            console.error('[Cubism]', err);
            self.say(pick(LINES.error), 4000);
            self.petEl.classList.add('ready');
        });

        return self;
    };

    Live2dPet.prototype.destroy = function () {
        this.destroyed = true;
        (this._unsubs || []).forEach(function (off) {
            try {
                off();
            } catch (_) {}
        });
        this._unsubs = [];
        this.closeTalk();
        if (this.model) {
            try {
                this.model.destroy();
            } catch (_) {}
            this.model = null;
        }
        if (this.app) {
            try {
                this.app.destroy(true, { children: true, texture: true, baseTexture: true });
            } catch (_) {}
            this.app = null;
        }
        var root = document.getElementById(ROOT_ID);
        if (root && root.parentNode) root.parentNode.removeChild(root);
        var css = document.getElementById(CSS_ID);
        if (css && css.parentNode) css.parentNode.removeChild(css);
        delete window.LingEchoWidget;
        delete window.LanlanPet;
    };

    Live2dPet.prototype.getState = function () {
        return {
            kind: 'cubism',
            pet: 'mao.js',
            name: this.name,
            modelUrl: this.modelUrl,
            idleMotion: this.idleMotion,
            tapMotion: this.tapMotion,
            hasLlm: this.hasLlm(),
            pos: Object.assign({}, this.pos),
        };
    };

    var instance = null;

    function mount(cfg) {
        if (instance) instance.destroy();
        instance = new Live2dPet(Object.assign({}, CFG, cfg || {}));
        instance.mount();
        return instance;
    }

    function destroy() {
        if (instance) {
            instance.destroy();
            instance = null;
        }
    }

    var api = {
        mount: mount,
        destroy: destroy,
        toggle: function (force) {
            if (instance) instance.toggleTalk(force);
        },
        toggleVoice: function () {
            if (instance) instance.say('暂未接语音，可用文字对话～', 2800);
            return Promise.resolve(false);
        },
        play: function (motion) {
            return instance ? instance.playMotion(motion) : false;
        },
        say: function (t, ms) {
            return instance ? instance.say(t, ms) : null;
        },
        ask: function (t) {
            return instance ? instance.ask(t) : Promise.resolve(null);
        },
        openTalk: function () {
            if (instance) instance.openTalk();
        },
        setExpression: function (name) {
            if (instance) instance.setExpression(name);
        },
        getState: function () {
            return instance ? instance.getState() : null;
        },
        get instance() {
            return instance;
        },
        notifyCodingKey: function () {
            if (instance) instance.playMotion(instance.tapMotion);
        },
    };

    window.LanlanPet = api;
    window.LingEchoWidget = api;

    if (CFG.autoMount !== false) {
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', function () {
                mount(CFG);
            });
        } else {
            mount(CFG);
        }
    }
})();
