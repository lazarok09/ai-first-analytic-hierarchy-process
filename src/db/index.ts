/**
 * Drizzle client for Next.js Route Handlers (Node runtime).
 *
 * IMPORTANT: Do not switch this file to `bun:sqlite`.
 * `next dev` / `next start` load Route Handlers via Node; `bun:sqlite` is not
 * available there. Seed scripts may use bun separately; the app API must use
 * better-sqlite3.
 */
import Database from "better-sqlite3";
import { drizzle } from "drizzle-orm/better-sqlite3";
import { mkdirSync } from "node:fs";
import { dirname, resolve } from "node:path";
import * as schema from "./schema";

const DEFAULT_DB_PATH = resolve(process.cwd(), "data", "ahp.db");

export function resolveDbPath(): string {
  return process.env.AHP_DB_PATH
    ? resolve(process.env.AHP_DB_PATH)
    : DEFAULT_DB_PATH;
}

/** Bootstrap tables until drizzle-kit migrations are the sole source of truth. */
function ensureSchema(sqlite: Database.Database): void {
  sqlite.exec(`
    CREATE TABLE IF NOT EXISTS users (
      id TEXT PRIMARY KEY NOT NULL,
      name TEXT,
      email TEXT UNIQUE,
      emailVerified INTEGER,
      image TEXT,
      locale TEXT NOT NULL DEFAULT 'en',
      created_at INTEGER NOT NULL,
      updated_at INTEGER NOT NULL
    );

    CREATE TABLE IF NOT EXISTS accounts (
      userId TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
      type TEXT NOT NULL,
      provider TEXT NOT NULL,
      providerAccountId TEXT NOT NULL,
      refresh_token TEXT,
      access_token TEXT,
      expires_at INTEGER,
      token_type TEXT,
      scope TEXT,
      id_token TEXT,
      session_state TEXT,
      PRIMARY KEY (provider, providerAccountId)
    );

    CREATE TABLE IF NOT EXISTS sessions (
      sessionToken TEXT PRIMARY KEY NOT NULL,
      userId TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
      expires INTEGER NOT NULL
    );

    CREATE TABLE IF NOT EXISTS verification_tokens (
      identifier TEXT NOT NULL,
      token TEXT NOT NULL,
      expires INTEGER NOT NULL,
      PRIMARY KEY (identifier, token)
    );

    CREATE TABLE IF NOT EXISTS api_keys (
      id TEXT PRIMARY KEY NOT NULL,
      user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
      name TEXT NOT NULL,
      key_hash TEXT NOT NULL,
      prefix TEXT NOT NULL,
      scopes TEXT NOT NULL DEFAULT '[]',
      created_at INTEGER NOT NULL,
      last_used_at INTEGER,
      revoked_at INTEGER
    );
    CREATE INDEX IF NOT EXISTS api_keys_user_id_idx ON api_keys(user_id);

    CREATE TABLE IF NOT EXISTS decisions (
      id TEXT PRIMARY KEY NOT NULL,
      user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
      title TEXT NOT NULL,
      description TEXT,
      status TEXT NOT NULL DEFAULT 'draft',
      created_at INTEGER NOT NULL,
      updated_at INTEGER NOT NULL
    );
    CREATE INDEX IF NOT EXISTS decisions_user_id_idx ON decisions(user_id);

    CREATE TABLE IF NOT EXISTS chats (
      id TEXT PRIMARY KEY NOT NULL,
      user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
      external_session_id TEXT,
      title TEXT NOT NULL,
      decision_id TEXT REFERENCES decisions(id) ON DELETE SET NULL,
      created_at INTEGER NOT NULL,
      updated_at INTEGER NOT NULL
    );
    CREATE INDEX IF NOT EXISTS chats_user_id_idx ON chats(user_id);
    CREATE INDEX IF NOT EXISTS chats_decision_id_idx ON chats(decision_id);

    CREATE TABLE IF NOT EXISTS criteria (
      id TEXT PRIMARY KEY NOT NULL,
      decision_id TEXT NOT NULL REFERENCES decisions(id) ON DELETE CASCADE,
      parent_id TEXT,
      name TEXT NOT NULL,
      description TEXT,
      sort_order INTEGER NOT NULL DEFAULT 0,
      source TEXT NOT NULL DEFAULT 'user',
      created_at INTEGER NOT NULL,
      updated_at INTEGER NOT NULL,
      FOREIGN KEY (parent_id) REFERENCES criteria(id) ON DELETE CASCADE
    );
    CREATE INDEX IF NOT EXISTS criteria_decision_id_idx ON criteria(decision_id);

    CREATE TABLE IF NOT EXISTS alternatives (
      id TEXT PRIMARY KEY NOT NULL,
      decision_id TEXT NOT NULL REFERENCES decisions(id) ON DELETE CASCADE,
      name TEXT NOT NULL,
      description TEXT,
      sort_order INTEGER NOT NULL DEFAULT 0,
      source TEXT NOT NULL DEFAULT 'user',
      created_at INTEGER NOT NULL,
      updated_at INTEGER NOT NULL
    );
    CREATE INDEX IF NOT EXISTS alternatives_decision_id_idx ON alternatives(decision_id);

    CREATE TABLE IF NOT EXISTS pairwise_judgments (
      id TEXT PRIMARY KEY NOT NULL,
      decision_id TEXT NOT NULL REFERENCES decisions(id) ON DELETE CASCADE,
      parent_criterion_id TEXT REFERENCES criteria(id) ON DELETE CASCADE,
      left_id TEXT NOT NULL,
      right_id TEXT NOT NULL,
      level TEXT NOT NULL,
      value REAL NOT NULL,
      status TEXT NOT NULL DEFAULT 'proposal',
      created_at INTEGER NOT NULL,
      updated_at INTEGER NOT NULL
    );
    CREATE INDEX IF NOT EXISTS pairwise_decision_id_idx ON pairwise_judgments(decision_id);
    CREATE UNIQUE INDEX IF NOT EXISTS pairwise_unique_idx
      ON pairwise_judgments(decision_id, parent_criterion_id, level, left_id, right_id)
      WHERE parent_criterion_id IS NOT NULL;
    CREATE UNIQUE INDEX IF NOT EXISTS pairwise_unique_root_idx
      ON pairwise_judgments(decision_id, level, left_id, right_id)
      WHERE parent_criterion_id IS NULL;

    CREATE TABLE IF NOT EXISTS snapshots (
      id TEXT PRIMARY KEY NOT NULL,
      decision_id TEXT NOT NULL REFERENCES decisions(id) ON DELETE CASCADE,
      chat_id TEXT REFERENCES chats(id) ON DELETE SET NULL,
      label TEXT NOT NULL,
      prompt TEXT NOT NULL,
      output_summary TEXT,
      state TEXT NOT NULL,
      created_at INTEGER NOT NULL
    );
    CREATE INDEX IF NOT EXISTS snapshots_decision_id_idx ON snapshots(decision_id);
    CREATE INDEX IF NOT EXISTS snapshots_chat_id_idx ON snapshots(chat_id);

    CREATE TABLE IF NOT EXISTS data_source_records (
      id TEXT PRIMARY KEY NOT NULL,
      decision_id TEXT NOT NULL REFERENCES decisions(id) ON DELETE CASCADE,
      entity_type TEXT NOT NULL,
      entity_id TEXT NOT NULL,
      url TEXT,
      provider TEXT,
      raw_excerpt TEXT,
      quality_score REAL,
      discrepancy_flags TEXT,
      created_at INTEGER NOT NULL
    );
    CREATE INDEX IF NOT EXISTS data_source_decision_id_idx ON data_source_records(decision_id);
  `);
}

function createDb() {
  const dbPath = resolveDbPath();
  mkdirSync(dirname(dbPath), { recursive: true });
  const sqlite = new Database(dbPath);
  sqlite.pragma("journal_mode = WAL");
  sqlite.pragma("foreign_keys = ON");
  ensureSchema(sqlite);
  return drizzle(sqlite, { schema });
}

/** Singleton Drizzle client (SQLite MVP via better-sqlite3). */
export const db = createDb();

export { schema };
export type Db = typeof db;
