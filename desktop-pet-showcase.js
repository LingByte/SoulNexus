// ============================================================
//  SoulNexus JS Template Showcase — Desktop Pet "Mochi" 🐱
//  v3 — SVG illustrated cat with:
//    • Curved realistic ears with independent rotation
//    • Rich idle behaviors: stretch, yawn, groom, look-around
//    • Smart AI: cursor following, multi-step wandering, edge walk
//    • Eye tracking, blink, tail wag, breathe, whisker twitch
//    • Mood / hunger / energy system
//    • Drag & drop, click interactions, mini-game
//    • Particle effects, speech bubbles, status bar, context menu
//  Pure vanilla JS — zero dependencies.
// ============================================================

(function () {
  'use strict';

  // ─── Reset ─────────────────────────────────────────────────
  document.body.innerHTML = '';
  document.body.style.cssText = 'margin:0;overflow:hidden;width:100vw;height:100vh;background:linear-gradient(135deg,#e0f2fe 0%,#f0e6ff 50%,#fce7f3 100%);font-family:"Segoe UI",system-ui,-apple-system,sans-serif;user-select:none;-webkit-user-select:none;cursor:default;';

  const root = document.createElement('div');
  root.style.cssText = 'position:fixed;inset:0;overflow:hidden;';
  document.body.appendChild(root);

  // ─── State ─────────────────────────────────────────────────
  const ST = {
    mood: 'happy', hunger: 80, energy: 90, petCount: 0,
    sleeping: false, walking: false, dragging: false, playing: false,
    facingRight: true,
    x: innerWidth / 2 - 75, y: innerHeight - 210,
    walkTarget: null, walkSpeed: 1.3,
    behavior: 'idle', behaviorTime: 0, behaviorData: {},
    blinkTimer: null, chatTimer: null, menuOpen: false,
  };

  // ─── Utils ─────────────────────────────────────────────────
  const rand = (a, b) => Math.random() * (b - a) + a;
  const randInt = (a, b) => Math.floor(rand(a, b + 1));
  const pick = (a) => a[randInt(0, a.length - 1)];
  const clamp = (v, lo, hi) => Math.max(lo, Math.min(hi, v));
  const lerp = (a, b, t) => a + (b - a) * t;
  const ease = (t) => t < .5 ? 2 * t * t : -1 + (4 - 2 * t) * t;

  // ─── SVG Helper ────────────────────────────────────────────
  const NS = 'http://www.w3.org/2000/svg';
  const S = (tag, a, p) => {
    const e = document.createElementNS(NS, tag);
    if (a) for (const k in a) e.setAttribute(k, String(a[k]));
    if (p) p.appendChild(e);
    return e;
  };

  // ─── Speech Library ────────────────────────────────────────
  const talk = {
    happy:   ['喵~ 💕', '今天心情真好！', '摸摸我嘛~', '噜噜噜~', '喵呜！', '嘿嘿 (*≧ω≦)', '尾巴摇起来~', '蹭蹭你~', '想玩！'],
    neutral: ['喵？', '有什么事吗…', '嗯…', '在想事情…', '喵。', '你好呀', '看看窗外…', '今天好无聊'],
    sleepy:  ['好困…💤', '想睡觉了…', '打个盹…', 'Zzz…', '眼皮好重…', '晚安喵…', '做个好梦…'],
    hungry:  ['饿了…🍗', '想吃小鱼干！', '肚子咕咕叫…', '投喂我！', '没力气了…', '闻到了好吃的…'],
    petted:  ['好舒服~ ❤️', '再摸摸！', '呼噜噜~', '最喜欢你了！', '翻肚皮给你看！', '还要~', '尾巴翘起来~'],
    boop:    ['哎呀！', '戳我干嘛！', '鼻子不要戳~', '( >﹏<。)', '哼！', '别闹~'],
    fed:     ['好好吃！🍗', '满足~', '还要！', '吃饱了好开心~', '嗝~', '小鱼干真香~'],
    stretch: ['伸个懒腰~', '喵啊~（打哈欠）', '活动一下筋骨'],
    groom:   ['舔舔毛~', '整理仪容中…', '喵·精致'],
  };

  // ─── Particle System ───────────────────────────────────────
  const particles = [];
  function emit(x, y, type) {
    const sym = { heart:['❤️','💕','💗'], star:['⭐','✨','🌟'], sparkle:['✨','💫','⚡'], food:['🍗','🐟','🍖'] };
    const d = document.createElement('div');
    d.textContent = pick(sym[type] || sym.heart);
    d.style.cssText = `position:fixed;font-size:${rand(14,22)}px;pointer-events:none;z-index:10000;`;
    root.appendChild(d);
    particles.push({ el:d, x, y, vx:rand(-2,2), vy:rand(-4,-1.5), life:1, dec:rand(.012,.025), g:.06 });
  }
  function tickParticles() {
    for (let i = particles.length - 1; i >= 0; i--) {
      const p = particles[i];
      p.vy += p.g; p.x += p.vx; p.y += p.vy; p.life -= p.dec;
      if (p.life <= 0) { p.el.remove(); particles.splice(i, 1); }
      else { Object.assign(p.el.style, { left:p.x+'px', top:p.y+'px', opacity:p.life, transform:`scale(${.5+p.life*.5})` }); }
    }
  }

  // ═══════════════════════════════════════════════════════════
  //  SVG CAT — Curved realistic ears, detailed features
  // ═══════════════════════════════════════════════════════════
  const R = {};

  function buildCat() {
    const svg = S('svg', { viewBox:'0 0 150 180', width:150, height:180 });
    const defs = S('defs', {}, svg);

    // ── Gradients ──
    const gBody = S('linearGradient', { id:'gB', x1:0, y1:0, x2:0, y2:1 }, defs);
    S('stop', { offset:'0%','stop-color':'#ddd6fe' }, gBody);
    S('stop', { offset:'100%','stop-color':'#8b5cf6' }, gBody);

    const gHead = S('linearGradient', { id:'gH', x1:0, y1:0, x2:0, y2:1 }, defs);
    S('stop', { offset:'0%','stop-color':'#ede9fe' }, gHead);
    S('stop', { offset:'55%','stop-color':'#c4b5fd' }, gHead);
    S('stop', { offset:'100%','stop-color':'#a78bfa' }, gHead);

    const gEar = S('linearGradient', { id:'gE', x1:0, y1:1, x2:0, y2:0 }, defs);
    S('stop', { offset:'0%','stop-color':'#a78bfa' }, gEar);
    S('stop', { offset:'100%','stop-color':'#c4b5fd' }, gEar);

    const gIris = S('radialGradient', { id:'gI', cx:.4, cy:.35, r:.6 }, defs);
    S('stop', { offset:'0%','stop-color':'#4338ca' }, gIris);
    S('stop', { offset:'100%','stop-color':'#1e1b4b' }, gIris);

    // Filters
    const fSh = S('filter', { id:'fSh', x:'-20%', y:'-10%', width:'140%', height:'140%' }, defs);
    S('feDropShadow', { dx:0, dy:2, stdDeviation:3, 'flood-color':'rgba(88,28,135,0.15)' }, fSh);

    const fGl = S('filter', { id:'fGl', x:'-50%', y:'-50%', width:'200%', height:'200%' }, defs);
    S('feGaussianBlur', { in:'SourceGraphic', stdDeviation:1.5, result:'b' }, fGl);
    const fm = S('feMerge', {}, fGl);
    S('feMergeNode', { in:'b' }, fm);
    S('feMergeNode', { in:'SourceGraphic' }, fm);

    // ── Ground shadow ──
    R.groundShadow = S('ellipse', { cx:75, cy:175, rx:42, ry:6, fill:'rgba(0,0,0,0.07)' }, svg);

    // ── Tail ──
    R.tail = S('g', {}, svg);
    S('path', { d:'M30,140 C6,128 0,92 16,74 C26,62 40,68 36,82', stroke:'#7c3aed', 'stroke-width':10, fill:'none', 'stroke-linecap':'round' }, R.tail);
    S('path', { d:'M30,140 C6,128 0,92 16,74 C26,62 40,68 36,82', stroke:'#a78bfa', 'stroke-width':5, fill:'none', 'stroke-linecap':'round' }, R.tail);
    S('circle', { cx:34, cy:78, r:4, fill:'#c4b5fd', opacity:.6 }, R.tail);

    // ── Body ──
    R.body = S('ellipse', { cx:75, cy:132, rx:44, ry:35, fill:'url(#gB)', filter:'url(#fSh)' }, svg);
    S('ellipse', { cx:75, cy:138, rx:26, ry:22, fill:'#ede9fe', opacity:.75 }, svg);
    // Fur stripes
    S('path', { d:'M54,118 Q59,113 64,118', stroke:'#c4b5fd', 'stroke-width':1.5, fill:'none', opacity:.4 }, svg);
    S('path', { d:'M86,118 Q91,113 96,118', stroke:'#c4b5fd', 'stroke-width':1.5, fill:'none', opacity:.4 }, svg);

    // ── Back paws (walking only) ──
    R.pawBL = S('ellipse', { cx:52, cy:164, rx:11, ry:7, fill:'#9673d8', opacity:0 }, svg);
    R.pawBR = S('ellipse', { cx:98, cy:164, rx:11, ry:7, fill:'#9673d8', opacity:0 }, svg);

    // ── Front paws ──
    R.pawL = S('g', { transform:'translate(52,160)' }, svg);
    S('ellipse', { cx:0, cy:0, rx:13, ry:8, fill:'#a78bfa' }, R.pawL);
    S('ellipse', { cx:-2, cy:1, rx:7, ry:4, fill:'#f5d0fe', opacity:.6 }, R.pawL);
    S('circle', { cx:-5, cy:-1, r:1.5, fill:'#e9d5ff', opacity:.5 }, R.pawL);
    S('circle', { cx:1, cy:-2, r:1.5, fill:'#e9d5ff', opacity:.5 }, R.pawL);

    R.pawR = S('g', { transform:'translate(98,160)' }, svg);
    S('ellipse', { cx:0, cy:0, rx:13, ry:8, fill:'#a78bfa' }, R.pawR);
    S('ellipse', { cx:2, cy:1, rx:7, ry:4, fill:'#f5d0fe', opacity:.6 }, R.pawR);
    S('circle', { cx:5, cy:-1, r:1.5, fill:'#e9d5ff', opacity:.5 }, R.pawR);
    S('circle', { cx:-1, cy:-2, r:1.5, fill:'#e9d5ff', opacity:.5 }, R.pawR);

    // ── Head ──
    R.head = S('g', {}, svg);
    S('circle', { cx:75, cy:60, r:46, fill:'url(#gH)', filter:'url(#fSh)' }, R.head);
    S('ellipse', { cx:62, cy:42, rx:22, ry:16, fill:'#ede9fe', opacity:.35 }, R.head);

    // ── Ears (curved realistic) ──
    R.earL = S('g', { transform:'rotate(0, 44, 28)' }, R.head);
    // Outer ear — curved triangle with rounded tip
    S('path', {
      d:'M44,30 C42,20 34,6 30,0 C28,-2 27,-3 29,-4 C32,-5 40,-3 44,0 C48,3 52,10 54,16 C53,22 48,28 44,30 Z',
      fill:'url(#gE)', stroke:'#a78bfa', 'stroke-width':.5
    }, R.earL);
    // Inner ear pink
    S('path', {
      d:'M44,26 C43,20 38,10 35,5 C34,3 33,2 34,1 C36,0 40,1 43,4 C46,7 48,12 49,16 C48,20 46,24 44,26 Z',
      fill:'#f9a8d4', opacity:.85
    }, R.earL);
    // Inner ear fur texture
    S('path', { d:'M41,22 C42,18 40,12 39,9', stroke:'#f0abfc', 'stroke-width':.8, fill:'none', opacity:.5 }, R.earL);
    S('path', { d:'M44,24 C44,20 43,14 42,11', stroke:'#f0abfc', 'stroke-width':.8, fill:'none', opacity:.4 }, R.earL);
    S('path', { d:'M47,22 C46,18 45,13 44,11', stroke:'#f0abfc', 'stroke-width':.8, fill:'none', opacity:.3 }, R.earL);

    R.earR = S('g', { transform:'rotate(0, 106, 28)' }, R.head);
    S('path', {
      d:'M106,30 C108,20 116,6 120,0 C122,-2 123,-3 121,-4 C118,-5 110,-3 106,0 C102,3 98,10 96,16 C97,22 102,28 106,30 Z',
      fill:'url(#gE)', stroke:'#a78bfa', 'stroke-width':.5
    }, R.earR);
    S('path', {
      d:'M106,26 C107,20 112,10 115,5 C116,3 117,2 116,1 C114,0 110,1 107,4 C104,7 102,12 101,16 C102,20 104,24 106,26 Z',
      fill:'#f9a8d4', opacity:.85
    }, R.earR);
    S('path', { d:'M109,22 C108,18 110,12 111,9', stroke:'#f0abfc', 'stroke-width':.8, fill:'none', opacity:.5 }, R.earR);
    S('path', { d:'M106,24 C106,20 107,14 108,11', stroke:'#f0abfc', 'stroke-width':.8, fill:'none', opacity:.4 }, R.earR);
    S('path', { d:'M103,22 C104,18 105,13 106,11', stroke:'#f0abfc', 'stroke-width':.8, fill:'none', opacity:.3 }, R.earR);

    // ── Eyes ──
    R.eyeL = S('g', {}, R.head);
    S('ellipse', { cx:56, cy:54, rx:11, ry:12, fill:'#fff' }, R.eyeL);
    R.irisL = S('ellipse', { cx:56, cy:55, rx:9, ry:10, fill:'url(#gI)' }, R.eyeL);
    R.pupilL = S('ellipse', { cx:56, cy:55, rx:3.5, ry:9, fill:'#0f0a2e' }, R.eyeL);
    S('circle', { cx:59, cy:50, r:3.5, fill:'#fff', filter:'url(#fGl)' }, R.eyeL);
    S('circle', { cx:53, cy:58, r:1.8, fill:'#fff', opacity:.7 }, R.eyeL);
    S('path', { d:'M45,48 Q56,42 67,48', stroke:'#7c3aed', 'stroke-width':1.5, fill:'none' }, R.eyeL);

    R.eyeR = S('g', {}, R.head);
    S('ellipse', { cx:94, cy:54, rx:11, ry:12, fill:'#fff' }, R.eyeR);
    R.irisR = S('ellipse', { cx:94, cy:55, rx:9, ry:10, fill:'url(#gI)' }, R.eyeR);
    R.pupilR = S('ellipse', { cx:94, cy:55, rx:3.5, ry:9, fill:'#0f0a2e' }, R.eyeR);
    S('circle', { cx:97, cy:50, r:3.5, fill:'#fff', filter:'url(#fGl)' }, R.eyeR);
    S('circle', { cx:91, cy:58, r:1.8, fill:'#fff', opacity:.7 }, R.eyeR);
    S('path', { d:'M83,48 Q94,42 105,48', stroke:'#7c3aed', 'stroke-width':1.5, fill:'none' }, R.eyeR);

    // Blink overlays
    R.blinkL = S('ellipse', { cx:56, cy:54, rx:12, ry:0, fill:'#c4b5fd' }, R.head);
    R.blinkR = S('ellipse', { cx:94, cy:54, rx:12, ry:0, fill:'#c4b5fd' }, R.head);

    // Sleepy eyes
    R.sleepL = S('path', { d:'M45,55 Q56,46 67,55', stroke:'#1e1b4b', 'stroke-width':2.5, fill:'none', 'stroke-linecap':'round', display:'none' }, R.head);
    R.sleepR = S('path', { d:'M83,55 Q94,46 105,55', stroke:'#1e1b4b', 'stroke-width':2.5, fill:'none', 'stroke-linecap':'round', display:'none' }, R.head);

    // ── Nose ──
    S('path', { d:'M72,63 L75,67 L78,63 Z', fill:'#ec4899', 'stroke-linejoin':'round' }, R.head);
    S('ellipse', { cx:75, cy:63.5, rx:2, ry:1, fill:'#f9a8d4', opacity:.6 }, R.head);

    // ── Mouth ──
    R.mouth = S('path', { d:'M66,69 Q70,75 75,69 Q80,75 84,69', stroke:'#6b21a8', 'stroke-width':1.8, fill:'none', 'stroke-linecap':'round' }, R.head);

    // ── Whiskers ──
    R.whiskers = S('g', { opacity:.35 }, R.head);
    const wkData = [[18,58,44,63],[16,65,43,67],[20,72,45,71],[132,58,106,63],[134,65,107,67],[130,72,105,71]];
    wkData.forEach(([x1,y1,x2,y2]) => {
      S('line', { x1, y1, x2, y2, stroke:'#7c3aed', 'stroke-width':1.2, 'stroke-linecap':'round' }, R.whiskers);
    });

    // ── Blush ──
    R.blushL = S('ellipse', { cx:40, cy:65, rx:7, ry:5, fill:'#f9a8d4', opacity:0 }, R.head);
    R.blushR = S('ellipse', { cx:110, cy:65, rx:7, ry:5, fill:'#f9a8d4', opacity:0 }, R.head);

    // ── Tongue (for grooming, hidden) ──
    R.tongue = S('ellipse', { cx:75, cy:73, rx:4, ry:3, fill:'#f472b6', opacity:0, display:'none' }, R.head);

    return svg;
  }

  // ─── Mount ─────────────────────────────────────────────────
  const catWrap = document.createElement('div');
  catWrap.style.cssText = 'position:fixed;z-index:9990;cursor:grab;filter:drop-shadow(0 4px 12px rgba(88,28,135,0.12));';
  root.appendChild(catWrap);
  catWrap.appendChild(buildCat());

  function posCat() {
    catWrap.style.left = ST.x + 'px';
    catWrap.style.top = ST.y + 'px';
    const flip = ST.facingRight ? 1 : -1;
    const tilt = ST.behavior === 'looking' ? (ST.behaviorData.lookDir || 0) * 3 : 0;
    catWrap.style.transform = `scaleX(${flip}) rotate(${tilt}deg)`;
  }
  posCat();

  // ═══════════════════════════════════════════════════════════
  //  ANIMATION
  // ═══════════════════════════════════════════════════════════
  const A = {
    tailAngle: 0, tailDir: 1,
    blinkProg: 0, blinking: false, nextBlink: 2500,
    breathPh: 0, walkPh: 0,
    eyeX: 0, eyeY: 0, tgtEyeX: 0, tgtEyeY: 0,
    earLAngle: 0, earRAngle: 0,
    earLTgt: 0, earRTgt: 0,
    earTwitchNext: 3000,
    whiskerPhase: 0,
    bodyTilt: 0, bodyTiltTgt: 0,
    headTilt: 0, headTiltTgt: 0,
    stretchProg: -1,  // -1 = not stretching
    yawnProg: -1,
    groomPhase: 0, grooming: false,
  };

  function tickAnim(dt) {
    // ── Tail wag ──
    if (!ST.sleeping) {
      A.tailAngle += A.tailDir * 0.8 * (dt / 16);
      if (A.tailAngle > 14) A.tailDir = -1;
      if (A.tailAngle < -14) A.tailDir = 1;
    } else {
      A.tailAngle *= .95;
    }
    R.tail.setAttribute('transform', `rotate(${A.tailAngle}, 30, 140)`);

    // ── Blink ──
    A.nextBlink -= dt;
    if (A.nextBlink <= 0 && !ST.sleeping && A.yawnProg < 0) {
      A.blinking = true; A.blinkProg = 0;
      A.nextBlink = rand(2500, 5500);
    }
    if (A.blinking) {
      A.blinkProg += dt * .012;
      const ry = A.blinkProg < 1 ? Math.sin(A.blinkProg * Math.PI) * 13 : 0;
      R.blinkL.setAttribute('ry', ry);
      R.blinkR.setAttribute('ry', ry);
      if (A.blinkProg >= 1) {
        A.blinking = false;
        R.blinkL.setAttribute('ry', 0);
        R.blinkR.setAttribute('ry', 0);
        if (Math.random() < .3) A.nextBlink = 200;
      }
    }

    // ── Breathing ──
    A.breathPh += dt * .0015;
    const bScale = 1 + Math.sin(A.breathPh) * .015;
    R.body.setAttribute('ry', 35 * bScale);

    // ── Eye tracking ──
    A.eyeX += (A.tgtEyeX - A.eyeX) * .1;
    A.eyeY += (A.tgtEyeY - A.eyeY) * .1;
    const ox = clamp(A.eyeX * 2.5, -3, 3);
    const oy = clamp(A.eyeY * 2.5, -2.5, 2.5);
    R.pupilL.setAttribute('cx', 56 + ox); R.pupilL.setAttribute('cy', 55 + oy);
    R.pupilR.setAttribute('cx', 94 + ox); R.pupilR.setAttribute('cy', 55 + oy);
    R.irisL.setAttribute('cx', 56 + ox * .6); R.irisL.setAttribute('cy', 55 + oy * .6);
    R.irisR.setAttribute('cx', 94 + ox * .6); R.irisR.setAttribute('cy', 55 + oy * .6);

    // ── Ear rotation ──
    A.earLAngle += (A.earLTgt - A.earLAngle) * .08;
    A.earRAngle += (A.earRTgt - A.earRAngle) * .08;
    R.earL.setAttribute('transform', `rotate(${A.earLAngle}, 44, 28)`);
    R.earR.setAttribute('transform', `rotate(${A.earRAngle}, 106, 28)`);

    // Random ear twitches
    A.earTwitchNext -= dt;
    if (A.earTwitchNext <= 0 && !ST.sleeping) {
      const side = Math.random() < .5 ? 'L' : 'R';
      const angle = rand(-8, 8);
      if (side === 'L') A.earLTgt = angle; else A.earRTgt = angle;
      setTimeout(() => { if (side === 'L') A.earLTgt = 0; else A.earRTgt = 0; }, 200);
      A.earTwitchNext = rand(2000, 6000);
    }

    // Mood-based ear position
    if (ST.mood === 'happy') { A.earLTgt = lerp(A.earLTgt, 3, .02); A.earRTgt = lerp(A.earRTgt, -3, .02); }
    else if (ST.mood === 'sleepy') { A.earLTgt = lerp(A.earLTgt, -4, .01); A.earRTgt = lerp(A.earRTgt, 4, .01); }
    else if (ST.mood === 'hungry') { A.earLTgt = lerp(A.earLTgt, 5, .02); A.earRTgt = lerp(A.earRTgt, -5, .02); }

    // ── Whisker twitch ──
    A.whiskerPhase += dt * .003;
    const wTilt = Math.sin(A.whiskerPhase) * 2;
    R.whiskers.setAttribute('transform', `rotate(${wTilt}, 75, 65)`);

    // ── Head tilt ──
    A.headTilt += (A.headTiltTgt - A.headTilt) * .05;
    R.head.setAttribute('transform', `rotate(${A.headTilt}, 75, 60)`);

    // ── Body tilt (walking lean) ──
    A.bodyTilt += (A.bodyTiltTgt - A.bodyTilt) * .06;

    // ── Walking paws ──
    if (ST.walking) {
      A.walkPh += dt * .008;
      const offL = Math.sin(A.walkPh) * 5;
      const offR = Math.sin(A.walkPh + Math.PI) * 5;
      R.pawL.setAttribute('transform', `translate(52,${160 + offL})`);
      R.pawR.setAttribute('transform', `translate(98,${160 + offR})`);
      R.pawBL.setAttribute('opacity', '.5'); R.pawBL.setAttribute('cy', 164 + offR);
      R.pawBR.setAttribute('opacity', '.5'); R.pawBR.setAttribute('cy', 164 + offL);
      A.bodyTiltTgt = ST.facingRight ? 1.5 : -1.5;
    } else {
      R.pawL.setAttribute('transform', 'translate(52,160)');
      R.pawR.setAttribute('transform', 'translate(98,160)');
      R.pawBL.setAttribute('opacity', '0');
      R.pawBR.setAttribute('opacity', '0');
      A.walkPh = 0;
      A.bodyTiltTgt = 0;
    }

    // ── Stretch animation ──
    if (A.stretchProg >= 0 && A.stretchProg <= 1) {
      A.stretchProg += dt * .002;
      const t = ease(clamp(A.stretchProg, 0, 1));
      // Body elongates, paws extend
      const stretchX = 1 + t * .15;
      const stretchY = 1 - t * .08;
      R.body.setAttribute('rx', 44 * stretchX);
      R.body.setAttribute('ry', 35 * stretchY);
      // Front paws extend forward
      const pawOff = t * 12;
      R.pawL.setAttribute('transform', `translate(${52 - pawOff},${160 + pawOff * .5})`);
      R.pawR.setAttribute('transform', `translate(${98 + pawOff},${160 + pawOff * .5})`);
      // Head tilts up
      A.headTiltTgt = -t * 8;
      // Rear goes up slightly
      if (t > .3) {
        const upT = (t - .3) / .7;
        R.body.setAttribute('cy', 132 - upT * 6);
      }
      if (A.stretchProg > 1) {
        A.stretchProg = -1;
        R.body.setAttribute('rx', 44); R.body.setAttribute('ry', 35);
        R.body.setAttribute('cy', 132);
        R.pawL.setAttribute('transform', 'translate(52,160)');
        R.pawR.setAttribute('transform', 'translate(98,160)');
        A.headTiltTgt = 0;
      }
    }

    // ── Yawn animation ──
    if (A.yawnProg >= 0 && A.yawnProg <= 1) {
      A.yawnProg += dt * .0015;
      const t = ease(clamp(A.yawnProg, 0, 1));
      // Mouth opens wide
      const mouthOpen = t < .5 ? t * 2 : (1 - t) * 2;
      const mh = 6 + mouthOpen * 10;
      R.mouth.setAttribute('d', `M64,69 Q70,${69 + mh} 75,${67 + mh * .8} Q80,${69 + mh} 86,69`);
      R.mouth.setAttribute('stroke-width', 1.8 + mouthOpen);
      // Eyes close
      const eyeClose = t < .3 ? 0 : Math.min((t - .3) / .2, 1) * 13;
      R.blinkL.setAttribute('ry', eyeClose);
      R.blinkR.setAttribute('ry', eyeClose);
      // Head tilts back
      A.headTiltTgt = -t * 6;
      // Ears flatten
      A.earLTgt = -t * 10;
      A.earRTgt = t * 10;
      if (A.yawnProg > 1) {
        A.yawnProg = -1;
        R.mouth.setAttribute('d', 'M66,69 Q70,75 75,69 Q80,75 84,69');
        R.mouth.setAttribute('stroke-width', 1.8);
        R.blinkL.setAttribute('ry', 0); R.blinkR.setAttribute('ry', 0);
        A.headTiltTgt = 0; A.earLTgt = 0; A.earRTgt = 0;
      }
    }

    // ── Groom animation ──
    if (A.grooming) {
      A.groomPhase += dt * .004;
      const t = A.groomPhase % (Math.PI * 2);
      // Head turns to side
      A.headTiltTgt = Math.sin(t) * 12;
      // Tongue appears/disappears
      const tongueVis = Math.sin(t * 2) > 0;
      R.tongue.setAttribute('opacity', tongueVis ? '.8' : '0');
      R.tongue.setAttribute('display', tongueVis ? '' : 'none');
    }
  }

  // ═══════════════════════════════════════════════════════════
  //  BEHAVIOR AI
  // ═══════════════════════════════════════════════════════════
  let mouseX = innerWidth / 2, mouseY = innerHeight / 2;
  document.addEventListener('mousemove', (e) => { mouseX = e.clientX; mouseY = e.clientY; });

  function setBehavior(name, dur, data) {
    // Cleanup previous
    if (ST.behavior === 'grooming') { A.grooming = false; A.groomPhase = 0; R.tongue.setAttribute('opacity', '0'); R.tongue.setAttribute('display', 'none'); A.headTiltTgt = 0; }
    ST.behavior = name;
    ST.behaviorTime = dur || 3000;
    ST.behaviorData = data || {};
  }

  function tickBehavior(dt) {
    if (ST.sleeping || ST.dragging || ST.playing) return;
    ST.behaviorTime -= dt;

    const distToMouse = Math.hypot(mouseX - (ST.x + 75), mouseY - (ST.y + 60));

    switch (ST.behavior) {
      case 'idle': {
        // Look at cursor sometimes
        if (distToMouse < 250 && Math.random() < .01) {
          ST.facingRight = mouseX > ST.x + 75;
          A.tgtEyeX = clamp((mouseX - ST.x - 75) / 200, -1, 1);
          A.tgtEyeY = clamp((mouseY - ST.y - 60) / 200, -1, 1);
        }
        // Random head movements
        if (Math.random() < .005) A.headTiltTgt = rand(-4, 4);
        if (Math.random() < .003) A.headTiltTgt = 0;

        // Decide next behavior
        if (ST.behaviorTime <= 0) {
          const roll = Math.random();
          if (ST.energy < 20 && roll < .3) { toggleSleep(); }
          else if (ST.energy < 40 && roll < .5) { doStretch(); }
          else if (roll < .15) { doYawn(); }
          else if (roll < .3) { doGroom(); }
          else if (roll < .5 && distToMouse < 200) { doFollow(); }
          else if (roll < .65) { doLookAround(); }
          else { doWander(); }
        }
        break;
      }

      case 'walking': {
        if (!ST.walkTarget) { setBehavior('idle', rand(2000, 5000)); break; }
        const dx = ST.walkTarget.x - ST.x;
        const dy = ST.walkTarget.y - ST.y;
        const dist = Math.hypot(dx, dy);
        if (dist < 5 || ST.behaviorTime <= 0) {
          ST.walking = false; ST.walkTarget = null;
          setBehavior('idle', rand(3000, 6000));
          break;
        }
        const spd = ST.walkSpeed * (dt / 16);
        ST.x += (dx / dist) * spd;
        ST.y += (dy / dist) * spd;
        ST.facingRight = dx > 0;
        posCat();
        break;
      }

      case 'following': {
        // Walk toward cursor
        const fx = mouseX - ST.x - 75;
        const fy = mouseY - ST.y - 60;
        const fd = Math.hypot(fx, fy);
        if (fd < 60 || ST.behaviorTime <= 0) {
          ST.walking = false;
          setBehavior('idle', rand(1500, 3000));
          break;
        }
        const fspd = ST.walkSpeed * .8 * (dt / 16);
        ST.x += (fx / fd) * fspd;
        ST.y += (fy / fd) * fspd;
        ST.facingRight = fx > 0;
        ST.walking = true;
        posCat();
        break;
      }

      case 'looking': {
        // Scan around
        const t = ST.behaviorTime;
        const phase = (3000 - t) / 3000;
        ST.behaviorData.lookDir = Math.sin(phase * Math.PI * 4) * 1.5;
        A.tgtEyeX = Math.sin(phase * Math.PI * 6);
        A.tgtEyeY = Math.cos(phase * Math.PI * 3) * .5;
        ST.facingRight = Math.sin(phase * Math.PI * 4) > 0;
        if (ST.behaviorTime <= 0) {
          ST.behaviorData.lookDir = 0;
          A.tgtEyeX = 0; A.tgtEyeY = 0;
          setBehavior('idle', rand(2000, 4000));
        }
        break;
      }

      case 'grooming': {
        A.grooming = true;
        if (ST.behaviorTime <= 0) {
          A.grooming = false; A.groomPhase = 0;
          R.tongue.setAttribute('opacity', '0'); R.tongue.setAttribute('display', 'none');
          A.headTiltTgt = 0;
          setBehavior('idle', rand(2000, 4000));
        }
        break;
      }

      case 'stretching': {
        if (A.stretchProg < 0) { setBehavior('idle', rand(2000, 4000)); break; }
        break;
      }

      case 'yawning': {
        if (A.yawnProg < 0) { setBehavior('idle', rand(2000, 4000)); break; }
        break;
      }
    }
  }

  function doWander() {
    const tx = rand(40, innerWidth - 190);
    const ty = rand(innerHeight - 230, innerHeight - 80);
    ST.walkTarget = { x: tx, y: ty };
    ST.walking = true;
    ST.walkSpeed = rand(1, 1.8);
    setBehavior('walking', 8000);
  }

  function doFollow() {
    setBehavior('following', 4000);
    ST.walking = true;
    say(pick(['跟你好近~', '你在看什么？', '喵？好奇~']), 1500);
  }

  function doLookAround() {
    setBehavior('looking', rand(2500, 4000));
    say(pick(['看看这边…', '那边有什么？', '观察中…']), 1500);
  }

  function doStretch() {
    A.stretchProg = 0;
    setBehavior('stretching', 2000);
    say(pick(talk.stretch), 2000);
  }

  function doYawn() {
    A.yawnProg = 0;
    setBehavior('yawning', 2500);
    say(pick(['啊~（打哈欠）', '好困…💤']), 2000);
  }

  function doGroom() {
    setBehavior('grooming', rand(2500, 4500));
    say(pick(talk.groom), 2000);
  }

  // ═══════════════════════════════════════════════════════════
  //  Zzz
  // ═══════════════════════════════════════════════════════════
  const zzzBox = document.createElement('div');
  zzzBox.style.cssText = 'position:absolute;top:-20px;right:-20px;pointer-events:none;display:none;';
  catWrap.appendChild(zzzBox);
  let lastZzz = 0;
  const zzzCSS = document.createElement('style');
  zzzCSS.textContent = '@keyframes zUp{0%{opacity:1;transform:translateY(0) scale(1)}100%{opacity:0;transform:translateY(-30px) scale(1.3)}}';
  document.head.appendChild(zzzCSS);

  function spawnZzz() {
    if (!ST.sleeping) return;
    const z = document.createElement('div');
    z.textContent = 'Z';
    z.style.cssText = `position:absolute;font-size:${rand(11,17)}px;color:#8b5cf6;font-weight:bold;animation:zUp 1.4s ease-out forwards;right:${rand(0,18)}px;top:0;`;
    zzzBox.appendChild(z);
    setTimeout(() => z.remove(), 1500);
  }

  // ═══════════════════════════════════════════════════════════
  //  STATUS BAR
  // ═══════════════════════════════════════════════════════════
  const bar = document.createElement('div');
  bar.style.cssText = 'position:fixed;top:12px;left:50%;transform:translateX(-50%);display:flex;gap:12px;background:rgba(255,255,255,0.85);backdrop-filter:blur(12px);border-radius:16px;padding:8px 20px;box-shadow:0 2px 16px rgba(0,0,0,0.08);z-index:9999;font-size:12px;align-items:center;border:1px solid rgba(255,255,255,0.6);';
  root.appendChild(bar);

  function drawBar() {
    const me = { happy:'😊', neutral:'😐', sleepy:'😴', hungry:'🥺' }[ST.mood];
    const ec = ST.energy > 60 ? '#22c55e' : ST.energy > 30 ? '#f59e0b' : '#ef4444';
    const hc = ST.hunger > 60 ? '#22c55e' : ST.hunger > 30 ? '#f59e0b' : '#ef4444';
    const bBar = (v, c) => `<span style="display:inline-block;width:48px;height:6px;background:#e5e7eb;border-radius:3px;vertical-align:middle;overflow:hidden"><span style="display:block;width:${v}%;height:100%;background:${c};border-radius:3px;transition:width .3s"></span></span>`;
    bar.innerHTML = `
      <span style="font-weight:600;color:#6d28d9">🐾 Mochi</span>
      <span style="color:#9ca3af">|</span>
      <span>${me} ${ST.mood}</span>
      <span style="color:#9ca3af">|</span>
      <span>⚡ ${bBar(ST.energy, ec)} ${Math.round(ST.energy)}%</span>
      <span>🍖 ${bBar(ST.hunger, hc)} ${Math.round(ST.hunger)}%</span>
      <span style="color:#9ca3af">|</span>
      <span>💕 ${ST.petCount}</span>`;
  }

  // ═══════════════════════════════════════════════════════════
  //  SPEECH BUBBLE
  // ═══════════════════════════════════════════════════════════
  const bubble = document.createElement('div');
  bubble.style.cssText = 'position:fixed;z-index:9998;background:#fff;border-radius:16px;padding:8px 16px;font-size:13px;color:#374151;box-shadow:0 4px 20px rgba(0,0,0,0.1);pointer-events:none;opacity:0;transition:opacity .3s,transform .3s;max-width:200px;text-align:center;border:1px solid rgba(0,0,0,0.05);white-space:pre-line;';
  root.appendChild(bubble);

  function say(text, ms) {
    clearTimeout(ST.chatTimer);
    bubble.textContent = text;
    bubble.style.opacity = '1';
    bubble.style.left = (ST.x + 75) + 'px';
    bubble.style.top = (ST.y - 45) + 'px';
    bubble.style.transform = 'translate(-50%,0) scale(1)';
    ST.chatTimer = setTimeout(() => { bubble.style.opacity = '0'; bubble.style.transform = 'translate(-50%,8px) scale(.9)'; }, ms || 2500);
  }
  function randomSay() { say(pick(talk[ST.mood] || talk.neutral)); }

  // ═══════════════════════════════════════════════════════════
  //  CONTEXT MENU
  // ═══════════════════════════════════════════════════════════
  const menu = document.createElement('div');
  menu.style.cssText = 'position:fixed;z-index:10001;background:rgba(255,255,255,0.95);backdrop-filter:blur(12px);border-radius:12px;padding:6px;box-shadow:0 8px 32px rgba(0,0,0,0.12);display:none;min-width:170px;border:1px solid rgba(0,0,0,0.06);';
  root.appendChild(menu);

  const menuDef = [
    { ic:'🤚', lb:'Pet Mochi',      fn:petCat },
    { ic:'👃', lb:'Boop nose',       fn:boopNose },
    { ic:'🍗', lb:'Feed treat',      fn:feedCat },
    { ic:'🎮', lb:'Play (catch)',    fn:startGame },
    { ic:'😴', lb:'Sleep / Wake',    fn:toggleSleep },
    { ic:'🦴', lb:'Stretch',         fn:doStretch },
    { ic:'🪥', lb:'Groom',           fn:doGroom },
    { ic:'🚶', lb:'Go for a walk',   fn:() => { doWander(); } },
    { ic:'💭', lb:'Say something',   fn:randomSay },
    { ic:'🧹', lb:'Clear particles', fn:() => { particles.forEach(p => p.el.remove()); particles.length = 0; } },
  ];

  function showMenu(mx, my) {
    ST.menuOpen = true;
    menu.style.display = 'block';
    menu.style.left = clamp(mx, 10, innerWidth - 190) + 'px';
    menu.style.top = clamp(my, 10, innerHeight - menu.offsetHeight - 10) + 'px';
    menu.innerHTML = menuDef.map((m, i) => `
      <div data-i="${i}" style="display:flex;align-items:center;gap:8px;padding:8px 12px;border-radius:8px;cursor:pointer;font-size:13px;color:#374151;transition:background .15s">
        <span style="font-size:15px">${m.ic}</span><span>${m.lb}</span></div>`).join('');
    menu.querySelectorAll('[data-i]').forEach(d => {
      d.onmouseenter = () => d.style.background = 'rgba(109,40,217,0.08)';
      d.onmouseleave = () => d.style.background = '';
      d.onclick = (e) => { e.stopPropagation(); menuDef[+d.dataset.i].fn(); hideMenu(); };
    });
  }
  function hideMenu() { ST.menuOpen = false; menu.style.display = 'none'; }

  // ═══════════════════════════════════════════════════════════
  //  MOOD
  // ═══════════════════════════════════════════════════════════
  function updateMood() {
    if (ST.sleeping) { ST.mood = 'sleepy'; return; }
    if (ST.hunger < 25) { ST.mood = 'hungry'; return; }
    if (ST.energy < 30) { ST.mood = 'sleepy'; return; }
    ST.mood = (ST.petCount > 5 || (ST.hunger > 60 && ST.energy > 60)) ? 'happy' : 'neutral';
  }

  // ═══════════════════════════════════════════════════════════
  //  INTERACTIONS
  // ═══════════════════════════════════════════════════════════
  function petCat() {
    ST.petCount++;
    R.blushL.setAttribute('opacity', '.7');
    R.blushR.setAttribute('opacity', '.7');
    R.mouth.setAttribute('d', 'M64,68 Q70,77 75,68 Q80,77 86,68');
    say(pick(talk.petted), 2000);
    for (let i = 0; i < 6; i++) setTimeout(() => emit(ST.x + 75 + rand(-30,30), ST.y - 10 + rand(-20,20), 'heart'), i * 80);
    // Happy ear wiggle
    A.earLTgt = -6; A.earRTgt = 6;
    setTimeout(() => { A.earLTgt = 0; A.earRTgt = 0; }, 600);
    ST.energy = clamp(ST.energy + 2, 0, 100);
    ST.hunger = clamp(ST.hunger - 1, 0, 100);
    updateMood(); drawBar();
    setTimeout(() => { R.blushL.setAttribute('opacity', '0'); R.blushR.setAttribute('opacity', '0'); R.mouth.setAttribute('d', 'M66,69 Q70,75 75,69 Q80,75 84,69'); }, 1800);
  }

  function boopNose() {
    say(pick(talk.boop), 1800);
    let sh = 0;
    const si = setInterval(() => {
      catWrap.style.transform = `scaleX(${ST.facingRight?1:-1}) translateX(${sh%2?5:-5}px)`;
      sh++;
      if (sh > 6) { clearInterval(si); posCat(); }
    }, 50);
    // Startled ears
    A.earLTgt = -12; A.earRTgt = 12;
    setTimeout(() => { A.earLTgt = 0; A.earRTgt = 0; }, 800);
    for (let i = 0; i < 3; i++) setTimeout(() => emit(ST.x + 75, ST.y - 10, 'sparkle'), i * 100);
  }

  function feedCat() {
    ST.hunger = clamp(ST.hunger + 25, 0, 100);
    say(pick(talk.fed), 2200);
    let ch = 0;
    const ci = setInterval(() => {
      R.mouth.setAttribute('d', ch % 2 === 0
        ? 'M66,69 Q70,75 75,69 Q80,75 84,69'
        : 'M66,69 Q70,73 75,71 Q80,73 84,69');
      ch++;
      if (ch > 6) { clearInterval(ci); R.mouth.setAttribute('d', 'M66,69 Q70,75 75,69 Q80,75 84,69'); }
    }, 180);
    for (let i = 0; i < 4; i++) setTimeout(() => emit(ST.x + 75 + rand(-20,20), ST.y + 10, 'food'), i * 130);
    // Happy ears
    A.earLTgt = -5; A.earRTgt = 5;
    setTimeout(() => { A.earLTgt = 0; A.earRTgt = 0; }, 1000);
    updateMood(); drawBar();
  }

  function toggleSleep() {
    ST.sleeping = !ST.sleeping;
    if (ST.sleeping) {
      ST.walking = false; ST.walkTarget = null;
      setBehavior('sleeping', Infinity);
      R.eyeL.style.display = 'none'; R.eyeR.style.display = 'none';
      R.blinkL.style.display = 'none'; R.blinkR.style.display = 'none';
      R.sleepL.removeAttribute('display'); R.sleepR.removeAttribute('display');
      zzzBox.style.display = 'block';
      // Ears flatten when sleeping
      A.earLTgt = -6; A.earRTgt = 6;
      say('💤 Zzz...', 2000);
    } else {
      ST.walking = false;
      setBehavior('idle', 1000);
      R.eyeL.style.display = ''; R.eyeR.style.display = '';
      R.blinkL.style.display = ''; R.blinkR.style.display = '';
      R.sleepL.setAttribute('display', 'none'); R.sleepR.setAttribute('display', 'none');
      zzzBox.style.display = 'none';
      A.earLTgt = 0; A.earRTgt = 0;
      // Auto-stretch after waking
      setTimeout(doStretch, 300);
      say(pick(['早安~ ☀️', '睡醒了！', '喵！活力满满！']), 2000);
    }
    updateMood(); drawBar();
  }

  // ═══════════════════════════════════════════════════════════
  //  MINI GAME
  // ═══════════════════════════════════════════════════════════
  let gameOn = false, gameCur = null, gameScore = 0, gameTimer = null;

  function startGame() {
    if (gameOn) return;
    if (ST.sleeping) { say('太困了不想玩…💤', 2000); return; }
    gameOn = true; gameScore = 0; ST.playing = true;
    setBehavior('playing', Infinity);
    say('来抓我呀~ 🐭', 2000);
    gameCur = document.createElement('div');
    gameCur.textContent = '🐭';
    gameCur.style.cssText = 'position:fixed;width:28px;height:28px;z-index:9995;pointer-events:none;display:flex;align-items:center;justify-content:center;font-size:22px;filter:drop-shadow(0 2px 4px rgba(0,0,0,.2));';
    root.appendChild(gameCur);
    moveGameCur();
    gameTimer = setTimeout(endGame, 15000);
  }

  function moveGameCur() {
    if (!gameOn || !gameCur) return;
    gameCur.style.left = rand(40, innerWidth - 60) + 'px';
    gameCur.style.top = rand(40, innerHeight - 60) + 'px';
    gameCur.style.transition = `left ${rand(.8,2)}s ease-in-out, top ${rand(.8,2)}s ease-in-out`;
    setTimeout(moveGameCur, rand(800, 2000));
  }

  function catchCursor() {
    if (!gameOn) return;
    gameScore++;
    say(`抓到了！×${gameScore} 🎉`, 1000);
    for (let i = 0; i < 6; i++) emit(ST.x + 75, ST.y - 10, 'star');
    ST.energy = clamp(ST.energy - 2, 0, 100);
    moveGameCur();
  }

  function endGame() {
    gameOn = false; ST.playing = false;
    clearTimeout(gameTimer);
    if (gameCur) { gameCur.remove(); gameCur = null; }
    setBehavior('idle', 2000);
    say(gameScore > 0 ? `游戏结束！抓了 ${gameScore} 次 🏆` : '下次再来玩吧~', 3000);
    drawBar();
  }

  // ═══════════════════════════════════════════════════════════
  //  DRAG & DROP
  // ═══════════════════════════════════════════════════════════
  let dragOff = { x:0, y:0 };

  catWrap.addEventListener('mousedown', (e) => {
    e.preventDefault();
    if (ST.sleeping) { toggleSleep(); return; }
    if (gameOn) { catchCursor(); return; }
    ST.dragging = true; ST.walking = false; ST.walkTarget = null;
    setBehavior('dragged', Infinity);
    dragOff.x = e.clientX - ST.x; dragOff.y = e.clientY - ST.y;
    catWrap.style.cursor = 'grabbing';
    bubble.style.opacity = '0';
  });

  document.addEventListener('mousemove', (e) => {
    if (ST.dragging) {
      ST.x = e.clientX - dragOff.x; ST.y = e.clientY - dragOff.y;
      ST.facingRight = e.movementX >= 0;
      posCat();
    }
    if (!ST.sleeping) {
      const hx = ST.x + 75, hy = ST.y + 55;
      A.tgtEyeX = clamp((e.clientX - hx) / 300, -1, 1);
      A.tgtEyeY = clamp((e.clientY - hy) / 300, -1, 1);
    }
  });

  document.addEventListener('mouseup', () => {
    if (!ST.dragging) return;
    ST.dragging = false; catWrap.style.cursor = 'grab';
    ST.y = clamp(ST.y, innerHeight - 240, innerHeight - 60);
    ST.x = clamp(ST.x, 10, innerWidth - 160);
    posCat();
    setBehavior('idle', rand(1000, 2000));
  });

  // ═══════════════════════════════════════════════════════════
  //  CLICK / KEYBOARD
  // ═══════════════════════════════════════════════════════════
  let lastTap = 0;
  catWrap.addEventListener('click', (e) => {
    e.stopPropagation();
    const now = Date.now();
    if (now - lastTap < 300) boopNose(); else petCat();
    lastTap = now;
  });

  catWrap.addEventListener('contextmenu', (e) => { e.preventDefault(); e.stopPropagation(); showMenu(e.clientX, e.clientY); });
  root.addEventListener('contextmenu', (e) => e.preventDefault());
  document.addEventListener('click', (e) => { if (ST.menuOpen && !menu.contains(e.target)) hideMenu(); });

  document.addEventListener('keydown', (e) => {
    const k = e.key.toLowerCase();
    if (k === 'p') petCat();
    if (k === 'f') feedCat();
    if (k === 's') toggleSleep();
    if (k === 'w') doWander();
    if (k === 'r') doStretch();
    if (k === 'g') doGroom();
    if (k === ' ') { e.preventDefault(); randomSay(); }
  });

  window.addEventListener('resize', () => {
    ST.x = clamp(ST.x, 10, innerWidth - 160);
    ST.y = clamp(ST.y, innerHeight - 240, innerHeight - 60);
    posCat();
  });

  // ═══════════════════════════════════════════════════════════
  //  STAT DECAY
  // ═══════════════════════════════════════════════════════════
  setInterval(() => {
    ST.hunger = clamp(ST.hunger - .5, 0, 100);
    ST.energy = clamp(ST.energy - (ST.sleeping ? -.8 : .3), 0, 100);
    updateMood(); drawBar();
    if (ST.hunger < 15 && Math.random() < .3) say('饿得不行了…🍗', 2000);
    if (ST.energy < 10 && !ST.sleeping && Math.random() < .3) { say('好困…💤', 2000); if (Math.random() < .4) toggleSleep(); }
  }, 5000);

  // ═══════════════════════════════════════════════════════════
  //  MAIN LOOP
  // ═══════════════════════════════════════════════════════════
  let prevT = performance.now();
  function loop(now) {
    const dt = Math.min(now - prevT, 50);
    prevT = now;
    tickAnim(dt);
    tickParticles();
    tickBehavior(dt);
    if (ST.sleeping && now - lastZzz > 2200) { spawnZzz(); lastZzz = now; }
    requestAnimationFrame(loop);
  }
  requestAnimationFrame(loop);

  // ─── Init ──────────────────────────────────────────────────
  setTimeout(() => say('你好呀~ 我是 Mochi！🐾\n点击=摸 | 右键=菜单\nP=摸 F=喂 S=睡 R=伸展 G=梳理', 5000), 600);
  drawBar();

})();
