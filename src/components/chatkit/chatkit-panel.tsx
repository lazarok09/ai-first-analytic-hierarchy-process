"use client";

import Script from "next/script";
import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useLocale, useTranslations } from "next-intl";
import { ChatKit, useChatKit } from "@openai/chatkit-react";
import { IconMessageChatbot, IconX } from "@tabler/icons-react";
import { apiFetch } from "@/lib/api/client";

type BootstrapResponse = {
  userId: string;
  agentToken: string;
  expiresAt: number;
  chatkitUrl: string;
  domainKey: string;
};

function ChatKitSession({ bootstrap }: { bootstrap: BootstrapResponse }) {
  const t = useTranslations("chatkit");
  const locale = useLocale();
  const router = useRouter();

  const ensureLineage = useCallback(
    async (threadId: string | null) => {
      if (!threadId) return;
      try {
        await apiFetch("/api/v1/chats/ensure", {
          method: "POST",
          body: JSON.stringify({
            externalSessionId: threadId,
            title: t("defaultThreadTitle"),
          }),
        });
      } catch {
        // Lineage is best-effort; agent tools also call ensure.
      }
    },
    [t],
  );

  const { control } = useChatKit({
    api: {
      url: bootstrap.chatkitUrl,
      domainKey: bootstrap.domainKey,
      fetch: ((input, init) => {
        const headers = new Headers(init?.headers);
        headers.set("Authorization", `Bearer ${bootstrap.agentToken}`);
        return fetch(input, { ...init, headers, credentials: "include" });
      }) as typeof fetch,
    },
    theme: "light",
    locale: locale === "pt-BR" ? "pt-BR" : "en",
    startScreen: {
      greeting: t("greeting"),
      prompts: [
        {
          label: t("promptCompare"),
          prompt: t("promptCompareText"),
        },
        {
          label: t("promptCriteria"),
          prompt: t("promptCriteriaText"),
        },
        {
          label: t("promptJudgments"),
          prompt: t("promptJudgmentsText"),
        },
      ],
    },
    composer: {
      placeholder: t("placeholder"),
    },
    onClientTool: async ({ name, params }) => {
      if (name === "refresh_decision_ui") {
        router.refresh();
        const decisionId =
          typeof params.decisionId === "string" ? params.decisionId : null;
        if (decisionId) {
          router.push(`/app/decisions/${decisionId}`);
        }
        return { ok: true };
      }
      return { ok: false, error: `Unknown client tool: ${name}` };
    },
    onThreadChange: ({ threadId }) => {
      void ensureLineage(threadId);
    },
    onResponseEnd: () => {
      router.refresh();
    },
    onError: ({ error }) => {
      console.error("[chatkit]", error);
    },
  });

  return <ChatKit control={control} className="chatkit-frame" />;
}

export function ChatKitPanel() {
  const t = useTranslations("chatkit");
  const [scriptReady, setScriptReady] = useState(false);
  const [bootstrap, setBootstrap] = useState<BootstrapResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [open, setOpen] = useState(false);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const res = await apiFetch<BootstrapResponse>("/api/chatkit/bootstrap", {
          method: "POST",
        });
        if (!cancelled) {
          setBootstrap(res);
          setError(null);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : t("bootstrapError"));
        }
      }
    }
    void load();
    return () => {
      cancelled = true;
    };
  }, [t]);

  return (
    <>
      <Script
        src="https://cdn.platform.openai.com/deployments/chatkit/chatkit.js"
        strategy="afterInteractive"
        onLoad={() => setScriptReady(true)}
      />

      <button
        type="button"
        className="chatkit-fab"
        aria-expanded={open}
        aria-controls="ahp-chatkit-panel"
        onClick={() => setOpen((v) => !v)}
        title={t("toggle")}
      >
        {open ? (
          <IconX size={22} stroke={1.75} />
        ) : (
          <IconMessageChatbot size={22} stroke={1.75} />
        )}
        <span className="sr-only">{t("toggle")}</span>
      </button>

      {open ? (
        <div
          id="ahp-chatkit-panel"
          className="chatkit-panel"
          role="dialog"
          aria-label={t("title")}
        >
          <header className="chatkit-panel-header">
            <div>
              <strong>{t("title")}</strong>
              <p className="faint">{t("subtitle")}</p>
            </div>
            <button
              type="button"
              className="btn btn-ghost"
              onClick={() => setOpen(false)}
            >
              <IconX size={18} stroke={1.75} />
            </button>
          </header>

          {error ? (
            <p className="callout callout-danger" style={{ margin: "0.75rem" }}>
              {error}
            </p>
          ) : null}

          {!bootstrap || !scriptReady ? (
            <p className="faint" style={{ padding: "1rem" }}>
              {t("loading")}
            </p>
          ) : (
            <ChatKitSession bootstrap={bootstrap} />
          )}
        </div>
      ) : null}
    </>
  );
}
