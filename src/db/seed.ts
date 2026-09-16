/**
 * Optional demo seed — run with: bun run db:seed
 */
import { db } from "./index";
import {
  alternatives,
  chats,
  criteria,
  decisions,
  pairwiseJudgments,
  users,
} from "./schema";

async function seed() {
  const [user] = await db
    .insert(users)
    .values({
      name: "Demo User",
      email: "demo@example.com",
      locale: "en",
    })
    .onConflictDoNothing()
    .returning();

  const existing =
    user ??
    (await db.query.users.findFirst({
      where: (u, { eq }) => eq(u.email, "demo@example.com"),
    }));

  if (!existing) {
    throw new Error("Failed to create or find demo user");
  }

  const [decision] = await db
    .insert(decisions)
    .values({
      userId: existing.id,
      title: "Which phone should I buy?",
      description: "Demo AHP decision for local development",
      status: "active",
    })
    .returning();

  await db.insert(chats).values({
    userId: existing.id,
    title: "Phone shopping session",
    decisionId: decision.id,
  });

  const [price, camera, battery] = await db
    .insert(criteria)
    .values([
      {
        decisionId: decision.id,
        name: "Price",
        sortOrder: 0,
        source: "user" as const,
      },
      {
        decisionId: decision.id,
        name: "Camera",
        sortOrder: 1,
        source: "agent" as const,
      },
      {
        decisionId: decision.id,
        name: "Battery",
        sortOrder: 2,
        source: "user" as const,
      },
    ])
    .returning();

  const [phoneA, phoneB] = await db
    .insert(alternatives)
    .values([
      {
        decisionId: decision.id,
        name: "Phone A",
        sortOrder: 0,
        source: "user" as const,
      },
      {
        decisionId: decision.id,
        name: "Phone B",
        sortOrder: 1,
        source: "import" as const,
      },
    ])
    .returning();

  await db.insert(pairwiseJudgments).values([
    {
      decisionId: decision.id,
      parentCriterionId: null,
      level: "criteria" as const,
      leftId: price.id,
      rightId: camera.id,
      value: 3,
      status: "committed" as const,
    },
    {
      decisionId: decision.id,
      parentCriterionId: null,
      level: "criteria" as const,
      leftId: price.id,
      rightId: battery.id,
      value: 2,
      status: "proposal" as const,
    },
    {
      decisionId: decision.id,
      parentCriterionId: price.id,
      level: "alternatives" as const,
      leftId: phoneA.id,
      rightId: phoneB.id,
      value: 1 / 3,
      status: "committed" as const,
    },
  ]);

  console.log("Seeded demo user, decision, criteria, alternatives, judgments.");
  console.log(`  user:     ${existing.email} (${existing.id})`);
  console.log(`  decision: ${decision.title} (${decision.id})`);
}

seed().catch((err) => {
  console.error(err);
  process.exit(1);
});
