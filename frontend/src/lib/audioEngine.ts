export type EqPresetId = 'flat' | 'spotify' | 'bass' | 'vocal' | 'electronic';
export type NormalizeLevel = 'quiet' | 'normal' | 'loud';
export type Spatial8DSpeed = 'slow' | 'normal' | 'fast';

const EQ_BANDS: ReadonlyArray<{ freq: number; type: BiquadFilterType; q: number }> = [
  { freq: 65, type: 'lowshelf', q: 0.7 },
  { freq: 260, type: 'peaking', q: 0.9 },
  { freq: 1000, type: 'peaking', q: 0.85 },
  { freq: 3500, type: 'peaking', q: 1.1 },
  { freq: 12000, type: 'highshelf', q: 0.65 }
];

const EQ_PRESETS: Record<EqPresetId, readonly [number, number, number, number, number]> = {
  flat: [0, 0, 0, 0, 0],
  spotify: [1.5, -1.5, 0.5, -1.8, 1.8],
  bass: [6.5, 2.5, -1.0, -1.5, 0.5],
  vocal: [-3.0, -2.0, 4.5, 2.5, 1.0],
  electronic: [5.0, -2.0, 0.5, 1.5, 4.5]
};

const NORMALIZE_OFFSETS_DB: Record<NormalizeLevel, number> = {
  quiet: -5.0,  // -19 LUFS
  normal: 0.0,  // -14 LUFS (Spotify default)
  loud: 3.0     // -11 LUFS
};

const SPATIAL_8D_PERIOD_SEC: Record<Spatial8DSpeed, number> = {
  slow: 14.0,
  normal: 9.5,
  fast: 6.0
};

function createStereoHallImpulse(ctx: AudioContext): AudioBuffer {
  const sampleRate = ctx.sampleRate || 48000;
  const durationSec = 1.25;
  const length = Math.floor(sampleRate * durationSec);
  const buffer = ctx.createBuffer(2, length, sampleRate);
  const chL = buffer.getChannelData(0);
  const chR = buffer.getChannelData(1);

  // Energy-mirrored early reflections after 32 ms pre-delay so L/R clarity is identical
  const earlyTaps: ReadonlyArray<[number, number, number]> = [
    [0.033, 0.24, 0.18],
    [0.036, 0.18, 0.24],
    [0.051, 0.20, 0.14],
    [0.054, 0.14, 0.20]
  ];
  for (const [sec, gainL, gainR] of earlyTaps) {
    const idx = Math.min(length - 1, Math.floor(sec * sampleRate));
    chL[idx] += gainL;
    chR[idx] += gainR;
  }

  // Decorrelated airy stereo diffuse tail with smooth fade-in and tight decay
  let seedL = 1337;
  let seedR = 7331;
  const nextNoiseL = () => {
    seedL = (seedL * 1664525 + 1013904223) >>> 0;
    return (seedL / 0xffffffff) * 2 - 1;
  };
  const nextNoiseR = () => {
    seedR = (seedR * 1664525 + 1013904223) >>> 0;
    return (seedR / 0xffffffff) * 2 - 1;
  };

  let lpL = 0;
  let lpR = 0;
  const dampAlpha = 0.78;
  const preDelaySamples = Math.floor(0.032 * sampleRate);

  for (let i = preDelaySamples; i < length; i++) {
    const t = (i - preDelaySamples) / Math.max(1, length - preDelaySamples);
    const fadeIn = Math.min(1, (i - preDelaySamples) / (0.035 * sampleRate));
    const decay = Math.pow(1 - t, 3.8) * fadeIn * 0.18;

    lpL = lpL + dampAlpha * (nextNoiseL() - lpL);
    lpR = lpR + dampAlpha * (nextNoiseR() - lpR);
    chL[i] += lpL * decay;
    chR[i] += lpR * decay;
  }

  return buffer;
}

export class VeloraAudioEngine {
  private ctx: AudioContext | null = null;
  private source: MediaElementAudioSourceNode | null = null;
  private attachedEl: HTMLAudioElement | null = null;

  private lufsGainNode: GainNode | null = null;
  private eqNodes: BiquadFilterNode[] = [];
  private postEqNode: AudioNode | null = null;
  private postSpatialBus: GainNode | null = null;
  private loudLimiter: DynamicsCompressorNode | null = null;
  private postLimiterBus: GainNode | null = null;

