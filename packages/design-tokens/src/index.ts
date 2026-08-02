/**
 * @mgd/design-tokens 公共入口。
 * 运行时消费方（packages/ui、apps/*）从 generated/tokens.ts 读取令牌值，
 * 样式侧通过 generated/tokens.css 与 generated/theme.css 消费 CSS 变量。
 */

export {
  BREAKPOINT_TOKENS,
  COLOR_TOKENS,
  FOCUS_RING_TOKENS,
  FONT_TOKENS,
  LAYOUT_TOKENS,
  SEMANTIC_TOKEN_NAMES,
  SPACE_TOKENS,
  TARGET_SIZE_TOKENS,
} from '../generated/tokens.ts';

export {
  contrastRatio,
  CONTRAST_THRESHOLD,
  meetsThreshold,
  parseHexColor,
  relativeLuminance,
  roundRatio,
  type Rgb,
  type TextSize,
} from './contrast.ts';
