export const PREVIEW_STATES = [
  'ready',
  'empty',
  'loading',
  'error',
  'forbidden',
  'recovering',
] as const;

export type PreviewState = (typeof PREVIEW_STATES)[number];

export function normalizePreviewState(value: string | undefined): PreviewState {
  return PREVIEW_STATES.includes(value as PreviewState) ? (value as PreviewState) : 'ready';
}