  // True-Stereo Binaural 8D DSP nodes (only connected when spatial8DEnabled === true)
  private spatialInputBus: GainNode | null = null;
  private stereoSplitter: ChannelSplitterNode | null = null;
  private gainLL: GainNode | null = null;
  private gainLR: GainNode | null = null;
  private gainRL: GainNode | null = null;
  private gainRR: GainNode | null = null;
  private earSumL: GainNode | null = null;
  private earSumR: GainNode | null = null;
  private itdDelayL: DelayNode | null = null;
  private itdDelayR: DelayNode | null = null;
  private proxShelfL: BiquadFilterNode | null = null;
  private proxShelfR: BiquadFilterNode | null = null;
  private blauertBandL: BiquadFilterNode | null = null;
  private blauertBandR: BiquadFilterNode | null = null;
  private headShadowL: BiquadFilterNode | null = null;
  private headShadowR: BiquadFilterNode | null = null;
  private pinnaPeakL: BiquadFilterNode | null = null;
  private pinnaPeakR: BiquadFilterNode | null = null;
  private pinnaNotchL: BiquadFilterNode | null = null;
  private pinnaNotchR: BiquadFilterNode | null = null;
  private pinnaAirL: BiquadFilterNode | null = null;
  private pinnaAirR: BiquadFilterNode | null = null;
  private msMidGainL: GainNode | null = null;
  private msMidGainR: GainNode | null = null;
  private msSubCompShelf: BiquadFilterNode | null = null;
  private msSideGainL: GainNode | null = null;
  private msSideGainR: GainNode | null = null;
  private msSideHighpass: BiquadFilterNode | null = null;
  private msSideOutL: GainNode | null = null;
  private msSideOutR: GainNode | null = null;
  private stereoMerger: ChannelMergerNode | null = null;
  private spatialDryGain: GainNode | null = null;
  private erSplitter: ChannelSplitterNode | null = null;
  private erDelayL: DelayNode | null = null;
  private erDelayR: DelayNode | null = null;
  private erCrossGainLtoR: GainNode | null = null;
  private erCrossGainRtoL: GainNode | null = null;
  private erMerger: ChannelMergerNode | null = null;
  private erHighpass: BiquadFilterNode | null = null;
  private erLowpass: BiquadFilterNode | null = null;
  private reverbDirSend: GainNode | null = null;
  private reverbOmniSend: GainNode | null = null;
  private reverbHighpass: BiquadFilterNode | null = null;
  private reverbLowpass: BiquadFilterNode | null = null;
  private reverbConvolver: ConvolverNode | null = null;
  private reverbWetGain: GainNode | null = null;
  private reverbBufferReady = false;

  private fadeGain: GainNode | null = null;
  private masterGain: GainNode | null = null;
  private analyser: AnalyserNode | null = null;
  private analyserBuf: Float32Array<ArrayBuffer> | null = null;

  private currentVolume = 100;
  private currentMuted = false;
  private currentPreset: EqPresetId = 'flat';
  private normalizeEnabled = true;
  private normalizeLevel: NormalizeLevel = 'normal';
  private trackLoudnessDb: number | null = null;

  private spatial8DEnabled = false;
  private spatial8DSpeed: Spatial8DSpeed = 'normal';
  private spatialConnected = false;
  private loudLimiterConnected = false;
  private orbitTimer: number | null = null;
  private orbitAngle = 0;
  private lastOrbitTickAt = 0;

