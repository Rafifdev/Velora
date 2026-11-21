export type EqPresetId = 'spotify' | 'flat' | 'bass' | 'vocal' | 'electronic';

const EQ_BANDS: ReadonlyArray<{ freq: number; type: BiquadFilterType; q: number }> = [
  { freq: 65, type: 'lowshelf', q: 0.7 },
  { freq: 260, type: 'peaking', q: 0.9 },
  { freq: 1000, type: 'peaking', q: 0.85 },
  { freq: 3500, type: 'peaking', q: 1.1 },
  { freq: 12000, type: 'highshelf', q: 0.65 }
];

const EQ_PRESETS: Record<EqPresetId, readonly [number, number, number, number, number]> = {
  spotify: [1.5, -1.5, 0.5, -1.8, 1.8],
  flat: [0, 0, 0, 0, 0],
  bass: [6.5, 2.5, -1.0, -1.5, 0.5],
  vocal: [-3.0, -2.0, 4.5, 2.5, 1.0],
  electronic: [5.0, -2.0, 0.5, 1.5, 4.5]
};

export class VeloraAudioEngine {
  private ctx: AudioContext | null = null;
  private source: MediaElementAudioSourceNode | null = null;
  private attachedEl: HTMLAudioElement | null = null;
  private highpass: BiquadFilterNode | null = null;
  private lowpass: BiquadFilterNode | null = null;
  private eqNodes: BiquadFilterNode[] = [];
  private makeupGain: GainNode | null = null;
  private leveler: DynamicsCompressorNode | null = null;
  private limiter: DynamicsCompressorNode | null = null;
  private fadeGain: GainNode | null = null;
  private masterGain: GainNode | null = null;
  private analyser: AnalyserNode | null = null;
  private analyserBuf: Float32Array<ArrayBuffer> | null = null;

  private currentVolume = 80;
  private currentMuted = false;
  private currentPreset: EqPresetId = 'spotify';
  private normalizeEnabled = true;

  attach(audioEl: HTMLAudioElement): void {
    if (this.attachedEl === audioEl && this.ctx) return;

    const AudioCtx = window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext;
    if (!AudioCtx) return;

    const ctx = new AudioCtx({ latencyHint: 'playback' });
    const source = ctx.createMediaElementSource(audioEl);

    const highpass = ctx.createBiquadFilter();
    highpass.type = 'highpass';
    highpass.frequency.value = 24;
    highpass.Q.value = 0.707;

    const lowpass = ctx.createBiquadFilter();
    lowpass.type = 'lowpass';
    lowpass.frequency.value = 18500;
    lowpass.Q.value = 0.707;

    const eqNodes = EQ_BANDS.map((band) => {
      const node = ctx.createBiquadFilter();
      node.type = band.type;
      node.frequency.value = band.freq;
      node.Q.value = band.q;
      node.gain.value = 0;
      return node;
    });

    // -3.3 dB trim for -14 LUFS headroom
    const leveler = ctx.createDynamicsCompressor();
    leveler.threshold.value = -12;
    leveler.knee.value = 24;
    leveler.ratio.value = 1.5;
    leveler.attack.value = 0.025;
    leveler.release.value = 0.28;

    const makeupGain = ctx.createGain();
    makeupGain.gain.value = 0.68;

    const limiter = ctx.createDynamicsCompressor();
    limiter.threshold.value = -1.5;
    limiter.knee.value = 2;
    limiter.ratio.value = 16;
    limiter.attack.value = 0.002;
    limiter.release.value = 0.08;

    const fadeGain = ctx.createGain();
    fadeGain.gain.value = 1.0;

    const masterGain = ctx.createGain();

    const analyser = ctx.createAnalyser();
    analyser.fftSize = 512;

    let prev: AudioNode = source;
    prev.connect(highpass);
    highpass.connect(lowpass);
    prev = lowpass;
    for (const eq of eqNodes) {
      prev.connect(eq);
      prev = eq;
    }
    prev.connect(makeupGain);
    makeupGain.connect(leveler);
    leveler.connect(limiter);
    limiter.connect(analyser);
    limiter.connect(fadeGain);
    fadeGain.connect(masterGain);
    masterGain.connect(ctx.destination);

    audioEl.volume = 1;
    (audioEl as HTMLAudioElement & { preservesPitch?: boolean }).preservesPitch = true;

    this.ctx = ctx;
    this.source = source;
    this.attachedEl = audioEl;
    this.highpass = highpass;
    this.lowpass = lowpass;
    this.eqNodes = eqNodes;
    this.leveler = leveler;
    this.makeupGain = makeupGain;
    this.limiter = limiter;
    this.fadeGain = fadeGain;
    this.masterGain = masterGain;
    this.analyser = analyser;
    this.analyserBuf = new Float32Array(new ArrayBuffer(analyser.fftSize * 4));

    this.applyPreset();
    this.applyNormalize();
    this.applyVolume();
  }

