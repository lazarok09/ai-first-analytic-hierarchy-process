export type DecisionStatus = "draft" | "active" | "archived";
export type CriterionSource = "user" | "agent" | "import";
export type PairwiseStatus = "proposal" | "committed";
export type PairwiseLevel = "criteria" | "alternatives";

export type Decision = {
  id: string;
  userId: string;
  title: string;
  description: string | null;
  status: DecisionStatus;
  createdAt: string | Date;
  updatedAt: string | Date;
};

export type Criterion = {
  id: string;
  decisionId: string;
  parentId: string | null;
  name: string;
  description: string | null;
  sortOrder: number;
  source: CriterionSource;
  createdAt: string | Date;
  updatedAt: string | Date;
};

export type PairwiseJudgment = {
  id: string;
  decisionId: string;
  parentCriterionId: string | null;
  level: PairwiseLevel;
  leftId: string;
  rightId: string;
  value: number;
  status: PairwiseStatus;
  createdAt: string | Date;
  updatedAt: string | Date;
};

export type Snapshot = {
  id: string;
  decisionId: string;
  chatId: string | null;
  label: string;
  prompt: string;
  outputSummary: string | null;
  state: Record<string, unknown>;
  createdAt: string | Date;
};

export type ApiKeyPublic = {
  id: string;
  name: string;
  prefix: string;
  scopes: string[];
  createdAt: string | Date;
  lastUsedAt: string | Date | null;
  revokedAt: string | Date | null;
};

export type CreatedApiKey = ApiKeyPublic & { key: string };

export type ApiErrorBody = {
  error: {
    code: string;
    message: string;
    details?: unknown;
  };
};
