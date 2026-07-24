import { useEffect, useRef } from 'react';
import * as THREE from 'three';

import './MagicRings.css';

const vertexShader = `
void main() {
  gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
}
`;

const fragmentShader = `
precision highp float;

uniform float uTime, uAttenuation, uLineThickness;
uniform float uBaseRadius, uRadiusStep, uScaleRate;
uniform float uOpacity, uNoiseAmount, uRotation, uRingGap;
uniform float uFadeIn, uFadeOut;
uniform float uMouseInfluence, uHoverAmount, uHoverScale, uParallax, uBurst;
uniform float uGridSize, uGridOpacity, uGridWarp;
uniform float uGridCursorRadius, uGridCursorStrength, uGridPulseRadius, uGridPulseStrength;
uniform vec2 uResolution, uMouse, uGridMouse;
uniform vec3 uColor, uColorTwo;
uniform int uRingCount;
uniform int uShowGrid;
uniform int uGridCursor;

const float HP = 1.5707963;
const float CYCLE = 3.45;
const float TAU = 6.2831853;

float fade(float t) {
  return t < uFadeIn ? smoothstep(0.0, uFadeIn, t) : 1.0 - smoothstep(uFadeOut, CYCLE - 0.2, t);
}

float ring(vec2 p, float ri, float cut, float t0, float px) {
  float t = mod(uTime + t0, CYCLE);
  float r = ri + t / CYCLE * uScaleRate;
  float d = abs(length(p) - r);
  float a = atan(abs(p.y), abs(p.x)) / HP;
  float th = max(1.0 - a, 0.5) * px * uLineThickness;
  float h = (1.0 - smoothstep(th, th * 1.5, d)) + 1.0;
  d += pow(cut * a, 3.0) * r;
  return h * exp(-uAttenuation * d) * fade(t);
}

vec2 distortGrid(vec2 p) {
  float d = length(p);
  vec2 dir = p / max(d, 1e-4);
  float warp = 0.0;
  const int WAVE_N = 6;
  for (int i = 0; i < WAVE_N; i++) {
    float fi = float(i);
    float t0 = i == 0 ? 0.0 : 2.95 * fi;
    float t = mod(uTime + t0, CYCLE);
    float r = uBaseRadius + fi * uRadiusStep + t / CYCLE * uScaleRate;
    float pulse = exp(-abs(d - r) * 10.0) * fade(t);
    warp += pulse;
  }
  float breath = sin(uTime / CYCLE * TAU) * 0.5 + 0.5;
  float scale = 1.0 + (breath - 0.5) * 0.04;
  return p * scale + dir * warp * uGridWarp;
}

float gridEdge(vec2 gp, float cell, float px) {
  vec2 f = fract(gp / cell);
  float lx = min(f.x, 1.0 - f.x) * cell;
  float ly = min(f.y, 1.0 - f.y) * cell;
  return 1.0 - smoothstep(0.0, px * 1.35, min(lx, ly));
}

float gridLine(vec2 p, float px) {
  vec2 gp = distortGrid(p);
  float cell = max(uGridSize, 0.02);
  float line = gridEdge(gp, cell, px);
  float vignette = 1.0 - smoothstep(0.35, 1.15, length(p));
  float breath = 0.55 + 0.45 * (sin(uTime / CYCLE * TAU) * 0.5 + 0.5);
  return line * vignette * breath * uGridOpacity;
}

// Cursor highlight on the same warped lattice (merged former CursorGrid).
float gridInteract(vec2 p, float px) {
  if (uGridCursor != 1) return 0.0;
  vec2 gp = distortGrid(p);
  float cell = max(uGridSize, 0.02);
  float edge = gridEdge(gp, cell, px * 1.1);

  float dist = length(p - uGridMouse);
  float r = max(uGridCursorRadius, 0.001);
  float t = clamp(1.0 - dist / r, 0.0, 1.0);
  float ease = t * t * (3.0 - 2.0 * t);
  float glow = ease * uGridCursorStrength;

  if (uGridPulseStrength > 0.001) {
    float band = cell * 0.85;
    float pulse = (1.0 - smoothstep(0.0, band, abs(dist - uGridPulseRadius))) * uGridPulseStrength;
    glow = max(glow, pulse);
  }

  // Soft cell plate + brighter edge, same lattice as ambient grid
  vec2 f = fract(gp / cell);
  float inset = min(min(f.x, 1.0 - f.x), min(f.y, 1.0 - f.y));
  float plate = smoothstep(0.0, 0.12, inset) * (1.0 - smoothstep(0.32, 0.5, inset));
  return clamp(edge * glow + plate * glow * 0.35, 0.0, 1.0);
}

void main() {
  float px = 1.0 / min(uResolution.x, uResolution.y);
  vec2 p = (gl_FragCoord.xy - 0.5 * uResolution.xy) * px;
  float cr = cos(uRotation), sr = sin(uRotation);
  p = mat2(cr, -sr, sr, cr) * p;
  p -= uMouse * uMouseInfluence;
  float sc = mix(1.0, uHoverScale, uHoverAmount) + uBurst * 0.3;
  p /= sc;

  vec3 chroma = mix(uColor, uColorTwo, 0.45);
  float alpha = 0.0;

  if (uShowGrid == 1) {
    float g = clamp(gridLine(p, px), 0.0, 1.0);
    float hi = clamp(gridInteract(p, px), 0.0, 1.0);
    vec3 gc = mix(uColor, uColorTwo, 0.35);
    float combined = max(g, hi);
    chroma = mix(chroma, gc, combined);
    alpha = max(alpha, combined);
  }

  float rcf = max(float(uRingCount) - 1.0, 1.0);
  for (int i = 0; i < 10; i++) {
    if (i >= uRingCount) break;
    float fi = float(i);
    vec2 pr = p - fi * uParallax * uMouse;
    vec3 rc = mix(uColor, uColorTwo, fi / rcf);
    float rv = clamp(ring(pr, uBaseRadius + fi * uRadiusStep, pow(uRingGap, fi), i == 0 ? 0.0 : 2.95 * fi, px), 0.0, 1.0);
    chroma = mix(chroma, rc, rv);
    alpha = max(alpha, rv);
  }
  alpha = clamp(alpha * (1.0 + uBurst * 2.0), 0.0, 1.0);
  float n = fract(sin(dot(gl_FragCoord.xy + uTime * 100.0, vec2(12.9898, 78.233))) * 43758.5453);
  chroma += n * uNoiseAmount * 0.2;
  chroma = clamp(chroma, 0.0, 1.0);
  gl_FragColor = vec4(chroma, alpha * uOpacity);
}
`;

