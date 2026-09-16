"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useTranslations } from "next-intl";
import { IconLogout, IconHierarchy2 } from "@tabler/icons-react";
import { LocaleSwitcher } from "@/components/locale-switcher";
import { ChatKitPanel } from "@/components/chatkit/chatkit-panel";
import { ReferencesFooter } from "@/components/references-footer";

type AppShellProps = {
  children: React.ReactNode;
  userLabel?: string | null;
  signOutAction: () => Promise<void>;
};

export function AppShell({
  children,
  userLabel,
  signOutAction,
}: AppShellProps) {
  const t = useTranslations();
  const pathname = usePathname();

  const links = [
    {
      href: "/app",
      label: t("nav.decisions"),
      match: (p: string) => p === "/app" || p.startsWith("/app/decisions"),
    },
    {
      href: "/app/connect",
      label: t("nav.connect"),
      match: (p: string) => p.startsWith("/app/connect"),
    },
  ];

  return (
    <div className="app-shell">
      <header className="app-header">
        <div className="row" style={{ gap: "1rem" }}>
          <Link href="/app" className="app-brand row" style={{ gap: "0.4rem" }}>
            <IconHierarchy2 size={20} stroke={1.75} />
            {t("common.appName")}
          </Link>
          <nav className="app-nav" aria-label="App">
            {links.map((link) => (
              <Link
                key={link.href}
                href={link.href}
                data-active={link.match(pathname) ? "true" : "false"}
              >
                {link.label}
              </Link>
            ))}
          </nav>
        </div>
        <div className="row" style={{ gap: "0.85rem" }}>
          <LocaleSwitcher />
          {userLabel ? <span className="faint">{userLabel}</span> : null}
          <form action={signOutAction}>
            <button
              type="submit"
              className="btn btn-ghost"
              title={t("common.signOut")}
            >
              <IconLogout size={18} stroke={1.75} />
              {t("common.signOut")}
            </button>
          </form>
        </div>
      </header>
      <main className="app-main">{children}</main>
      <ReferencesFooter />
      <ChatKitPanel />
    </div>
  );
}
