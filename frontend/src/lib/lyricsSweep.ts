export interface SweepWordItem {
  word: string;
  prog: number;
}

const clamp01 = (v: number): number => Math.max(0, Math.min(1, v));

const easeOutCubic = (t: number): number => 1 - Math.pow(1 - clamp01(t), 3);

export function computeLineSweepProgress(
  lineText: string,
  startSec: number,
  nextStartSec: number | undefined,
  currentTimeSec: number
): number {
  const words = (lineText || '♪').trim().split(/\s+/).filter(Boolean);
  const wordCount = Math.max(1, words.length);
  const charCount = words.reduce((acc, w) => acc + w.length, 0);

  const rawGap = nextStartSec !== undefined ? nextStartSec - startSec : 3.6;
  const gap = Math.max(0.45, rawGap);

  const estimatedVocal = wordCount * 0.22 + charCount * 0.042;
  const minSing = Math.min(0.55, gap * 0.88);
  const maxSing = Math.max(minSing, Math.min(gap * 0.86, gap - 0.14));
  const singDur = Math.max(minSing, Math.min(maxSing, estimatedVocal));

  // Lead by +0.14s so words start illuminating right on vocal attack
  const elapsed = currentTimeSec + 0.14 - startSec;
  return clamp01(elapsed / singDur);
}

export function computeWordProgressList(lineText: string, lineProgress: number): SweepWordItem[] {
  const words = (lineText || '♪').split(/\s+/).filter((w) => w.length > 0);
  if (words.length === 0) {
    return [{ word: '♪', prog: 1 }];
  }

  const p = clamp01(lineProgress);
  const weights = words.map((w) => Math.max(1.6, w.length * 0.85));
  const totalWeight = weights.reduce((a, b) => a + b, 0);

  const out: SweepWordItem[] = [];
  let cursor = 0;
  for (let i = 0; i < words.length; i++) {
    const startW = cursor / totalWeight;
    cursor += weights[i];
    const endW = cursor / totalWeight;
    const span = Math.max(0.02, endW - startW);
    const local = easeOutCubic((p - startW) / span);
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
  const softStart = Math.max(0, edge - 14).toFixed(1);
  const softEnd = Math.min(100, edge + 8).toFixed(1);
  return `linear-gradient(90deg, ${litColor} 0%, ${litColor} ${softStart}%, ${dimColor} ${softEnd}%, ${dimColor} 100%)`;
}
