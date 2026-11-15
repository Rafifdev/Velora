import React, { useEffect, useState } from 'react';
import { AbsoluteFill, continueRender, delayRender, useCurrentFrame, useVideoConfig } from 'remotion';
import '@fontsource/poppins/700.css';
import {
  BALL_R,
  DRAW_LEFT_D,
  DRAW_RIGHT_D,
  DROP_LAND,
  LETTER_STAGGER,
  LOGO_CENTER_D,
  LOGO_LEFT_D,
  LOGO_RIGHT_D,
  T_LOGO_DONE,
  WAVE_DUR,
  WAVE_T0,
  WORD_T0,
  ballState,
  clamp01,
  deformSeg,
  dropletPath,
  dropletState,
  earcupFill,
  easeInOutCubic,
  easeOutQuart,
  glowAmount,
  impactRipples,
  lerp,
  lineSag,
  logoPulse,
  morphProgress,
  morphShapes,
  segsToPath,
  sprayParticles,
  springStep
} from './motion';

export type SplashProps = { theme: 'dark' | 'light' };

const ACCENT = '#FF9E64';
const THEMES = {
  dark: { line: '#B8EAFF', text: '#FFFFFF' },
  light: { line: '#006684', text: '#0B0C0E' }
} as const;

const DRAW_T0 = 0.04;
const DRAW_T1 = 0.6;
const STROKE = 40;
const LETTERS = 'Velora'.split('');
const TEXT_WAVE_D =
  'M 4 12 C 14 2, 24 2, 34 12 C 44 22, 54 22, 64 12 C 74 2, 84 2, 94 12 C 104 22, 114 22, 124 12 C 134 2, 144 2, 154 12 C 162 20, 170 20, 176 12';

const strokeProps = {
  strokeWidth: STROKE,
  strokeLinecap: 'round' as const,
  strokeLinejoin: 'round' as const,
  fill: 'none'
};

