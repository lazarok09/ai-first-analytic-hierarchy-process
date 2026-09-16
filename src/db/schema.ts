import { sql } from "drizzle-orm";
import {
  type AnySQLiteColumn,
  foreignKey,
  index,
  integer,
  primaryKey,
  real,
  sqliteTable,
  text,
  uniqueIndex,
} from "drizzle-orm/sqlite-core";

const newId = () => crypto.randomUUID();

const createdAt = () =>
  integer("created_at", { mode: "timestamp_ms" })
    .notNull()
    .$defaultFn(() => new Date());

const updatedAt = () =>
  integer("updated_at", { mode: "timestamp_ms" })
    .notNull()
    .$defaultFn(() => new Date());

// --- Auth.js-compatible tables ---
// Column names match @auth/drizzle-adapter SQLite defaults (camelCase).

export const users = sqliteTable("users", {
  id: text("id").primaryKey().$defaultFn(newId),
  name: text("name"),
  email: text("email").unique(),
  emailVerified: integer("emailVerified", { mode: "timestamp_ms" }),
  image: text("image"),
  locale: text("locale", { enum: ["en", "pt-BR"] }).notNull().default("en"),
  createdAt: createdAt(),
  updatedAt: updatedAt(),
});

export const accounts = sqliteTable(
  "accounts",
  {
    userId: text("userId")
      .notNull()
      .references(() => users.id, { onDelete: "cascade" }),
    type: text("type").notNull(),
    provider: text("provider").notNull(),
    providerAccountId: text("providerAccountId").notNull(),
    refresh_token: text("refresh_token"),
    access_token: text("access_token"),
    expires_at: integer("expires_at"),
    token_type: text("token_type"),
    scope: text("scope"),
    id_token: text("id_token"),
    session_state: text("session_state"),
  },
  (t) => [primaryKey({ columns: [t.provider, t.providerAccountId] })],
);

export const sessions = sqliteTable("sessions", {
  sessionToken: text("sessionToken").primaryKey(),
  userId: text("userId")
    .notNull()
    .references(() => users.id, { onDelete: "cascade" }),
  expires: integer("expires", { mode: "timestamp_ms" }).notNull(),
});

export const verificationTokens = sqliteTable(
  "verification_tokens",
  {
    identifier: text("identifier").notNull(),
    token: text("token").notNull(),
    expires: integer("expires", { mode: "timestamp_ms" }).notNull(),
  },
  (t) => [primaryKey({ columns: [t.identifier, t.token] })],
);

// --- Domain tables ---

export const apiKeys = sqliteTable(
  "api_keys",
  {
    id: text("id").primaryKey().$defaultFn(newId),
    userId: text("user_id")
      .notNull()
      .references(() => users.id, { onDelete: "cascade" }),
    name: text("name").notNull(),
    keyHash: text("key_hash").notNull(),
    prefix: text("prefix").notNull(),
    scopes: text("scopes", { mode: "json" }).$type<string[]>().notNull().default([]),
    createdAt: createdAt(),
    lastUsedAt: integer("last_used_at", { mode: "timestamp_ms" }),
    revokedAt: integer("revoked_at", { mode: "timestamp_ms" }),
  },
  (t) => [index("api_keys_user_id_idx").on(t.userId)],
);

export const decisions = sqliteTable(
  "decisions",
  {
    id: text("id").primaryKey().$defaultFn(newId),
    userId: text("user_id")
      .notNull()
      .references(() => users.id, { onDelete: "cascade" }),
    title: text("title").notNull(),
    description: text("description"),
    status: text("status", { enum: ["draft", "active", "archived"] })
      .notNull()
      .default("draft"),
    createdAt: createdAt(),
    updatedAt: updatedAt(),
  },
  (t) => [index("decisions_user_id_idx").on(t.userId)],
);

export const chats = sqliteTable(
  "chats",
  {
    id: text("id").primaryKey().$defaultFn(newId),
    userId: text("user_id")
      .notNull()
      .references(() => users.id, { onDelete: "cascade" }),
    externalSessionId: text("external_session_id"),
    title: text("title").notNull(),
    decisionId: text("decision_id").references(() => decisions.id, {
      onDelete: "set null",
    }),
    createdAt: createdAt(),
    updatedAt: updatedAt(),
  },
  (t) => [
    index("chats_user_id_idx").on(t.userId),
    index("chats_decision_id_idx").on(t.decisionId),
  ],
);

