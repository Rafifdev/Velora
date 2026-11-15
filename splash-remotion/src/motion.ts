// Velora splash — physically-based motion model (pure functions of time in seconds).
// The final frame resolves to the exact VeloraLogo geometry (see VeloraLogo.svelte).

export type Pt = [number, number];
export type Seg = [Pt, Pt, Pt, Pt];

export const lerp = (a: number, b: number, t: number) => a + (b - a) * t;
export const clamp01 = (v: number) => Math.max(0, Math.min(1, v));

export const easeInOutCubic = (x: number) =>
  x < 0.5 ? 4 * x * x * x : 1 - Math.pow(-2 * x + 2, 3) / 2;
export const easeOutCubic = (x: number) => 1 - Math.pow(1 - x, 3);
export const easeOutQuart = (x: number) => 1 - Math.pow(1 - x, 4);
export const easeInOutSine = (x: number) => -(Math.cos(Math.PI * x) - 1) / 2;

/** Closed-form damped spring step response 0 -> 1 (may overshoot when zeta < 1). */
export function springStep(t: number, omega: number, zeta: number): number {
  if (t <= 0) return 0;
  const wd = omega * Math.sqrt(1 - zeta * zeta);
  return 1 - Math.exp(-zeta * omega * t) * (Math.cos(wd * t) + ((zeta * omega) / wd) * Math.sin(wd * t));
}

/** Damped impulse response (starts at 0, unit initial velocity scaled to peak ~1). */
export function springImpulse(t: number, omega: number, zeta: number): number {
  if (t <= 0) return 0;
  const wd = omega * Math.sqrt(1 - zeta * zeta);
  return Math.exp(-zeta * omega * t) * Math.sin(wd * t);
}

// ───────────────────────────── Geometry ─────────────────────────────

const mirrorSeg = (s: Seg): Seg => s.map(([x, y]) => [512 - x, y]) as Seg;
const lerpPt = (a: Pt, b: Pt, t: number): Pt => [lerp(a[0], b[0], t), lerp(a[1], b[1], t)];
export const lerpSeg = (a: Seg, b: Seg, t: number): Seg =>
  [lerpPt(a[0], b[0], t), lerpPt(a[1], b[1], t), lerpPt(a[2], b[2], t), lerpPt(a[3], b[3], t)];
const reverseSeg = (s: Seg): Seg => [s[3], s[2], s[1], s[0]];

export function segsToPath(segs: Seg[]): string {
  if (!segs.length) return '';
  const f = (n: number) => n.toFixed(2);
  let d = `M ${f(segs[0][0][0])} ${f(segs[0][0][1])}`;
  for (const s of segs) d += ` C ${f(s[1][0])} ${f(s[1][1])}, ${f(s[2][0])} ${f(s[2][1])}, ${f(s[3][0])} ${f(s[3][1])}`;
  return d;
}

// Stage 1: horizontal soundwave
export const WAVE_LEFT: Seg = [[52, 248], [76, 304], [120, 304], [148, 212]];
export const WAVE_RIGHT: Seg = mirrorSeg(WAVE_LEFT);
export const WAVE_CENTER: Seg[] = [
  [[148, 212], [164, 188], [176, 210], [192, 244]],
  [[192, 244], [210, 282], [230, 296], [256, 296]],
  [[256, 296], [282, 296], [302, 282], [320, 244]],
  [[320, 244], [336, 210], [348, 188], [364, 212]]
];

