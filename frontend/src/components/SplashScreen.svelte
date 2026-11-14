<script lang="ts">
  import { onMount } from 'svelte';

  interface Props {
    onComplete: () => void;
  }

  let { onComplete }: Props = $props();

  type Pt = [number, number];
  type Seg = [Pt, Pt, Pt, Pt];
  type Ghost = { cy: number; rx: number; ry: number; op: number };
  type DropGhost = { lx: number; ly: number; rx: number; ry: number; r: number; d: string; flip: number; angle: number; op: number };

  function lerp(a: number, b: number, t: number): number {
    return a + (b - a) * t;
  }

  function lerpPt(a: Pt, b: Pt, t: number): Pt {
    return [lerp(a[0], b[0], t), lerp(a[1], b[1], t)];
  }

  function lerpSeg(a: Seg, b: Seg, t: number): Seg {
    return [lerpPt(a[0], b[0], t), lerpPt(a[1], b[1], t), lerpPt(a[2], b[2], t), lerpPt(a[3], b[3], t)];
  }

  function mirrorSeg(s: Seg): Seg {
    return [
      [512 - s[0][0], s[0][1]],
      [512 - s[1][0], s[1][1]],
      [512 - s[2][0], s[2][1]],
      [512 - s[3][0], s[3][1]]
    ];
  }

  function segsToPath(segs: Seg[]): string {
    if (segs.length === 0) return '';
    let d = `M ${segs[0][0][0].toFixed(1)} ${segs[0][0][1].toFixed(1)}`;
    for (const s of segs) {
      d += ` C ${s[1][0].toFixed(1)} ${s[1][1].toFixed(1)}, ${s[2][0].toFixed(1)} ${s[2][1].toFixed(1)}, ${s[3][0].toFixed(1)} ${s[3][1].toFixed(1)}`;
    }
    return d;
  }

  function easeInQuad(x: number): number {
    return x * x;
  }

  function easeOutQuad(x: number): number {
    return 1 - (1 - x) * (1 - x);
  }

  function easeOutCubic(x: number): number {
    return 1 - Math.pow(1 - x, 3);
  }

  function easeInOutCubic(x: number): number {
    return x < 0.5 ? 4 * x * x * x : 1 - Math.pow(-2 * x + 2, 3) / 2;
  }

  function clamp01(v: number): number {
    return Math.max(0, Math.min(1, v));
  }

  // Hang-time easing: noticeably slower & floaty near the apex (t = 0.5)
  function hangTimeEase(t: number): number {
    const u = 2 * clamp01(t) - 1;
    return 0.5 + 0.5 * (0.35 * u + 0.65 * u * u * u);
  }

  function getBallState(ms: number): { cy: number; rx: number; ry: number; op: number } {
    if (ms < 0) return { cy: 56, rx: 0, ry: 0, op: 0 };
    if (ms < 400) {
      const a = easeOutCubic(clamp01((ms - 100) / 240));
      return { cy: 56, rx: 64 * a, ry: 64 * a, op: a };
    }
    // 1. Jatoh (400..650ms) + Impact 1 squash (650..780ms)
    if (ms < 650) {
      const d = easeInQuad(clamp01((ms - 400) / 250));
      const st = Math.sin(d * Math.PI) * 3;
      return { cy: lerp(56, 212, d), rx: 64 - st * 0.5, ry: 64 + st, op: 1 };
    }
    if (ms < 780) {
      const t = easeOutQuad(clamp01((ms - 650) / 130));
      const sq = Math.sin(t * Math.PI);
      return { cy: lerp(212, 256, t), rx: 64 + sq * 8, ry: 64 - sq * 8, op: 1 };
    }
    // 2. Bounce 1: up (780..1020ms) -> 108, down (1020..1220ms) -> 240
    if (ms < 1020) {
      const up = easeOutQuad(clamp01((ms - 780) / 240));
      const st = Math.sin(up * Math.PI) * 4;
      return { cy: lerp(256, 108, up), rx: 64 - st * 0.5, ry: 64 + st, op: 1 };
    }
    if (ms < 1220) {
      const dn = easeInQuad(clamp01((ms - 1020) / 200));
      const st = Math.sin(dn * Math.PI) * 3;
      return { cy: lerp(108, 240, dn), rx: 64 - st * 0.5, ry: 64 + st, op: 1 };
    }
    // 3. Bounce 2: up (1220..1400ms) -> 152, down kelewat ke bawah (1400..1560ms) -> 248
    if (ms < 1400) {
      const up = easeOutQuad(clamp01((ms - 1220) / 180));
      const st = Math.sin(up * Math.PI) * 3;
      return { cy: lerp(240, 152, up), rx: 64 - st * 0.5, ry: 64 + st, op: 1 };
    }
    if (ms < 1560) {
      const dn = easeInQuad(clamp01((ms - 1400) / 160));
      const sq = Math.max(0, (dn - 0.7) / 0.3);
      return { cy: lerp(152, 248, dn), rx: lerp(64, 69, sq), ry: lerp(64, 59, sq), op: 1 };
    }
    // Naik dikit (1560..1645ms) -> 225, terus nge-set (1645..1730ms) -> 232
    if (ms < 1645) {
      const t = easeInOutCubic(clamp01((ms - 1560) / 85));
      return { cy: lerp(248, 225, t), rx: lerp(69, 62, t), ry: lerp(59, 66, t), op: 1 };
    }
    if (ms < 1730) {
      const t = easeOutCubic(clamp01((ms - 1645) / 85));
      return { cy: lerp(225, 232, t), rx: lerp(62, 64, t), ry: lerp(66, 64, t), op: 1 };
    }
    return { cy: 232, rx: 64, ry: 64, op: 1 };
  }

  function buildDropletPath(s: number): string {
    const topY = lerp(-1, -1.65, s);
    const sideX = lerp(1, 0.96, s);
    const sideY = lerp(0, 0.36, s);
    const botY = lerp(1, 1.24, s);
    const c1x = lerp(0.5523, 0.22, s);
    const c1y = lerp(-1, -0.92, s);
    const c2x = lerp(1, 0.96, s);
    const c2y = lerp(-0.5523, -0.12, s);
    const c3x = lerp(1, 0.96, s);
    const c3y = lerp(0.5523, 0.92, s);
    const c4x = lerp(0.5523, 0.54, s);
    const c4y = lerp(1, 1.24, s);
    return `M 0 ${topY.toFixed(3)} C ${c1x.toFixed(3)} ${c1y.toFixed(3)}, ${c2x.toFixed(3)} ${c2y.toFixed(3)}, ${sideX.toFixed(3)} ${sideY.toFixed(3)} C ${c3x.toFixed(3)} ${c3y.toFixed(3)}, ${c4x.toFixed(3)} ${c4y.toFixed(3)}, 0 ${botY.toFixed(3)} C ${(-c4x).toFixed(3)} ${c4y.toFixed(3)}, ${(-c3x).toFixed(3)} ${c3y.toFixed(3)}, ${(-sideX).toFixed(3)} ${sideY.toFixed(3)} C ${(-c2x).toFixed(3)} ${c2y.toFixed(3)}, ${(-c1x).toFixed(3)} ${c1y.toFixed(3)}, 0 ${topY.toFixed(3)} Z`;
  }

  function getDropletAt(dtRaw: number): {
    lx: number;
    ly: number;
    rx: number;
    ry: number;
    r: number;
    d: string;
    flip: number;
    angle: number;
    op: number;
  } {
    if (dtRaw <= 0 || dtRaw >= 1) {
      return { lx: 204, ly: 224, rx: 308, ry: 224, r: 0, d: buildDropletPath(0), flip: 1, angle: 0, op: 0 };
    }
    const dt = hangTimeEase(dtRaw);
    const arcLift = Math.sin(dt * Math.PI) * 128;
    const r = 13 * Math.sin(clamp01(dt * 1.1) * Math.PI * 0.85 + 0.15);
    const op = dt < 0.88 ? 1 : 1 - (dt - 0.88) / 0.12;
    // Dynamic tip: points down (-1 flip) while rising, perfect circle (s=0) at apex, points up (+1 flip) while falling
    const cosPhase = Math.cos(dt * Math.PI);
    const s = Math.pow(Math.abs(cosPhase), 0.85);
    const flip = cosPhase >= 0 ? -1 : 1;
    // Tangent velocity along track for left droplet: vx = -116, vy = -48 - 128*PI*cos(dt*PI)
    const vx = -116;
    const vy = -48 - 128 * Math.PI * cosPhase;
    const rawRad = flip === -1 ? Math.atan2(vx, -vy) : Math.atan2(-vx, vy);
    const angle = (rawRad * 180) / Math.PI;
    return {
      lx: lerp(204, 88, dt),
      ly: lerp(224, 176, dt) - arcLift,
      rx: lerp(308, 424, dt),
      ry: lerp(224, 176, dt) - arcLift,
      r,
      d: buildDropletPath(s),
      flip,
      angle,
      op
    };
  }

  function getWaveShake(ms: number): { x: number; y: number } {
    const impactShake = (startMs: number, durMs: number, ampY: number, ampX: number) => {
      if (ms < startMs || ms > startMs + durMs) return { x: 0, y: 0 };
      const t = (ms - startMs) / durMs;
      const decay = Math.pow(1 - t, 2);
      return {
        x: ampX * decay * Math.sin(t * Math.PI * 9),
        y: ampY * decay * Math.sin(t * Math.PI * 7)
      };
    };
    const s1 = impactShake(650, 170, 4.2, 1.5);
    const s2 = impactShake(1190, 150, 3.2, 1.2);
    const s3 = impactShake(1520, 160, 3.8, 1.4);
    return { x: s1.x + s2.x + s3.x, y: s1.y + s2.y + s3.y };
  }

  function getWaveMorphProgress(ms: number): number {
    if (ms < 650) return 0;
    if (ms <= 870) return lerp(0, 0.4, easeInOutCubic(clamp01((ms - 650) / 220)));
    if (ms < 1190) return 0.4;
    if (ms <= 1330) return lerp(0.4, 0.8, easeInOutCubic(clamp01((ms - 1190) / 140)));
    if (ms < 1520) return 0.8;
    return lerp(0.8, 1.0, easeInOutCubic(clamp01((ms - 1520) / 160)));
  }

  // Stage 1: Continuous horizontal soundwave with wide center valley
  const WAVE_LEFT: Seg[] = [[[52, 248], [76, 304], [120, 304], [148, 212]]];
  const WAVE_RIGHT_DRAW: Seg[] = [[[364, 212], [392, 304], [436, 304], [460, 248]]];
  const WAVE_RIGHT: Seg[] = [mirrorSeg(WAVE_LEFT[0])];
  const WAVE_CENTER: Seg[] = [
    [[148, 212], [164, 188], [176, 210], [192, 244]],
    [[192, 244], [210, 282], [230, 296], [256, 296]],
    [[256, 296], [282, 296], [302, 282], [320, 244]],
    [[320, 244], [336, 210], [348, 188], [364, 212]]
  ];

  // Stage 3: 100% formed Velora 5D Concentric Arc Earcups logo
  const FINAL_LEFT: Seg[] = [[[88, 176], [77, 213], [77, 251], [88, 288]]];
  const FINAL_RIGHT: Seg[] = [mirrorSeg(FINAL_LEFT[0])];
  const FINAL_CENTER: Seg[] = [
    [[205, 128], [162, 148], [140, 188], [140, 232]],
    [[140, 232], [140, 286], [174, 328], [256, 384]],
    [[256, 384], [338, 328], [372, 286], [372, 232]],
    [[372, 232], [372, 188], [350, 148], [307, 128]]
  ];

  // Stage 2 (Midpoint): smooth open U-arc without top shoulder hooks
  const IMPACT50_LEFT: Seg[] = [lerpSeg(WAVE_LEFT[0], FINAL_LEFT[0], 0.5)];
  const IMPACT50_RIGHT: Seg[] = [mirrorSeg(IMPACT50_LEFT[0])];
  const IMPACT50_CENTER: Seg[] = [
    [[168, 176], [162, 204], [164, 232], [176, 260]],
    [[176, 260], [192, 298], [218, 336], [256, 336]],
    [[256, 336], [294, 336], [320, 298], [336, 260]],
    [[336, 260], [348, 232], [350, 204], [344, 176]]
  ];

  let leftPath = $state(segsToPath(WAVE_LEFT));
  let rightPath = $state(segsToPath(WAVE_RIGHT));
  let centerPath = $state(segsToPath(WAVE_CENTER));
  const FULL_WAVE_PATH = segsToPath([...WAVE_LEFT, ...WAVE_CENTER, ...WAVE_RIGHT_DRAW]);

  let waveDrawProgress = $state(0);
  let ballCy = $state(56);
  let ballRx = $state(0);
  let ballRy = $state(0);
  let ballOpacity = $state(0);
  let ballGhosts = $state<Ghost[]>([]);

  let dropLeftX = $state(204);
  let dropLeftY = $state(224);
  let dropRightX = $state(308);
  let dropRightY = $state(224);
  let dropR = $state(0);
  let dropPath = $state(buildDropletPath(0));
  let dropFlip = $state(1);
  let dropAngle = $state(0);
  let dropOpacity = $state(0);
  let dropGhosts = $state<DropGhost[]>([]);
  let waveShakeX = $state(0);
  let waveShakeY = $state(0);

  let earcupFill = $state(0);
  let wordmarkOpacity = $state(0);
  let wordmarkY = $state(12);
  let textWaveProgress = $state(0);
  let exiting = $state(false);
  let finished = $state(false);

  function finishSplash() {
    if (exiting || finished) return;
    exiting = true;
    setTimeout(() => {
      finished = true;
      onComplete();
    }, 380);
  }

  onMount(() => {
    let rafId = 0;
    let start = 0;

    function frame(now: number) {
      if (start === 0) start = now;
      const elapsed = (now - start) * 0.78;

      waveDrawProgress = easeInOutCubic(clamp01(elapsed / 440));

      // Main ball state + subtle motion trail ghosts during drop & bounce back
      const b = getBallState(elapsed);
      ballCy = b.cy;
      ballRx = b.rx;
      ballRy = b.ry;
      ballOpacity = b.op;

      if (elapsed >= 410 && elapsed <= 1710) {
        const ghosts: Ghost[] = [];
        for (let k = 1; k <= 3; k++) {
          const g = getBallState(elapsed - k * 28);
          const scale = 1 - k * 0.08;
          const fade = (0.2 - k * 0.05) * (elapsed > 1560 ? 1 - clamp01((elapsed - 1560) / 150) : 1);
          if (g.op > 0 && fade > 0.01) {
            ghosts.push({ cy: g.cy, rx: g.rx * scale, ry: g.ry * scale, op: fade });
          }
        }
        ballGhosts = ghosts;
      } else {
        ballGhosts = [];
      }

      // Line morph: Impact 1 -> 70% | Bounce 1 -> 80% | Bounce 2 -> 90% | Bounce 3 -> 100%
      const mp = getWaveMorphProgress(elapsed);
      if (mp <= 0) {
        leftPath = segsToPath(WAVE_LEFT);
        rightPath = segsToPath(WAVE_RIGHT);
        centerPath = segsToPath(WAVE_CENTER);
      } else if (mp <= 0.5) {
        const t = mp / 0.5;
        leftPath = segsToPath([lerpSeg(WAVE_LEFT[0], IMPACT50_LEFT[0], t)]);
        rightPath = segsToPath([lerpSeg(WAVE_RIGHT[0], IMPACT50_RIGHT[0], t)]);
        centerPath = segsToPath(WAVE_CENTER.map((s, i) => lerpSeg(s, IMPACT50_CENTER[i], t)));
      } else {
        const t = (mp - 0.5) / 0.5;
        leftPath = segsToPath([lerpSeg(IMPACT50_LEFT[0], FINAL_LEFT[0], t)]);
        rightPath = segsToPath([lerpSeg(IMPACT50_RIGHT[0], FINAL_RIGHT[0], t)]);
        centerPath = segsToPath(IMPACT50_CENTER.map((s, i) => lerpSeg(s, FINAL_CENTER[i], t)));
      }

      // Line impact vibration
      const shake = getWaveShake(elapsed);
      waveShakeX = shake.x;
      waveShakeY = shake.y;

      // 2 water droplets (1520..1980ms = 460ms) on final bounce impact with hang-time, dynamic tip & motion trail
      const DROP_DUR = 460;
      if (elapsed >= 1520 && elapsed <= 1520 + DROP_DUR) {
        const dtRaw = (elapsed - 1520) / DROP_DUR;
        const d = getDropletAt(dtRaw);
        dropLeftX = d.lx;
        dropLeftY = d.ly;
        dropRightX = d.rx;
        dropRightY = d.ry;
        dropR = d.r;
        dropPath = d.d;
        dropFlip = d.flip;
        dropAngle = d.angle;
        dropOpacity = d.op;

        const dGhosts: DropGhost[] = [];
        for (let k = 1; k <= 4; k++) {
          const g = getDropletAt(dtRaw - k * 0.05);
          if (g.op > 0) {
            dGhosts.push({
              lx: g.lx,
              ly: g.ly,
              rx: g.rx,
              ry: g.ry,
              r: g.r * (1 - k * 0.16),
              d: g.d,
              flip: g.flip,
              angle: g.angle,
              op: g.op * (0.3 - k * 0.06)
            });
          }
        }
        dropGhosts = dGhosts;
      } else {
        dropOpacity = 0;
        dropR = 0;
        dropGhosts = [];
      }

      // Top-to-bottom color sweep on earcups after droplets land (1950..2310ms)
      if (elapsed >= 1950) {
        earcupFill = easeInOutCubic(clamp01((elapsed - 1950) / 360));
      } else {
        earcupFill = 0;
      }

      // 2250..2570ms: "Velora" text appears
      if (elapsed > 2250) {
        const wt = easeInOutCubic(clamp01((elapsed - 2250) / 320));
        wordmarkOpacity = wt;
        wordmarkY = lerp(12, 0, wt);
      }

      // 2490..3320ms: Horizontal wave sweeps left -> right above "Velora"
      if (elapsed > 2490) {
        textWaveProgress = easeInOutCubic(clamp01((elapsed - 2490) / 830));
      }

      // Enter app after wave reaches right end
      if (elapsed >= 3500) {
        finishSplash();
        return;
      }

      rafId = requestAnimationFrame(frame);
    }

    rafId = requestAnimationFrame(() => {
      rafId = requestAnimationFrame(frame);
    });

    return () => cancelAnimationFrame(rafId);
  });