export const criteria = sqliteTable(
  "criteria",
  {
    id: text("id").primaryKey().$defaultFn(newId),
    decisionId: text("decision_id")
      .notNull()
      .references(() => decisions.id, { onDelete: "cascade" }),
    parentId: text("parent_id"),
    name: text("name").notNull(),
    description: text("description"),
    sortOrder: integer("sort_order").notNull().default(0),
    source: text("source", { enum: ["user", "agent", "import"] })
      .notNull()
      .default("user"),
    createdAt: createdAt(),
    updatedAt: updatedAt(),
  },
  (t) => [
    index("criteria_decision_id_idx").on(t.decisionId),
    foreignKey({
      columns: [t.parentId],
      foreignColumns: [t.id as AnySQLiteColumn],
      name: "criteria_parent_fk",
    }).onDelete("cascade"),
  ],
);

export const alternatives = sqliteTable(
  "alternatives",
  {
    id: text("id").primaryKey().$defaultFn(newId),
    decisionId: text("decision_id")
      .notNull()
      .references(() => decisions.id, { onDelete: "cascade" }),
    name: text("name").notNull(),
    description: text("description"),
    sortOrder: integer("sort_order").notNull().default(0),
    source: text("source", { enum: ["user", "agent", "import"] })
      .notNull()
      .default("user"),
    createdAt: createdAt(),
    updatedAt: updatedAt(),
  },
  (t) => [index("alternatives_decision_id_idx").on(t.decisionId)],
);

export const pairwiseJudgments = sqliteTable(
  "pairwise_judgments",
  {
    id: text("id").primaryKey().$defaultFn(newId),
    decisionId: text("decision_id")
      .notNull()
      .references(() => decisions.id, { onDelete: "cascade" }),
    /** Null = comparing under the goal/root. */
    parentCriterionId: text("parent_criterion_id").references(() => criteria.id, {
      onDelete: "cascade",
    }),
    level: text("level", { enum: ["criteria", "alternatives"] }).notNull(),
    leftId: text("left_id").notNull(),
    rightId: text("right_id").notNull(),
    value: real("value").notNull(),
    status: text("status", { enum: ["proposal", "committed"] })
      .notNull()
      .default("proposal"),
    createdAt: createdAt(),
    updatedAt: updatedAt(),
  },
  (t) => [
    index("pairwise_decision_id_idx").on(t.decisionId),
    // Split unique indexes so SQLite NULL parent_criterion_id still enforces uniqueness
    uniqueIndex("pairwise_unique_idx")
      .on(t.decisionId, t.parentCriterionId, t.level, t.leftId, t.rightId)
      .where(sql`${t.parentCriterionId} is not null`),
    uniqueIndex("pairwise_unique_root_idx")
      .on(t.decisionId, t.level, t.leftId, t.rightId)
      .where(sql`${t.parentCriterionId} is null`),
  ],
);

export const snapshots = sqliteTable(
  "snapshots",
  {
    id: text("id").primaryKey().$defaultFn(newId),
    decisionId: text("decision_id")
      .notNull()
      .references(() => decisions.id, { onDelete: "cascade" }),
    chatId: text("chat_id").references(() => chats.id, {
      onDelete: "set null",
    }),
    label: text("label").notNull(),
    prompt: text("prompt").notNull(),
    outputSummary: text("output_summary"),
    state: text("state", { mode: "json" })
      .notNull()
      .$type<Record<string, unknown>>(),
    createdAt: createdAt(),
  },
  (t) => [
    index("snapshots_decision_id_idx").on(t.decisionId),
    index("snapshots_chat_id_idx").on(t.chatId),
  ],
);

export const dataSourceRecords = sqliteTable(
  "data_source_records",
  {
    id: text("id").primaryKey().$defaultFn(newId),
    decisionId: text("decision_id")
      .notNull()
      .references(() => decisions.id, { onDelete: "cascade" }),
    entityType: text("entity_type").notNull(),
    entityId: text("entity_id").notNull(),
    url: text("url"),
    provider: text("provider"),
    rawExcerpt: text("raw_excerpt"),
    qualityScore: real("quality_score"),
    discrepancyFlags: text("discrepancy_flags", { mode: "json" }).$type<
      string[] | Record<string, unknown>
    >(),
    createdAt: createdAt(),
  },
  (t) => [index("data_source_decision_id_idx").on(t.decisionId)],
);

export type User = typeof users.$inferSelect;
export type Account = typeof accounts.$inferSelect;
export type Session = typeof sessions.$inferSelect;
export type ApiKey = typeof apiKeys.$inferSelect;
export type Decision = typeof decisions.$inferSelect;
export type Chat = typeof chats.$inferSelect;
export type Criterion = typeof criteria.$inferSelect;
export type Alternative = typeof alternatives.$inferSelect;
export type PairwiseJudgment = typeof pairwiseJudgments.$inferSelect;
export type Snapshot = typeof snapshots.$inferSelect;
export type DataSourceRecord = typeof dataSourceRecords.$inferSelect;
