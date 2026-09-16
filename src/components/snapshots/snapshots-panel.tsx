"use client";

import { useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { IconCamera, IconDeviceFloppy } from "@tabler/icons-react";
import { apiFetch } from "@/lib/api/client";
import type { Snapshot } from "@/lib/api/types";

type Props = {
  decisionId: string;
  initialSnapshots: Snapshot[];
};

export function SnapshotsPanel({ decisionId, initialSnapshots }: Props) {
  const t = useTranslations("snapshots");
  const tc = useTranslations("common");
  const locale = useLocale();
  const [snapshots, setSnapshots] = useState(initialSnapshots);
  const [label, setLabel] = useState("");
  const [prompt, setPrompt] = useState("");
  const [summary, setSummary] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const dateFmt = new Intl.DateTimeFormat(locale, {
    dateStyle: "medium",
    timeStyle: "short",
  });

  async function createSnapshot(input: {
    label: string;
    prompt?: string;
    outputSummary?: string;
  }) {
    setBusy(true);
    setError(null);
    try {
      const res = await apiFetch<{ snapshot: Snapshot }>(
        `/api/v1/decisions/${decisionId}/snapshots`,
        {
          method: "POST",
          body: JSON.stringify({
            label: input.label,
            prompt: input.prompt || null,
            outputSummary: input.outputSummary || null,
          }),
        },
      );
      setSnapshots((prev) => [res.snapshot, ...prev]);
      setLabel("");
      setPrompt("");
      setSummary("");
    } catch (err) {
      setError(err instanceof Error ? err.message : tc("error"));
    } finally {
      setBusy(false);
    }
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!label.trim()) return;
    await createSnapshot({
      label: label.trim(),
      prompt: prompt.trim(),
      outputSummary: summary.trim(),
    });
  }

  async function quickSave() {
    const stamp = dateFmt.format(new Date());
    await createSnapshot({
      label: `${t("quickSave")} — ${stamp}`,
      prompt: "",
      outputSummary: "",
    });
  }

  return (
    <section className="section" aria-labelledby="snapshots-heading">
      <div className="row" style={{ justifyContent: "space-between" }}>
        <div>
          <h2 id="snapshots-heading">{t("title")}</h2>
          <p className="section-lead">{t("subtitle")}</p>
        </div>
        <button
          type="button"
          className="btn btn-secondary"
          disabled={busy}
          onClick={() => void quickSave()}
        >
          <IconCamera size={18} />
          {t("quickSave")}
        </button>
      </div>

      <form className="form-panel" onSubmit={onSubmit}>
        <div className="field">
          <label htmlFor="snap-label">{t("label")}</label>
          <input
            id="snap-label"
            className="input"
            value={label}
            onChange={(e) => setLabel(e.target.value)}
            placeholder={t("labelPlaceholder")}
            required
          />
        </div>
        <div className="field">
          <label htmlFor="snap-prompt">
            {t("prompt")} <span className="hint">({tc("optional")})</span>
          </label>
          <textarea
            id="snap-prompt"
            className="textarea"
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder={t("promptPlaceholder")}
          />
        </div>
        <div className="field">
          <label htmlFor="snap-summary">
            {t("summary")} <span className="hint">({tc("optional")})</span>
          </label>
          <textarea
            id="snap-summary"
            className="textarea"
            value={summary}
            onChange={(e) => setSummary(e.target.value)}
            placeholder={t("summaryPlaceholder")}
          />
        </div>
        {error ? <p className="error-text">{error}</p> : null}
        <div>
          <button type="submit" className="btn btn-primary" disabled={busy}>
            <IconDeviceFloppy size={18} />
            {t("create")}
          </button>
        </div>
      </form>

      {snapshots.length === 0 ? (
        <p className="muted" style={{ marginTop: "1rem" }}>
          {t("empty")}
        </p>
      ) : (
        <ul className="list" style={{ marginTop: "1rem" }}>
          {snapshots.map((s) => (
            <li key={s.id} className="list-item">
              <div className="stack" style={{ gap: "0.25rem", flex: 1 }}>
                <strong>{s.label}</strong>
                <p className="faint" style={{ margin: 0 }}>
                  {t("created", {
                    date: dateFmt.format(new Date(s.createdAt)),
                  })}
                </p>
                {s.prompt ? (
                  <p className="muted" style={{ margin: 0 }}>
                    {s.prompt}
                  </p>
                ) : null}
                {s.outputSummary ? (
                  <p className="muted" style={{ margin: 0 }}>
                    {s.outputSummary}
                  </p>
                ) : null}
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