  attach(audioEl: HTMLAudioElement): void {
    if (this.attachedEl === audioEl && this.ctx) return;

    const AudioCtx = window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext;
    if (!AudioCtx) return;

    const ctx = new AudioCtx({ latencyHint: 'playback' });
    const source = ctx.createMediaElementSource(audioEl);

    const lufsGainNode = ctx.createGain();
    lufsGainNode.gain.value = 1.0;

    const eqNodes = EQ_BANDS.map((band) => {
      const node = ctx.createBiquadFilter();
      node.type = band.type;
      node.frequency.value = band.freq;
      node.Q.value = band.q;
      node.gain.value = 0;
      return node;
    });

    const postSpatialBus = ctx.createGain();
    postSpatialBus.gain.value = 1.0;

    // Spotify Loud-mode (-11 LUFS) safety limiter: -1.0 dB, 5 ms attack, 100 ms release
    const loudLimiter = ctx.createDynamicsCompressor();
    loudLimiter.threshold.value = -1.0;
    loudLimiter.knee.value = 0;
    loudLimiter.ratio.value = 20;
    loudLimiter.attack.value = 0.005;
    loudLimiter.release.value = 0.1;

    const postLimiterBus = ctx.createGain();
    postLimiterBus.gain.value = 1.0;

    // True-Stereo Binaural 8D DSP branch (kept disconnected until enabled)
    const spatialInputBus = ctx.createGain();
    spatialInputBus.gain.value = 1.0;

    const stereoSplitter = ctx.createChannelSplitter(2);
    const gainLL = ctx.createGain();
    const gainLR = ctx.createGain();
    const gainRL = ctx.createGain();
    const gainRR = ctx.createGain();
    const earSumL = ctx.createGain();
    const earSumR = ctx.createGain();

    const itdDelayL = ctx.createDelay(0.01);
    const itdDelayR = ctx.createDelay(0.01);

    const createEarChain = () => {
      const prox = ctx.createBiquadFilter();
      prox.type = 'lowshelf';
      prox.frequency.value = 135;
      prox.gain.value = 0;

      const blauert = ctx.createBiquadFilter();
      blauert.type = 'peaking';
      blauert.frequency.value = 1100;
      blauert.Q.value = 1.35;
      blauert.gain.value = 0;

      const shadow = ctx.createBiquadFilter();
      shadow.type = 'highshelf';
      shadow.frequency.value = 1650;
      shadow.gain.value = 0;

      const peak = ctx.createBiquadFilter();
      peak.type = 'peaking';
      peak.frequency.value = 3700;
      peak.Q.value = 1.1;
      peak.gain.value = 0;

      const notch = ctx.createBiquadFilter();
      notch.type = 'peaking';
      notch.frequency.value = 8300;
      notch.Q.value = 3.6;
      notch.gain.value = 0;

      const air = ctx.createBiquadFilter();
      air.type = 'highshelf';
      air.frequency.value = 11500;
      air.gain.value = 0;

      return { prox, blauert, shadow, peak, notch, air };
    };

    const earL = createEarChain();
    const earR = createEarChain();

    // Elliptical Mid/Side sub-bass anchor (< 85 Hz centered with 0 dB crossover ripple)
    const msMidGainL = ctx.createGain();
    msMidGainL.gain.value = 0.5;
    const msMidGainR = ctx.createGain();
    msMidGainR.gain.value = 0.5;

    const msSubCompShelf = ctx.createBiquadFilter();
    msSubCompShelf.type = 'lowshelf';
    msSubCompShelf.frequency.value = 80;
    msSubCompShelf.gain.value = 0;

    const msSideGainL = ctx.createGain();
    msSideGainL.gain.value = 0.5;
    const msSideGainR = ctx.createGain();
    msSideGainR.gain.value = -0.5;

    const msSideHighpass = ctx.createBiquadFilter();
    msSideHighpass.type = 'highpass';
    msSideHighpass.frequency.value = 85;
    msSideHighpass.Q.value = 0.707;

    const msSideOutL = ctx.createGain();
    msSideOutL.gain.value = 1.0;
    const msSideOutR = ctx.createGain();
    msSideOutR.gain.value = -1.0;

    const stereoMerger = ctx.createChannelMerger(2);
    const spatialDryGain = ctx.createGain();
    spatialDryGain.gain.value = 0.94;

    // Cross-wall early reflections for strong out-of-head externalization
    const erSplitter = ctx.createChannelSplitter(2);
    const erDelayL = ctx.createDelay(0.05);
    erDelayL.delayTime.value = 0.022;
    const erDelayR = ctx.createDelay(0.05);
    erDelayR.delayTime.value = 0.022;
    const erCrossGainLtoR = ctx.createGain();
    erCrossGainLtoR.gain.value = 0.13;
    const erCrossGainRtoL = ctx.createGain();
    erCrossGainRtoL.gain.value = 0.13;
    const erMerger = ctx.createChannelMerger(2);
    const erHighpass = ctx.createBiquadFilter();
    erHighpass.type = 'highpass';
    erHighpass.frequency.value = 240;
    erHighpass.Q.value = 0.707;
    const erLowpass = ctx.createBiquadFilter();
    erLowpass.type = 'lowpass';
    erLowpass.frequency.value = 7800;
    erLowpass.Q.value = 0.707;

    const reverbDirSend = ctx.createGain();
    reverbDirSend.gain.value = 0.68;
    const reverbOmniSend = ctx.createGain();
    reverbOmniSend.gain.value = 0.32;

    const reverbHighpass = ctx.createBiquadFilter();
    reverbHighpass.type = 'highpass';
    reverbHighpass.frequency.value = 280;
    reverbHighpass.Q.value = 0.707;

    const reverbLowpass = ctx.createBiquadFilter();
    reverbLowpass.type = 'lowpass';
    reverbLowpass.frequency.value = 10500;
    reverbLowpass.Q.value = 0.707;

    const reverbConvolver = ctx.createConvolver();
    const reverbWetGain = ctx.createGain();
    reverbWetGain.gain.value = 0.14;

    const fadeGain = ctx.createGain();
    fadeGain.gain.value = 1.0;

    const masterGain = ctx.createGain();

    const analyser = ctx.createAnalyser();
    analyser.fftSize = 512;

    // Core transparent chain: source -> lufsGainNode -> eqNodes
    let prev: AudioNode = source;
    prev.connect(lufsGainNode);
    prev = lufsGainNode;
    for (const eq of eqNodes) {
      prev.connect(eq);
      prev = eq;
    }

    // Internal static wiring for the True-Stereo Binaural 8D branch
    spatialInputBus.connect(stereoSplitter);
    stereoSplitter.connect(gainLL, 0);
    stereoSplitter.connect(gainLR, 0);
    stereoSplitter.connect(gainRL, 1);
    stereoSplitter.connect(gainRR, 1);
    gainLL.connect(earSumL);
    gainRL.connect(earSumL);
    gainLR.connect(earSumR);
    gainRR.connect(earSumR);

    earSumL.connect(itdDelayL);
    itdDelayL.connect(earL.prox);
    earL.prox.connect(earL.blauert);
    earL.blauert.connect(earL.shadow);
    earL.shadow.connect(earL.peak);
    earL.peak.connect(earL.notch);
    earL.notch.connect(earL.air);

    earSumR.connect(itdDelayR);
    itdDelayR.connect(earR.prox);
    earR.prox.connect(earR.blauert);
    earR.blauert.connect(earR.shadow);
    earR.shadow.connect(earR.peak);
    earR.peak.connect(earR.notch);
    earR.notch.connect(earR.air);

    earL.air.connect(msMidGainL);
    earR.air.connect(msMidGainR);
    msMidGainL.connect(msSubCompShelf);
    msMidGainR.connect(msSubCompShelf);

    earL.air.connect(msSideGainL);
    earR.air.connect(msSideGainR);
    msSideGainL.connect(msSideHighpass);
    msSideGainR.connect(msSideHighpass);

    msSubCompShelf.connect(stereoMerger, 0, 0);
    msSubCompShelf.connect(stereoMerger, 0, 1);
    msSideHighpass.connect(msSideOutL);
    msSideHighpass.connect(msSideOutR);
    msSideOutL.connect(stereoMerger, 0, 0);
    msSideOutR.connect(stereoMerger, 0, 1);

    stereoMerger.connect(spatialDryGain);
    spatialDryGain.connect(postSpatialBus);

    // Opposite-wall early reflections
    stereoMerger.connect(erSplitter);
    erSplitter.connect(erDelayL, 0);
    erDelayL.connect(erCrossGainLtoR);
    erCrossGainLtoR.connect(erMerger, 0, 1);
    erSplitter.connect(erDelayR, 1);
    erDelayR.connect(erCrossGainRtoL);
    erCrossGainRtoL.connect(erMerger, 0, 0);
    erMerger.connect(erHighpass);
    erHighpass.connect(erLowpass);
    erLowpass.connect(postSpatialBus);

    // Directionally-followed stereo hall aux
    stereoMerger.connect(reverbDirSend);
    reverbDirSend.connect(reverbHighpass);
    spatialInputBus.connect(reverbOmniSend);
    reverbOmniSend.connect(reverbHighpass);
    reverbHighpass.connect(reverbLowpass);
    reverbLowpass.connect(reverbConvolver);
    reverbConvolver.connect(reverbWetGain);
    reverbWetGain.connect(postSpatialBus);

    postLimiterBus.connect(analyser);
    postLimiterBus.connect(fadeGain);
    fadeGain.connect(masterGain);
    masterGain.connect(ctx.destination);

    audioEl.volume = 1;
    (audioEl as HTMLAudioElement & { preservesPitch?: boolean }).preservesPitch = true;

    this.ctx = ctx;
    this.source = source;
    this.attachedEl = audioEl;
    this.lufsGainNode = lufsGainNode;
    this.eqNodes = eqNodes;
    this.postEqNode = prev;
    this.postSpatialBus = postSpatialBus;
    this.loudLimiter = loudLimiter;
    this.postLimiterBus = postLimiterBus;
    this.spatialInputBus = spatialInputBus;
    this.stereoSplitter = stereoSplitter;
    this.gainLL = gainLL;
    this.gainLR = gainLR;
    this.gainRL = gainRL;
    this.gainRR = gainRR;
    this.earSumL = earSumL;
    this.earSumR = earSumR;
    this.itdDelayL = itdDelayL;
    this.itdDelayR = itdDelayR;
    this.proxShelfL = earL.prox;
    this.proxShelfR = earR.prox;
    this.blauertBandL = earL.blauert;
    this.blauertBandR = earR.blauert;
    this.headShadowL = earL.shadow;
    this.headShadowR = earR.shadow;
    this.pinnaPeakL = earL.peak;
    this.pinnaPeakR = earR.peak;
    this.pinnaNotchL = earL.notch;
    this.pinnaNotchR = earR.notch;
    this.pinnaAirL = earL.air;
    this.pinnaAirR = earR.air;
    this.msMidGainL = msMidGainL;
    this.msMidGainR = msMidGainR;
    this.msSubCompShelf = msSubCompShelf;
    this.msSideGainL = msSideGainL;
    this.msSideGainR = msSideGainR;
    this.msSideHighpass = msSideHighpass;
    this.msSideOutL = msSideOutL;
    this.msSideOutR = msSideOutR;
    this.stereoMerger = stereoMerger;
    this.spatialDryGain = spatialDryGain;
    this.erSplitter = erSplitter;
    this.erDelayL = erDelayL;
    this.erDelayR = erDelayR;
    this.erCrossGainLtoR = erCrossGainLtoR;
    this.erCrossGainRtoL = erCrossGainRtoL;
    this.erMerger = erMerger;
    this.erHighpass = erHighpass;
    this.erLowpass = erLowpass;
    this.reverbDirSend = reverbDirSend;
    this.reverbOmniSend = reverbOmniSend;
    this.reverbHighpass = reverbHighpass;
    this.reverbLowpass = reverbLowpass;
    this.reverbConvolver = reverbConvolver;
    this.reverbWetGain = reverbWetGain;
    this.reverbBufferReady = false;
    this.fadeGain = fadeGain;
    this.masterGain = masterGain;
    this.analyser = analyser;
    this.analyserBuf = new Float32Array(new ArrayBuffer(analyser.fftSize * 4));
    this.spatialConnected = false;
    this.loudLimiterConnected = false;

    this.updateSpatialRouting();
    this.updateLimiterRouting();
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
      preset === 'spotify' || preset === 'bass' || preset === 'vocal' || preset === 'electronic' ? preset : 'flat';
    this.currentPreset = valid;
    this.applyPreset();
  }

