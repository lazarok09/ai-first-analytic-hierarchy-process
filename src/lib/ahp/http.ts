import { NextResponse } from "next/server";
import { ZodError } from "zod";
import { AppError, isAppError } from "@/lib/ahp/errors";

export type ApiErrorBody = {
  error: {
    code: string;
    message: string;
    details?: unknown;
  };
};

export function jsonOk<T>(data: T, status = 200): NextResponse {
  return NextResponse.json(data, { status });
}

export function jsonError(
  code: string,
  message: string,
  status: number,
  details?: unknown,
): NextResponse<ApiErrorBody> {
  const body: ApiErrorBody = {
    error: details === undefined ? { code, message } : { code, message, details },
  };
  return NextResponse.json(body, { status });
}

export function handleRouteError(err: unknown): NextResponse<ApiErrorBody> {
  if (isAppError(err)) {
    return jsonError(err.code, err.message, err.status, err.details);
  }
  if (err instanceof ZodError) {
    return jsonError("VALIDATION_ERROR", "Invalid request body", 422, err.issues);
  }
  console.error(err);
  return jsonError("INTERNAL_ERROR", "Internal server error", 500);
}

export async function parseJsonBody(request: Request): Promise<unknown> {
  try {
    return await request.json();
  } catch {
    throw new AppError("VALIDATION_ERROR", "Request body must be JSON", 422);
  }
}