// Stage 2: open U-arc midpoint
export const FINAL_LEFT: Seg = [[88, 176], [77, 213], [77, 251], [88, 288]];
export const FINAL_RIGHT: Seg = mirrorSeg(FINAL_LEFT);
export const FINAL_CENTER: Seg[] = [
  [[205, 128], [162, 148], [140, 188], [140, 232]],
  [[140, 232], [140, 286], [174, 328], [256, 384]],
  [[256, 384], [338, 328], [372, 286], [372, 232]],
  [[372, 232], [372, 188], [350, 148], [307, 128]]
];
const MID_LEFT: Seg = lerpSeg(WAVE_LEFT, FINAL_LEFT, 0.5);
const MID_RIGHT: Seg = mirrorSeg(MID_LEFT);
const MID_CENTER: Seg[] = [
  [[168, 176], [162, 204], [164, 232], [176, 260]],
  [[176, 260], [192, 298], [218, 336], [256, 336]],
  [[256, 336], [294, 336], [320, 298], [336, 260]],
  [[336, 260], [348, 232], [350, 204], [344, 176]]
];

// Exact final logo (identical to VeloraLogo.svelte)
export const LOGO_LEFT_D = 'M 88 176 A 188 188 0 0 0 88 288';
export const LOGO_RIGHT_D = 'M 424 176 A 188 188 0 0 1 424 288';
export const LOGO_CENTER_D =
  'M 205 128 C 162 148, 140 188, 140 232 C 140 286, 174 328, 256 384 C 338 328, 372 286, 372 232 C 372 188, 350 148, 307 128';

// Intro draw paths: grow outward from the centre of the wave
export const DRAW_LEFT_D = segsToPath([reverseSeg(WAVE_CENTER[1]), reverseSeg(WAVE_CENTER[0]), reverseSeg(WAVE_LEFT)]);
export const DRAW_RIGHT_D = segsToPath([WAVE_CENTER[2], WAVE_CENTER[3], reverseSeg(WAVE_RIGHT)]);

/** Morph geometry for progress m (0 = wave, 0.5 = U-arc, 1 = logo); m may slightly overshoot. */
export function morphShapes(m: number): { left: Seg; right: Seg; center: Seg[] } {
  if (m <= 0.5) {
    const t = m / 0.5;
    return {
      left: lerpSeg(WAVE_LEFT, MID_LEFT, t),
      right: lerpSeg(WAVE_RIGHT, MID_RIGHT, t),
      center: WAVE_CENTER.map((s, i) => lerpSeg(s, MID_CENTER[i], t))
    };
  }
  const t = (m - 0.5) / 0.5;
  return {
    left: lerpSeg(MID_LEFT, FINAL_LEFT, t),
    right: lerpSeg(MID_RIGHT, FINAL_RIGHT, t),
    center: MID_CENTER.map((s, i) => lerpSeg(s, FINAL_CENTER[i], t))
  };
}

/** Elastic sag of the line under the ball, falling off smoothly away from the centre. */
export function deformSeg(s: Seg, depth: number): Seg {
  if (Math.abs(depth) < 0.01) return s;
  return s.map(([x, y]) => [x, y + depth * Math.exp(-Math.pow((x - 256) / 118, 2))]) as Seg;
}

// ───────────────────────────── Ball physics ─────────────────────────────

const G = 2700; // px / s²
export const BALL_R = 64;
export const T_APPEAR = 0.16;
const T_ANTICIPATE = 0.46;
export const T_DROP = 0.64;
const Y_HOVER = 56;
const Y_DROP = 49;

type Phase =
  | { kind: 'fall'; t0: number; t1: number; y0: number; v0: number }
  | { kind: 'contact'; t0: number; t1: number; yc: number; vin: number; omega: number; zeta: number; settle?: boolean };

function makeContact(t0: number, yc: number, vin: number, vout: number | null, tc: number, settleOmega = 21, settleZeta = 0.36): Phase {
  if (vout === null) return { kind: 'contact', t0, t1: Infinity, yc, vin, omega: settleOmega, zeta: settleZeta, settle: true };
  const e = vout / vin;
  const ln = Math.log(e);
  const zeta = -ln / Math.sqrt(Math.PI * Math.PI + ln * ln);
  const wd = Math.PI / tc;
  const omega = wd / Math.sqrt(1 - zeta * zeta);
  return { kind: 'contact', t0, t1: t0 + tc, yc, vin, omega, zeta };
}