  setNormalize(enabled: boolean, level: NormalizeLevel = 'normal'): void {
    const validLevel: NormalizeLevel = level === 'quiet' || level === 'loud' ? level : 'normal';
    this.normalizeEnabled = enabled;
    this.normalizeLevel = validLevel;
    this.updateLimiterRouting();
    this.applyNormalize();
  }

  setTrackLoudness(loudnessDb: number | null): void {
    this.trackLoudnessDb = loudnessDb !== null && Number.isFinite(loudnessDb) ? loudnessDb : null;
    this.applyNormalize();
  }

  setSpatial8D(enabled: boolean, speed: Spatial8DSpeed = 'normal'): void {
    const validSpeed: Spatial8DSpeed = speed === 'slow' || speed === 'fast' ? speed : 'normal';
    this.spatial8DEnabled = enabled;
    this.spatial8DSpeed = validSpeed;
    this.updateSpatialRouting();
    this.syncOrbitTimer();
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

  private updateSpatialRouting(): void {
    if (!this.ctx || !this.postEqNode || !this.postSpatialBus || !this.spatialInputBus) {
      return;
    }

    if (this.spatial8DEnabled) {
      if (!this.reverbBufferReady && this.reverbConvolver) {
        this.reverbConvolver.buffer = createStereoHallImpulse(this.ctx);
        this.reverbBufferReady = true;
      }
      if (!this.spatialConnected) {
        try {
          this.postEqNode.disconnect(this.postSpatialBus);
        } catch {
        }
        this.postEqNode.connect(this.spatialInputBus);
        this.spatialConnected = true;
      } else {
        try {
          this.postEqNode.disconnect(this.postSpatialBus);
        } catch {
        }
      }
      this.applyOrbitPosition(true);
    } else {
      if (this.spatialConnected) {
        try {
          this.postEqNode.disconnect(this.spatialInputBus);
        } catch {
        }
        this.spatialConnected = false;
      }
      try {
        this.postEqNode.disconnect(this.postSpatialBus);
      } catch {
      }
      this.postEqNode.connect(this.postSpatialBus);
    }
  }

  private updateLimiterRouting(): void {
    if (!this.postSpatialBus || !this.loudLimiter || !this.postLimiterBus) return;
    const useLimiter = this.normalizeEnabled && this.normalizeLevel === 'loud';

    if (useLimiter) {
      if (!this.loudLimiterConnected) {
        try {
          this.postSpatialBus.disconnect(this.postLimiterBus);
        } catch {
        }
        this.postSpatialBus.connect(this.loudLimiter);
        this.loudLimiter.connect(this.postLimiterBus);
        this.loudLimiterConnected = true;
      }
    } else {
      if (this.loudLimiterConnected) {
        try {
          this.postSpatialBus.disconnect(this.loudLimiter);
        } catch {
        }
        try {
          this.loudLimiter.disconnect(this.postLimiterBus);
        } catch {
        }
        this.loudLimiterConnected = false;
      }
      try {
        this.postSpatialBus.disconnect(this.postLimiterBus);
      } catch {
      }
      this.postSpatialBus.connect(this.postLimiterBus);
    }
  }

  private syncOrbitTimer(): void {
    if (this.spatial8DEnabled && this.ctx) {
      if (this.orbitTimer === null) {
        this.lastOrbitTickAt = performance.now();
        this.orbitTimer = window.setInterval(() => {
          this.stepOrbit();
        }, 16);
      }
    } else if (this.orbitTimer !== null) {
      clearInterval(this.orbitTimer);
      this.orbitTimer = null;
    }
  }

  private stepOrbit(): void {
    const now = performance.now();
    const dtSec = Math.min(0.25, Math.max(0, (now - this.lastOrbitTickAt) / 1000));
    this.lastOrbitTickAt = now;

    if (!this.spatial8DEnabled || !this.attachedEl || this.attachedEl.paused) {
      return;
    }
    const period = SPATIAL_8D_PERIOD_SEC[this.spatial8DSpeed] || 9.5;
    this.orbitAngle = (this.orbitAngle + (Math.PI * 2 * dtSec) / period) % (Math.PI * 2);
    this.applyOrbitPosition(false);
  }

  private setParam(param: AudioParam, value: number, time: number, immediate: boolean): void {
    if (immediate) {
      param.setValueAtTime(value, time);
    } else {
      param.setTargetAtTime(value, time, 0.024);
    }
  }

  private applyOrbitPosition(immediate: boolean): void {
    if (
      !this.ctx ||
      !this.gainLL ||
      !this.gainLR ||
      !this.gainRL ||
      !this.gainRR ||
      !this.itdDelayL ||
      !this.itdDelayR ||
      !this.proxShelfL ||
      !this.proxShelfR ||
      !this.blauertBandL ||
      !this.blauertBandR ||
      !this.headShadowL ||
      !this.headShadowR ||
      !this.pinnaPeakL ||
      !this.pinnaPeakR ||
      !this.pinnaNotchL ||
      !this.pinnaNotchR ||
      !this.pinnaAirL ||
      !this.pinnaAirR ||
      !this.msSubCompShelf ||
      !this.erDelayL ||
      !this.erDelayR ||
      !this.erCrossGainLtoR ||
      !this.erCrossGainRtoL ||
      !this.reverbWetGain
    ) {
      return;
    }

    // Organic elliptical pacing & distance parallax
    const warpedAngle = this.orbitAngle + 0.08 * Math.sin(this.orbitAngle * 2);
    const s = Math.sin(warpedAngle); // -1 Left .. +1 Right
    const c = Math.cos(warpedAngle); // +1 Front .. -1 Back
    const absS = Math.abs(s);
    const sideProx = s * s;
    // Concentrate rear cue strictly behind the head (150°..210°) so Left (270°) arrives 100% open & crisp
    const rearness = Math.pow(Math.max(0, -c), 2.2);

    const distGain = 1.0 - 0.09 * (1 - sideProx) - 0.03 * rearness;

    // Dual virtual studio monitors: wide discrete stereo in front/back, 100% equal L+R (0.0 dB bias) at both Left & Right
    const delta = 0.46 + 0.32 * (c * c);
    const cosDelta = Math.cos(delta);
    const sSrcL = Math.max(-1, Math.min(1, Math.sin(warpedAngle - delta) / cosDelta));
    const sSrcR = Math.max(-1, Math.min(1, Math.sin(warpedAngle + delta) / cosDelta));

    const alphaL = Math.PI * 0.25 * (sSrcL + 1);
    const alphaR = Math.PI * 0.25 * (sSrcR + 1);
    const scale = 0.5 * distGain;

    const ll = (0.075 + 1.08 * Math.pow(Math.max(0, Math.cos(alphaL)), 0.78)) * scale;
    const lr = (0.075 + 1.08 * Math.pow(Math.max(0, Math.sin(alphaL)), 0.78)) * scale;
    const rl = (0.075 + 1.08 * Math.pow(Math.max(0, Math.cos(alphaR)), 0.78)) * scale;
    const rr = (0.075 + 1.08 * Math.pow(Math.max(0, Math.sin(alphaR)), 0.78)) * scale;

    // Full Woodworth spherical-head ITD (0.06 ms base .. 0.66 ms max)
    const latRad = Math.asin(Math.min(1, absS));
    const woodworth = (latRad + absS) / (Math.PI * 0.5 + 1);
    const delayL = 0.00006 + (s > 0 ? woodworth * 0.00066 : 0);
    const delayR = 0.00006 + (s < 0 ? woodworth * 0.00066 : 0);

    // Near-field proximity warmth (135 Hz lowshelf, symmetric)
    const proxL = 1.3 * Math.pow(Math.max(0, -s), 1.5);
    const proxR = 1.3 * Math.pow(Math.max(0, s), 1.5);

    // Asymmetric spherical head-shadow + near-ear baffle clarity boost (1650 Hz)
    const shadowL = s < 0 ? 1.8 * Math.max(0, -s) : -7.5 * Math.pow(Math.max(0, s), 0.85);
    const shadowR = s > 0 ? 1.8 * Math.max(0, s) : -7.5 * Math.pow(Math.max(0, -s), 0.85);

    // Per-ear concha clarity (3700 Hz) so BOTH Left & Right ears get +1.8 dB crisp presence when facing the source
    const leftFocus = Math.max(0, -s);
    const rightFocus = Math.max(0, s);
    const frontFocus = Math.max(0, c);
    const pinnaPeakL = 1.5 * frontFocus + 1.8 * leftFocus - 1.4 * rearness * (1 - 0.7 * leftFocus);
    const pinnaPeakR = 1.5 * frontFocus + 1.8 * rightFocus - 1.4 * rearness * (1 - 0.7 * rightFocus);

    // Blauert rear band & pinna notch strictly confined to rear arc; ultra-high air always open
    const blauertDb = 2.0 * rearness - 0.6 * frontFocus;
    const notchFreq = 8300 + 1100 * c;
    const notchGainDb = -4.6 * rearness;
    const pinnaAirDb = 0.4 + 0.7 * (1 - rearness);

    // Dynamic cross-wall early reflections for out-of-head externalization
    const erDelayLSec = 0.016 + 0.012 * leftFocus + 0.005 * rearness;
    const erDelayRSec = 0.016 + 0.012 * rightFocus + 0.005 * rearness;
    const erCrossGain = 0.105 + 0.045 * sideProx;

    // Sub-bass side-pan level compensation & rear hall bloom
    const subCompDb = 2.6 * sideProx;
    const wetGain = 0.11 + 0.06 * rearness;

    const t = this.ctx.currentTime;
    this.setParam(this.gainLL.gain, ll, t, immediate);
    this.setParam(this.gainRL.gain, rl, t, immediate);
    this.setParam(this.gainLR.gain, lr, t, immediate);
    this.setParam(this.gainRR.gain, rr, t, immediate);
    this.setParam(this.itdDelayL.delayTime, delayL, t, immediate);
    this.setParam(this.itdDelayR.delayTime, delayR, t, immediate);
    this.setParam(this.proxShelfL.gain, proxL, t, immediate);
    this.setParam(this.proxShelfR.gain, proxR, t, immediate);
    this.setParam(this.blauertBandL.gain, blauertDb, t, immediate);
    this.setParam(this.blauertBandR.gain, blauertDb, t, immediate);
    this.setParam(this.headShadowL.gain, shadowL, t, immediate);
    this.setParam(this.headShadowR.gain, shadowR, t, immediate);
    this.setParam(this.pinnaPeakL.gain, pinnaPeakL, t, immediate);
    this.setParam(this.pinnaPeakR.gain, pinnaPeakR, t, immediate);
    this.setParam(this.pinnaNotchL.frequency, notchFreq, t, immediate);
    this.setParam(this.pinnaNotchR.frequency, notchFreq, t, immediate);
    this.setParam(this.pinnaNotchL.gain, notchGainDb, t, immediate);
    this.setParam(this.pinnaNotchR.gain, notchGainDb, t, immediate);
    this.setParam(this.pinnaAirL.gain, pinnaAirDb, t, immediate);
    this.setParam(this.pinnaAirR.gain, pinnaAirDb, t, immediate);
    this.setParam(this.msSubCompShelf.gain, subCompDb, t, immediate);
    this.setParam(this.erDelayL.delayTime, erDelayLSec, t, immediate);
    this.setParam(this.erDelayR.delayTime, erDelayRSec, t, immediate);
    this.setParam(this.erCrossGainLtoR.gain, erCrossGain, t, immediate);
    this.setParam(this.erCrossGainRtoL.gain, erCrossGain, t, immediate);
    this.setParam(this.reverbWetGain.gain, wetGain, t, immediate);
  }

  private applyVolume(): void {
    const linear = this.currentMuted
      ? 0
      : this.currentVolume >= 100
        ? 1.0
        : Math.pow(this.currentVolume / 100, 1.65);

    if (!this.ctx || !this.masterGain) {
      if (this.attachedEl) {
        this.attachedEl.volume = linear;
      }
      return;
    }
    this.masterGain.gain.setTargetAtTime(linear, this.ctx.currentTime, 0.015);
  }

  private applyPreset(): void {
    if (!this.ctx || this.eqNodes.length === 0) return;
    const gains = EQ_PRESETS[this.currentPreset] ?? EQ_PRESETS.flat;
    const now = this.ctx.currentTime;
    this.eqNodes.forEach((node, idx) => {
      node.gain.setTargetAtTime(gains[idx] ?? 0, now, 0.04);
    });
  }

  private applyNormalize(): void {
    if (!this.ctx || !this.lufsGainNode) return;
    const now = this.ctx.currentTime;
    if (!this.normalizeEnabled || this.trackLoudnessDb === null) {
      this.lufsGainNode.gain.setTargetAtTime(1.0, now, 0.03);
      return;
    }

    // Spotify-style static ITU-R BS.1770 LUFS gain with headroom safeguard
    const targetOffsetDb = NORMALIZE_OFFSETS_DB[this.normalizeLevel] ?? 0;
    const maxBoostDb = this.normalizeLevel === 'loud' ? 6.0 : 3.5;
    const deltaDb = Math.max(-18.0, Math.min(maxBoostDb, targetOffsetDb - this.trackLoudnessDb));
    const gain = Math.pow(10, deltaDb / 20);
    this.lufsGainNode.gain.setTargetAtTime(gain, now, 0.03);
  }

  destroy(): void {
    if (this.orbitTimer !== null) {
      clearInterval(this.orbitTimer);
      this.orbitTimer = null;
    }
    try {
      this.source?.disconnect();
      this.lufsGainNode?.disconnect();
      this.eqNodes.forEach((n) => n.disconnect());
      this.spatialInputBus?.disconnect();
      this.stereoSplitter?.disconnect();
      this.gainLL?.disconnect();
      this.gainLR?.disconnect();
      this.gainRL?.disconnect();
      this.gainRR?.disconnect();
      this.earSumL?.disconnect();
      this.earSumR?.disconnect();
      this.itdDelayL?.disconnect();
      this.itdDelayR?.disconnect();
      this.proxShelfL?.disconnect();
      this.proxShelfR?.disconnect();
      this.blauertBandL?.disconnect();
      this.blauertBandR?.disconnect();
      this.headShadowL?.disconnect();
      this.headShadowR?.disconnect();
      this.pinnaPeakL?.disconnect();
      this.pinnaPeakR?.disconnect();
      this.pinnaNotchL?.disconnect();
      this.pinnaNotchR?.disconnect();
      this.pinnaAirL?.disconnect();
      this.pinnaAirR?.disconnect();
      this.msMidGainL?.disconnect();
      this.msMidGainR?.disconnect();
      this.msSubCompShelf?.disconnect();
      this.msSideGainL?.disconnect();
      this.msSideGainR?.disconnect();
      this.msSideHighpass?.disconnect();
      this.msSideOutL?.disconnect();
      this.msSideOutR?.disconnect();
      this.stereoMerger?.disconnect();
      this.spatialDryGain?.disconnect();
      this.erSplitter?.disconnect();
      this.erDelayL?.disconnect();
      this.erDelayR?.disconnect();
      this.erCrossGainLtoR?.disconnect();
      this.erCrossGainRtoL?.disconnect();
      this.erMerger?.disconnect();
      this.erHighpass?.disconnect();
      this.erLowpass?.disconnect();
      this.reverbDirSend?.disconnect();
      this.reverbOmniSend?.disconnect();
      this.reverbHighpass?.disconnect();
      this.reverbLowpass?.disconnect();
      this.reverbConvolver?.disconnect();
      this.reverbWetGain?.disconnect();
      this.postSpatialBus?.disconnect();
      this.loudLimiter?.disconnect();
      this.postLimiterBus?.disconnect();
      this.fadeGain?.disconnect();
      this.masterGain?.disconnect();
      void this.ctx?.close();
    } catch {
    }
    this.ctx = null;
    this.source = null;
    this.attachedEl = null;
    this.spatialConnected = false;
    this.loudLimiterConnected = false;
  }
}
