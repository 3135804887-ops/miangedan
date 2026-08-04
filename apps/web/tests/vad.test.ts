import { describe, expect, it, vi } from 'vitest';

import { rms, SpeechDetector } from '../src/lib/vad';

describe('rms', () => {
  it('computes root mean square', () => {
    expect(rms(new Float32Array([1, -1, 1, -1]))).toBeCloseTo(1, 5);
    expect(rms(new Float32Array([0.5, -0.5, 0.5, -0.5]))).toBeCloseTo(0.5, 5);
    expect(rms(new Float32Array(8))).toBe(0);
  });
});

describe('SpeechDetector', () => {
  it('fires onSpeechStart only after attackFrames', () => {
    const onSpeechStart = vi.fn();
    const detector = new SpeechDetector(
      { threshold: 0.1, attackFrames: 2, hangoverFrames: 3 },
      { onSpeechStart },
    );
    detector.feed({ atMs: 0, rms: 0.01 });
    detector.feed({ atMs: 40, rms: 0.2 });
    expect(onSpeechStart).not.toHaveBeenCalled();
    expect(detector.speaking).toBe(false);
    detector.feed({ atMs: 80, rms: 0.2 });
    expect(onSpeechStart).toHaveBeenCalledTimes(1);
    expect(detector.speaking).toBe(true);
  });

  it('fires onSpeechEnd after hangoverFrames', () => {
    const onSpeechEnd = vi.fn();
    const detector = new SpeechDetector(
      { threshold: 0.1, attackFrames: 1, hangoverFrames: 2 },
      { onSpeechEnd },
    );
    detector.feed({ atMs: 0, rms: 0.5 });
    expect(detector.speaking).toBe(true);
    detector.feed({ atMs: 40, rms: 0.01 });
    expect(onSpeechEnd).not.toHaveBeenCalled();
    detector.feed({ atMs: 80, rms: 0.01 });
    expect(onSpeechEnd).toHaveBeenCalledTimes(1);
    expect(detector.speaking).toBe(false);
  });

  it('does not re-trigger while speaking', () => {
    const onSpeechStart = vi.fn();
    const detector = new SpeechDetector(
      { threshold: 0.1, attackFrames: 1, hangoverFrames: 1 },
      { onSpeechStart },
    );
    detector.feed({ atMs: 0, rms: 0.5 });
    detector.feed({ atMs: 40, rms: 0.5 });
    expect(onSpeechStart).toHaveBeenCalledTimes(1);
  });
});