function fallTime(y0: number, v0: number, y1: number): number {
  return (-v0 + Math.sqrt(v0 * v0 + 2 * G * (y1 - y0))) / G;
}

const CONTACTS = [
  { yc: 212, apex: 108, tc: 0.15 },
  { yc: 222, apex: 150, tc: 0.13 },
  { yc: 232, apex: null as number | null, tc: 0 }
];

export const PHASES: Phase[] = [];
export const IMPACTS: { t: number; yc: number; tExit: number; vExit: number }[] = [];
{
  let t = T_DROP;
  let y = Y_DROP;
  let v = 0;
  for (const c of CONTACTS) {
    const tf = fallTime(y, v, c.yc);
    PHASES.push({ kind: 'fall', t0: t, t1: t + tf, y0: y, v0: v });
    t += tf;
    const vin = v + G * tf;
    const vout = c.apex === null ? null : Math.sqrt(2 * G * (c.yc - c.apex));
    const ph = makeContact(t, c.yc, vin, vout, c.tc);
    PHASES.push(ph);
    IMPACTS.push({ t, yc: c.yc, tExit: ph.t1, vExit: vout === null ? 0 : -vout });
    if (vout === null) break;
    t = ph.t1;
    y = c.yc;
    v = -vout;
  }
}
export const T_SETTLE = IMPACTS[IMPACTS.length - 1].t;

function contactState(p: Extract<Phase, { kind: 'contact' }>, tau: number) {
  const wd = p.omega * Math.sqrt(1 - p.zeta * p.zeta);
  const env = Math.exp(-p.zeta * p.omega * tau);
  const y = p.yc + (p.vin / wd) * env * Math.sin(wd * tau);
  const v = (p.vin / wd) * env * (wd * Math.cos(wd * tau) - p.zeta * p.omega * Math.sin(wd * tau));
  return { y, v };
}

export function ballState(t: number): { cy: number; rx: number; ry: number; scale: number; opacity: number; pen: number; v: number } {
  // Pop in with a soft overshoot, then a gentle anticipation lift before the drop.
  if (t < T_DROP) {
    const s = springStep(t - T_APPEAR, 17, 0.52);
    const lift = easeInOutSine(clamp01((t - T_ANTICIPATE) / (T_DROP - T_ANTICIPATE)));
    const cy = lerp(Y_HOVER, Y_DROP, lift);
    // slight vertical stretch during the anticipation (preparing to drop)
    const k = 0.035 * Math.sin(lift * Math.PI);
    return { cy, rx: BALL_R / (1 + k), ry: BALL_R * (1 + k), scale: Math.max(0, s), opacity: clamp01((t - T_APPEAR) / 0.08), pen: 0, v: 0 };
  }
  const ph = PHASES.find((p) => t >= p.t0 && t < p.t1) ?? PHASES[PHASES.length - 1];
  let y: number;
  let v: number;
  let pen = 0;
  if (ph.kind === 'fall') {
    const tau = t - ph.t0;
    y = ph.y0 + ph.v0 * tau + 0.5 * G * tau * tau;
    v = ph.v0 + G * tau;
  } else {
    ({ y, v } = contactState(ph, t - ph.t0));
    pen = (y - ph.yc) * (ph.settle ? 0.55 : 1);
  }
  // Squash & stretch: velocity stretches along motion, penetration squashes; area is preserved.
  const stretch = 0.13 * Math.min(Math.abs(v) / 950, 1.15);
  const squash = 0.2 * Math.max(-0.35, Math.min(pen / 46, 1));
  const k = stretch - squash;
  const ry = BALL_R * (1 + k);
  const rx = BALL_R / (1 + k);
  // keep the contact edge planted while squashed
  const cy = y + (BALL_R - ry) * (pen > 0 ? 0.85 : 0);
  return { cy, rx, ry, scale: 1, opacity: 1, pen, v };
}

