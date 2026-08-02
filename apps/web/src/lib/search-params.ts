export type PageSearchParams = Promise<Record<string, string | ReadonlyArray<string> | undefined>>;

export function readSearchParam(
  values: Record<string, string | ReadonlyArray<string> | undefined>,
  key: string,
): string | undefined {
  const value = values[key];
  return typeof value === 'string' ? value : value?.[0];
}