</script>

{#if !finished}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    onclick={finishSplash}
    class="splash-backdrop fixed inset-0 z-50 flex flex-col items-center justify-center select-none cursor-pointer"
    style="
      opacity: {exiting ? 0 : 1};
      transform: scale({exiting ? 1.03 : 1});
      transition: opacity 380ms ease, transform 380ms ease;
    "
  >
    <div class="relative flex flex-col items-center justify-center">
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 512 512"
        class="w-56 h-56 sm:w-64 sm:h-64 relative z-10 overflow-visible"
        fill="none"
        aria-label="Velora"
      >
        <defs>
          <linearGradient id="earcup-drip-grad" gradientUnits="userSpaceOnUse" x1="0" y1="156" x2="0" y2="312">
            <stop offset="0%" stop-color="#FF9E64" stop-opacity="1" />
            <stop offset="{Math.min(100, earcupFill * 100).toFixed(1)}%" stop-color="#FF9E64" stop-opacity="1" />
            <stop offset="{Math.min(100, earcupFill * 118).toFixed(1)}%" stop-color="#FF9E64" stop-opacity="{earcupFill >= 0.98 ? 1 : 0}" />
            <stop offset="100%" stop-color="#FF9E64" stop-opacity="{earcupFill >= 0.98 ? 1 : 0}" />
          </linearGradient>
        </defs>

        {#if waveDrawProgress < 1}
          <path
            d={FULL_WAVE_PATH}
            pathLength="1"
            stroke="currentColor"
            stroke-width="40"
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-dasharray="1"
            stroke-dashoffset={1 - waveDrawProgress}
            style="opacity: {waveDrawProgress > 0.01 ? 1 : 0};"
          />
        {:else}
          <g transform="translate({waveShakeX.toFixed(2)}, {waveShakeY.toFixed(2)})">
            <path d={leftPath} stroke="currentColor" stroke-width="40" stroke-linecap="round" stroke-linejoin="round" />
            <path d={rightPath} stroke="currentColor" stroke-width="40" stroke-linecap="round" stroke-linejoin="round" />

            {#if earcupFill > 0}
              <path d={leftPath} stroke="url(#earcup-drip-grad)" stroke-width="40" stroke-linecap="round" stroke-linejoin="round" />
              <path d={rightPath} stroke="url(#earcup-drip-grad)" stroke-width="40" stroke-linecap="round" stroke-linejoin="round" />
            {/if}

            <path d={centerPath} stroke="currentColor" stroke-width="40" stroke-linecap="round" stroke-linejoin="round" />
          </g>
        {/if}

        <!-- Subtle Ball Motion Trail -->
        {#each ballGhosts as g}
          <ellipse cx="256" cy={g.cy} rx={g.rx} ry={g.ry} fill="#FF9E64" style="opacity: {g.op};" />
        {/each}

        <!-- Subtle Water Droplet Motion Trail -->
        {#each dropGhosts as dg}
          <path
            d={dg.d}
            fill="#FF9E64"
            transform="translate({dg.lx}, {dg.ly}) rotate({dg.angle}) scale({dg.r}, {dg.r * dg.flip})"
            style="opacity: {dg.op};"
          />
          <path
            d={dg.d}
            fill="#FF9E64"
            transform="translate({dg.rx}, {dg.ry}) rotate({-dg.angle}) scale({dg.r}, {dg.r * dg.flip})"
            style="opacity: {dg.op};"
          />
        {/each}

        <!-- Splash Water Droplets -->
        {#if dropOpacity > 0}
          <path
            d={dropPath}
            fill="#FF9E64"
            transform="translate({dropLeftX}, {dropLeftY}) rotate({dropAngle}) scale({dropR}, {dropR * dropFlip})"
            style="opacity: {dropOpacity};"
          />
          <path
            d={dropPath}
            fill="#FF9E64"
            transform="translate({dropRightX}, {dropRightY}) rotate({-dropAngle}) scale({dropR}, {dropR * dropFlip})"
            style="opacity: {dropOpacity};"
          />
        {/if}

        <!-- Main Bouncing Center Ball -->
        <ellipse
          cx="256"
          cy={ballCy}
          rx={Math.max(0, ballRx)}
          ry={Math.max(0, ballRy)}
          fill="#FF9E64"
          style="opacity: {ballOpacity};"
        />
      </svg>

      <div
        class="-mt-2 flex flex-col items-center relative z-10"
        style="
          opacity: {wordmarkOpacity};
          transform: translateY({wordmarkY}px);
        "
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 180 24"
          class="w-40 h-5 mb-2 overflow-visible"
          fill="none"
          aria-hidden="true"
        >
          <defs>
            <linearGradient id="velora-wave-grad" x1="0%" y1="0%" x2="100%" y2="0%">
              <stop offset="0%" stop-color="currentColor" />
              <stop offset="100%" stop-color="#FF9E64" />
            </linearGradient>
          </defs>
          <path
            d="M 4 12 C 14 2, 24 2, 34 12 C 44 22, 54 22, 64 12 C 74 2, 84 2, 94 12 C 104 22, 114 22, 124 12 C 134 2, 144 2, 154 12 C 162 20, 170 20, 176 12"
            pathLength="1"
            stroke="url(#velora-wave-grad)"
            stroke-width="3.5"
            stroke-linecap="round"
            stroke-dasharray="1"
            stroke-dashoffset={1 - textWaveProgress}
            style="opacity: {textWaveProgress > 0.01 ? 1 : 0};"
          />
        </svg>

        <span
          class="velora-wordmark text-[44px] leading-none font-bold tracking-tight"
          style="font-family: 'Poppins', sans-serif;"
        >
          Velora
        </span>
      </div>
    </div>
  </div>
{/if}

<style>
  .splash-backdrop {
    color: #B8EAFF;
    background-color: rgba(11, 12, 14, 0.56);
    backdrop-filter: blur(28px);
    -webkit-backdrop-filter: blur(28px);
  }
  :global([data-theme='light']) .splash-backdrop {
    color: #006684;
    background-color: rgba(255, 255, 255, 0.56);
  }
  .velora-wordmark {
    color: #ffffff;
  }
  :global([data-theme='light']) .velora-wordmark {
    color: #0b0c0e;
  }
</style>
