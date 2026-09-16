"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { IconPlus } from "@tabler/icons-react";
import { apiFetch } from "@/lib/api/client";
import type { Decision } from "@/lib/api/types";

type Props = {
  initialDecisions: Decision[];
};

export function DecisionsPanel({ initialDecisions }: Props) {
  const t = useTranslations("decisions");
  const tc = useTranslations("common");
  const locale = useLocale();
  const router = useRouter();
  const [decisions, setDecisions] = useState(initialDecisions);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!title.trim()) return;
    setBusy(true);
    setError(null);
    try {
      const res = await apiFetch<{ decision: Decision }>("/api/v1/decisions", {
        method: "POST",
        body: JSON.stringify({
          title: title.trim(),
          description: description.trim() || null,
        }),
      });
      setDecisions((prev) => [res.decision, ...prev]);
      setTitle("");
      setDescription("");
      router.push(`/app/decisions/${res.decision.id}`);
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : tc("error"));
    } finally {
      setBusy(false);
    }
  }

  const dateFmt = new Intl.DateTimeFormat(locale, {
    dateStyle: "medium",
    timeStyle: "short",
  });

  return (
    <div className="stack" style={{ gap: "1.75rem" }}>
      <form className="form-panel" onSubmit={onCreate}>
        <h2 style={{ margin: 0, fontFamily: "var(--font-display)", fontSize: "1.15rem" }}>
          {t("createTitle")}
        </h2>
        <div className="field">
          <label htmlFor="decision-title">{t("titleLabel")}</label>
          <input
            id="decision-title"
            className="input"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder={t("titlePlaceholder")}
            required
          />
        </div>
        <div className="field">
          <label htmlFor="decision-desc">
            {t("descriptionLabel")}{" "}
            <span className="hint">({tc("optional")})</span>
          </label>
          <textarea
            id="decision-desc"
            className="textarea"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder={t("descriptionPlaceholder")}
          />
        </div>
        {error ? <p className="error-text">{error}</p> : null}
        <div>
          <button type="submit" className="btn btn-primary" disabled={busy}>
            <IconPlus size={18} stroke={1.75} />
            {t("create")}
          </button>
        </div>
      </form>

      {decisions.length === 0 ? (
        <p className="muted">{t("empty")}</p>
      ) : (
        <ul className="list">
          {decisions.map((d) => (
            <li key={d.id} className="list-item">
              <div className="stack" style={{ gap: "0.25rem", flex: 1 }}>
                <Link
                  href={`/app/decisions/${d.id}`}
                  style={{ fontWeight: 600, fontSize: "1.05rem" }}
                >
                  {d.title}
                </Link>
                {d.description ? (
                  <p className="muted" style={{ margin: 0 }}>
                    {d.description}
                  </p>
                ) : null}
                <p className="faint" style={{ margin: 0 }}>
                  <span className="badge badge-user" style={{ marginRight: "0.5rem" }}>
                    {t(`status.${d.status}`)}
                  </span>
                  {t("updated", {
                    date: dateFmt.format(new Date(d.updatedAt)),
                  })}
                </p>
              </div>
              <Link href={`/app/decisions/${d.id}`} className="btn btn-secondary">
                {t("open")}
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
