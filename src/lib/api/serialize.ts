/** Make Drizzle Date fields safe to pass into Client Components. */
export function toClientJson<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T;
}
