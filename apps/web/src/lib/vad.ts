/** 本地 VAD 打断检测（NFR-009：用户说话 → 数字人停止发声）。 */

export interface VadConfig {
  /** RMS 阈值（0~1），超过视为有声。 */
  readonly threshold: number;
  /** 连续超过阈值帧数后才触发 onSpeechStart（防瞬态噪声）。 */
  readonly attackFrames: number;
  /** 连续低于阈值帧数后才触发 onSpeechEnd（防句间断音）。 */
  readonly hangoverFrames: number;
}

export interface VadCallbacks {
  readonly onSpeechStart?: () => void;
  readonly onSpeechEnd?: () => void;
}

export interface VadFrame {
  readonly atMs: number;
  readonly rms: number;
}

export function rms(samples: Float32Array): number {
  let sum = 0;
  for (let i = 0; i < samples.length; i += 1) {
    const value = samples[i] ?? 0;
    sum += value * value;
  }
  return Math.sqrt(sum / samples.length);
}

export class SpeechDetector {
  private readonly config: VadConfig;
  private readonly callbacks: VadCallbacks;
  private over = 0;
  private under = 0;
  speaking = false;

  constructor(config: VadConfig, callbacks: VadCallbacks = {}) {
    this.config = config;
    this.callbacks = callbacks;
  }

  feed(frame: VadFrame): void {
    if (frame.rms >= this.config.threshold) {
      this.over += 1;
      this.under = 0;
    } else {
      this.under += 1;
      this.over = 0;
    }
    if (!this.speaking && this.over >= this.config.attackFrames) {
      this.speaking = true;
      this.callbacks.onSpeechStart?.();
    } else if (this.speaking && this.under >= this.config.hangoverFrames) {
      this.speaking = false;
      this.callbacks.onSpeechEnd?.();
    }
  }
}

export interface VadHandle {
  readonly stop: () => void;
}

/** 将麦克风（或任意音频流）接入本地 VAD，每 40ms 输出一帧 RMS。 */
export function createVadFromStream(
  stream: MediaStream,
  config: VadConfig,
  callbacks: VadCallbacks,
): VadHandle {
  const audioContext = new AudioContext();
  const source = audioContext.createMediaStreamSource(stream);
  const analyser = audioContext.createAnalyser();
  analyser.fftSize = 512;
  analyser.smoothingTimeConstant = 0.2;
  source.connect(analyser);
  const samples = new Float32Array(analyser.fftSize);
  const detector = new SpeechDetector(config, callbacks);
  const frameMs = 40;
  let running = true;

  const tick = (): void => {
    if (!running) {
      return;
    }
    analyser.getFloatTimeDomainData(samples);
    detector.feed({
      atMs: audioContext.currentTime * 1000,
      rms: rms(samples),
    });
    window.setTimeout(tick, frameMs);
  };
  tick();

  return {
    stop(): void {
      running = false;
      source.disconnect();
      analyser.disconnect();
      void audioContext.close();
    },
  };
}