// ───────────────────────────── Line response ─────────────────────────────

const MORPH_TARGETS = [0.4, 0.8, 1.0];

export function morphProgress(t: number): number {
  let m = 0;
  let prev = 0;
  IMPACTS.forEach((imp, i) => {
    const target = MORPH_TARGETS[i];
    const isLast = i === IMPACTS.length - 1;
    m += (target - prev) * springStep(t - imp.t, isLast ? 20 : 25, isLast ? 0.78 : 0.7);
    prev = target;
  });
  return m;
}

/** Elastic vertical sag at the centre of the line (positive = down). */
export function lineSag(t: number): number {
  let d = 0;
  const b = ballState(t);
  IMPACTS.forEach((imp, i) => {
    const isLast = i === IMPACTS.length - 1;
    if (isLast) {
      if (t >= imp.t) d += 0.32 * (b.cy - imp.yc);
      return;
    }
    if (t >= imp.t && t < imp.tExit) {
      d += 0.6 * (b.cy - imp.yc);
    } else if (t >= imp.tExit) {
      const ws = 34;
      const zs = 0.2;
      const wd = ws * Math.sqrt(1 - zs * zs);
      d += ((0.6 * imp.vExit) / wd) * springImpulse(t - imp.tExit, ws, zs);
    }
  });
  return d;
}

// ───────────────────────────── Droplets & splash ─────────────────────────────

export const DROP_T0 = T_SETTLE + 0.012;
export const DROP_DUR = 0.52;
export const DROP_LAND = DROP_T0 + DROP_DUR;
const D_Y0 = 250;
const D_X0 = 256 - 50;
const D_X1 = 88;
const D_Y1 = 176;
const D_APEX = 70;
const D_G = Math.pow((Math.sqrt(2 * (D_Y0 - D_APEX)) + Math.sqrt(2 * (D_Y1 - D_APEX))) / DROP_DUR, 2);
const D_VY0 = -Math.sqrt(2 * D_G * (D_Y0 - D_APEX));
const D_VX = (D_X1 - D_X0) / DROP_DUR;

/** Left droplet state (right one is mirrored). */
export function dropletState(t: number): { x: number; y: number; r: number; angle: number; s: number; opacity: number } | null {
  const tau = t - DROP_T0;
  if (tau <= 0 || tau >= DROP_DUR) return null;
  const x = D_X0 + D_VX * tau;
  const y = D_Y0 + D_VY0 * tau + 0.5 * D_G * tau * tau;
  const vx = D_VX;
  const vy = D_VY0 + D_G * tau;
  const speed = Math.hypot(vx, vy);
  const grow = springStep(tau, 34, 0.6);
  const shrink = 1 - easeInOutCubic(clamp01((tau - (DROP_DUR - 0.07)) / 0.07));
  const r = 12 * grow * lerp(0.55, 1, shrink) ;
  // tip trails behind the motion; the drop becomes round near the apex where speed is lowest
  const s = clamp01((speed - 330) / 900);
  const angle = (Math.atan2(vy, vx) * 180) / Math.PI - 90;
  return { x, y, r, angle, s, opacity: shrink > 0.02 ? 1 : 0 };
}

