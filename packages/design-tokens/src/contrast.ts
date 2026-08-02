/**
 * WCAG 2.2 相对亮度与对比度计算（docs/design/ACCESSIBILITY.md 第 4.1 节）。
 * 正文 ≥4.5:1、大文本 ≥3:1；本模块只做计算，阈值判定在 check.ts。
 */

export type TextSize = 'body' | 'large';

export const CONTRAST_THRESHOLD: Readonly<Record<TextSize, number>> = {
  body: 4.5,
  large: 3,
};

export interface Rgb {
  readonly r: number;
  readonly g: number;
  readonly b: number;
}

/** 解析 #RGB 或 #RRGGBB；非法输入抛错而不是静默回退，避免令牌拼写错误被吞掉。 */
export function parseHexColor(hex: string): Rgb {
  const normalized = hex.trim().replace(/^#/, '');
  const expanded =
    normalized.length === 3
      ? normalized
          .split('')
          .map((c) => c + c)
          .join('')
      : normalized;

  if (!/^[0-9a-fA-F]{6}$/.test(expanded)) {
    throw new Error(`非法颜色值 "${hex}"，期望 #RGB 或 #RRGGBB`);
  }

  return {
    r: Number.parseInt(expanded.slice(0, 2), 16),
    g: Number.parseInt(expanded.slice(2, 4), 16),
    b: Number.parseInt(expanded.slice(4, 6), 16),
  };
}

function channelToLinear(value8Bit: number): number {
  const c = value8Bit / 255;
  return c <= 0.04045 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
}

/** WCAG 相对亮度 L。 */
export function relativeLuminance(color: Rgb): number {
  return (
    0.2126 * channelToLinear(color.r) +
    0.7152 * channelToLinear(color.g) +
    0.0722 * channelToLinear(color.b)
  );
}

/** 对比度比值，返回值 ≥1；与前后景顺序无关。 */
export function contrastRatio(foreground: string, background: string): number {
  const l1 = relativeLuminance(parseHexColor(foreground));
  const l2 = relativeLuminance(parseHexColor(background));
  const lighter = Math.max(l1, l2);
  const darker = Math.min(l1, l2);
  return (lighter + 0.05) / (darker + 0.05);
}

/** 保留两位小数，用于稳定的报告输出与断言。 */
export function roundRatio(ratio: number): number {
  return Math.round(ratio * 100) / 100;
}

export function meetsThreshold(ratio: number, textSize: TextSize): boolean {
  return roundRatio(ratio) >= CONTRAST_THRESHOLD[textSize];
}
