import { redirect } from "next/navigation";
import { getTranslations } from "next-intl/server";
import { auth, authProvidersAvailable, signIn } from "@/auth";
import { LocaleSwitcher } from "@/components/locale-switcher";
import { ReferencesFooter } from "@/components/references-footer";

type SearchParams = Promise<{ callbackUrl?: string }>;

export default async function LoginPage({
  searchParams,
}: {
  searchParams: SearchParams;
}) {
  const t = await getTranslations("login");
  const session = await auth();
  const params = await searchParams;
  const callbackUrl =
    typeof params.callbackUrl === "string" && params.callbackUrl.startsWith("/")
      ? params.callbackUrl
      : "/app";

  if (session?.user?.id) {
    redirect(callbackUrl);
  }

  const providers = authProvidersAvailable();
  const hasAny = providers.github || providers.google || providers.devLogin;

  return (
    <div style={{ minHeight: "100%", display: "flex", flexDirection: "column" }}>
    <main
      className="stack"
      style={{
        maxWidth: "28rem",
        margin: "0 auto",
        flex: 1,
        justifyContent: "center",
        padding: "3rem 1.25rem",
        gap: "1.5rem",
        width: "100%",
      }}
    >
      <div className="row" style={{ justifyContent: "flex-end" }}>
        <LocaleSwitcher />
      </div>

      <div className="stack" style={{ gap: "0.5rem" }}>
        <p className="faint" style={{ margin: 0, letterSpacing: "0.06em", textTransform: "uppercase" }}>
          {t("eyebrow")}
        </p>
        <h1
          style={{
            margin: 0,
            fontFamily: "var(--font-display), Georgia, serif",
            fontSize: "1.85rem",
          }}
        >
          {t("title")}
        </h1>
        <p className="muted" style={{ margin: 0 }}>
          {t("subtitle")}
        </p>
      </div>

      {!hasAny ? (
        <p className="callout callout-warn">{t("noProviders")}</p>
      ) : null}

      <div className="stack" style={{ gap: "0.65rem" }}>
        {providers.github ? (
          <form
            action={async () => {
              "use server";
              await signIn("github", { redirectTo: callbackUrl });
            }}
          >
            <button type="submit" className="btn btn-primary" style={{ width: "100%" }}>
              {t("github")}
            </button>
          </form>
        ) : null}

        {providers.google ? (
          <form
            action={async () => {
              "use server";
              await signIn("google", { redirectTo: callbackUrl });
            }}
          >
            <button type="submit" className="btn btn-secondary" style={{ width: "100%" }}>
              {t("google")}
            </button>
          </form>
        ) : null}

        {providers.devLogin ? (
          <form
            className="stack"
            style={{
              marginTop: "0.5rem",
              paddingTop: "1rem",
              borderTop: "1px solid var(--line)",
              gap: "0.75rem",
            }}
            action={async (formData) => {
              "use server";
              const email = String(formData.get("email") ?? "").trim();
              await signIn("dev-login", {
                email: email || "demo@example.com",
                redirectTo: callbackUrl,
              });
            }}
          >
            <div className="field">
              <label htmlFor="dev-email">{t("devEmail")}</label>
              <input
                id="dev-email"
                name="email"
                type="email"
                defaultValue="demo@example.com"
                placeholder="demo@example.com"
                className="input"
              />
            </div>
            <button type="submit" className="btn btn-primary" style={{ width: "100%" }}>
              {t("devLogin")}
            </button>
          </form>
        ) : null}
      </div>
    </main>
    <ReferencesFooter />
    </div>
  );
}