/** Teardrop path (unit radius, tip towards -y). s = 0 -> circle, s = 1 -> elongated drop. */
export function dropletPath(s: number): string {
  const topY = lerp(-1, -1.75, s);
  const sideY = lerp(0, 0.3, s);
  const botY = lerp(1, 1.18, s);
  const c1x = lerp(0.5523, 0.2, s);
  const c1y = lerp(-1, -0.95, s);
  const c2x = 1;
  const c2y = lerp(-0.5523, -0.18, s);
  const c3y = lerp(0.5523, 0.86, s);
  const c4x = lerp(0.5523, 0.56, s);
  const c4y = botY;
  const f = (n: number) => n.toFixed(3);
  return `M 0 ${f(topY)} C ${f(c1x)} ${f(c1y)}, ${f(c2x)} ${f(c2y)}, 1 ${f(sideY)} C 1 ${f(c3y)}, ${f(c4x)} ${f(c4y)}, 0 ${f(botY)} C ${f(-c4x)} ${f(c4y)}, -1 ${f(c3y)}, -1 ${f(sideY)} C ${f(-c2x)} ${f(c2y)}, ${f(-c1x)} ${f(c1y)}, 0 ${f(topY)} Z`;
}

/** Small spray particles on the first two impacts (deterministic). */
export type Particle = { x: number; y: number; r: number; opacity: number };
const SPRAY = [
  { dx: -1, vx: 230, vy: -520, r: 6.5 },
  { dx: -1, vx: 330, vy: -360, r: 4.5 },
  { dx: -1, vx: 150, vy: -640, r: 3.5 },
  { dx: 1, vx: 240, vy: -500, r: 6 },
  { dx: 1, vx: 350, vy: -380, r: 4 },
  { dx: 1, vx: 160, vy: -620, r: 3.5 }
];
export function sprayParticles(t: number): Particle[] {
  const out: Particle[] = [];
  IMPACTS.slice(0, 2).forEach((imp, i) => {
    const tau = t - imp.t - 0.03;
    const life = 0.5 - i * 0.08;
    if (tau <= 0 || tau >= life) return;
    const power = i === 0 ? 1 : 0.72;
    for (const p of SPRAY) {
      const x = 256 + p.dx * (34 + p.vx * power * tau);
      const y = imp.yc + 72 + p.vy * power * tau + 0.5 * G * tau * tau;
      const fade = 1 - easeInOutCubic(clamp01(tau / life));
      out.push({ x, y, r: p.r * power * lerp(0.6, 1, fade), opacity: 0.9 * fade });
    }
  });
  return out;
}

/** Expanding ripple rings at line impacts. */
export function impactRipples(t: number): { cy: number; rx: number; ry: number; w: number; opacity: number }[] {
  const out: { cy: number; rx: number; ry: number; w: number; opacity: number }[] = [];
  IMPACTS.slice(0, 2).forEach((imp, i) => {
    const tau = t - imp.t;
    const life = 0.6;
    if (tau <= 0 || tau >= life) return;
    const p = easeOutQuart(tau / life);
    const power = i === 0 ? 1 : 0.75;
    out.push({
      cy: imp.yc + 86 + 18 * p,
      rx: lerp(50, 200 * power, p),
      ry: lerp(8, 26 * power, p),
      w: lerp(7, 0.5, p),
      opacity: 0.42 * power * (1 - p)
    });
  });
  return out;
}

// ───────────────────────────── Finale ─────────────────────────────

export const FILL_T0 = DROP_LAND - 0.03;
export const FILL_DUR = 0.42;
export const T_LOGO_DONE = FILL_T0 + FILL_DUR;
export const earcupFill = (t: number) => easeInOutCubic(clamp01((t - FILL_T0) / FILL_DUR));

/** Subtle "breath" of the completed logo once the earcups are filled. */
export const logoPulse = (t: number) => 1 + 0.022 * springImpulse(t - (T_LOGO_DONE - 0.12), 13, 0.42);
/** Soft warm glow behind the ball. */
export const glowAmount = (t: number) => {
  const a = springImpulse(t - (DROP_LAND - 0.1), 6.5, 0.55);
  return Math.max(0, a);
};

export const WORD_T0 = T_LOGO_DONE - 0.18;
export const LETTER_STAGGER = 0.048;
export const WAVE_T0 = WORD_T0 + 0.3;
export const WAVE_DUR = 0.82;
export const T_END = WAVE_T0 + WAVE_DUR + 0.34;
