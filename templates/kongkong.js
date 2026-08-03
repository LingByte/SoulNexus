/**
 * LingEchoX — 空空（Kongkong）孙悟空桌宠
 *
 * 精灵图：
 *   idle.png  — 站立=第 0 帧；跑步=其余帧循环
 *   jump.png  — 跳跃一拍
 *
 * 操作：
 *   ← → / A D  跑步移动（自动左右翻转）
 *   Space / W / ↑  跳跃
 *   拖拽角色任意放置
 *
 * Usage:
 *   <script>
 *     window.__KongkongConfig = {
 *       apiBase: 'https://your-host/api',
 *       size: 88,
 *       position: 'right',
 *       name: '空空',
 *       autoMount: true,
 *       persist: true,
 *       autoWander: true
 *     };
 *   </script>
 *   <script src=".../embed.js" async></script>
 *
 * API: KongkongPet.mount / destroy / play / wanderTo / getState
 */
(function () {
    'use strict';

    var CFG = Object.assign({}, window.__LingEchoConfig || {}, window.__KongkongConfig || {});
    var ROOT_ID = 'lingecho-embed-root';
    var CSS_ID = 'kongkong-embed-css';
    var CDN = 'https://cdn.lingecho.com/kongkong';
    var STORAGE_KEY = 'kongkong-pet-v1';

    /** 显示宽度；高度按精灵格 256×439 比例（默认偏小，避免占屏） */
    var PET_W = Math.max(64, Math.min(180, Number(CFG.size) || 88));
    var PET_H = Math.round(PET_W * (439 / 256));

    /**
     * idle.png 2560×4390 → 10×10 格，单格 256×439
     * jump.png 2304×3951 → 9×9 格，单格 256×439
     */
    var ACTIONS = {
        idle: {
            url: CDN + '/idle.png',
            cols: 10,
            rows: 10,
            frames: 1,
            fps: 1,
            loop: true,
            kind: 'idle',
        },
        run: {
            url: CDN + '/idle.png',
            cols: 10,
            rows: 10,
            frames: 81,
            fps: 14,
            loop: true,
            from: 1,
            loopFrom: 1,
            kind: 'move',
        },
        jump: {
            url: CDN + '/jump.png',
            cols: 9,
            rows: 9,
            frames: 73,
            fps: 16,
            loop: false,
            kind: 'move',
        },
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

    function clamp(n, a, b) { return Math.max(a, Math.min(b, n)); }
    function lerp(a, b, t) { return a + (b - a) * t; }

    function preloadImage(src) {
        return new Promise(function (resolve) {
            var img = new Image();
            img.onload = function () {
                if (img.decode) img.decode().then(function () { resolve(true); }).catch(function () { resolve(true); });
                else resolve(true);
            };
            img.onerror = function () { resolve(false); };
            img.src = src;
        });
    }

    function injectCSS() {
        if (document.getElementById(CSS_ID)) return;
        var css = [
            '#' + ROOT_ID + '{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;pointer-events:none}',
            '#' + ROOT_ID + ' *{box-sizing:border-box}',
            '#' + ROOT_ID + ' .kk-stage{position:fixed;inset:0;z-index:2147483000;pointer-events:none}',
            '#' + ROOT_ID + ' .kk-wrap{position:absolute;width:' + PET_W + 'px;height:' + PET_H + 'px;pointer-events:auto}',
            '#' + ROOT_ID + ' .kk-pet{width:100%;height:100%;overflow:hidden;cursor:grab;user-select:none;touch-action:none;',
            'filter:drop-shadow(0 8px 16px rgba(0,0,0,.18));opacity:0;transition:opacity .22s ease}',
            '#' + ROOT_ID + ' .kk-pet.ready{opacity:1}',
            '#' + ROOT_ID + ' .kk-pet.dragging{cursor:grabbing;opacity:.95}',
            '#' + ROOT_ID + ' .kk-sprite{display:block;width:100%;height:100%;background-repeat:no-repeat;image-rendering:auto;',
            'transform-origin:center bottom;will-change:background-position,transform}',
            '#' + ROOT_ID + ' .kk-hint{position:absolute;left:50%;bottom:calc(100% - 8px);transform:translateX(-50%);',
            'max-width:200px;padding:6px 10px;border-radius:10px;background:#fff;border:1px solid rgba(0,0,0,.08);',
            'box-shadow:0 4px 14px rgba(0,0,0,.1);font-size:12px;color:#3f3f46;opacity:0;visibility:hidden;',
            'pointer-events:none;transition:opacity .18s ease,visibility .18s ease;white-space:nowrap}',
            '#' + ROOT_ID + ' .kk-hint.show{opacity:1;visibility:visible}',
            '#' + ROOT_ID + ' .kk-hint::after{content:"";position:absolute;left:50%;bottom:-5px;width:10px;height:10px;',
            'background:#fff;border-right:1px solid rgba(0,0,0,.08);border-bottom:1px solid rgba(0,0,0,.08);',
            'transform:translateX(-50%) rotate(45deg)}',
        ].join('');
        document.head.appendChild(el('style', { id: CSS_ID, text: css }));
    }

    function SpritePlayer(viewport) {
        this.viewport = viewport;
        this.sprite = viewport.querySelector('.kk-sprite') || viewport;
        this.spec = ACTIONS.idle;
        this.action = 'idle';
        this.frame = 0;
        this.loopFrom = 0;
        this.playing = false;
        this.loop = false;
        this.timer = null;
        this.onComplete = null;
        this.locked = false;
        this.facing = 1;
    }

    SpritePlayer.prototype.applyFacing = function () {
        this.sprite.style.transform = this.facing < 0 ? 'scaleX(-1)' : 'none';
    };

    SpritePlayer.prototype.setFacing = function (dir) {
        if (!dir) return;
        this.facing = dir < 0 ? -1 : 1;
        this.applyFacing();
    };

    SpritePlayer.prototype.setFrame = function (gridIndex) {
        var spec = this.spec;
        var col = gridIndex % spec.cols;
        var row = Math.floor(gridIndex / spec.cols);
        this.sprite.style.backgroundImage = 'url("' + spec.url + '")';
        this.sprite.style.backgroundSize = (spec.cols * PET_W) + 'px ' + (spec.rows * PET_H) + 'px';
        this.sprite.style.backgroundPosition = (-col * PET_W) + 'px ' + (-row * PET_H) + 'px';
        this.applyFacing();
    };

    SpritePlayer.prototype.stop = function () {
        this.playing = false;
        if (this.timer) { clearTimeout(this.timer); this.timer = null; }
    };

    SpritePlayer.prototype.play = function (action, opts) {
        var self = this;
        opts = opts || {};
        var spec = ACTIONS[action];
        if (!spec) return Promise.resolve(false);
        if (self.locked && !opts.force) return Promise.resolve(false);
        self.stop();
        self.spec = spec;
        self.action = action;
        self.loop = opts.loop != null ? !!opts.loop : !!spec.loop;
        var loopFrom = opts.loopFrom != null ? opts.loopFrom : (spec.loopFrom != null ? spec.loopFrom : 0);
        if (loopFrom < 0 || loopFrom >= spec.frames) loopFrom = 0;
        self.loopFrom = loopFrom;
        var start = opts.from != null ? opts.from : (spec.from != null ? spec.from : 0);
        if (start < 0) start = 0;
        if (start >= spec.frames) start = loopFrom;
        self.frame = start;
        self.playing = true;
        self.locked = !!opts.lock;
        self._painted = true;
        var interval = 1000 / (opts.fps || spec.fps || 12);
        return new Promise(function (resolve) {
            self.onComplete = function () {
                self.locked = false;
                if (opts.onComplete) opts.onComplete();
                resolve(true);
            };
            function tick() {
                if (!self.playing) return;
                self.setFrame(self.frame);
                self.frame += 1;
                if (self.frame >= spec.frames) {
                    if (self.loop) {
                        self.frame = self.loopFrom;
                        self.timer = setTimeout(tick, interval);
                    } else {
                        self.playing = false;
                        self.locked = false;
                        if (self.onComplete) self.onComplete();
                    }
                    return;
                }
                self.timer = setTimeout(tick, interval);
            }
            tick();
        });
    };

    function Pet(cfg) {
        this.cfg = Object.assign({}, CFG, cfg || {});
        this.name = this.cfg.name || '空空';
        this.root = null;
        this.wrap = null;
        this.petEl = null;
        this.hintEl = null;
        this.player = null;
        this.pos = { left: 0, top: 0 };
        this.baseTop = 0;
        this.facing = 1;
        this.keys = { left: false, right: false, jump: false };
        this.vx = 0;
        this.vy = 0;
        this.onGround = true;
        this.jumping = false;
        this.moveRaf = null;
        this.lastTs = 0;
        this.wandering = false;
        this.wanderTarget = null;
        this.wanderRaf = null;
        this.hintTimer = null;
        this.destroyed = false;
        this._keyHandler = null;
        this._keyUpHandler = null;
        this._blurHandler = null;
    }

    Pet.prototype.say = function (text, ms) {
        var self = this;
        if (!self.hintEl) return;
        self.hintEl.textContent = text;
        self.hintEl.classList.add('show');
        if (self.hintTimer) clearTimeout(self.hintTimer);
        self.hintTimer = setTimeout(function () {
            self.hintEl.classList.remove('show');
        }, ms || 2200);
    };

    Pet.prototype.layoutAt = function (left, top) {
        this.pos.left = left;
        this.pos.top = top;
        if (!this.jumping) this.baseTop = top;
        if (this.wrap) {
            this.wrap.style.left = left + 'px';
            this.wrap.style.top = top + 'px';
        }
    };

    Pet.prototype.bounds = function () {
        var pad = 8;
        return {
            minL: pad,
            maxL: Math.max(pad, window.innerWidth - PET_W - pad),
            minT: pad,
            maxT: Math.max(pad, window.innerHeight - PET_H - pad),
        };
    };

    Pet.prototype.setFacing = function (dir) {
        if (!dir) return;
        this.facing = dir < 0 ? -1 : 1;
        if (this.player) this.player.setFacing(this.facing);
    };

    Pet.prototype.playIdle = function (force) {
        if (this.jumping && !force) return;
        if (!this.player) return;
        // 首帧必须强制 play：构造时 action 已是 idle，否则永远不 setFrame
        if (force || this.player.action !== 'idle' || !this.player._painted) {
            this.player.play('idle', { loop: true, force: true });
            this.player._painted = true;
        }
    };

    Pet.prototype.playRun = function () {
        if (this.jumping) return;
        if (!this.player) return;
        if (this.player.action !== 'run') {
            this.player.play('run', { loop: true, from: 1, loopFrom: 1, force: true });
            this.player._painted = true;
        }
    };

    Pet.prototype.jump = function () {
        var self = this;
        if (self.jumping || !self.onGround || self.destroyed) return;
        self.jumping = true;
        self.onGround = false;
        self.vy = -520;
        self.baseTop = self.pos.top;
        self.player.play('jump', {
            loop: false,
            lock: true,
            force: true,
            onComplete: function () {
                if (!self.jumping) self.syncMoveAnim();
            },
        });
    };

    Pet.prototype.syncMoveAnim = function () {
        if (this.jumping) return;
        if (Math.abs(this.vx) > 8 || this.keys.left || this.keys.right || this.wandering) {
            this.playRun();
        } else {
            this.playIdle();
        }
    };

    Pet.prototype.startMoveLoop = function () {
        var self = this;
        if (self.moveRaf) return;
        self.lastTs = 0;
        function step(ts) {
            if (self.destroyed) return;
            self.moveRaf = requestAnimationFrame(step);
            if (!self.lastTs) { self.lastTs = ts; return; }
            var dt = Math.min(0.05, (ts - self.lastTs) / 1000);
            self.lastTs = ts;
            self.tickMove(dt);
        }
        self.moveRaf = requestAnimationFrame(step);
    };

    Pet.prototype.tickMove = function (dt) {
        var self = this;
        var speed = 220;
        var want = 0;
        if (self.keys.left) want -= 1;
        if (self.keys.right) want += 1;

        if (want !== 0) {
            self.stopWander(false);
            self.vx = want * speed;
            self.setFacing(want);
        } else if (!self.wandering) {
            self.vx *= Math.pow(0.001, dt);
            if (Math.abs(self.vx) < 4) self.vx = 0;
        }

        var b = self.bounds();
        var nextL = clamp(self.pos.left + self.vx * dt, b.minL, b.maxL);
        if (nextL === b.minL || nextL === b.maxL) self.vx = 0;

        var nextT = self.pos.top;
        if (self.jumping) {
            self.vy += 1400 * dt;
            nextT = self.pos.top + self.vy * dt;
            if (nextT >= self.baseTop) {
                nextT = self.baseTop;
                self.vy = 0;
                self.jumping = false;
                self.onGround = true;
                if (self.player && self.player.action === 'jump') {
                    self.player.locked = false;
                }
            }
        } else {
            nextT = clamp(self.baseTop, b.minT, b.maxT);
        }

        self.layoutAt(nextL, nextT);
        self.syncMoveAnim();
        self.scheduleSave();
    };

    Pet.prototype.bindKeys = function () {
        var self = this;
        function setKey(code, down) {
            if (code === 'ArrowLeft' || code === 'KeyA') self.keys.left = down;
            if (code === 'ArrowRight' || code === 'KeyD') self.keys.right = down;
            if (code === 'ArrowUp' || code === 'KeyW' || code === 'Space') {
                if (down && !self.keys.jump) self.jump();
                self.keys.jump = down;
            }
        }
        self._keyHandler = function (e) {
            if (e.repeat) return;
            var tag = (e.target && e.target.tagName) || '';
            if (tag === 'INPUT' || tag === 'TEXTAREA' || (e.target && e.target.isContentEditable)) return;
            if (['ArrowLeft', 'ArrowRight', 'ArrowUp', 'KeyA', 'KeyD', 'KeyW', 'Space'].indexOf(e.code) >= 0) {
                e.preventDefault();
                setKey(e.code, true);
            }
        };
        self._keyUpHandler = function (e) {
            setKey(e.code, false);
        };
        self._blurHandler = function () {
            self.keys.left = self.keys.right = self.keys.jump = false;
        };
        window.addEventListener('keydown', self._keyHandler, { passive: false });
        window.addEventListener('keyup', self._keyUpHandler);
        window.addEventListener('blur', self._blurHandler);
    };

    Pet.prototype.bindDrag = function () {
        var self = this;
        var dragging = false;
        var ox = 0;
        var oy = 0;

        function onDown(ev) {
            if (ev.button != null && ev.button !== 0) return;
            dragging = true;
            self.stopWander(true);
            self.petEl.classList.add('dragging');
            var pt = ev.touches ? ev.touches[0] : ev;
            ox = pt.clientX - self.pos.left;
            oy = pt.clientY - self.pos.top;
            ev.preventDefault();
        }
        function onMove(ev) {
            if (!dragging) return;
            var pt = ev.touches ? ev.touches[0] : ev;
            var b = self.bounds();
            var left = clamp(pt.clientX - ox, b.minL, b.maxL);
            var top = clamp(pt.clientY - oy, b.minT, b.maxT);
            self.layoutAt(left, top);
            self.baseTop = top;
            ev.preventDefault();
        }
        function onUp() {
            if (!dragging) return;
            dragging = false;
            self.petEl.classList.remove('dragging');
            self.scheduleSave();
            if (self.cfg.autoWander !== false) self.scheduleWander();
        }

        self.petEl.addEventListener('mousedown', onDown);
        self.petEl.addEventListener('touchstart', onDown, { passive: false });
        window.addEventListener('mousemove', onMove, { passive: false });
        window.addEventListener('touchmove', onMove, { passive: false });
        window.addEventListener('mouseup', onUp);
        window.addEventListener('touchend', onUp);
    };

    Pet.prototype.stopWander = function (cancelAnim) {
        this.wandering = false;
        this.wanderTarget = null;
        if (this.wanderRaf) {
            cancelAnimationFrame(this.wanderRaf);
            this.wanderRaf = null;
        }
        if (cancelAnim) this.vx = 0;
    };

    Pet.prototype.wanderTo = function (left, top) {
        var self = this;
        self.stopWander(false);
        var b = self.bounds();
        self.wanderTarget = {
            left: clamp(left, b.minL, b.maxL),
            top: clamp(top, b.minT, b.maxT),
        };
        self.wandering = true;
        var start = { left: self.pos.left, top: self.pos.top };
        var dx = self.wanderTarget.left - start.left;
        if (Math.abs(dx) > 2) self.setFacing(dx);
        var dist = Math.hypot(dx, self.wanderTarget.top - start.top);
        var dur = clamp(dist / 180, 0.6, 2.8);
        var t0 = performance.now();

        function step(now) {
            if (!self.wandering || self.destroyed) return;
            var t = clamp((now - t0) / (dur * 1000), 0, 1);
            var ease = t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2;
            var L = lerp(start.left, self.wanderTarget.left, ease);
            var T = lerp(start.top, self.wanderTarget.top, ease);
            self.vx = (L - self.pos.left) * 40;
            self.layoutAt(L, self.jumping ? self.pos.top : T);
            if (!self.jumping) self.baseTop = T;
            self.syncMoveAnim();
            if (t < 1) {
                self.wanderRaf = requestAnimationFrame(step);
            } else {
                self.wandering = false;
                self.vx = 0;
                self.syncMoveAnim();
                self.scheduleSave();
                self.scheduleWander();
            }
        }
        self.wanderRaf = requestAnimationFrame(step);
    };

    Pet.prototype.scheduleWander = function () {
        var self = this;
        if (self.cfg.autoWander === false) return;
        if (self._wanderTimer) clearTimeout(self._wanderTimer);
        self._wanderTimer = setTimeout(function () {
            if (self.destroyed || self.keys.left || self.keys.right || self.jumping) {
                self.scheduleWander();
                return;
            }
            var b = self.bounds();
            var left = b.minL + Math.random() * (b.maxL - b.minL);
            var top = clamp(self.baseTop + (Math.random() - 0.5) * 40, b.minT, b.maxT);
            self.wanderTo(left, top);
        }, 4000 + Math.random() * 5000);
    };

    Pet.prototype.scheduleSave = function () {
        var self = this;
        if (self.cfg.persist === false) return;
        if (self._saveTimer) clearTimeout(self._saveTimer);
        self._saveTimer = setTimeout(function () {
            try {
                localStorage.setItem(STORAGE_KEY, JSON.stringify({
                    left: self.pos.left,
                    top: self.baseTop,
                    facing: self.facing,
                }));
            } catch (_) { /* ignore */ }
        }, 400);
    };

    Pet.prototype.restore = function () {
        if (this.cfg.persist === false) return null;
        try {
            var raw = localStorage.getItem(STORAGE_KEY);
            return raw ? JSON.parse(raw) : null;
        } catch (_) {
            return null;
        }
    };

    Pet.prototype.mount = function () {
        var self = this;
        if (self.root) return self;
        injectCSS();

        var existing = document.getElementById(ROOT_ID);
        if (existing) existing.remove();

        self.root = el('div', { id: ROOT_ID });
        var stage = el('div', { className: 'kk-stage' });
        self.wrap = el('div', { className: 'kk-wrap' });
        self.hintEl = el('div', { className: 'kk-hint', text: '' });
        self.petEl = el('div', { className: 'kk-pet' }, [
            el('div', { className: 'kk-sprite' }),
        ]);
        self.wrap.appendChild(self.hintEl);
        self.wrap.appendChild(self.petEl);
        stage.appendChild(self.wrap);
        self.root.appendChild(stage);
        document.body.appendChild(self.root);

        self.player = new SpritePlayer(self.petEl);
        self.player.setFacing(self.facing);

        var b = self.bounds();
        var saved = self.restore();
        var left = saved && saved.left != null ? saved.left : (self.cfg.position === 'left' ? 24 : b.maxL - 24);
        var top = saved && saved.top != null ? saved.top : b.maxT - 24;
        if (saved && saved.facing) self.setFacing(saved.facing);
        self.layoutAt(clamp(left, b.minL, b.maxL), clamp(top, b.minT, b.maxT));
        self.baseTop = self.pos.top;

        self.bindDrag();
        self.bindKeys();
        self.startMoveLoop();

        // 立刻绘制 idle 第 0 帧并显示，避免等 CDN 预加载期间空白
        self.playIdle(true);
        self.petEl.classList.add('ready');

        Promise.all([
            preloadImage(ACTIONS.idle.url),
            preloadImage(ACTIONS.jump.url),
        ]).then(function () {
            // 预加载完成后再刷一帧，确保解码后的位图可见
            if (self.player) self.player.setFrame(self.player.frame || 0);
            self.say(self.name + '来啦 · ←→跑步 Space跳', 2800);
            if (self.cfg.autoWander !== false) self.scheduleWander();
        });

        window.addEventListener('resize', function () {
            var bb = self.bounds();
            self.layoutAt(clamp(self.pos.left, bb.minL, bb.maxL), clamp(self.baseTop, bb.minT, bb.maxT));
            self.baseTop = self.pos.top;
        });

        return self;
    };

    Pet.prototype.destroy = function () {
        this.destroyed = true;
        this.stopWander(true);
        if (this.moveRaf) cancelAnimationFrame(this.moveRaf);
        if (this.player) this.player.stop();
        if (this._keyHandler) window.removeEventListener('keydown', this._keyHandler);
        if (this._keyUpHandler) window.removeEventListener('keyup', this._keyUpHandler);
        if (this._blurHandler) window.removeEventListener('blur', this._blurHandler);
        if (this.root && this.root.parentNode) this.root.parentNode.removeChild(this.root);
        this.root = null;
    };

    Pet.prototype.play = function (action, opts) {
        if (action === 'run') { this.playRun(); return Promise.resolve(true); }
        if (action === 'idle') { this.playIdle(); return Promise.resolve(true); }
        if (action === 'jump') { this.jump(); return Promise.resolve(true); }
        return this.player ? this.player.play(action, opts) : Promise.resolve(false);
    };

    Pet.prototype.getState = function () {
        return {
            name: this.name,
            action: this.player ? this.player.action : 'idle',
            facing: this.facing,
            left: this.pos.left,
            top: this.pos.top,
            jumping: this.jumping,
        };
    };

    var api = {
        _pet: null,
        mount: function (cfg) {
            if (api._pet) api._pet.destroy();
            api._pet = new Pet(cfg).mount();
            return api._pet;
        },
        destroy: function () {
            if (api._pet) api._pet.destroy();
            api._pet = null;
        },
        play: function (a, o) { return api._pet ? api._pet.play(a, o) : Promise.resolve(false); },
        wanderTo: function (l, t) { if (api._pet) api._pet.wanderTo(l, t); },
        getState: function () { return api._pet ? api._pet.getState() : null; },
        say: function (t, ms) { if (api._pet) api._pet.say(t, ms); },
    };

    window.KongkongPet = api;
    window.LingEchoWidget = window.LingEchoWidget || api;

    if (CFG.autoMount !== false) {
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', function () { api.mount(); });
        } else {
            api.mount();
        }
    }
})();
