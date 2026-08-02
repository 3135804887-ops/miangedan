/**
 * 面试便利设置枚举。唯一事实源：ai/schemas/turn-evidence.schema.json 的
 * properties.accommodations_in_effect.items.enum，与 docs/design/ACCESSIBILITY.md 第 5.1 节一致。
 *
 * 红线（ACCESSIBILITY 第 6 节）：便利设置只作记录，不进入评分证据；文字模式口语项标记
 * 「未评估」而不记 0 分。前端只负责会前可选、开始后冻结与跨轮继承的展示。
 */

export const ACCOMMODATION_KEYS = [
  'text_only',
  'mixed_input',
  'slower_avatar_speech',
  'repeat_questions',
  'extended_time',
  'silence_threshold_adjusted',
  'no_proactive_interruption',
  'reduced_motion',
  'tool_keyboard_alternative',
] as const;

export type AccommodationKey = (typeof ACCOMMODATION_KEYS)[number];

const ACCOMMODATION_SET: ReadonlySet<string> = new Set(ACCOMMODATION_KEYS);

export function isAccommodationKey(value: string): value is AccommodationKey {
  return ACCOMMODATION_SET.has(value);
}

/** 默认值：全部关闭，逐项独立开启（不使用捆绑或诱导，ACCESSIBILITY 第 5.2 节第 1 条）。 */
export function defaultAccommodations(): Readonly<Record<AccommodationKey, boolean>> {
  return Object.freeze(
    Object.fromEntries(ACCOMMODATION_KEYS.map((key) => [key, false])) as Record<
      AccommodationKey,
      boolean
    >,
  );
}
