import type { ApiErrorBody } from "./types";

export class ApiClientError extends Error {
  status: number;
  code: string;
  details?: unknown;

  constructor(status: number, body: ApiErrorBody | null, fallback: string) {
    super(body?.error?.message ?? fallback);
    this.name = "ApiClientError";
    this.status = status;
    this.code = body?.error?.code ?? "UNKNOWN";
    this.details = body?.error?.details;
  }
}

export async function apiFetch<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const headers = new Headers(init?.headers);
  if (init?.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const res = await fetch(path, {
    ...init,
    credentials: "include",
    headers,
  });

  if (!res.ok) {
    let body: ApiErrorBody | null = null;
    try {
      body = (await res.json()) as ApiErrorBody;
    } catch {
      body = null;
    }
    throw new ApiClientError(res.status, body, res.statusText);
  }

  if (res.status === 204) {
    return undefined as T;
  }

  return (await res.json()) as T;
}
