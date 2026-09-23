"use client";

import { useState } from "react";
import { SITE } from "@/lib/site";

export function InstallCommand() {
  const [copied, setCopied] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(SITE.install);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      setCopied(false);
    }
  }

  return (
    <button
      type="button"
      onClick={copy}
      className="install-chip group cursor-pointer text-left"
      aria-label="Copy install command"
    >
      <span className="shrink-0 text-amber">$</span>
      <code className="truncate">{SITE.install}</code>
      <span className="ml-auto shrink-0 text-[0.65rem] tracking-[0.18em] text-steel uppercase group-hover:text-amber-hot">
        {copied ? "copied" : "copy"}
      </span>
    </button>
  );
}
