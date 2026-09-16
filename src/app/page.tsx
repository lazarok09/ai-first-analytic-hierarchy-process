import Link from "next/link";
import { getTranslations } from "next-intl/server";
import { auth } from "@/auth";
import { LocaleSwitcher } from "@/components/locale-switcher";
import { ReferencesFooter } from "@/components/references-footer";

export default async function HomePage() {
  const t = await getTranslations("home");
  const session = await auth();
  const signedIn = Boolean(session?.user?.id);

  return (
    <main
      style={{
        minHeight: "100%",
        display: "flex",
        flexDirection: "column",
        padding: "1.25rem 1.5rem 0",
      }}
    >
      <div
        className="row"
        style={{ justifyContent: "space-between", marginBottom: "auto" }}
      >
        <span className="app-brand">{t("title")}</span>
        <LocaleSwitcher />
      </div>

      <div
        style={{
          maxWidth: "36rem",
          margin: "4rem auto",
          width: "100%",
          paddingBottom: "3rem",
        }}
      >
        <h1
          style={{
            fontFamily: "var(--font-display), Georgia, serif",
            fontSize: "clamp(2.2rem, 5vw, 3.1rem)",
            letterSpacing: "-0.03em",
            lineHeight: 1.15,
            margin: "0 0 1rem",
          }}
        >
          {t("title")}
        </h1>
        <p className="muted" style={{ fontSize: "1.1rem", marginBottom: "1.75rem" }}>
          {t("subtitle")}
        </p>
        <div className="row">
          {signedIn ? (
            <Link href="/app" className="btn btn-primary">
              {t("ctaOpenApp")}
            </Link>
          ) : (
            <Link href="/login" className="btn btn-primary">
              {t("ctaSignIn")}
            </Link>
          )}
        </div>
      </div>

      <ReferencesFooter />
    </main>
  );
}
