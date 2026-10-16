const siteUrl = (
  process.env.NEXT_PUBLIC_SITE_URL ??
  "https://lazarok09.github.io/ai-first-analytic-hierarchy-process"
).replace(/\/$/, "");

export const SITE = {
  name: "ahp",
  tagline: "Ask. Compare. Decide — with Analytic Hierarchy Process.",
  url: siteUrl,
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
    title: "AHP",
    kicker: "01 — AHP",
    line: "A decision made by science.",
    sub: "A laptop, a hire, a vendor. The calls that cost real money. You leave with one winner, and a reason strong enough to show anyone who asks.",
  },
  {
    id: "lie",
    title: "Prove it",
    kicker: "02 — PROVE IT",
    line: "Prove your guts were right.",
    sub: "In most real decisions we justify the choice by feel — and never show how good it was next to the others. Now you can.",
  },
  {
    id: "stance",
    title: "Yours",
    kicker: "03 — YOURS",
    line: "Your questions will be safe.",
    sub: "No cloud, no third parties. The question never leaves your computer.",
  },
  {
    id: "agent",
    title: "Contract",
    kicker: "04 — CONTRACT",
    line: "Your agent knows what to do.",
    sub: "It gathers the options, runs the comparisons, and does the math. With the Analytic Hierarchy Process, every decision is one you can trust.",
  },
  {
    id: "proof",
    title: "Report",
    kicker: "05 — REPORT",
    line: "See the thinking behind the decision.",
    sub: "The HTML report shows your comparisons, the weights they produced, and which comparison contradicts the others, so you can correct that one and trust the ranking.",
  },
  {
    id: "close",
    title: "Install",
    kicker: "06 — INSTALL",
    line: "Never be unsure about a call again.",
    sub: "Start using it now — bring structure to any decision you’ve been making blind.",
  },
] as const;

export type SectionId = (typeof SECTIONS)[number]["id"];
