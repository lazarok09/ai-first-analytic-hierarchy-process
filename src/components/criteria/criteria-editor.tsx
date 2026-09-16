"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import {
  IconPencil,
  IconPlus,
  IconTrash,
  IconCheck,
  IconX,
} from "@tabler/icons-react";
import { apiFetch } from "@/lib/api/client";
import type { Criterion, CriterionSource } from "@/lib/api/types";

type Props = {
  decisionId: string;
  initialCriteria: Criterion[];
  onCriteriaChange?: (criteria: Criterion[]) => void;
};

export function CriteriaEditor({
  decisionId,
  initialCriteria,
  onCriteriaChange,
}: Props) {
  const t = useTranslations("criteria");
  const tc = useTranslations("common");
  const [criteria, setCriteria] = useState(initialCriteria);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editName, setEditName] = useState("");
  const [editDescription, setEditDescription] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function sync(next: Criterion[]) {
    setCriteria(next);
    onCriteriaChange?.(next);
  }

  async function onAdd(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    setBusy(true);
    setError(null);
    try {
      const res = await apiFetch<{ criterion: Criterion }>(
        `/api/v1/decisions/${decisionId}/criteria`,
        {
          method: "POST",
          body: JSON.stringify({
            name: name.trim(),
            description: description.trim() || null,
            source: "user",
            sortOrder: criteria.length,
          }),
        },
      );
      sync([...criteria, res.criterion]);
      setName("");
      setDescription("");
    } catch (err) {
      setError(err instanceof Error ? err.message : tc("error"));
    } finally {
      setBusy(false);
    }
  }

  function startEdit(c: Criterion) {
    setEditingId(c.id);
    setEditName(c.name);
    setEditDescription(c.description ?? "");
  }

  async function saveEdit(id: string) {
    setBusy(true);
    setError(null);
    try {
      const res = await apiFetch<{ criterion: Criterion }>(
        `/api/v1/criteria/${id}`,
        {
          method: "PATCH",
          body: JSON.stringify({
            name: editName.trim(),
            description: editDescription.trim() || null,
          }),
        },
      );
      sync(criteria.map((c) => (c.id === id ? res.criterion : c)));
      setEditingId(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : tc("error"));
    } finally {
      setBusy(false);
    }
  }

  async function onDelete(id: string) {
    if (!window.confirm(t("deleteConfirm"))) return;
    setBusy(true);
    setError(null);
    try {
      await apiFetch<{ ok: boolean }>(`/api/v1/criteria/${id}`, {
        method: "DELETE",
      });
      sync(criteria.filter((c) => c.id !== id));
    } catch (err) {
      setError(err instanceof Error ? err.message : tc("error"));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="section" aria-labelledby="criteria-heading">
      <h2 id="criteria-heading">{t("title")}</h2>
      <p className="section-lead">{t("subtitle")}</p>

      <form className="form-panel" onSubmit={onAdd}>
        <div className="field">
          <label htmlFor="criterion-name">{t("name")}</label>
          <input
            id="criterion-name"
            className="input"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t("namePlaceholder")}
            required
          />
        </div>
        <div className="field">
          <label htmlFor="criterion-desc">
            {t("description")} <span className="hint">({tc("optional")})</span>
          </label>
          <textarea
            id="criterion-desc"
            className="textarea"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder={t("descriptionPlaceholder")}
          />
        </div>
        <div>
          <button type="submit" className="btn btn-primary" disabled={busy}>
            <IconPlus size={18} stroke={1.75} />
            {t("add")}
          </button>
        </div>
      </form>

      {error ? <p className="error-text">{error}</p> : null}

      {criteria.length === 0 ? (
        <p className="muted" style={{ marginTop: "1rem" }}>
          {t("empty")}
        </p>
      ) : (
        <ul className="list" style={{ marginTop: "1rem" }}>
          {criteria.map((c) => (
            <li key={c.id} className="list-item">
              {editingId === c.id ? (
                <div className="stack" style={{ flex: 1 }}>
                  <input
                    className="input"
                    value={editName}
                    onChange={(e) => setEditName(e.target.value)}
                    aria-label={t("name")}
                  />
                  <textarea
                    className="textarea"
                    value={editDescription}
                    onChange={(e) => setEditDescription(e.target.value)}
                    aria-label={t("description")}
                  />
                  <div className="row">
                    <button
                      type="button"
                      className="btn btn-primary"
                      disabled={busy}
                      onClick={() => void saveEdit(c.id)}
                    >
                      <IconCheck size={16} />
                      {tc("save")}
                    </button>
                    <button
                      type="button"
                      className="btn btn-ghost"
                      onClick={() => setEditingId(null)}
                    >
                      <IconX size={16} />
                      {tc("cancel")}
                    </button>
                  </div>
                </div>
              ) : (
                <>
                  <div className="stack" style={{ gap: "0.25rem", flex: 1 }}>
                    <div className="row" style={{ gap: "0.5rem" }}>
                      <strong>{c.name}</strong>
                      <SourceBadge source={c.source} />
                    </div>
                    {c.description ? (
                      <p className="muted" style={{ margin: 0 }}>
                        {c.description}
                      </p>
                    ) : null}
                  </div>
                  <div className="row">
                    <button
                      type="button"
                      className="btn btn-ghost"
                      onClick={() => startEdit(c)}
                      title={tc("edit")}
                    >
                      <IconPencil size={16} />
                    </button>
                    <button
                      type="button"
                      className="btn btn-danger"
                      disabled={busy}
                      onClick={() => void onDelete(c.id)}
                      title={tc("delete")}
                    >
                      <IconTrash size={16} />
                    </button>
                  </div>
                </>
              )}
            </li>
          ))}
        </ul>
      )}

      {criteria.length === 1 ? (
        <p className="callout callout-warn" style={{ marginTop: "1rem" }}>
          {t("needMore")}
        </p>
      ) : null}
    </section>
  );
}

function SourceBadge({ source }: { source: CriterionSource }) {
  const t = useTranslations("criteria.sources");
  const className =
    source === "agent"
      ? "badge badge-agent"
      : source === "import"
        ? "badge badge-import"
        : "badge badge-user";
  return <span className={className}>{t(source)}</span>;
}
