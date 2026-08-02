/**
 * 控件清册工具（需求 G9 第 1 条、B4-2 第 4 条）。
 *
 * UI_Kit 的每个交互组件都输出 data-mgd-control 属性，本模块把页面渲染结果收敛成
 * 可断言、可快照的控件清单，用于「0 个改分控件」红线校验。
 */

export interface ControlEntry {
  readonly controlId: string;
  readonly tagName: string;
  readonly accessibleName: string;
  readonly disabled: boolean;
}

/**
 * 禁用词表：任何控件的 controlId 或可访问名命中即视为红线违规。
 * 覆盖编辑个人分数、编辑解锁状态、编辑证据正文三类能力（FR-039）。
 */
export const FORBIDDEN_CONTROL_PATTERNS: readonly RegExp[] = [
  /score/i,
  /scoring/i,
  /unlock/i,
  /override/i,
  /\bgate\b/i,
  /evidence[-_]?(text|body|content)/i,
  /分数/,
  /评分结果/,
  /改分/,
  /解锁/,
  /证据正文/,
];

function accessibleNameOf(element: Element): string {
  const ariaLabel = element.getAttribute('aria-label');
  if (ariaLabel !== null && ariaLabel.trim().length > 0) return ariaLabel.trim();
  return (element.textContent ?? '').replace(/\s+/g, ' ').trim();
}

/** 收集容器内全部带 data-mgd-control 的控件，按 controlId 排序以获得稳定快照。 */
export function collectControls(container: ParentNode): readonly ControlEntry[] {
  const entries: ControlEntry[] = [];

  for (const element of container.querySelectorAll('[data-mgd-control]')) {
    const controlId = element.getAttribute('data-mgd-control');
    if (controlId === null) continue;

    entries.push({
      controlId,
      tagName: element.tagName.toLowerCase(),
      accessibleName: accessibleNameOf(element),
      disabled:
        element.getAttribute('aria-disabled') === 'true' || element.hasAttribute('disabled'),
    });
  }

  return entries.sort((a, b) => a.controlId.localeCompare(b.controlId));
}

export interface ForbiddenControlHit {
  readonly controlId: string;
  readonly accessibleName: string;
  readonly pattern: string;
}

/** 找出命中禁用词表的控件；返回空数组即通过红线校验。 */
export function findForbiddenControls(
  controls: readonly ControlEntry[],
): readonly ForbiddenControlHit[] {
  const hits: ForbiddenControlHit[] = [];

  for (const control of controls) {
    for (const pattern of FORBIDDEN_CONTROL_PATTERNS) {
      if (pattern.test(control.controlId) || pattern.test(control.accessibleName)) {
        hits.push({
          controlId: control.controlId,
          accessibleName: control.accessibleName,
          pattern: pattern.source,
        });
        break;
      }
    }
  }

  return hits;
}
