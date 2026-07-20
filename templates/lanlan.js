/**
 * SoulNexus — 懒懒（Lanlan）熊猫桌宠
 *
 * - 22 CDN 帧序动作 + 情理化状态机
 * - 自主游荡 / 打盹 / 觅食反应
 * - 敲代码监听：coding / bug / bug_sleep / bug_fixed
 * - LLM 日常对话（气泡，无聊天面板）via dialog/v1 text API
 * - 语音对讲（WebRTC）快捷键唤起 / 结束
 *
 * Usage:
 *   <script>
 *     window.__LanlanConfig = {
 *       apiBase: 'https://your-host/api',
 *       apiKey: 'soulnexus_...',
 *       assistantId: '123',
 *       size: 160,
 *       position: 'right',
 *       name: '懒懒',
 *       autoMount: true,
 *       persist: true,
 *       autoWander: true,
 *       autoChat: true,
 *       watchCoding: true,
 *       voiceHotkey: 'Alt+Shift+V',
 *       talkHotkey: 'Alt+Shift+T'
 *     };
 *   </script>
 *   <script src="/lingecho/lanlan.js" async></script>
 *
 * API: LanlanPet.mount / destroy / say / ask / talk / feed / pet / poke
 *      LanlanPet.toggleVoice / startVoice / stopVoice / play / wanderTo / tidyDesktop / getState
 *      play('coding'|'bug'|'bug_sleep'|'bug_fixed') 敲代码相关动作
 */
