import type { LyricWord } from './types';

export interface SweepWordItem {
  word: string;
  prog: number;
}

const clamp01 = (v: number): number => Math.max(0, Math.min(1, v));

export function computeLineSweepProgress(
  lineText: string,
  startSec: number,
  nextStartSec: number | undefined,
  currentTimeSec: number
): number {
  const words = (lineText || '♪').trim().split(/\s+/).filter(Boolean);
  const wordCount = Math.max(1, words.length);
  const charCount = words.reduce((acc, w) => acc + w.length, 0);

  let singDur: number;
  if (nextStartSec !== undefined && nextStartSec > startSec) {
    const gap = Math.max(0.35, nextStartSec - startSec);
    const maxVocalDur = Math.max(2.2, wordCount * 0.45 + charCount * 0.075);
    singDur = Math.min(gap * 0.92, maxVocalDur);
    if (gap < 0.8) {
      singDur = Math.max(0.35, gap * 0.9);
    }
  } else {
    singDur = Math.max(2.0, Math.min(5.0, wordCount * 0.35 + charCount * 0.06));
  }

  const elapsed = currentTimeSec + 0.05 - startSec;
  return clamp01(elapsed / singDur);
}

export function computeWordProgressList(
  lineText: string,
  lineProgress: number,
  timedWords?: LyricWord[],
  currentTimeSec?: number
): SweepWordItem[] {
  if (timedWords && timedWords.length > 0 && currentTimeSec !== undefined) {
    const effTime = currentTimeSec + 0.05;
    return timedWords.map((tw) => {
      const dur = Math.max(0.12, tw.durationSec || 0.35);
      return {
        word: tw.text,
        prog: clamp01((effTime - tw.timeSec) / dur)
      };
    });
  }

  const words = (lineText || '♪').split(/\s+/).filter((w) => w.length > 0);
  if (words.length === 0) {
    return [{ word: '♪', prog: 1 }];
  }

  const p = clamp01(lineProgress);
  const weights = words.map((w) => Math.max(1.0, w.length + 0.5));
  const totalWeight = weights.reduce((a, b) => a + b, 0);

  const out: SweepWordItem[] = [];
  let cursor = 0;
  for (let i = 0; i < words.length; i++) {
    const startW = cursor / totalWeight;
    cursor += weights[i];
    const endW = cursor / totalWeight;
    const span = Math.max(0.01, endW - startW);
    const local = clamp01((p - startW) / span);
    out.push({ word: words[i], prog: local });
  }
  return out;
}

export function wordSweepBackground(prog: number, litColor: string, dimColor: string): string {
  if (prog <= 0.001) {
    return `linear-gradient(90deg, ${dimColor} 0%, ${dimColor} 100%)`;
  }
  if (prog >= 0.999) {
    return `linear-gradient(90deg, ${litColor} 0%, ${litColor} 100%)`;
  }
  const edge = prog * 100;
  const softStart = Math.max(0, edge - 4).toFixed(1);
  const softEnd = Math.min(100, edge + 4).toFixed(1);
  return `linear-gradient(90deg, ${litColor} 0%, ${litColor} ${softStart}%, ${dimColor} ${softEnd}%, ${dimColor} 100%)`;
}
