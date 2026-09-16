"use client";

import { useLocale, useTranslations } from "next-intl";
import { useRouter } from "next/navigation";
import { useTransition } from "react";
import { locales, type Locale } from "@/i18n/config";

export function LocaleSwitcher() {
  const t = useTranslations("locale");
  const locale = useLocale();
  const router = useRouter();
  const [pending, startTransition] = useTransition();

  async function onChange(next: Locale) {
    if (next === locale) return;
    await fetch("/api/locale", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ locale: next }),
    });
    startTransition(() => {
      router.refresh();
    });
  }

  return (
    <label className="row" style={{ gap: "0.4rem" }}>
      <span className="faint">{t("label")}</span>
      <select
        className="select"
        style={{ width: "auto", padding: "0.35rem 0.5rem" }}
        value={locale}
        disabled={pending}
        aria-label={t("label")}
        onChange={(e) => void onChange(e.target.value as Locale)}
      >
        {locales.map((code) => (
          <option key={code} value={code}>
            {t(code)}
          </option>
        ))}
      </select>
    </label>
  );
}