  async resume(): Promise<void> {
    if (this.ctx && this.ctx.state === 'suspended') {
      try {
        await this.ctx.resume();
      } catch {
      }
    }
  }

  setVolume(volPercent: number, muted: boolean): void {
    this.currentVolume = Math.max(0, Math.min(100, volPercent));
    this.currentMuted = muted;
    this.applyVolume();
  }

  setPreset(preset: string): void {
    const valid: EqPresetId =
      preset === 'flat' || preset === 'bass' || preset === 'vocal' || preset === 'electronic' ? preset : 'spotify';
    this.currentPreset = valid;
    this.applyPreset();
  }

  setNormalize(enabled: boolean): void {
    this.normalizeEnabled = enabled;
    this.applyNormalize();
  }

  fadeIn(durationSec = 0.03): void {
    if (!this.ctx || !this.fadeGain) return;
    const now = this.ctx.currentTime;
    this.fadeGain.gain.cancelScheduledValues(now);
    this.fadeGain.gain.setValueAtTime(0.001, now);
    this.fadeGain.gain.exponentialRampToValueAtTime(1.0, now + durationSec);
  }

  setFadeLevel(level: number): void {
    if (!this.ctx || !this.fadeGain) return;
    const clamped = Math.max(0.001, Math.min(1.0, level));
    this.fadeGain.gain.setTargetAtTime(clamped, this.ctx.currentTime, 0.04);
  }

  getRmsLevel(): number {
    if (!this.analyser || !this.analyserBuf) return 1;
    this.analyser.getFloatTimeDomainData(this.analyserBuf);
    let sum = 0;
    for (let i = 0; i < this.analyserBuf.length; i++) {
      const s = this.analyserBuf[i];
      sum += s * s;
    }
    return Math.sqrt(sum / this.analyserBuf.length);
  }

  private applyVolume(): void {
    if (!this.ctx || !this.masterGain) {
      if (this.attachedEl) {
        this.attachedEl.volume = this.currentMuted ? 0 : Math.pow(this.currentVolume / 100, 1.9);
      }
      return;
    }
    const linear = this.currentMuted ? 0 : Math.pow(this.currentVolume / 100, 1.9);
    this.masterGain.gain.setTargetAtTime(linear, this.ctx.currentTime, 0.015);
  }

  private applyPreset(): void {
    if (!this.ctx || this.eqNodes.length === 0) return;
    const gains = EQ_PRESETS[this.currentPreset];
    const now = this.ctx.currentTime;
    this.eqNodes.forEach((node, idx) => {
      node.gain.setTargetAtTime(gains[idx] ?? 0, now, 0.04);
    });
  }

  private applyNormalize(): void {
    if (!this.ctx || !this.leveler || !this.makeupGain) return;
    const now = this.ctx.currentTime;
    if (this.normalizeEnabled) {
      this.leveler.threshold.setTargetAtTime(-12, now, 0.03);
      this.leveler.ratio.setTargetAtTime(1.5, now, 0.03);
      this.makeupGain.gain.setTargetAtTime(0.68, now, 0.03);
    } else {
      this.leveler.threshold.setTargetAtTime(0, now, 0.03);
      this.leveler.ratio.setTargetAtTime(1, now, 0.03);
      this.makeupGain.gain.setTargetAtTime(1.0, now, 0.03);
    }
  }

  destroy(): void {
    try {
      this.source?.disconnect();
      this.highpass?.disconnect();
      this.lowpass?.disconnect();
      this.eqNodes.forEach((n) => n.disconnect());
      this.leveler?.disconnect();
      this.makeupGain?.disconnect();
      this.limiter?.disconnect();
      this.fadeGain?.disconnect();
      this.masterGain?.disconnect();
      void this.ctx?.close();
    } catch {
    }
    this.ctx = null;
    this.source = null;
    this.attachedEl = null;
  }
}
