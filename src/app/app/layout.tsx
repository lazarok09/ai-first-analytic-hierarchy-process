import { redirect } from "next/navigation";
import { auth, signOut } from "@/auth";
import { AppShell } from "@/components/app-shell";

/**
 * Protects /app/* UI routes. Prefer layout auth over Edge proxy so the
 * Drizzle/SQLite auth module is not pulled into the proxy runtime.
 */
export default async function AppLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const session = await auth();
  if (!session?.user?.id) {
    redirect("/login?callbackUrl=/app");
  }

  async function signOutAction() {
    "use server";
    await signOut({ redirectTo: "/login" });
  }

  return (
    <AppShell
      userLabel={session.user?.email ?? session.user?.name ?? null}
      signOutAction={signOutAction}
    >
      {children}
    </AppShell>
  );
}
