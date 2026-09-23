export const SITE = {
  name: "ahp",
  tagline: "Ask. Compare. Decide — with Analytic Hierarchy Process.",
  url: "https://lazarok09.github.io/ai-first-analytic-hierarchy-process",
  github: "https://github.com/lazarok09/ai-first-analytic-hierarchy-process",
  releases:
    "https://github.com/lazarok09/ai-first-analytic-hierarchy-process/releases",
  license:
    "https://github.com/lazarok09/ai-first-analytic-hierarchy-process/blob/main/LICENSE",
  install: "go install github.com/lazarok09/ahp-method/cmd/ahp@latest",
  description:
    "Local Analytic Hierarchy Process CLI: ask a question, name your criteria, get a ranked decision from pairwise judgment. Agents draft; you sign.",
} as const;

export const SECTIONS = [
  {
    id: "hook",
    kicker: "01 — AHP",
    line: "A decision made by science.",
    sub: "Ask a question. Name what matters. What you get back is ranked by the Analytic Hierarchy Process — options compared side by side with science, so you can trust the answer. Agents collect and calculate; you pick the winner.",
  },
  {
    id: "lie",
    kicker: "02 — PROVE IT",
    line: "Gut can pick a winner. It can’t prove the race.",
    sub: "In most real decisions we justify the choice by feel — and never show how good it was next to the others. Now you can.",
  },
  {
    id: "stance",
    kicker: "03 — YOURS",
    line: "The hierarchy lives in your files.",
    sub: "We don’t hold your data. No cloud, cookies, or third parties. You are safe to ask anything.",
  },
  {
    id: "agent",
    kicker: "04 — CONTRACT",
    line: "Let them draft. You sign.",
    sub: "Your agent gathers the options, runs the comparisons, and does the math. You just choose the best one.",
  },
  {
    id: "proof",
    kicker: "05 — REPORT",
    line: "See the thinking behind the decision.",
    sub: "The HTML report shows your comparisons, the resulting weights, and where consistency broke — so you can fix the judgment, not guess the winner.",
  },
  {
    id: "close",
    kicker: "06 — INSTALL",
    line: "Never be unsure about a call.",
    sub: "Start using it now — bring structure to any decision you’ve been making blind.",
  },
] as const;

export type SectionId = (typeof SECTIONS)[number]["id"];