(function () {
    'use strict';

    var CFG = Object.assign({}, window.__LingEchoConfig || {}, window.__LanlanConfig || {});
    var ROOT_ID = 'lingecho-embed-root';
    var CSS_ID = 'lingecho-embed-css';
    var CDN = 'https://cdn.lingecho.com';
    var STORAGE_KEY = 'lanlan-pet-v1';
    var PET_SIZE = Math.max(96, Math.min(256, Number(CFG.size) || 160));

    /** 帧序图：多数 8×8/61（2048²）；blink·cookie·yawn·coding·bug·bug_fixed 为 8×7/51（2048×1792）；drowsy 63 */
    var ACTIONS = {
        blink: { url: CDN + '/blink.png', cols: 8, rows: 7, frames: 51, fps: 9, loop: true, kind: 'idle' },
        sway: { url: CDN + '/sway.png', cols: 8, rows: 8, frames: 61, fps: 10, loop: true, kind: 'idle' },
        zen: { url: CDN + '/zen.png', cols: 8, rows: 8, frames: 61, fps: 7, loop: true, kind: 'calm' },
        sleep: { url: CDN + '/sleep.png', cols: 8, rows: 8, frames: 61, fps: 5, loop: true, kind: 'sleep' },
        drowsy: { url: CDN + '/drowsy.png', cols: 8, rows: 8, frames: 63, fps: 8, loop: false, kind: 'sleep' },
        yawn: { url: CDN + '/yawn.png', cols: 8, rows: 7, frames: 51, fps: 8, loop: false, kind: 'sleep' },
        bored: { url: CDN + '/bored.png', cols: 8, rows: 8, frames: 61, fps: 9, loop: false, kind: 'idle' },
        shy: { url: CDN + '/shy.png', cols: 8, rows: 8, frames: 61, fps: 11, loop: false, kind: 'react' },
        giggle: { url: CDN + '/giggle.png', cols: 8, rows: 8, frames: 61, fps: 13, loop: false, kind: 'react' },
        celebrate: { url: CDN + '/celebrate.png', cols: 8, rows: 8, frames: 61, fps: 13, loop: false, kind: 'happy' },
        cookie: { url: CDN + '/cookie.png', cols: 8, rows: 7, frames: 51, fps: 11, loop: false, kind: 'eat' },
        bamboo: { url: CDN + '/bamboo.png', cols: 8, rows: 8, frames: 61, fps: 11, loop: false, kind: 'eat' },
        money: { url: CDN + '/money.png', cols: 8, rows: 8, frames: 61, fps: 11, loop: false, kind: 'happy' },
        ignore: { url: CDN + '/ignore.png', cols: 8, rows: 8, frames: 61, fps: 9, loop: false, kind: 'mood' },
        sulk: { url: CDN + '/sulk.png', cols: 8, rows: 8, frames: 61, fps: 9, loop: false, kind: 'mood' },
        cry: { url: CDN + '/cry.png', cols: 8, rows: 8, frames: 61, fps: 9, loop: false, kind: 'sad' },
        roll_cry: { url: CDN + '/roll_cry.png', cols: 8, rows: 8, frames: 61, fps: 11, loop: false, kind: 'sad' },
        chaos: { url: CDN + '/chaos.png', cols: 8, rows: 8, frames: 61, fps: 13, loop: false, kind: 'special' },
        coding: { url: CDN + '/coding_1.png', cols: 8, rows: 8, frames: 60, fps: 12, loop: true, loopFrom: 56, kind: 'code' },
        bug: { url: CDN + '/bug.png', cols: 8, rows: 7, frames: 51, fps: 11, loop: false, kind: 'code' },
        bug_sleep: { url: CDN + '/bug_sleep.png', cols: 8, rows: 8, frames: 61, fps: 6, loop: true, kind: 'code' },
        bug_fixed: { url: CDN + '/bug_fixed.png', cols: 8, rows: 7, frames: 51, fps: 11, loop: false, kind: 'code' },
    };

    var ACTION_LABELS = {
        blink: '眨眼', sway: '晃悠', zen: '发呆', sleep: '睡觉', drowsy: '犯困', yawn: '打哈欠',
        bored: '无聊', shy: '害羞', giggle: '偷笑', celebrate: '庆祝', cookie: '吃饼干',
        bamboo: '啃竹子', money: '发财', ignore: '装没看见', sulk: '生闷气', cry: '哭哭',
        roll_cry: '滚地大哭', chaos: '整活',
        coding: '敲代码', bug: '改出 bug', bug_sleep: '敲睡着了', bug_fixed: 'bug 修好了',
    };

    /** 敲代码监听阈值（可被 __LanlanConfig 覆盖） */
    var CODING_DEFAULTS = {
        pauseMs: 7000,       // 停敲多久算「停了」
        longMs: 150000,      // 连续敲多久 → 睡着
        fixedMinMs: 18000,   // 至少敲多久才播「修好了」
        fixedMinKeys: 10,    // 至少敲多少键才播「修好了」
        cooldownMs: 90000,   // 修好后多久内再敲 → 懊恼
    };

    var LINES = {
        greet: ['嗯……你来啦', '懒懒在，别催', '摸头可以，催醒不行'],
        pet: ['嘿嘿……再来一下', '舒服', '你手挺暖的'],
        poke: ['干嘛戳我', '……懒得理你', '哼'],
        pokeMad: ['再戳真哭了啊', '讨厌……', '呜'],
        feedBamboo: ['竹子！活过来了', '嘎吱嘎吱……幸福'],
        feedCookie: ['饼干！偏爱我了吧', '甜的……可以再来'],
        full: ['吃不下了……先躺会儿', '饱了饱了'],
        sleep: ['zzzz……别吵', '梦里有竹子'],
        wake: ['再五分钟……好吧', '醒了醒了'],
        bored: ['好闲……陪我玩？', '有没有竹子外卖'],
        celebrate: ['耶——！', '今天运气不错'],
        sad: ['呜……心情掉到地板了', '需要竹子急救'],
        money: ['发财！躺平基金 +1'],
        zen: ['空……', '不急，慢慢来'],
        chaos: ['？？？世界线错乱中'],
        ignore: ['（看天）', '没听见'],
        menu: ['想干嘛？', '点我干什么'],
        listen: ['我在听……', '说吧，别太长'],
        listenOff: ['好，先这样', '语音关了，摸头也行'],
        thinking: ['……想一想', '脑子转得很慢'],
        noLlm: ['没接上脑子（检查 apiBase / assistantId）', '现在只会卖萌，接上 LLM 才能聊天'],
        wander: ['换个地方躺', '这边风水一般'],
        coding: ['敲！敲！认真写呢', '代码进竹子里了……', '别催，在改'],
        codingSleep: ['敲太久……先眯一下', 'zzz 键盘还开着', '梦里全是红字'],
        codingFixed: ['改好了？可以躺了', 'bug 退散！', '收工收工'],
        codingBug: ['怎么又炸了……', '明明刚才还好的', '又来？哼'],
        codingWake: ['醒了醒了，继续敲', '红字还在……起来改'],
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

    function pick(arr) { return arr[Math.floor(Math.random() * arr.length)]; }
    function clamp(n, a, b) { return Math.max(a, Math.min(b, n)); }
    function now() { return Date.now(); }
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

    function safeParseJson(raw) {
        if (!raw || !String(raw).trim()) throw new Error('服务端返回为空');
        try { return JSON.parse(raw); }
        catch (e) { throw new Error('后端返回格式异常'); }
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
                    return safeParseJson(raw);
                });
            })
            .catch(function (err) {
                if (timer) clearTimeout(timer);
                if (err && err.name === 'AbortError') throw new Error('请求超时');
                throw err;
            });
    }

    function inferApiBase(cfg) {
        if (cfg && cfg.apiBase) return String(cfg.apiBase).replace(/\/$/, '');
        var scripts = document.getElementsByTagName('script');
        for (var i = scripts.length - 1; i >= 0; i--) {
            var src = scripts[i].src || '';
            var m = src.match(/^(.*)\/lingecho\/(?:lanlan\.js|embed\/v1\/(?:t\/[^/]+\/)?embed\.js)(?:\?|$)/);
            if (m) return m[1].replace(/\/$/, '');
        }
        return '/api';
    }

    function parseHotkey(spec) {
        var parts = String(spec || '').split('+').map(function (s) { return s.trim().toLowerCase(); }).filter(Boolean);
        var need = { alt: false, ctrl: false, shift: false, meta: false, key: '' };
        parts.forEach(function (p) {
            if (p === 'alt' || p === 'option') need.alt = true;
            else if (p === 'ctrl' || p === 'control' || p === 'controlorcommand' || p === 'commandorcontrol') need.ctrl = true;
            else if (p === 'shift') need.shift = true;
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
        } else if (mod) {
            return false;
        }
        return true;
    }

    function loadState() {
        if (CFG.persist === false) return null;
        try {
            var raw = localStorage.getItem(STORAGE_KEY);
            return raw ? JSON.parse(raw) : null;
        } catch (_) { return null; }
    }

    function saveState(pet) {
        if (CFG.persist === false || !pet) return;
        try {
            localStorage.setItem(STORAGE_KEY, JSON.stringify({
                left: pet.pos.left,
                top: pet.pos.top,
                stats: pet.stats,
                sleeping: pet.sleeping,
                savedAt: now(),
            }));
        } catch (_) {}
    }

    function injectCSS(size) {
        if (document.getElementById(CSS_ID)) return;
        var css = [
            '#' + ROOT_ID + '{position:fixed;z-index:2147483000;pointer-events:none;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Hiragino Sans GB",sans-serif;color:#3f3f46}',
            '#' + ROOT_ID + ' *{box-sizing:border-box}',
            '#' + ROOT_ID + ' .ll-wrap{position:relative;width:' + size + 'px;height:' + size + 'px;pointer-events:auto}',
            '#' + ROOT_ID + ' .ll-pet{width:' + size + 'px;height:' + size + 'px;overflow:hidden;cursor:grab;user-select:none;-webkit-user-select:none;touch-action:none;filter:drop-shadow(0 8px 18px rgba(0,0,0,.14));opacity:0;transform:scale(.92);transition:opacity .28s ease,transform .28s ease}',
            '#' + ROOT_ID + ' .ll-pet.ready{opacity:1;transform:none}',
            '#' + ROOT_ID + ' .ll-pet.dragging{cursor:grabbing;opacity:.94}',
            '#' + ROOT_ID + ' .ll-pet.listening{filter:drop-shadow(0 0 0 rgba(0,0,0,0)) drop-shadow(0 8px 18px rgba(0,0,0,.14))}',
            '#' + ROOT_ID + ' .ll-sprite{display:block;width:100%;height:100%;background-repeat:no-repeat;image-rendering:auto}',
            '#' + ROOT_ID + ' .ll-ring{position:absolute;inset:-10px;border-radius:50%;border:2px solid rgba(34,197,94,.55);opacity:0;pointer-events:none;transition:opacity .2s ease}',
            '#' + ROOT_ID + ' .ll-wrap.listening .ll-ring{opacity:1;animation:llPulse 1.4s ease-in-out infinite}',
            '@keyframes llPulse{0%,100%{transform:scale(.92);opacity:.35}50%{transform:scale(1.05);opacity:.9}}',
            '#' + ROOT_ID + ' .ll-hint{position:absolute;left:50%;bottom:calc(100% - 8px);transform:translateX(-50%);z-index:3;max-width:240px;min-width:88px;padding:8px 12px;border-radius:14px;background:rgba(255,255,255,.96);border:1px solid rgba(0,0,0,.08);box-shadow:0 6px 22px rgba(0,0,0,.12);font-size:12px;line-height:1.45;color:#3f3f46;text-align:center;white-space:normal;word-break:break-word;pointer-events:none;opacity:0;visibility:hidden;transition:opacity .2s ease,transform .2s ease,visibility .2s ease}',
            '#' + ROOT_ID + ' .ll-hint.show{opacity:1;visibility:visible;animation:llPop .28s ease both}',
            '#' + ROOT_ID + ' .ll-hint::after{content:"";position:absolute;left:50%;bottom:-5px;width:10px;height:10px;background:rgba(255,255,255,.96);border-right:1px solid rgba(0,0,0,.08);border-bottom:1px solid rgba(0,0,0,.08);transform:translateX(-50%) rotate(45deg)}',
            '@keyframes llPop{from{opacity:0;transform:translateX(-50%) translateY(6px) scale(.96)}to{opacity:1;transform:translateX(-50%) translateY(0) scale(1)}}',
            '#' + ROOT_ID + ' .ll-talk{position:absolute;left:50%;bottom:calc(100% + 8px);transform:translateX(-50%);z-index:5;display:none;width:min(260px,70vw);padding:8px;border-radius:14px;background:rgba(255,255,255,.98);border:1px solid rgba(0,0,0,.08);box-shadow:0 10px 28px rgba(0,0,0,.14);pointer-events:auto;gap:6px}',
            '#' + ROOT_ID + ' .ll-talk.open{display:flex;flex-direction:column;animation:llPop .2s ease both}',
            '#' + ROOT_ID + ' .ll-talk input{width:100%;border:0;border-bottom:1px solid #e4e4e7;outline:none;font-size:13px;padding:8px 2px;background:transparent;color:#18181b}',
            '#' + ROOT_ID + ' .ll-talk-actions{display:flex;justify-content:flex-end;gap:6px}',
            '#' + ROOT_ID + ' .ll-talk button{border:0;border-radius:8px;padding:6px 10px;font-size:12px;cursor:pointer;background:#f4f4f5;color:#27272a}',
            '#' + ROOT_ID + ' .ll-talk button.primary{background:#27272a;color:#fff}',
            '#' + ROOT_ID + ' .ll-menu{position:absolute;left:50%;bottom:calc(100% + 6px);transform:translateX(-50%);z-index:4;min-width:148px;max-width:168px;padding:4px;border-radius:12px;background:rgba(255,255,255,.98);border:1px solid rgba(0,0,0,.08);box-shadow:0 8px 24px rgba(0,0,0,.12);display:none;pointer-events:auto}',
            '#' + ROOT_ID + ' .ll-menu.open{display:block;animation:llPop .2s ease both}',
            '#' + ROOT_ID + ' .ll-menu-hd{display:flex;align-items:center;justify-content:space-between;gap:6px;padding:5px 8px 6px}',
            '#' + ROOT_ID + ' .ll-menu-name{font-size:12px;font-weight:500;color:#27272a;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}',
            '#' + ROOT_ID + ' .ll-menu-stat-btn{border:0;background:#f4f4f5;color:#71717a;border-radius:6px;font-size:10px;line-height:1;padding:3px 6px;cursor:pointer;flex-shrink:0}',
            '#' + ROOT_ID + ' .ll-menu-stat-btn.on{background:#eef2ff;color:#4f46e5}',
            '#' + ROOT_ID + ' .ll-menu button{display:block;width:100%;text-align:left;border:0;background:transparent;padding:7px 8px;border-radius:8px;font-size:12px;color:#27272a;cursor:pointer}',
            '#' + ROOT_ID + ' .ll-menu button:hover{background:#f4f4f5}',
            '#' + ROOT_ID + ' .ll-menu button.sub{padding-left:14px;font-size:11px;color:#52525b}',
            '#' + ROOT_ID + ' .ll-menu-group{margin:1px 0}',
            '#' + ROOT_ID + ' .ll-menu-group-hd{display:flex;align-items:center;justify-content:space-between;padding:7px 8px;border-radius:8px;font-size:12px;color:#3f3f46;cursor:pointer;user-select:none}',
            '#' + ROOT_ID + ' .ll-menu-group-hd:hover{background:#f4f4f5}',
            '#' + ROOT_ID + ' .ll-menu-group-hd span{flex:1}',
            '#' + ROOT_ID + ' .ll-menu-group-hd i{font-style:normal;font-size:10px;color:#a1a1aa;transition:transform .15s ease}',
            '#' + ROOT_ID + ' .ll-menu-group.open .ll-menu-group-hd i{transform:rotate(90deg)}',
            '#' + ROOT_ID + ' .ll-menu-group-body{display:none;padding:0 0 2px}',
            '#' + ROOT_ID + ' .ll-menu-group.open .ll-menu-group-body{display:block}',
            '#' + ROOT_ID + ' .ll-stats{position:absolute;right:-2px;top:-2px;z-index:2;display:flex;gap:2px;padding:3px 4px;border-radius:8px;background:rgba(255,255,255,.9);border:1px solid rgba(0,0,0,.06);box-shadow:0 2px 8px rgba(0,0,0,.08);pointer-events:none;opacity:0;visibility:hidden;transition:opacity .18s ease,visibility .18s ease}',
            '#' + ROOT_ID + ' .ll-stats.show{opacity:1;visibility:visible}',
            '#' + ROOT_ID + ' .ll-stat{width:3px;height:10px;border-radius:99px;background:#e4e4e7;overflow:hidden;position:relative}',
            '#' + ROOT_ID + ' .ll-stat b{position:absolute;left:0;right:0;bottom:0;height:0%;border-radius:99px;transition:height .3s ease}',
            '#' + ROOT_ID + ' .ll-stat.mood b{background:#f59e0b}',
            '#' + ROOT_ID + ' .ll-stat.energy b{background:#22c55e}',
            '#' + ROOT_ID + ' .ll-stat.hunger b{background:#fb7185}',
            '#' + ROOT_ID + ' .ll-stat.love b{background:#60a5fa}',
        ].join('');
        document.head.appendChild(el('style', { id: CSS_ID, text: css }));
    }

    function SpritePlayer(viewport) {
        this.viewport = viewport;
        this.sprite = viewport.querySelector('.ll-sprite') || viewport;
        this.spec = ACTIONS.blink;
        this.action = 'blink';
        this.frame = 0;
        this.loopFrom = 0;
        this.playing = false;
        this.loop = false;
        this.timer = null;
        this.onComplete = null;
        this.locked = false;
    }

    SpritePlayer.prototype.setFrame = function (gridIndex) {
        var spec = this.spec;
        var col = gridIndex % spec.cols;
        var row = Math.floor(gridIndex / spec.cols);
        this.sprite.style.backgroundImage = 'url("' + spec.url + '")';
        this.sprite.style.backgroundSize = spec.cols * PET_SIZE + 'px ' + spec.rows * PET_SIZE + 'px';
        this.sprite.style.backgroundPosition = -col * PET_SIZE + 'px ' + -row * PET_SIZE + 'px';
    };

    SpritePlayer.prototype.stop = function () {
        this.playing = false;
        if (this.timer) { clearTimeout(this.timer); this.timer = null; }
    };

    /**
     * @param {string} action
     * @param {object} [opts]
     * @param {boolean} [opts.loop]
     * @param {boolean} [opts.force]
     * @param {boolean} [opts.lock]
     * @param {number} [opts.fps]
     * @param {number} [opts.from]      起始帧（默认 0；跳过过渡可传 loopFrom）
     * @param {number} [opts.loopFrom]  循环回跳帧（默认取 ACTIONS[action].loopFrom 或 0）
     * @param {function} [opts.onComplete]
     */
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
        var start = opts.from != null ? opts.from : 0;
        if (start < 0) start = 0;
        if (start >= spec.frames) start = loopFrom;
        self.frame = start;
        self.playing = true;
        self.locked = !!opts.lock;
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

    /** 从回复里抽出 [action:xxx]，并做关键词兜底 */
    function parseReplyEmotion(text) {
        var raw = String(text || '');
        var action = '';
        var m = raw.match(/\[\s*action\s*:\s*([a-z_]+)\s*\]/i);
        if (m && ACTIONS[m[1].toLowerCase()]) {
            action = m[1].toLowerCase();
            raw = raw.replace(m[0], '');
        }
        raw = raw.replace(/\s*\[action:[a-z_]+\]\s*/gi, '').trim();
        if (!action) {
            if (/哭|难过|委屈|呜/.test(raw)) action = 'cry';
            else if (/生气|哼|不理/.test(raw)) action = 'sulk';
            else if (/哈欠|困|睡/.test(raw)) action = /睡/.test(raw) ? 'drowsy' : 'yawn';
            else if (/哈哈|嘿嘿|笑|开心/.test(raw)) action = 'giggle';
            else if (/庆祝|耶|太好了/.test(raw)) action = 'celebrate';
            else if (/竹子|饿|吃/.test(raw)) action = 'bamboo';
            else if (/钱|发财/.test(raw)) action = 'money';
            else if (/乱|疯|整活/.test(raw)) action = 'chaos';
            else if (/害羞|不好意思/.test(raw)) action = 'shy';
            else if (/冥想|静|空/.test(raw)) action = 'zen';
            else if (/没听见|装死|无视/.test(raw)) action = 'ignore';
            else if (/无聊/.test(raw)) action = 'bored';
            else if (/修好|改好|搞定|收工/.test(raw)) action = 'bug_fixed';
            else if (/bug|炸了|报错|红字/.test(raw)) action = 'bug';
            else if (/敲代码|写代码|coding/.test(raw)) action = 'coding';
        }
        return { text: raw, action: action };
    }

    function Pet(cfg) {
        this.cfg = Object.assign({}, CFG, cfg || {});
        this.name = this.cfg.name || '懒懒';
        this.apiBase = inferApiBase(this.cfg);
        this.root = null;
        this.wrap = null;
        this.petEl = null;
        this.hintEl = null;
        this.menuEl = null;
        this.talkEl = null;
        this.talkInput = null;
        this.statsEl = null;
        this.ringEl = null;
        this.player = null;
        this.pos = { left: 0, top: 0 };
        this.sleeping = false;
        this.busy = false;
        this.menuOpen = false;
        this.talkOpen = false;
        this.statsVisible = false;
        this.listening = false;
        this.wandering = false;
        this.wanderTarget = null;
        this.wanderRaf = null;
        this.hintTimer = null;
        this.aiTimer = null;
        this.decayTimer = null;
        this.chatTimer = null;
        this.saveTimer = null;
        this.clickCount = 0;
        this.clickTimer = null;
        this.pokeStreak = 0;
        this.pokeResetTimer = null;
        this.destroyed = false;
        this._awaitingContextMenu = false;
        this._suppressTapUntil = 0;
        this._menuStatBtn = null;
        this.textSessionId = null;
        this.voiceSessionId = null;
        this.pc = null;
        this.dc = null;
        this.localStream = null;
        this.remoteAudio = null;
        this.voiceHotkey = parseHotkey(this.cfg.voiceHotkey || 'Alt+Shift+V');
        this.talkHotkey = parseHotkey(this.cfg.talkHotkey || 'Alt+Shift+T');
        this.stats = { mood: 72, energy: 68, hunger: 42, love: 60 };
        this._unsubs = [];
        this._lastChatAt = 0;
        // 敲代码监听：idle | coding | sleepy | cooldown
        this.codingMode = 'idle';
        this.codingKeyCount = 0;
        this.codingSessionStart = 0;
        this.codingBurstStart = 0;
        this.codingLastKeyAt = 0;
        this.codingCooldownUntil = 0;
        this.codingTimer = null;
        this.codingOpts = Object.assign({}, CODING_DEFAULTS, {
            pauseMs: Number(this.cfg.codingPauseMs) || CODING_DEFAULTS.pauseMs,
            longMs: Number(this.cfg.codingLongMs) || CODING_DEFAULTS.longMs,
            fixedMinMs: Number(this.cfg.codingFixedMinMs) || CODING_DEFAULTS.fixedMinMs,
            fixedMinKeys: Number(this.cfg.codingFixedMinKeys) || CODING_DEFAULTS.fixedMinKeys,
            cooldownMs: Number(this.cfg.codingCooldownMs) || CODING_DEFAULTS.cooldownMs,
        });
        this._tidyingDesktop = false;
    }

    Pet.prototype.hasLlm = function () {
        var id = String(this.cfg.assistantId || '').trim();
        return !!(this.apiBase && id && id !== 'YOUR_ASSISTANT_ID');
    };

    Pet.prototype.authHeaders = function (forForm) {
        var h = {};
        if (!forForm) h['Content-Type'] = 'application/json';
        if (this.cfg.apiKey) h['X-API-Key'] = this.cfg.apiKey;
        if (this.cfg.token) h['Authorization'] = 'Bearer ' + this.cfg.token;
        return h;
    };

    Pet.prototype.mount = function () {
        var self = this;
        if (document.getElementById(ROOT_ID)) {
            console.warn('[Lanlan] already mounted');
            return self;
        }
        injectCSS(PET_SIZE);
        self.root = el('div', { id: ROOT_ID });
        self.ringEl = el('div', { className: 'll-ring' });
        self.hintEl = el('div', { className: 'll-hint' });
        self.menuEl = self.buildMenu();
        self.talkEl = self.buildTalkBox();
        self.statsEl = self.buildStats();
        self.petEl = el('div', {
            className: 'll-pet le-pet',
            title: self.name + ' · 拖动/点击/右键 · ' + (self.cfg.voiceHotkey || 'Alt+Shift+V') + ' 语音 · ' + (self.cfg.talkHotkey || 'Alt+Shift+T') + ' 说话',
        }, [el('div', { className: 'll-sprite' })]);
        self.wrap = el('div', { className: 'll-wrap le-pet-wrap' }, [
            self.ringEl, self.menuEl, self.talkEl, self.hintEl, self.petEl, self.statsEl,
        ]);
        self.root.appendChild(self.wrap);
        document.body.appendChild(self.root);

        self.player = new SpritePlayer(self.petEl);
        self.restore();
        self.bindInput();
        self.bindHotkeys();
        self.bindCodingWatch();
        self.startSystems();

        var priority = ['blink', 'sway', 'sleep', 'giggle', 'bamboo', 'celebrate', 'coding', 'bug_fixed'];
        var urls = priority.map(function (k) { return ACTIONS[k].url; });
        Object.keys(ACTIONS).forEach(function (k) {
            if (priority.indexOf(k) < 0) urls.push(ACTIONS[k].url);
        });
        Promise.all(urls.slice(0, 6).map(preloadImage)).then(function () {
            if (self.destroyed) return;
            self.petEl.classList.add('ready');
            if (self.sleeping) {
                self.player.play('sleep', { loop: true, force: true });
                self.say(pick(LINES.sleep), 2200);
            } else {
                self.player.play('sway', { loop: true, force: true });
                self.say(pick(LINES.greet), 2600);
                if (self.cfg.autoWander !== false) {
                    setTimeout(function () { self.scheduleWander(true); }, 4000);
                }
            }
            urls.slice(6).forEach(preloadImage);
        });
        return self;
    };

    Pet.prototype.buildTalkBox = function () {
        var self = this;
        var input = el('input', {
            type: 'text',
            placeholder: '跟' + self.name + '说点什么…',
            maxlength: '120',
        });
        self.talkInput = input;
        input.addEventListener('keydown', function (e) {
            if (e.key === 'Enter') {
                e.preventDefault();
                self.submitTalk();
            } else if (e.key === 'Escape') {
                self.closeTalk();
            }
        });
        return el('div', { className: 'll-talk' }, [
            input,
            el('div', { className: 'll-talk-actions' }, [
                el('button', { type: 'button', text: '取消', onClick: function () { self.closeTalk(); } }),
                el('button', {
                    type: 'button',
                    className: 'primary',
                    text: '发送',
                    onClick: function () { self.submitTalk(); },
                }),
            ]),
        ]);
    };

    Pet.prototype.buildMenu = function () {
        var self = this;

        function item(label, run, sub) {
            return el('button', {
                type: 'button',
                className: sub ? 'sub' : '',
                text: label,
                onClick: function (e) {
                    e.stopPropagation();
                    self.closeMenu();
                    run();
                },
            });
        }

        function group(title, children, open) {
            var body = el('div', { className: 'll-menu-group-body' }, children);
            var box = el('div', { className: 'll-menu-group' + (open ? ' open' : '') }, [
                el('div', {
                    className: 'll-menu-group-hd',
                    onClick: function (e) {
                        e.stopPropagation();
                        box.classList.toggle('open');
                    },
                }, [
                    el('span', { text: title }),
                    el('i', { text: '›' }),
                ]),
                body,
            ]);
            return box;
        }

        self._menuStatBtn = el('button', {
            type: 'button',
            className: 'll-menu-stat-btn',
            text: '状态',
            title: '显示/隐藏角标状态',
            onClick: function (e) {
                e.stopPropagation();
                self.toggleStats();
            },
        });

        self._sleepMenuBtn = item(self.sleeping ? '叫醒' : '入睡', function () { self.toggleSleep(); }, true);

        return el('div', { className: 'll-menu' }, [
            el('div', { className: 'll-menu-hd' }, [
                el('div', { className: 'll-menu-name', text: self.name }),
                self._menuStatBtn,
            ]),
            item('说话', function () { self.openTalk(); }),
            item('语音对讲', function () { self.toggleVoice(); }),
            item('摸摸头', function () { self.pet(); }),
            group('喂食', [
                item('竹子', function () { self.feed('bamboo'); }, true),
                item('饼干', function () { self.feed('cookie'); }, true),
            ]),
            group('更多', [self._sleepMenuBtn]),
        ]);
    };

    Pet.prototype.refreshMenuLabels = function () {
        if (this._sleepMenuBtn) {
            this._sleepMenuBtn.textContent = this.sleeping ? '叫醒' : '入睡';
        }
    };

    Pet.prototype.buildStats = function () {
        return el('div', { className: 'll-stats', title: '心情·精力·饿饿·亲密' }, [
            el('div', { className: 'll-stat mood' }, [el('b')]),
            el('div', { className: 'll-stat energy' }, [el('b')]),
            el('div', { className: 'll-stat hunger' }, [el('b')]),
            el('div', { className: 'll-stat love' }, [el('b')]),
        ]);
    };

    Pet.prototype.syncStatsUI = function () {
        if (!this.statsEl) return;
        var map = [['.mood b', this.stats.mood], ['.energy b', this.stats.energy], ['.hunger b', this.stats.hunger], ['.love b', this.stats.love]];
        var self = this;
        map.forEach(function (pair) {
            var node = self.statsEl.querySelector(pair[0]);
            if (node) node.style.height = clamp(pair[1], 0, 100) + '%';
        });
        if (self._menuStatBtn) {
            if (self.statsVisible) self._menuStatBtn.classList.add('on');
            else self._menuStatBtn.classList.remove('on');
        }
    };

    Pet.prototype.restore = function () {
        var saved = loadState();
        var margin = 16;
        var left = this.cfg.position === 'left' ? margin : window.innerWidth - PET_SIZE - margin;
        var top = window.innerHeight - PET_SIZE - margin - 24;
        if (saved) {
            if (typeof saved.left === 'number') left = saved.left;
            if (typeof saved.top === 'number') top = saved.top;
            if (saved.stats) this.stats = Object.assign(this.stats, saved.stats);
            if (saved.sleeping) this.sleeping = true;
            if (saved.savedAt) {
                var hours = (now() - saved.savedAt) / 3600000;
                this.stats.hunger = clamp(this.stats.hunger + hours * 5, 0, 100);
                this.stats.energy = clamp(this.stats.energy - hours * 2.5, 0, 100);
                if (hours > 2) this.stats.mood = clamp(this.stats.mood - hours * 1.5, 10, 100);
            }
        }
        this.layoutAt(left, top);
        this.syncStatsUI();
    };

    Pet.prototype.layoutAt = function (left, top) {
        var margin = 8;
        left = clamp(left, margin, window.innerWidth - PET_SIZE - margin);
        top = clamp(top, margin, window.innerHeight - PET_SIZE - margin);
        this.pos = { left: left, top: top };
        this.root.style.left = left + 'px';
        this.root.style.top = top + 'px';
        this.root.style.right = 'auto';
        this.root.style.bottom = 'auto';
        this.scheduleSave();
    };

    Pet.prototype.scheduleSave = function () {
        var self = this;
        if (self.saveTimer) clearTimeout(self.saveTimer);
        self.saveTimer = setTimeout(function () { saveState(self); }, 400);
    };

    Pet.prototype.say = function (text, ms) {
        var self = this;
        if (!self.hintEl || !text) return self;
        self.hintEl.textContent = text;
        self.hintEl.classList.add('show');
        if (self.hintTimer) clearTimeout(self.hintTimer);
        self.hintTimer = setTimeout(function () { self.hintEl.classList.remove('show'); }, ms || 2600);
        return self;
    };

    Pet.prototype.hideHint = function () {
        if (this.hintEl) this.hintEl.classList.remove('show');
        if (this.hintTimer) clearTimeout(this.hintTimer);
    };

    Pet.prototype.openMenu = function () {
        this.closeTalk();
        this.refreshMenuLabels();
        this.menuOpen = true;
        this._suppressTapUntil = now() + 480;
        if (this.menuEl) this.menuEl.classList.add('open');
        this.pauseWander();
        this.hideHint();
    };

    Pet.prototype.closeMenu = function () {
        this.menuOpen = false;
        if (this.menuEl) this.menuEl.classList.remove('open');
        if (!this.busy && !this.listening && !this.talkOpen) this.scheduleWander();
    };

    Pet.prototype.openTalk = function () {
        var self = this;
        self.closeMenu();
        self.pauseWander();
        if (self.sleeping) self.wake();
        self.talkOpen = true;
        self.talkEl.classList.add('open');
        setTimeout(function () { if (self.talkInput) self.talkInput.focus(); }, 30);
    };

    Pet.prototype.closeTalk = function () {
        this.talkOpen = false;
        if (this.talkEl) this.talkEl.classList.remove('open');
        if (this.talkInput) this.talkInput.value = '';
    };

    Pet.prototype.submitTalk = function () {
        var text = (this.talkInput && this.talkInput.value || '').trim();
        this.closeTalk();
        if (!text) return;
        this.ask(text);
    };

    Pet.prototype.toggleStats = function () {
        this.statsVisible = !this.statsVisible;
        if (this.statsVisible) this.statsEl.classList.add('show');
        else this.statsEl.classList.remove('show');
        this.syncStatsUI();
    };

    Pet.prototype.applyDelta = function (delta) {
        if (!delta) return;
        var s = this.stats;
        ['mood', 'energy', 'hunger', 'love'].forEach(function (k) {
            if (delta[k] != null) s[k] = clamp(s[k] + delta[k], 0, 100);
        });
        this.syncStatsUI();
        this.scheduleSave();
    };

    /** 情理化：默认待机动作 */
    Pet.prototype.pickIdleLoop = function () {
        if (this.sleeping) return 'sleep';
        if (this.stats.energy < 28) return 'blink';
        if (this.stats.mood > 80 && this.stats.love > 65) return Math.random() < 0.35 ? 'sway' : 'blink';
        return Math.random() < 0.6 ? 'sway' : 'blink';
    };

    Pet.prototype.playIdle = function () {
        return this.player.play(this.pickIdleLoop(), { loop: true, force: true });
    };

    Pet.prototype.playThenIdle = function (action, delta) {
        var self = this;
        if (!ACTIONS[action]) return Promise.resolve(false);
        if (self.listening) return Promise.resolve(false);
        if (self.menuOpen) return Promise.resolve(false);
        self.busy = true;
        self.pauseWander();
        if (delta) self.applyDelta(delta);
        return self.player.play(action, { loop: false, lock: true, force: true }).then(function () {
            self.busy = false;
            if (self.destroyed) return;
            if (self.sleeping) return self.player.play('sleep', { loop: true, force: true });
            if (self.codingMode === 'coding') return self.playCodingLoop(true);
            if (self.codingMode === 'sleepy') return self.player.play('bug_sleep', { loop: true, force: true });
            self.playIdle();
            self.scheduleWander();
        });
    };

    Pet.prototype.play = function (action, opts) {
        opts = opts || {};
        if (opts.say) this.say(opts.say);
        if (opts.delta) this.applyDelta(opts.delta);
        if (opts.thenIdle !== false && ACTIONS[action] && !ACTIONS[action].loop) {
            return this.playThenIdle(action);
        }
        return this.player.play(action, {
            loop: opts.loop != null ? opts.loop : !!ACTIONS[action].loop,
            force: true,
            lock: !!opts.lock,
        });
    };

    Pet.prototype.reactToReply = function (replyText) {
        var parsed = parseReplyEmotion(replyText);
        var text = parsed.text || replyText;
        var action = parsed.action;
        this.say(text, Math.min(8000, 1800 + text.length * 60));
        this.applyDelta({ love: 1, mood: 2 });

        if (action === 'sleep' || (action === 'drowsy' && this.stats.energy < 25)) {
            this.sleep();
            return;
        }
        if (action === 'zen') {
            this.enterZen(6000);
            return;
        }
        if (action === 'coding') {
            this.codingMode = 'coding';
            this.codingSessionStart = now();
            this.codingBurstStart = now();
            this.codingLastKeyAt = now();
            this.codingKeyCount = Math.max(this.codingKeyCount, this.codingOpts.fixedMinKeys);
            this.pauseWander();
            this.playCodingLoop(false);
            return;
        }
        if (action === 'bug_sleep') {
            this.codingMode = 'coding';
            this.enterCodingSleepy();
            return;
        }
        if (action && ACTIONS[action]) {
            var delta = {};
            if (action === 'giggle' || action === 'celebrate' || action === 'bug_fixed') delta = { mood: 4, love: 2 };
            if (action === 'cry' || action === 'sulk' || action === 'bug') delta = { mood: -3 };
            if (action === 'bamboo' || action === 'cookie') delta = { hunger: -8, mood: 3 };
            this.playThenIdle(action, delta);
            return;
        }
        if (this.stats.mood > 75) this.playThenIdle('giggle');
        else if (this.stats.mood < 35) this.playThenIdle('shy');
        else this.playIdle();
    };

    /* ---------------- LLM (text session, bubble only) ---------------- */

    Pet.prototype.ensureTextSession = function () {
        var self = this;
        if (!self.hasLlm()) return Promise.reject(new Error('no llm config'));
        if (self.textSessionId) return Promise.resolve(self.textSessionId);
        return fetchJSON(self.apiBase + '/lingecho/dialog/v1/conversations', {
            method: 'POST',
            headers: self.authHeaders(false),
            body: JSON.stringify({
                assistantId: String(self.cfg.assistantId || ''),
                channel: 'api',
            }),
        }, 30000).then(function (body) {
            if (!body || (body.code !== 200 && body.code !== 0) || !body.data) {
                throw new Error((body && body.msg) || '会话创建失败');
            }
            self.textSessionId = body.data.id;
            return self.textSessionId;
        });
    };

    Pet.prototype.buildContextPrefix = function (mode) {
        var s = this.stats;
        var bits = [
            '【桌宠状态】你是「' + this.name + '」，一只爱睡觉的熊猫。',
            '心情' + Math.round(s.mood) + ' 精力' + Math.round(s.energy) + ' 饥饿' + Math.round(s.hunger) + ' 亲密度' + Math.round(s.love) + '。',
            this.sleeping ? '当前在睡觉。' : '',
            '回复请短，1～2 句中文口语，懒懒风格。',
            '若想表达情绪，可在末尾加且仅加一个标签，例如 [action:giggle]、[action:yawn]、[action:sulk]、[action:celebrate]、[action:bamboo]、[action:shy]、[action:zen]、[action:chaos]、[action:coding]、[action:bug]、[action:bug_fixed]。',
        ];
        if (mode === 'proactive') {
            bits.push('现在请主动跟主人说一句很短的话，不要提问清单。');
        }
        return bits.filter(Boolean).join('');
    };

    Pet.prototype.ask = function (userText, opts) {
        var self = this;
        opts = opts || {};
        if (!self.hasLlm()) {
            self.say(pick(LINES.noLlm), 3200);
            self.playThenIdle('ignore');
            return Promise.resolve(null);
        }
        if (self.busy && !opts.force) return Promise.resolve(null);
        self.pauseWander();
        if (self.sleeping && !opts.allowSleep) self.wake();
        self.busy = true;
        self.say(pick(LINES.thinking), 5000);
        self.player.play('zen', { loop: true, force: true });

        var payload = (opts.raw ? '' : self.buildContextPrefix(opts.mode || 'chat') + '\n主人说：') + String(userText || '').trim();

        return self.ensureTextSession()
            .then(function (sid) {
                return fetchJSON(
                    self.apiBase + '/lingecho/dialog/v1/conversations/' + encodeURIComponent(sid) + '/messages',
                    {
                        method: 'POST',
                        headers: self.authHeaders(false),
                        body: JSON.stringify({ text: payload }),
                    },
                    120000,
                );
            })
            .then(function (body) {
                self.busy = false;
                self._lastChatAt = now();
                if ((body.code === 200 || body.code === 0) && body.data && body.data.reply) {
                    self.reactToReply(body.data.reply);
                    return body.data.reply;
                }
                self.say((body && body.msg) || '脑子卡住了…', 2800);
                self.playThenIdle('sulk', { mood: -2 });
                return null;
            })
            .catch(function (err) {
                self.busy = false;
                self.say((err && err.message) || '连不上脑子', 2800);
                self.playThenIdle('ignore', { mood: -1 });
                return null;
            });
    };

    Pet.prototype.talk = function (text) {
        return this.ask(text);
    };

    Pet.prototype.proactiveChat = function () {
        var self = this;
        if (!self.hasLlm() || self.cfg.autoChat === false) return;
        if (self.busy || self.listening || self.talkOpen || self.menuOpen || self.sleeping || self.isCodingBusy()) return;
        if (now() - self._lastChatAt < 90000) return;
        var prompt = '根据当前状态随便嘟囔一句。';
        if (self.stats.hunger > 75) prompt = '你有点饿，小声抱怨一下。';
        else if (self.stats.energy < 30) prompt = '你很困，打着哈欠说话。';
        else if (self.stats.mood < 35) prompt = '你有点闷，说一句软乎乎的抱怨。';
        else if (self.stats.mood > 80) prompt = '你挺开心，说一句懒懒的快乐话。';
        self.ask(prompt, { mode: 'proactive', force: true });
    };

    /* ---------------- Voice (WebRTC + hotkey) ---------------- */

    Pet.prototype.setListeningUI = function (on) {
        this.listening = !!on;
        if (this.wrap) {
            if (on) this.wrap.classList.add('listening');
            else this.wrap.classList.remove('listening');
        }
        if (this.petEl) {
            if (on) this.petEl.classList.add('listening');
            else this.petEl.classList.remove('listening');
        }
    };

    Pet.prototype.cleanupVoice = function () {
        var self = this;
        self.setListeningUI(false);
        if (self.dc) {
            try { self.dc.close(); } catch (_) {}
            self.dc = null;
        }
        if (self.pc) {
            try { self.pc.close(); } catch (_) {}
            self.pc = null;
        }
        if (self.localStream) {
            try { self.localStream.getTracks().forEach(function (t) { t.stop(); }); } catch (_) {}
            self.localStream = null;
        }
        if (self.remoteAudio) {
            self.remoteAudio.srcObject = null;
            if (self.remoteAudio.parentNode) self.remoteAudio.parentNode.removeChild(self.remoteAudio);
            self.remoteAudio = null;
        }
        if (self.voiceSessionId) {
            var sid = self.voiceSessionId;
            self.voiceSessionId = null;
            fetchJSON(
                self.apiBase + '/lingecho/voice-session/v1/sessions/' + encodeURIComponent(sid),
                { method: 'DELETE', headers: self.authHeaders(true) },
                10000,
            ).catch(function () {});
        }
    };

    Pet.prototype.bindVoiceChannel = function (dc) {
        var self = this;
        if (self.dc && self.dc !== dc) {
            try { self.dc.close(); } catch (_) {}
        }
        self.dc = dc;
        dc.binaryType = 'arraybuffer';
        dc.onmessage = function (ev) {
            var fr = null;
            try {
                if (typeof ev.data === 'string') fr = JSON.parse(ev.data);
                else if (ev.data && ev.data.byteLength != null) {
                    fr = JSON.parse(new TextDecoder().decode(ev.data));
                }
            } catch (_) { return; }
            if (!fr || !fr.type) return;
            if (fr.type === 'transcript.user' && fr.text) {
                self.say('你：' + fr.text, 2200);
            } else if (fr.type === 'transcript.assistant' && fr.text) {
                self.reactToReply(fr.text);
            } else if (fr.type === 'error') {
                self.say(fr.message || '语音异常', 2600);
            }
        };
    };

    Pet.prototype.startVoice = function () {
        var self = this;
        if (!self.hasLlm()) {
            self.say(pick(LINES.noLlm), 3000);
            return Promise.resolve(false);
        }
        if (self.listening) return Promise.resolve(true);
        if (typeof RTCPeerConnection === 'undefined') {
            self.say('当前环境不支持 WebRTC 语音', 3000);
            return Promise.resolve(false);
        }
        self.pauseWander();
        self.closeMenu();
        self.closeTalk();
        if (self.sleeping) self.wake();
        self.busy = true;
        self.say(pick(LINES.listen), 2400);
        self.player.play('shy', { loop: false, force: true });

        return fetchJSON(self.apiBase + '/lingecho/voice-session/v1/sessions', {
            method: 'POST',
            headers: self.authHeaders(false),
            body: JSON.stringify({
                transport: 'webrtc',
                assistantId: String(self.cfg.assistantId || ''),
                sampleRateHz: 16000,
            }),
        }, 30000)
            .then(function (body) {
                if (!body || (body.code !== 200 && body.code !== 0) || !body.data) {
                    throw new Error((body && body.msg) || '语音会话失败');
                }
                self.voiceSessionId = body.data.sessionId;
                var pc = new RTCPeerConnection({ iceServers: [{ urls: 'stun:stun.l.google.com:19302' }] });
                self.pc = pc;
                pc.ondatachannel = function (ev) {
                    if (ev.channel && ev.channel.label === 'dialog') self.bindVoiceChannel(ev.channel);
                };
                self.bindVoiceChannel(pc.createDataChannel('dialog', { ordered: true }));

                var audio = document.createElement('audio');
                audio.autoplay = true;
                audio.setAttribute('playsinline', 'true');
                audio.style.display = 'none';
                document.body.appendChild(audio);
                self.remoteAudio = audio;
                pc.ontrack = function (ev) {
                    if (ev.streams && ev.streams[0]) {
                        audio.srcObject = ev.streams[0];
                        var p = audio.play();
                        if (p && p.catch) p.catch(function () {});
                    }
                };

                return navigator.mediaDevices.getUserMedia({
                    audio: { echoCancellation: true, noiseSuppression: true },
                    video: false,
                }).then(function (stream) {
                    self.localStream = stream;
                    stream.getTracks().forEach(function (t) { pc.addTrack(t, stream); });
                    return pc.createOffer({ offerToReceiveAudio: true });
                });
            })
            .then(function (offer) {
                return self.pc.setLocalDescription(offer).then(function () { return offer; });
            })
            .then(function (offer) {
                return new Promise(function (resolve) {
                    if (self.pc.iceGatheringState === 'complete') { resolve(); return; }
                    var check = function () {
                        if (self.pc.iceGatheringState === 'complete') {
                            self.pc.removeEventListener('icegatheringstatechange', check);
                            resolve();
                        }
                    };
                    self.pc.addEventListener('icegatheringstatechange', check);
                    setTimeout(resolve, 2500);
                }).then(function () {
                    return fetchJSON(self.apiBase + '/lingecho/voice-session/v1/webrtc/offer', {
                        method: 'POST',
                        headers: self.authHeaders(false),
                        body: JSON.stringify({
                            sessionId: self.voiceSessionId,
                            sdp: (self.pc.localDescription && self.pc.localDescription.sdp) || offer.sdp || '',
                            type: 'offer',
                        }),
                    }, 30000);
                });
            })
            .then(function (body) {
                if (!body || (body.code !== 200 && body.code !== 0) || !body.data || !body.data.sdp) {
                    throw new Error((body && body.msg) || 'WebRTC 协商失败');
                }
                return self.pc.setRemoteDescription({ type: 'answer', sdp: body.data.sdp });
            })
            .then(function () {
                self.busy = false;
                self.setListeningUI(true);
                self.player.play('sway', { loop: true, force: true });
                self.say('语音已开 · 再按 ' + (self.cfg.voiceHotkey || 'Alt+Shift+V') + ' 结束', 3200);
                return true;
            })
            .catch(function (err) {
                self.busy = false;
                self.cleanupVoice();
                self.say((err && err.message) || '语音开启失败', 3000);
                self.playThenIdle('sulk');
                return false;
            });
    };

    Pet.prototype.stopVoice = function () {
        if (!this.listening && !this.voiceSessionId) return Promise.resolve(false);
        this.cleanupVoice();
        this.say(pick(LINES.listenOff), 2200);
        this.playIdle();
        this.scheduleWander();
        return Promise.resolve(true);
    };

    Pet.prototype.toggleVoice = function () {
        return this.listening ? this.stopVoice() : this.startVoice();
    };

    /* ---------------- Interactions (情理化) ---------------- */

    Pet.prototype.pet = function () {
        if (this.listening) return Promise.resolve(false);
        this.pokeStreak = 0;
        if (this.sleeping) {
            this.say('……zzz 摸着也行', 1600);
            this.applyDelta({ love: 2, mood: 1 });
            return this.playThenIdle('shy');
        }
        this.say(pick(LINES.pet));
        this.applyDelta({ love: 5, mood: 5, energy: 1 });
        if (this.stats.love > 75 && this.stats.mood > 60) return this.playThenIdle('giggle');
        if (this.stats.love < 40) return this.playThenIdle('shy');
        return this.playThenIdle(Math.random() < 0.5 ? 'shy' : 'sway');
    };

    Pet.prototype.poke = function () {
        var self = this;
        if (self.listening) return Promise.resolve(false);
        if (self.sleeping) { self.wake(); return Promise.resolve(true); }
        self.pokeStreak += 1;
        if (self.pokeResetTimer) clearTimeout(self.pokeResetTimer);
        self.pokeResetTimer = setTimeout(function () { self.pokeStreak = 0; }, 5000);
        self.applyDelta({ mood: -2, love: 0.5 });

        if (self.pokeStreak >= 6) {
            self.pokeStreak = 0;
            self.say(pick(LINES.pokeMad));
            self.applyDelta({ mood: -10 });
            return self.playThenIdle('roll_cry');
        }
        if (self.pokeStreak >= 4) {
            self.say(pick(LINES.pokeMad));
            return self.playThenIdle('cry', { mood: -5 });
        }
        if (self.pokeStreak >= 2) {
            self.say(pick(LINES.poke));
            return self.playThenIdle('sulk', { mood: -3 });
        }
        self.say(pick(LINES.ignore));
        return self.playThenIdle('ignore');
    };

    Pet.prototype.feed = function (kind) {
        kind = kind === 'cookie' ? 'cookie' : 'bamboo';
        if (this.listening) return Promise.resolve(false);
        if (this.sleeping) this.wake();
        if (this.stats.hunger < 18) {
            this.say(pick(LINES.full));
            return this.playThenIdle('bored', { hunger: -2 });
        }
        if (kind === 'cookie') {
            this.say(pick(LINES.feedCookie));
            this.applyDelta({ hunger: -22, mood: 10, love: 6, energy: 3 });
            var after = this.stats.mood > 85 ? 'celebrate' : 'cookie';
            if (after === 'celebrate') {
                var self = this;
                return this.player.play('cookie', { lock: true, force: true }).then(function () {
                    self.say(pick(LINES.celebrate));
                    return self.playThenIdle('celebrate');
                });
            }
            return this.playThenIdle('cookie');
        }
        this.say(pick(LINES.feedBamboo));
        this.applyDelta({ hunger: -30, mood: 8, love: 3, energy: 6 });
        return this.playThenIdle('bamboo');
    };

    Pet.prototype.enterZen = function (ms) {
        var self = this;
        if (self.listening) return Promise.resolve(false);
        if (self.sleeping) self.wake();
        self.busy = true;
        self.pauseWander();
        self.say(pick(LINES.zen), 2400);
        self.applyDelta({ mood: 5, energy: 2 });
        self.player.play('zen', { loop: true, force: true, lock: true });
        setTimeout(function () {
            if (self.destroyed || self.listening) return;
            self.busy = false;
            self.playIdle();
            self.scheduleWander();
        }, ms || 7000);
        return Promise.resolve(true);
    };

    Pet.prototype.sleep = function () {
        var self = this;
        if (self.listening) self.stopVoice();
        self.codingMode = 'idle';
        self.codingCooldownUntil = 0;
        self.resetCodingSession();
        self.sleeping = true;
        self.busy = true;
        self.pauseWander();
        self.say(pick(LINES.sleep), 2000);
        self.scheduleSave();
        return self.player.play('yawn', { force: true, lock: true })
            .then(function () { return self.player.play('drowsy', { force: true, lock: true }); })
            .then(function () {
                self.busy = false;
                return self.player.play('sleep', { loop: true, force: true });
            });
    };

    Pet.prototype.wake = function () {
        var self = this;
        if (!self.sleeping) return Promise.resolve(false);
        self.sleeping = false;
        self.busy = true;
        self.say(pick(LINES.wake));
        self.applyDelta({ energy: 16, mood: 3 });
        self.scheduleSave();
        return self.player.play('yawn', { force: true, lock: true }).then(function () {
            self.busy = false;
            self.playIdle();
            self.scheduleWander();
        });
    };

    Pet.prototype.toggleSleep = function () {
        return this.sleeping ? this.wake() : this.sleep();
    };

    /* ---------------- Coding watch (打字监听状态机) ----------------
     *
     * idle ──(开始打字)──► coding ──(连续很久)──► sleepy
     *   ▲                    │                      │
     *   │                    │(停一会儿)            │(再敲)
     *   │                    ▼                      ▼
     *   └──(冷静期结束)── cooldown ◄── bug_fixed   coding
     *                         │
     *                         └──(冷静期内再敲)──► bug → coding
     */

    Pet.prototype.isCodingBusy = function () {
        return this.codingMode === 'coding' || this.codingMode === 'sleepy';
    };

    Pet.prototype.codingEnabled = function () {
        return this.cfg.watchCoding !== false;
    };

    Pet.prototype.isCodingKeyEvent = function (e) {
        if (!e || e.repeat) return false;
        if (e.isComposing || e.key === 'Process') return false;
        // 跟懒懒说话时不算敲代码
        if (this.talkOpen && this.talkInput && (e.target === this.talkInput || (this.talkEl && this.talkEl.contains(e.target)))) {
            return false;
        }
        var key = String(e.key || '');
        var code = String(e.code || '');
        if (key === 'Shift' || key === 'Control' || key === 'Alt' || key === 'Meta' || key === 'CapsLock' || key === 'Escape') return false;
        if (matchHotkey(e, this.voiceHotkey) || matchHotkey(e, this.talkHotkey)) return false;
        // 不必点进 input：页面任意焦点下，常见打字/编辑键都算
        return key.length === 1
            || key === 'Backspace' || key === 'Delete' || key === 'Enter' || key === 'Tab' || key === ' '
            || code.indexOf('Key') === 0 || code.indexOf('Digit') === 0 || code.indexOf('Numpad') === 0
            || code === 'Backspace' || code === 'Delete' || code === 'Enter' || code === 'NumpadEnter' || code === 'Tab' || code === 'Space';
    };

    Pet.prototype.resetCodingSession = function () {
        this.codingKeyCount = 0;
        this.codingSessionStart = 0;
        this.codingBurstStart = 0;
        this.codingLastKeyAt = 0;
    };

    /** 外部注入（桌宠全局键盘钩子 / CustomEvent），无需页面焦点 */
    Pet.prototype.notifyCodingKey = function () {
        var t = now();
        if (this._lastNotifyCodingAt && t - this._lastNotifyCodingAt < 30) return;
        this._lastNotifyCodingAt = t;
        this.onCodingKey();
    };

    Pet.prototype.enterCodingPose = function (opts) {
        var self = this;
        opts = opts || {};
        if (self.destroyed || self.listening || self.menuOpen) return;
        self.pauseWander();
        if (self.sleeping) {
            self.sleeping = false;
            self.scheduleSave();
        }
        if (opts.say) self.say(opts.say, opts.sayMs || 2000);
        // 已在敲代码循环段：不要重头播过渡
        if (self.player.action === 'coding' && self.player.playing && self.player.loop) {
            return;
        }
        self.playCodingLoop(!!opts.skipIntro);
    };

    /** 敲代码：先播过渡，再只循环末尾 loopFrom..frames；skipIntro 则直接进入循环段 */
    Pet.prototype.playCodingLoop = function (skipIntro) {
        var spec = ACTIONS.coding;
        var loopFrom = spec.loopFrom != null ? spec.loopFrom : 0;
        return this.player.play('coding', {
            loop: true,
            force: true,
            from: skipIntro ? loopFrom : 0,
            loopFrom: loopFrom,
        });
    };

    Pet.prototype.startCodingSession = function (fromCooldown) {
        var self = this;
        var t = now();
        self.codingMode = 'coding';
        self.codingSessionStart = t;
        self.codingBurstStart = t;
        self.codingKeyCount = 1;
        self.codingLastKeyAt = t;
        self.pauseWander();

        if (fromCooldown) {
            // 刚收工又开敲 → 懊恼（改出 bug），然后直接进敲代码循环段
            self.busy = true;
            self.say(pick(LINES.codingBug), 2200);
            self.applyDelta({ mood: -3, energy: -1 });
            self.player.play('bug', { loop: false, lock: true, force: true }).then(function () {
                self.busy = false;
                if (self.destroyed || self.codingMode !== 'coding') return;
                self.playCodingLoop(true);
            });
            return;
        }

        self.say(pick(LINES.coding), 1800);
        self.applyDelta({ energy: -0.5 });
        self.playCodingLoop(false);
    };

    Pet.prototype.onCodingKey = function () {
        var self = this;
        if (!self.codingEnabled() || self.destroyed || self.listening) return;
        if (self.menuOpen || self.talkOpen) return;

        var t = now();
        self.codingLastKeyAt = t;

        if (self.codingMode === 'cooldown') {
            self.startCodingSession(true);
            return;
        }

        if (self.codingMode === 'sleepy') {
            self.codingMode = 'coding';
            self.codingBurstStart = t;
            self.codingKeyCount += 1;
            self.say(pick(LINES.codingWake), 1800);
            self.applyDelta({ energy: 4, mood: 1 });
            self.playCodingLoop(true);
            return;
        }

        if (self.codingMode === 'idle') {
            self.startCodingSession(false);
            return;
        }

        // coding：持续敲
        self.codingKeyCount += 1;
        if (self.player && self.player.action !== 'coding' && !self.busy) {
            self.playCodingLoop(true);
        }
    };

    Pet.prototype.finishCodingAsFixed = function () {
        var self = this;
        var earned = self.codingKeyCount >= self.codingOpts.fixedMinKeys &&
            (now() - self.codingSessionStart) >= self.codingOpts.fixedMinMs;

        self.codingMode = 'cooldown';
        self.codingCooldownUntil = now() + self.codingOpts.cooldownMs;
        self.resetCodingSession();

        if (!earned) {
            // 敲太少，安静收工
            if (!self.busy && !self.listening && !self.sleeping) {
                self.playIdle();
                self.scheduleWander();
            }
            return;
        }

        self.busy = true;
        self.pauseWander();
        self.say(pick(LINES.codingFixed), 2400);
        self.applyDelta({ mood: 6, love: 2, energy: -2 });
        self.player.play('bug_fixed', { loop: false, lock: true, force: true }).then(function () {
            if (self.destroyed) return;
            // 播放期间又开始敲了：交给 coding / sleepy 流程，别抢动画
            if (self.codingMode === 'coding' || self.codingMode === 'sleepy') {
                if (!self.busy) {
                    if (self.codingMode === 'sleepy') self.player.play('bug_sleep', { loop: true, force: true });
                    else self.playCodingLoop(true);
                }
                return;
            }
            self.busy = false;
            self.playIdle();
            self.scheduleWander();
        });
    };

    Pet.prototype.enterCodingSleepy = function () {
        var self = this;
        if (self.codingMode !== 'coding') return;
        self.codingMode = 'sleepy';
        self.busy = false;
        self.pauseWander();
        self.say(pick(LINES.codingSleep), 2600);
        self.applyDelta({ energy: -8, mood: -1 });
        self.player.play('bug_sleep', { loop: true, force: true });
    };

    Pet.prototype.tickCodingWatch = function () {
        var self = this;
        if (!self.codingEnabled() || self.destroyed) return;
        var t = now();

        if (self.codingMode === 'cooldown') {
            if (t >= self.codingCooldownUntil) {
                self.codingMode = 'idle';
                self.codingCooldownUntil = 0;
            }
            return;
        }

        if (self.codingMode === 'coding') {
            if (self.busy || self.listening || self.menuOpen) return;
            // 停敲一会儿 → 修好了
            if (self.codingLastKeyAt && t - self.codingLastKeyAt >= self.codingOpts.pauseMs) {
                self.finishCodingAsFixed();
                return;
            }
            // 连续敲太久 → 睡着
            if (self.codingBurstStart && t - self.codingBurstStart >= self.codingOpts.longMs) {
                self.enterCodingSleepy();
            }
            return;
        }

        if (self.codingMode === 'sleepy') {
            // 睡着后若很久完全没再敲，慢慢退回 idle（不播 fixed，已经睡糊了）
            if (self.codingLastKeyAt && t - self.codingLastKeyAt >= self.codingOpts.pauseMs * 4) {
                self.codingMode = 'idle';
                self.resetCodingSession();
                if (!self.busy && !self.listening && !self.sleeping) {
                    self.playIdle();
                    self.scheduleWander();
                }
            }
        }
    };

    Pet.prototype.bindCodingWatch = function () {
        var self = this;
        if (!self.codingEnabled()) return;
        function onKey(e) {
            if (!self.isCodingKeyEvent(e)) return;
            self.onCodingKey();
        }
        function onExternal() {
            self.notifyCodingKey();
        }
        // 捕获阶段挂在 window，避免必须先点到桌宠 DOM
        window.addEventListener('keydown', onKey, true);
        window.addEventListener('lanlan:coding-key', onExternal);
        self._unsubs.push(function () { window.removeEventListener('keydown', onKey, true); });
        self._unsubs.push(function () { window.removeEventListener('lanlan:coding-key', onExternal); });
        // 桌宠 Electron：preload 注入的全局钩子
        if (window.electronPet && typeof window.electronPet.onCodingKey === 'function') {
            var off = window.electronPet.onCodingKey(function () { self.notifyCodingKey(); });
            if (typeof off === 'function') self._unsubs.push(off);
        }
        self.codingTimer = setInterval(function () { self.tickCodingWatch(); }, 1000);
    };

    /* ---------------- Desktop file tidy ---------------- */

    Pet.prototype.tidyDesktop = async function () {
        var self = this;
        if (self._tidyingDesktop) return false;
        if (!window.electronPet || typeof window.electronPet.tidyDesktop !== 'function') return false;

        self._tidyingDesktop = true;
        self.closeMenu();
        self.pauseWander();
        self.codingMode = 'idle';
        self.resetCodingSession();
        self.sleeping = false;
        self.busy = false;

        try {
            var preview = null;
            if (typeof window.electronPet.previewDesktop === 'function') {
                preview = await window.electronPet.previewDesktop();
            }
            if (preview && preview.ok && !preview.total) {
                self.say('桌面已经很干净啦～', 1800);
                self.playThenIdle('blink');
                return true;
            }

            var total = preview && preview.total ? preview.total : 1;
            // Finder 默认把桌面文件排列在右侧：先走到对应的文件区域。
            var target = {
                left: window.innerWidth - PET_SIZE - 36,
                top: clamp(54 + Math.min(total, 8) * 16, 54, window.innerHeight - PET_SIZE - 40),
            };
            var dist = Math.hypot(target.left - self.pos.left, target.top - self.pos.top);
            var moveMs = clamp(dist * 12, 1800, 7000);

            self.say('我来收拾一下～', 1800);
            self.wanderTo(target, false, true);
            await new Promise(function (resolve) { setTimeout(resolve, moveMs + 120); });
            self.pauseWander();

            // 到文件旁边先播放害羞动作，再真正归档文件。
            await self.player.play('shy', { loop: false, force: true, fps: 16 });
            var result = await window.electronPet.tidyDesktop();
            if (result && result.ok) {
                self.say(result.moved ? '都整理好啦～' : '没有需要整理的文件哦', 2200);
                if (result.moved) self.playThenIdle('celebrate');
                else self.playThenIdle('blink');
            } else {
                self.say('有几个文件没收拾成功…', 2200);
                self.playThenIdle('sulk');
            }
            return !!(result && result.ok);
        } catch (e) {
            console.warn('[Lanlan] tidyDesktop failed', e);
            self.say('整理时出了点小问题…', 2200);
            self.playThenIdle('sulk');
            return false;
        } finally {
            self._tidyingDesktop = false;
            self.scheduleWander();
        }
    };

    /* ---------------- Autonomous wander ---------------- */

    Pet.prototype.pauseWander = function () {
        this.wandering = false;
        this.wanderTarget = null;
        if (this.wanderRaf) {
            cancelAnimationFrame(this.wanderRaf);
            this.wanderRaf = null;
        }
    };

    Pet.prototype.scheduleWander = function (immediate) {
        var self = this;
        if (self.cfg.autoWander === false) return;
        if (self._wanderWait) clearTimeout(self._wanderWait);
        var delay = immediate ? 800 + Math.random() * 1200 : 8000 + Math.random() * 14000;
        self._wanderWait = setTimeout(function () {
            if (self.destroyed || self._tidyingDesktop || self.busy || self.sleeping || self.listening || self.menuOpen || self.talkOpen || self.isCodingBusy()) {
                self.scheduleWander();
                return;
            }
            if (Math.random() < 0.22) {
                self.enterZen(5000 + Math.random() * 4000);
                return;
            }
            self.wanderTo(null, false);
        }, delay);
    };

    Pet.prototype.randomWanderPoint = function () {
        var margin = 24;
        var maxL = Math.max(margin, window.innerWidth - PET_SIZE - margin);
        var maxT = Math.max(margin, window.innerHeight - PET_SIZE - margin);
        var left = margin + Math.random() * (maxL - margin);
        var top = margin + Math.random() * (maxT - margin);
        // 偏好下半屏，像桌宠贴着任务栏附近晃
        if (Math.random() < 0.55) top = maxT * (0.55 + Math.random() * 0.4);
        return { left: left, top: top };
    };

    Pet.prototype.wanderTo = function (target, announce, force) {
        var self = this;
        if (self.cfg.autoWander === false && !announce && !force) return;
        if (self.busy || self.sleeping || self.listening || self.talkOpen || self.isCodingBusy()) return;
        self.pauseWander();
        self.wanderTarget = target || self.randomWanderPoint();
        self.wandering = true;
        if (announce) self.say(pick(LINES.wander), 1600);
        self.applyDelta({ energy: -1.5 });
        // 移动时用 sway 表现「挪过去」
        if (!self.player.locked) self.player.play('sway', { loop: true, force: true, fps: 12 });

        var start = { left: self.pos.left, top: self.pos.top };
        var dist = Math.hypot(self.wanderTarget.left - start.left, self.wanderTarget.top - start.top);
        var duration = clamp(dist * 12, 1800, 7000);
        var t0 = now();

        function step() {
            if (!self.wandering || self.destroyed) return;
            if (self.busy || self.listening || self.sleeping) {
                self.pauseWander();
                return;
            }
            var t = clamp((now() - t0) / duration, 0, 1);
            // ease-in-out
            var e = t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2;
            self.layoutAt(
                lerp(start.left, self.wanderTarget.left, e),
                lerp(start.top, self.wanderTarget.top, e),
            );
            if (t < 1) {
                self.wanderRaf = requestAnimationFrame(step);
            } else {
                self.wandering = false;
                self.wanderTarget = null;
                self.wanderRaf = null;
                // 到达后偶尔眨眼或坐下发呆
                if (Math.random() < 0.3) self.playThenIdle('blink');
                else self.playIdle();
                self.scheduleWander();
            }
        }
        self.wanderRaf = requestAnimationFrame(step);
    };

    /* ---------------- Input / hotkeys / systems ---------------- */

    Pet.prototype.bindHotkeys = function () {
        var self = this;
        // Electron desktop registers these in the main process so they work
        // while other apps are focused. Avoid handling the same chord twice
        // when the pet window itself happens to have focus.
        if (window.__LINGECHO_EMBED_MODE__ === 'desktop') return;
        function onKey(e) {
            if (e.repeat) return;
            var tag = (e.target && e.target.tagName) || '';
            if (self.talkOpen && tag === 'INPUT') {
                if (matchHotkey(e, self.voiceHotkey)) {
                    e.preventDefault();
                    self.closeTalk();
                    self.toggleVoice();
                }
                return;
            }
            if (tag === 'INPUT' || tag === 'TEXTAREA' || (e.target && e.target.isContentEditable)) return;

            if (matchHotkey(e, self.voiceHotkey)) {
                e.preventDefault();
                self.toggleVoice();
                return;
            }
            if (matchHotkey(e, self.talkHotkey)) {
                e.preventDefault();
                if (self.talkOpen) self.closeTalk();
                else self.openTalk();
            }
        }
        document.addEventListener('keydown', onKey, true);
        self._unsubs.push(function () { document.removeEventListener('keydown', onKey, true); });
    };

    Pet.prototype.bindInput = function () {
        var self = this;
        var dragging = false;
        var moved = false;
        var startX = 0, startY = 0, startLeft = 0, startTop = 0;
        var DRAG_THRESHOLD = 5;
        var hoverTimer = null;

        function pointer(e) {
            if (e.touches && e.touches.length) return e.touches[0];
            if (e.changedTouches && e.changedTouches.length) return e.changedTouches[0];
            return e;
        }

        function onDown(e) {
            if (e.button === 2 || e.ctrlKey) {
                self._awaitingContextMenu = true;
                return;
            }
            if (self.menuOpen) {
                if (self.menuEl && self.menuEl.contains(e.target)) return;
                self.closeMenu();
            }
            dragging = true;
            moved = false;
            var pt = pointer(e);
            startX = pt.clientX; startY = pt.clientY;
            startLeft = self.pos.left; startTop = self.pos.top;
            self.petEl.classList.add('dragging');
            self.pauseWander();
            e.preventDefault();
        }

        function onMove(e) {
            if (!dragging) return;
            var pt = pointer(e);
            var dx = pt.clientX - startX;
            var dy = pt.clientY - startY;
            if (!moved && Math.abs(dx) + Math.abs(dy) < DRAG_THRESHOLD) return;
            moved = true;
            self.hideHint();
            self.layoutAt(startLeft + dx, startTop + dy);
            e.preventDefault();
        }

        function onUp(e) {
            if (self._awaitingContextMenu) {
                self._awaitingContextMenu = false;
                dragging = false;
                self.petEl.classList.remove('dragging');
                return;
            }
            if (now() < self._suppressTapUntil) {
                dragging = false;
                self.petEl.classList.remove('dragging');
                return;
            }
            if (!dragging) return;
            dragging = false;
            self.petEl.classList.remove('dragging');
            if (moved) {
                if (Math.random() < 0.18) {
                    self.say('搬来搬去好累…', 1500);
                    self.playThenIdle('sulk', { energy: -2 });
                } else {
                    self.scheduleWander();
                }
                moved = false;
                return;
            }
            if (self.menuOpen) {
                moved = false;
                return;
            }
            self.handleTap();
            moved = false;
        }

        function onContext(e) {
            e.preventDefault();
            e.stopPropagation();
            if (hoverTimer) clearTimeout(hoverTimer);
            self._awaitingContextMenu = false;
            self._suppressTapUntil = now() + 520;
            if (self.menuOpen) self.closeMenu();
            else self.openMenu();
        }

        function onDbl(e) {
            e.preventDefault();
            self.handleDouble();
        }

        function onEnter() {
            if (self.menuOpen) return;
            hoverTimer = setTimeout(function () {
                if (self.busy || self.sleeping || self.menuOpen || self.listening || self.wandering) return;
                if (Math.random() < 0.35) {
                    self.say('看我干嘛…', 1200);
                    self.playThenIdle('shy', { love: 1 });
                }
            }, 1200);
        }

        function onLeave() {
            if (hoverTimer) clearTimeout(hoverTimer);
        }

        function onDocDown(e) {
            if (self.menuOpen && self.menuEl && !self.menuEl.contains(e.target) && !self.petEl.contains(e.target)) {
                self.closeMenu();
            }
            if (self.talkOpen && self.talkEl && !self.talkEl.contains(e.target) && !self.petEl.contains(e.target)) {
                self.closeTalk();
            }
        }

        function onResize() {
            self.layoutAt(self.pos.left, self.pos.top);
        }

        self.petEl.addEventListener('mousedown', onDown);
        self.petEl.addEventListener('touchstart', onDown, { passive: false });
        document.addEventListener('mousemove', onMove);
        document.addEventListener('touchmove', onMove, { passive: false });
        document.addEventListener('mouseup', onUp);
        document.addEventListener('touchend', onUp);
        self.petEl.addEventListener('contextmenu', onContext);
        self.petEl.addEventListener('dblclick', onDbl);
        self.petEl.addEventListener('mouseenter', onEnter);
        self.petEl.addEventListener('mouseleave', onLeave);
        document.addEventListener('mousedown', onDocDown);
        window.addEventListener('resize', onResize);

        self._unsubs.push(
            function () { document.removeEventListener('mousemove', onMove); },
            function () { document.removeEventListener('mouseup', onUp); },
            function () { document.removeEventListener('mousedown', onDocDown); },
            function () { window.removeEventListener('resize', onResize); },
        );
    };

    Pet.prototype.handleTap = function () {
        var self = this;
        if (self.menuOpen || now() < self._suppressTapUntil) return;
        if (self.clickTimer) clearTimeout(self.clickTimer);
        self._tapBucket = (self._tapBucket || 0) + 1;
        self.clickTimer = setTimeout(function () {
            var n = self._tapBucket || 1;
            self._tapBucket = 0;
            if (n >= 2) self.handleDouble();
            else self.pet();
        }, 220);
    };

    Pet.prototype.handleDouble = function () {
        if (this.menuOpen || now() < this._suppressTapUntil) return;
        if (this.clickTimer) clearTimeout(this.clickTimer);
        this._tapBucket = 0;
        if (this.sleeping) { this.wake(); return; }
        if (this.listening) return;
        // 双击：优先喂食或打开说话，不再乱放 celebrate/chaos
        if (this.stats.hunger > 55) {
            this.feed('bamboo');
        } else if (this.hasLlm()) {
            this.openTalk();
        } else if (this.stats.mood > 70) {
            this.say(pick(LINES.celebrate));
            this.playThenIdle('giggle', { mood: 3 });
        } else {
            this.feed('cookie');
        }
    };

    Pet.prototype.startSystems = function () {
        var self = this;

        self.decayTimer = setInterval(function () {
            if (self.destroyed) return;
            if (self.sleeping) self.applyDelta({ energy: 2.2, hunger: 0.6, mood: 0.3 });
            else self.applyDelta({ energy: -0.7, hunger: 0.9, mood: -0.25 });

            if (self.busy || self.listening || self.menuOpen || self.talkOpen || self.isCodingBusy()) return;
            if (!self.sleeping && self.stats.energy < 12) {
                self.say('好困……要睡了', 1800);
                self.sleep();
            } else if (!self.sleeping && self.stats.hunger > 90 && Math.random() < 0.35) {
                self.say('饿扁了……', 1600);
                self.playThenIdle('sulk');
            }
        }, 14000);

        self.aiTimer = setInterval(function () {
            if (self.destroyed || self.busy || self.listening || self.menuOpen || self.talkOpen || self.wandering || self.isCodingBusy()) return;
            if (self.sleeping || document.hidden) return;
            var hour = new Date().getHours();
            var roll = Math.random();

            if ((hour >= 23 || hour < 6) && roll < 0.28) {
                self.sleep();
                return;
            }
            if (self.stats.energy < 35 && roll < 0.3) {
                self.say('哈啊……', 1200);
                self.playThenIdle('yawn', { energy: -1 });
                return;
            }
            if (self.stats.hunger > 70 && roll < 0.25) {
                self.playThenIdle('bored', { mood: -1 });
                return;
            }
            if (roll < 0.15) {
                self.enterZen();
                return;
            }
            if (roll < 0.28 && self.stats.mood > 70) {
                self.playThenIdle('giggle');
                return;
            }
            if (roll < 0.4) {
                self.playThenIdle('blink');
                return;
            }
            if (self.cfg.autoWander !== false && roll < 0.7) {
                self.wanderTo(null, Math.random() < 0.3);
                return;
            }
            self.playIdle();
        }, 14000 + Math.random() * 10000);

        if (self.cfg.autoChat !== false) {
            self.chatTimer = setInterval(function () {
                if (Math.random() < 0.4) self.proactiveChat();
            }, 110000 + Math.random() * 60000);
        }
    };

    Pet.prototype.destroy = function () {
        this.destroyed = true;
        this.pauseWander();
        if (this._wanderWait) clearTimeout(this._wanderWait);
        if (this.codingTimer) { clearInterval(this.codingTimer); this.codingTimer = null; }
        if (this.player) this.player.stop();
        this.cleanupVoice();
        if (this.textSessionId && this.apiBase) {
            fetchJSON(
                this.apiBase + '/lingecho/dialog/v1/conversations/' + encodeURIComponent(this.textSessionId) + '/end',
                { method: 'POST', headers: this.authHeaders(false), body: '{}' },
                8000,
            ).catch(function () {});
            this.textSessionId = null;
        }
        [this.aiTimer, this.decayTimer, this.chatTimer].forEach(function (t) {
            if (t) clearInterval(t);
        });
        [this.hintTimer, this.saveTimer, this.clickTimer, this.pokeResetTimer].forEach(function (t) {
            if (t) clearTimeout(t);
        });
        saveState(this);
        (this._unsubs || []).forEach(function (fn) { try { fn(); } catch (_) {} });
        var root = document.getElementById(ROOT_ID);
        if (root && root.parentNode) root.parentNode.removeChild(root);
        var css = document.getElementById(CSS_ID);
        if (css && css.parentNode) css.parentNode.removeChild(css);
        delete window.LingEchoWidget;
    };

    Pet.prototype.getState = function () {
        return {
            name: this.name,
            action: this.player ? this.player.action : null,
            sleeping: this.sleeping,
            busy: this.busy,
            listening: this.listening,
            wandering: this.wandering,
            codingMode: this.codingMode,
            stats: Object.assign({}, this.stats),
            pos: Object.assign({}, this.pos),
            hasLlm: this.hasLlm(),
            actions: Object.keys(ACTIONS),
            labels: ACTION_LABELS,
            hotkeys: {
                voice: this.cfg.voiceHotkey || 'Alt+Shift+V',
                talk: this.cfg.talkHotkey || 'Alt+Shift+T',
            },
        };
    };

    var instance = null;

    function mount(cfg) {
        if (instance) instance.destroy();
        instance = new Pet(cfg);
        instance.mount();
        return instance;
    }

    function destroy() {
        if (instance) { instance.destroy(); instance = null; }
    }

    function toggleWidget(force) {
        if (!instance) return;
        if (typeof force === 'boolean') {
            if (force) instance.openTalk();
            else instance.closeTalk();
            return;
        }
        if (instance.talkOpen) instance.closeTalk();
        else instance.openTalk();
    }

    var api = {
        mount: mount,
        destroy: destroy,
        toggle: toggleWidget,
        play: function (a, o) { return instance ? instance.play(a, o) : Promise.resolve(false); },
        say: function (t, ms) { return instance ? instance.say(t, ms) : null; },
        ask: function (t, o) { return instance ? instance.ask(t, o) : Promise.resolve(null); },
        talk: function (t) { return instance ? instance.talk(t) : Promise.resolve(null); },
        feed: function (k) { return instance ? instance.feed(k) : Promise.resolve(false); },
        pet: function () { return instance ? instance.pet() : Promise.resolve(false); },
        poke: function () { return instance ? instance.poke() : Promise.resolve(false); },
        sleep: function () { return instance ? instance.sleep() : Promise.resolve(false); },
        wake: function () { return instance ? instance.wake() : Promise.resolve(false); },
        toggleVoice: function () { return instance ? instance.toggleVoice() : Promise.resolve(false); },
        startVoice: function () { return instance ? instance.startVoice() : Promise.resolve(false); },
        stopVoice: function () { return instance ? instance.stopVoice() : Promise.resolve(false); },
        openTalk: function () { return instance ? instance.openTalk() : null; },
        wanderTo: function (t, a, f) { return instance ? instance.wanderTo(t, a, f) : null; },
        tidyDesktop: function () { return instance ? instance.tidyDesktop() : Promise.resolve(false); },
        notifyCodingKey: function () { return instance ? instance.notifyCodingKey() : null; },
        getState: function () { return instance ? instance.getState() : null; },
        get instance() { return instance; },
        ACTIONS: ACTIONS,
        ACTION_LABELS: ACTION_LABELS,
    };

    window.LanlanPet = api;
    window.LingEchoWidget = api;

    function autoMount() {
        mount(CFG);
    }

    if (CFG.autoMount !== false) {
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', autoMount);
        } else {
            autoMount();
        }
    }
})();