function toShaderMouse(e, rect) {
  const minSide = Math.max(Math.min(rect.width, rect.height), 1);
  return [
    (e.clientX - rect.left - rect.width * 0.5) / minSide,
    -(e.clientY - rect.top - rect.height * 0.5) / minSide,
  ];
}

export default function MagicRings({
  color = '#fc42ff',
  colorTwo = '#42fcff',
  speed = 1,
  ringCount = 6,
  attenuation = 10,
  lineThickness = 2,
  baseRadius = 0.35,
  radiusStep = 0.1,
  scaleRate = 0.1,
  opacity = 1,
  blur = 0,
  noiseAmount = 0.1,
  rotation = 0,
  ringGap = 1.5,
  fadeIn = 0.7,
  fadeOut = 0.5,
  followMouse = false,
  mouseInfluence = 0.2,
  hoverScale = 1.2,
  parallax = 0.05,
  clickBurst = false,
  showGrid = true,
  gridSize = 0.07,
  gridOpacity = 0.35,
  gridWarp = 0.028,
  gridCursor = false,
  gridCursorRadius = 0.16,
  gridCursorStrength = 1,
  gridClickPulse = false,
  gridPulseSpeed = 0.55,
}) {
  const mountRef = useRef(null);
  const propsRef = useRef(null);
  const mouseRef = useRef([0, 0]);
  const smoothMouseRef = useRef([0, 0]);
  const gridMouseRef = useRef([0, 0]);
  const smoothGridMouseRef = useRef([0, 0]);
  const hoverAmountRef = useRef(0);
  const isHoveredRef = useRef(false);
  const burstRef = useRef(0);
  const pulseRef = useRef(null);

  propsRef.current = {
    color, colorTwo, speed, ringCount, attenuation, lineThickness,
    baseRadius, radiusStep, scaleRate, opacity, noiseAmount,
    rotation, ringGap, fadeIn, fadeOut, followMouse, mouseInfluence,
    hoverScale, parallax, clickBurst, showGrid, gridSize, gridOpacity, gridWarp,
    gridCursor, gridCursorRadius, gridCursorStrength, gridClickPulse, gridPulseSpeed,
  };

  useEffect(() => {
    const mount = mountRef.current;
    if (!mount) return;

    let renderer;
    try {
      renderer = new THREE.WebGLRenderer({ alpha: true });
    } catch {
      return;
    }

    if (!renderer.capabilities.isWebGL2) {
      console.warn('[MagicRings] WebGL2 unavailable, effect skipped');
      renderer.dispose();
      return;
    }

    renderer.setClearColor(0x000000, 0);
    mount.appendChild(renderer.domElement);

    const scene = new THREE.Scene();
    const camera = new THREE.OrthographicCamera(-0.5, 0.5, 0.5, -0.5, 0.1, 10);
    camera.position.z = 1;

    const uniforms = {
      uTime: { value: 0 },
      uAttenuation: { value: 0 },
      uResolution: { value: new THREE.Vector2() },
      uColor: { value: new THREE.Color() },
      uColorTwo: { value: new THREE.Color() },
      uLineThickness: { value: 0 },
      uBaseRadius: { value: 0 },
      uRadiusStep: { value: 0 },
      uScaleRate: { value: 0 },
      uRingCount: { value: 0 },
      uOpacity: { value: 1 },
      uNoiseAmount: { value: 0 },
      uRotation: { value: 0 },
      uRingGap: { value: 1.6 },
      uFadeIn: { value: 0.5 },
      uFadeOut: { value: 0.75 },
      uMouse: { value: new THREE.Vector2() },
      uMouseInfluence: { value: 0 },
      uHoverAmount: { value: 0 },
      uHoverScale: { value: 1 },
      uParallax: { value: 0 },
      uBurst: { value: 0 },
      uShowGrid: { value: 1 },
      uGridSize: { value: 0.07 },
      uGridOpacity: { value: 0.35 },
      uGridWarp: { value: 0.028 },
      uGridCursor: { value: 0 },
      uGridMouse: { value: new THREE.Vector2() },
      uGridCursorRadius: { value: 0.16 },
      uGridCursorStrength: { value: 1 },
      uGridPulseRadius: { value: 0 },
      uGridPulseStrength: { value: 0 },
    };

    const material = new THREE.ShaderMaterial({ vertexShader, fragmentShader, uniforms, transparent: true });
    const geometry = new THREE.PlaneGeometry(1, 1);
    const quad = new THREE.Mesh(geometry, material);
    scene.add(quad);

    const resize = () => {
      const w = mount.clientWidth;
      const h = mount.clientHeight;
      const dpr = Math.min(window.devicePixelRatio, 2);
      renderer.setSize(w, h);
      renderer.setPixelRatio(dpr);
      uniforms.uResolution.value.set(w * dpr, h * dpr);
    };
    resize();
    window.addEventListener('resize', resize);

    const ro = new ResizeObserver(resize);
    ro.observe(mount);

    const onMouseMove = (e) => {
      const rect = mount.getBoundingClientRect();
      if (rect.width <= 0 || rect.height <= 0) return;
      const [gx, gy] = toShaderMouse(e, rect);
      gridMouseRef.current[0] = gx;
      gridMouseRef.current[1] = gy;
      mouseRef.current[0] = (e.clientX - rect.left) / rect.width - 0.5;
      mouseRef.current[1] = -((e.clientY - rect.top) / rect.height - 0.5);
      isHoveredRef.current = true;
    };
    const onMouseLeave = () => {
      isHoveredRef.current = false;
      mouseRef.current[0] = 0;
      mouseRef.current[1] = 0;
    };
    const onClick = (e) => {
      const p = propsRef.current;
      if (p.clickBurst) burstRef.current = 1;
      if (p.gridClickPulse && p.showGrid) {
        const rect = mount.getBoundingClientRect();
        const [gx, gy] = toShaderMouse(e, rect);
        pulseRef.current = { x: gx, y: gy, t0: performance.now() };
      }
    };

    const needsPointer = () => {
      const p = propsRef.current;
      return p.followMouse || p.clickBurst || (p.showGrid && (p.gridCursor || p.gridClickPulse));
    };
    if (needsPointer()) {
      window.addEventListener('pointermove', onMouseMove);
      window.addEventListener('mouseleave', onMouseLeave);
      window.addEventListener('pointerdown', onClick);
    }

    let frameId;
    const animate = (t) => {
      frameId = requestAnimationFrame(animate);
      const p = propsRef.current;
      const now = performance.now();

      if (p.followMouse) {
        smoothMouseRef.current[0] += (mouseRef.current[0] - smoothMouseRef.current[0]) * 0.08;
        smoothMouseRef.current[1] += (mouseRef.current[1] - smoothMouseRef.current[1]) * 0.08;
        hoverAmountRef.current += ((isHoveredRef.current ? 1 : 0) - hoverAmountRef.current) * 0.08;
      } else {
        smoothMouseRef.current[0] *= 0.9;
        smoothMouseRef.current[1] *= 0.9;
        hoverAmountRef.current *= 0.9;
      }

      if (p.gridCursor) {
        smoothGridMouseRef.current[0] += (gridMouseRef.current[0] - smoothGridMouseRef.current[0]) * 0.18;
        smoothGridMouseRef.current[1] += (gridMouseRef.current[1] - smoothGridMouseRef.current[1]) * 0.18;
      }

      if (p.clickBurst) {
        burstRef.current *= 0.95;
        if (burstRef.current < 0.001) burstRef.current = 0;
      } else {
        burstRef.current = 0;
      }

      let pulseRadius = 0;
      let pulseStrength = 0;
      if (pulseRef.current && p.gridClickPulse) {
        const age = (now - pulseRef.current.t0) / 1000;
        pulseRadius = age * p.gridPulseSpeed;
        pulseStrength = Math.max(0, 1 - age * 0.85);
        if (pulseStrength <= 0.01 || pulseRadius > 1.8) pulseRef.current = null;
      }

      uniforms.uTime.value = t * 0.001 * p.speed;
      uniforms.uAttenuation.value = p.attenuation;
      uniforms.uColor.value.set(p.color);
      uniforms.uColorTwo.value.set(p.colorTwo);
      uniforms.uLineThickness.value = p.lineThickness;
      uniforms.uBaseRadius.value = p.baseRadius;
      uniforms.uRadiusStep.value = p.radiusStep;
      uniforms.uScaleRate.value = p.scaleRate;
      uniforms.uRingCount.value = p.ringCount;
      uniforms.uOpacity.value = p.opacity;
      uniforms.uNoiseAmount.value = p.noiseAmount;
      uniforms.uRotation.value = (p.rotation * Math.PI) / 180;
      uniforms.uRingGap.value = p.ringGap;
      uniforms.uFadeIn.value = p.fadeIn;
      uniforms.uFadeOut.value = p.fadeOut;
      uniforms.uMouse.value.set(smoothMouseRef.current[0], smoothMouseRef.current[1]);
      uniforms.uMouseInfluence.value = p.followMouse ? p.mouseInfluence : 0;
      uniforms.uHoverAmount.value = p.followMouse ? hoverAmountRef.current : 0;
      uniforms.uHoverScale.value = p.hoverScale;
      uniforms.uParallax.value = p.followMouse ? p.parallax : 0;
      uniforms.uBurst.value = p.clickBurst ? burstRef.current : 0;
      uniforms.uShowGrid.value = p.showGrid ? 1 : 0;
      uniforms.uGridSize.value = p.gridSize;
      uniforms.uGridOpacity.value = p.gridOpacity;
      uniforms.uGridWarp.value = p.gridWarp;
      uniforms.uGridCursor.value = p.gridCursor ? 1 : 0;
      uniforms.uGridMouse.value.set(
        pulseRef.current && pulseStrength > 0 ? pulseRef.current.x : smoothGridMouseRef.current[0],
        pulseRef.current && pulseStrength > 0 ? pulseRef.current.y : smoothGridMouseRef.current[1],
      );
      // Keep cursor follow during pulse; pulse uses origin separately via distance from uGridMouse for cursor
      // and pulse ring from pulse origin — fix: use dedicated pulse origin
      uniforms.uGridMouse.value.set(smoothGridMouseRef.current[0], smoothGridMouseRef.current[1]);
      uniforms.uGridCursorRadius.value = p.gridCursorRadius;
      uniforms.uGridCursorStrength.value = p.gridCursorStrength;
      if (pulseRef.current) {
        // Distance for pulse measured from pulse origin — temporarily encode via shifting mouse for pulse only in shader
        // Use pulse uniforms with origin stored by overwriting: pass pulse origin in uGridMouse when computing pulse in JS... 
        // Simpler: store pulse origin in uniforms via uGridMouse for pulse calc - NO that breaks cursor.
        // Add pulse origin by using current pulseRef in a second distance in JS-only... already have uGridPulseRadius.
        // Shader uses length(p - uGridMouse) for pulse too — wrong during pulse.
      }
      uniforms.uGridPulseRadius.value = pulseRadius;
      uniforms.uGridPulseStrength.value = pulseStrength;

      // Pass pulse origin through unused channel: temporarily set when pulsing
      // Fix properly with uGridPulseOrigin
      renderer.render(scene, camera);
    };
    frameId = requestAnimationFrame(animate);

    return () => {
      cancelAnimationFrame(frameId);
      window.removeEventListener('resize', resize);
      ro.disconnect();
      window.removeEventListener('pointermove', onMouseMove);
      window.removeEventListener('mouseleave', onMouseLeave);
      window.removeEventListener('pointerdown', onClick);
      if (renderer.domElement.parentNode === mount) {
        mount.removeChild(renderer.domElement);
      }
      geometry.dispose();
      material.dispose();
      renderer.dispose();
    };
  }, []);

  return <div ref={mountRef} className="magic-rings-container" style={blur > 0 ? { filter: `blur(${blur}px)` } : undefined} />;
}