export const VeloraSplash: React.FC<SplashProps> = ({ theme }) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const t = frame / fps;
  const c = THEMES[theme] ?? THEMES.dark;

  const [fontHandle] = useState(() => delayRender('Loading Poppins'));
  useEffect(() => {
    document.fonts.load('700 88px Poppins').then(() => continueRender(fontHandle));
  }, [fontHandle]);

  const done = t >= T_LOGO_DONE;

  // ── Line geometry ──
  const drawP = easeInOutCubic(clamp01((t - DRAW_T0) / (DRAW_T1 - DRAW_T0)));
  const drawing = t < DRAW_T1;
  let leftD = LOGO_LEFT_D;
  let rightD = LOGO_RIGHT_D;
  let centerD = LOGO_CENTER_D;
  if (!done && !drawing) {
    const sag = lineSag(t);
    const shapes = morphShapes(morphProgress(t));
    leftD = segsToPath([deformSeg(shapes.left, sag)]);
    rightD = segsToPath([deformSeg(shapes.right, sag)]);
    centerD = segsToPath(shapes.center.map((s) => deformSeg(s, sag)));
  }

  // ── Ball ──
  const ball = done
    ? { cy: 232, rx: BALL_R, ry: BALL_R, scale: 1, opacity: 1, v: 0 }
    : ballState(t);
  const ballGhosts =
    !done && Math.abs(ball.v) > 320
      ? [1, 2, 3].map((k) => ({ ...ballState(t - k * 0.014), op: [0.18, 0.1, 0.05][k - 1] }))
      : [];

  // ── Droplets ──
  const drop = dropletState(t);
  const dropGhosts = drop
    ? [1, 2, 3]
        .map((k) => ({ d: dropletState(t - k * 0.016), op: [0.32, 0.18, 0.08][k - 1] }))
        .filter((g) => g.d)
    : [];
  const landTau = t - DROP_LAND;
  const landP = easeOutQuart(clamp01(landTau / 0.5));
  const showLandRing = landTau > 0 && landTau < 0.5;

  // ── Fill / glow / pulse ──
  const fill = earcupFill(t);
  const fillEdge = Math.min(100, fill * 114);
  const fillSolid = Math.max(0, fillEdge - 14);
  const glowA = Math.max(0, glowAmount(t));
  const pulse = logoPulse(t);

  // ── Wordmark ──
  const wordTau = t - WORD_T0;
  const waveP = easeInOutCubic(clamp01((t - WAVE_T0) / WAVE_DUR));
  const wordGroupY = lerp(14, 0, easeOutQuart(clamp01(wordTau / 0.6)));

  const drawOpacity = clamp01((t - DRAW_T0) / 0.05);

  return (
    <AbsoluteFill style={{ backgroundColor: 'transparent', alignItems: 'center', fontFamily: 'Poppins, sans-serif' }}>
      <div style={{ position: 'absolute', top: 48, left: 64, width: 512, height: 512 }}>
        <svg viewBox="0 0 512 512" width={512} height={512} style={{ overflow: 'visible' }} shapeRendering="geometricPrecision">
          <defs>
            <linearGradient id="earcup-fill" gradientUnits="userSpaceOnUse" x1="0" y1="154" x2="0" y2="310">
              <stop offset="0%" stopColor={ACCENT} stopOpacity={1} />
              <stop offset={`${fillSolid.toFixed(2)}%`} stopColor={ACCENT} stopOpacity={1} />
              <stop offset={`${fillEdge.toFixed(2)}%`} stopColor={ACCENT} stopOpacity={fill >= 0.999 ? 1 : 0} />
              <stop offset="100%" stopColor={ACCENT} stopOpacity={fill >= 0.999 ? 1 : 0} />
            </linearGradient>
            <radialGradient id="ball-glow" cx="256" cy="232" r="200" gradientUnits="userSpaceOnUse">
              <stop offset="0%" stopColor={ACCENT} stopOpacity={0.55} />
              <stop offset="45%" stopColor={ACCENT} stopOpacity={0.16} />
              <stop offset="100%" stopColor={ACCENT} stopOpacity={0} />
            </radialGradient>
          </defs>

          {glowA > 0.003 && <circle cx={256} cy={232} r={200} fill="url(#ball-glow)" opacity={0.5 * glowA} />}

          <g transform={`translate(256 256) scale(${pulse}) translate(-256 -256)`}>
            {/* Impact ripples */}
            {!done &&
              impactRipples(t).map((r, i) => (
                <ellipse key={`rp${i}`} cx={256} cy={r.cy} rx={r.rx} ry={r.ry} fill="none" stroke={c.line} strokeWidth={r.w} opacity={r.opacity} />
              ))}

            {/* Line */}
            {drawing ? (
              <g opacity={drawOpacity}>
                <path d={DRAW_LEFT_D} pathLength={1} stroke={c.line} strokeDasharray="1 1" strokeDashoffset={1 - drawP} {...strokeProps} />
                <path d={DRAW_RIGHT_D} pathLength={1} stroke={c.line} strokeDasharray="1 1" strokeDashoffset={1 - drawP} {...strokeProps} />
              </g>
            ) : (
              <g>
                <path d={leftD} stroke={done ? ACCENT : c.line} {...strokeProps} />
                <path d={rightD} stroke={done ? ACCENT : c.line} {...strokeProps} />
                {!done && fill > 0 && (
                  <>
                    <path d={leftD} stroke="url(#earcup-fill)" {...strokeProps} />
                    <path d={rightD} stroke="url(#earcup-fill)" {...strokeProps} />
                  </>
                )}
                <path d={centerD} stroke={c.line} {...strokeProps} />
              </g>
            )}

            {/* Landing rings on the earcups */}
            {showLandRing &&
              [88, 424].map((x) => (
                <circle
                  key={`lr${x}`}
                  cx={x}
                  cy={176}
                  r={lerp(14, 40, landP)}
                  fill="none"
                  stroke={ACCENT}
                  strokeWidth={lerp(6, 0.5, landP)}
                  opacity={0.6 * (1 - landP)}
                />
              ))}

            {/* Spray particles */}
            {!done && sprayParticles(t).map((p, i) => <circle key={`sp${i}`} cx={p.x} cy={p.y} r={p.r} fill={ACCENT} opacity={p.opacity} />)}

            {/* Droplets (left + mirrored right) */}
            {drop &&
              [false, true].map((mirror) => (
                <g key={`dg${mirror}`} transform={mirror ? 'translate(512 0) scale(-1 1)' : undefined}>
                  {dropGhosts.map((g, i) => (
                    <path
                      key={i}
                      d={dropletPath(g.d!.s)}
                      fill={ACCENT}
                      opacity={g.op}
                      transform={`translate(${g.d!.x} ${g.d!.y}) rotate(${g.d!.angle}) scale(${g.d!.r * (1 - (i + 1) * 0.12)})`}
                    />
                  ))}
                  <path d={dropletPath(drop.s)} fill={ACCENT} transform={`translate(${drop.x} ${drop.y}) rotate(${drop.angle}) scale(${drop.r})`} />
                </g>
              ))}

            {/* Ball trail */}
            {ballGhosts.map((g, i) => (
              <ellipse key={`bg${i}`} cx={256} cy={g.cy} rx={g.rx * g.scale} ry={g.ry * g.scale} fill={ACCENT} opacity={g.op * g.opacity} />
            ))}

            {/* Ball */}
            <ellipse
              cx={256}
              cy={ball.cy}
              rx={Math.max(0, ball.rx * ball.scale)}
              ry={Math.max(0, ball.ry * ball.scale)}
              fill={ACCENT}
              opacity={ball.opacity}
            />
          </g>
        </svg>
      </div>

      {/* Wordmark */}
      <div
        style={{
          position: 'absolute',
          top: 48 + 512 - 16,
          left: 0,
          right: 0,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          transform: `translateY(${wordGroupY}px)`
        }}
      >
        <svg viewBox="0 0 180 24" width={320} height={40} style={{ overflow: 'visible', marginBottom: 16, color: c.line }} fill="none">
          <defs>
            <linearGradient id="text-wave" x1="0%" y1="0%" x2="100%" y2="0%">
              <stop offset="0%" stopColor={c.line} />
              <stop offset="100%" stopColor={ACCENT} />
            </linearGradient>
          </defs>
          <path
            d={TEXT_WAVE_D}
            pathLength={1}
            stroke="url(#text-wave)"
            strokeWidth={3.5}
            strokeLinecap="round"
            strokeDasharray="1 1"
            strokeDashoffset={1 - waveP}
            opacity={waveP > 0.005 ? 1 : 0}
          />
        </svg>
        <div style={{ display: 'flex', fontSize: 88, lineHeight: 1, fontWeight: 700, letterSpacing: '-0.025em', color: c.text, height: 88 }}>
          {LETTERS.map((ch, i) => {
            const lt = wordTau - i * LETTER_STAGGER;
            const p = springStep(lt, 15, 0.7);
            const op = clamp01(lt / 0.22);
            const blur = Math.max(0, (1 - Math.min(p, 1)) * 10);
            return (
              <span
                key={i}
                style={{
                  display: 'inline-block',
                  opacity: op,
                  transform: `translateY(${((1 - p) * 34).toFixed(2)}px)`,
                  filter: blur > 0.05 ? `blur(${blur.toFixed(2)}px)` : undefined
                }}
              >
                {ch}
              </span>
            );
          })}
        </div>
      </div>
    </AbsoluteFill>
  );
};
