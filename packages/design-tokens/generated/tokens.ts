// 本文件由 packages/design-tokens/src/build-css.ts 生成，禁止手工编辑。
// 令牌事实源：packages/design-tokens/tokens/*.json

export const COLOR_TOKENS = {
  "primary": "#2B5CE6",
  "success": "#1F9D66",
  "successText": "#16794E",
  "warning": "#B7791F",
  "warningText": "#8A5A12",
  "danger": "#C53030",
  "info": "#2B6CB0",
  "neutral900": "#1A202C",
  "neutral600": "#4A5568",
  "neutral100": "#EDF2F7",
  "surface": "#FFFFFF",
  "focus": "#805AD5"
} as const;

export const FONT_TOKENS = {
  "display": {
    "size": "32px",
    "lineHeight": "40px",
    "usage": "落地页主标题"
  },
  "h1": {
    "size": "28px",
    "lineHeight": "36px",
    "usage": "页面标题"
  },
  "h2": {
    "size": "22px",
    "lineHeight": "30px",
    "usage": "区块标题"
  },
  "h3": {
    "size": "18px",
    "lineHeight": "26px",
    "usage": "卡片标题"
  },
  "body": {
    "size": "16px",
    "lineHeight": "26px",
    "usage": "正文，行高 1.625 满足中文可读性"
  },
  "caption": {
    "size": "13px",
    "lineHeight": "20px",
    "usage": "辅助说明、状态标签"
  }
} as const;

export const SPACE_TOKENS = {
  "1": "4px",
  "2": "8px",
  "3": "12px",
  "4": "16px",
  "6": "24px",
  "8": "32px",
  "12": "48px"
} as const;

export const TARGET_SIZE_TOKENS = {
  "min": "24px",
  "primary": "44px"
} as const;

export const FOCUS_RING_TOKENS = {
  "width": "2px",
  "offset": "2px"
} as const;

export const BREAKPOINT_TOKENS = {
  "tablet": "768px",
  "desktop": "1024px"
} as const;

export const LAYOUT_TOKENS = {
  "contentMaxWidth": "1200px"
} as const;

/** 语义令牌名称集合：稳定契约，与 tokens/NAMES.lock 一致。 */
export const SEMANTIC_TOKEN_NAMES = [
  "breakpoint.desktop",
  "breakpoint.tablet",
  "color.danger",
  "color.focus",
  "color.info",
  "color.neutral100",
  "color.neutral600",
  "color.neutral900",
  "color.primary",
  "color.success",
  "color.successText",
  "color.surface",
  "color.warning",
  "color.warningText",
  "focusRing.offset",
  "focusRing.width",
  "font.body.lineHeight",
  "font.body.size",
  "font.caption.lineHeight",
  "font.caption.size",
  "font.display.lineHeight",
  "font.display.size",
  "font.h1.lineHeight",
  "font.h1.size",
  "font.h2.lineHeight",
  "font.h2.size",
  "font.h3.lineHeight",
  "font.h3.size",
  "fontFamily.en",
  "fontFamily.mono",
  "fontFamily.zh",
  "layout.contentMaxWidth",
  "space.1",
  "space.12",
  "space.2",
  "space.3",
  "space.4",
  "space.6",
  "space.8",
  "targetSize.min",
  "targetSize.primary"
] as const;
