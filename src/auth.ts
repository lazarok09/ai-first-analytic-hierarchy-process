import { DrizzleAdapter } from "@auth/drizzle-adapter";
import { eq } from "drizzle-orm";
import NextAuth from "next-auth";
import type { Provider } from "next-auth/providers";
import Credentials from "next-auth/providers/credentials";
import GitHub from "next-auth/providers/github";
import Google from "next-auth/providers/google";
import { db } from "@/db";
import { accounts, sessions, users, verificationTokens } from "@/db/schema";

const DEFAULT_DEV_EMAIL = "demo@example.com";
const DEFAULT_DEV_NAME = "Demo User";

function buildProviders(): Provider[] {
  const providers: Provider[] = [];

  if (process.env.AUTH_GITHUB_ID && process.env.AUTH_GITHUB_SECRET) {
    providers.push(
      GitHub({
        clientId: process.env.AUTH_GITHUB_ID,
        clientSecret: process.env.AUTH_GITHUB_SECRET,
        allowDangerousEmailAccountLinking: true,
      }),
    );
  }

  if (process.env.AUTH_GOOGLE_ID && process.env.AUTH_GOOGLE_SECRET) {
    providers.push(
      Google({
        clientId: process.env.AUTH_GOOGLE_ID,
        clientSecret: process.env.AUTH_GOOGLE_SECRET,
        allowDangerousEmailAccountLinking: true,
      }),
    );
  }

  if (process.env.AUTH_DEV_LOGIN === "true") {
    providers.push(
      Credentials({
        id: "dev-login",
        name: "Dev Login",
        credentials: {
          email: {
            label: "Email",
            type: "email",
            placeholder: DEFAULT_DEV_EMAIL,
          },
        },
        async authorize(credentials) {
          const raw =
            typeof credentials?.email === "string"
              ? credentials.email.trim().toLowerCase()
              : "";
          const email = raw || DEFAULT_DEV_EMAIL;

          const existing = await db
            .select()
            .from(users)
            .where(eq(users.email, email))
            .limit(1);

          if (existing[0]) {
            return {
              id: existing[0].id,
              email: existing[0].email,
              name: existing[0].name,
              image: existing[0].image,
            };
          }

          const now = new Date();
          const [created] = await db
            .insert(users)
            .values({
              email,
              name: email === DEFAULT_DEV_EMAIL ? DEFAULT_DEV_NAME : email,
              locale: "en",
              createdAt: now,
              updatedAt: now,
            })
            .returning();

          return {
            id: created.id,
            email: created.email,
            name: created.name,
            image: created.image,
          };
        },
      }),
    );
  }

  return providers;
}

/**
 * Auth.js (NextAuth v5) — human web SSO.
 *
 * - OAuth: GitHub + Google when secrets are present
 * - Dev: Credentials provider when AUTH_DEV_LOGIN=true (email-only / demo user)
 * - Sessions: JWT (required for Credentials; adapter still persists users/accounts)
 * - Agents: Bearer API keys via requireUser (see docs/sso.md) — not Auth.js
 */
export const { handlers, auth, signIn, signOut } = NextAuth({
  adapter: DrizzleAdapter(db, {
    usersTable: users,
    accountsTable: accounts,
    sessionsTable: sessions,
    verificationTokensTable: verificationTokens,
  }),
  providers: buildProviders(),
  session: { strategy: "jwt" },
  pages: {
    signIn: "/login",
  },
  callbacks: {
    async jwt({ token, user }) {
      if (user?.id) {
        token.sub = user.id;
      }
      return token;
    },
    async session({ session, token }) {
      if (session.user && token.sub) {
        session.user.id = token.sub;
      }
      return session;
    },
  },
  trustHost: true,
});

export function authProvidersAvailable(): {
  github: boolean;
  google: boolean;
  devLogin: boolean;
} {
  return {
    github: Boolean(process.env.AUTH_GITHUB_ID && process.env.AUTH_GITHUB_SECRET),
    google: Boolean(process.env.AUTH_GOOGLE_ID && process.env.AUTH_GOOGLE_SECRET),
    devLogin: process.env.AUTH_DEV_LOGIN === "true",
  };
}
