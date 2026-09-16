import { NextResponse } from "next/server";
import { isLocale, localeCookieName } from "@/i18n/config";

export async function POST(request: Request) {
  let locale: string | undefined;
  try {
    const body = (await request.json()) as { locale?: string };
    locale = body.locale;
  } catch {
    return NextResponse.json(
      { error: { code: "VALIDATION_ERROR", message: "Invalid JSON" } },
      { status: 422 },
    );
  }

  if (!isLocale(locale)) {
    return NextResponse.json(
      { error: { code: "VALIDATION_ERROR", message: "Invalid locale" } },
      { status: 422 },
    );
  }

  const response = NextResponse.json({ locale });
  response.cookies.set(localeCookieName, locale, {
    path: "/",
    maxAge: 60 * 60 * 24 * 365,
    sameSite: "lax",
  });
  return response;
}
