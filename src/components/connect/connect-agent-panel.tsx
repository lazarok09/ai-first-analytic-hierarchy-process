"use client";

import { useMemo, useState } from "react";
import { useTranslations } from "next-intl";
import {
  IconCopy,
  IconKey,
  IconTrash,
  IconCheck,
} from "@tabler/icons-react";
import { apiFetch } from "@/lib/api/client";
import type { ApiKeyPublic, CreatedApiKey } from "@/lib/api/types";

type Props = {
  initialKeys: ApiKeyPublic[];
  mcpEndpoint: string;
};

const EXAMPLE_CONFIG = `{
  "mcpServers": {
    "ahp": {
      "url": "http://localhost:3000/api/mcp",
      "headers": {
        "Authorization": "Bearer YOUR_KEY_HERE"
      }
    }
  }
}`;

export function ConnectAgentPanel({ initialKeys, mcpEndpoint }: Props) {
  const t = useTranslations("connect");
  const tc = useTranslations("common");
  const [keys, setKeys] = useState(initialKeys.filter((k) => !k.revokedAt));
  const [name, setName] = useState("");
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const example = useMemo(
    () => EXAMPLE_CONFIG.replace("http://localhost:3000/api/mcp", mcpEndpoint),
    [mcpEndpoint],
  );

  async function onCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    setBusy(true);
    setError(null);
    setCopied(false);
    try {
      const res = await apiFetch<{ apiKey: CreatedApiKey }>("/api/v1/api-keys", {
        method: "POST",
        body: JSON.stringify({ name: name.trim() }),
      });
      const { key, ...pub } = res.apiKey;
      setKeys((prev) => [pub, ...prev]);
      setCreatedKey(key);
      setName("");
    } catch (err) {
      setError(err instanceof Error ? err.message : tc("error"));
    } finally {
      setBusy(false);
    }
  }

  async function onRevoke(id: string) {
    setBusy(true);
    setError(null);
    try {
      await apiFetch<{ apiKey: ApiKeyPublic }>(`/api/v1/api-keys/${id}`, {
        method: "DELETE",
      });
      setKeys((prev) => prev.filter((k) => k.id !== id));
    } catch (err) {
      setError(err instanceof Error ? err.message : tc("error"));
    } finally {
      setBusy(false);
    }
  }

  async function copyText(text: string) {
    await navigator.clipboard.writeText(text);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 2000);
  }

  return (
    <div className="stack" style={{ gap: "2rem" }}>
      <form className="form-panel" onSubmit={onCreate}>
        <h2 style={{ margin: 0, fontFamily: "var(--font-display)", fontSize: "1.15rem" }}>
          {t("keysTitle")}
        </h2>
        <div className="field">
          <label htmlFor="key-name">{t("name")}</label>
          <input
            id="key-name"
            className="input"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t("namePlaceholder")}
            required
          />
        </div>
        {error ? <p className="error-text">{error}</p> : null}
        <div>
          <button type="submit" className="btn btn-primary" disabled={busy}>
            <IconKey size={18} />
            {t("create")}
          </button>
        </div>
      </form>

      {createdKey ? (
        <div className="callout callout-warn">
          <p style={{ marginTop: 0, fontWeight: 600 }}>{t("createdOnce")}</p>
          <div className="secret-box">
            <code>{createdKey}</code>
            <button
              type="button"
              className="btn btn-secondary"
              onClick={() => void copyText(createdKey)}
            >
              {copied ? <IconCheck size={16} /> : <IconCopy size={16} />}
              {copied ? tc("copied") : tc("copy")}
            </button>
          </div>
        </div>
      ) : null}

      {keys.length === 0 ? (
        <p className="muted">{t("empty")}</p>
      ) : (
        <ul className="list">
          {keys.map((k) => (
            <li key={k.id} className="list-item">
              <div className="stack" style={{ gap: "0.2rem", flex: 1 }}>
                <strong>{k.name}</strong>
                <span className="faint mono">
                  {t("prefix")}: ahp_{k.prefix}_…
                </span>
              </div>
              <button
                type="button"
                className="btn btn-danger"
                disabled={busy}
                onClick={() => void onRevoke(k.id)}
              >
                <IconTrash size={16} />
                {t("revoke")}
              </button>
            </li>
          ))}
        </ul>
      )}

      <section className="section" style={{ marginTop: 0 }}>
        <h2>{t("instructionsTitle")}</h2>
        <p className="section-lead">{t("instructionsBody")}</p>
        <p className="muted">
          {t("endpoint")}: <code className="mono">{mcpEndpoint}</code>
        </p>
        <p className="faint">
          {t("examplePath")}
        </p>
        <pre
          className="mono"
          style={{
            margin: 0,
            padding: "1rem",
            background: "#1a2332",
            color: "#e8edf5",
            borderRadius: "var(--radius)",
            overflow: "auto",
            fontSize: "0.8rem",
          }}
        >
          {example}
        </pre>
        <div style={{ marginTop: "0.75rem" }}>
          <button
            type="button"
            className="btn btn-secondary"
            onClick={() => void copyText(example)}
          >
            <IconCopy size={16} />
            {t("exampleConfig")}
          </button>
        </div>
      </section>
    </div>
  );
}
