"use client";

import { useCallback, useEffect, useRef, useState } from "react";

type Props = {
  text: string;
  active: boolean;
  /** characters per second */
  cps?: number;
  className?: string;
  as?: "h2" | "p" | "span";
  id?: string;
  showCursor?: boolean;
  onDone?: () => void;
  delayMs?: number;
  /** click-to-finish window in ms */
  rushMs?: number;
};

/**
 * Streams text character-by-character when `active` becomes true.
 * Full text is always measured invisibly so typing never shifts layout.
 * Click (or Enter/Space) while typing rushes the rest to finish in ~rushMs.
 */
export function StreamText({
  text,
  active,
  cps = 28,
  className,
  as: Tag = "span",
  id,
  showCursor = true,
  onDone,
  delayMs = 0,
  rushMs = 2000,
}: Props) {
  const [shown, setShown] = useState("");
  const [done, setDone] = useState(false);
  const startedRef = useRef(false);
  const indexRef = useRef(0);
  const onDoneRef = useRef(onDone);
  const rushUntilRef = useRef(0);
  const timerRef = useRef(0);
  const stepRef = useRef<() => void>(() => {});

  useEffect(() => {
    onDoneRef.current = onDone;
  }, [onDone]);

  useEffect(() => {
    if (!active || done) return;
    if (!startedRef.current) {
      startedRef.current = true;
      indexRef.current = 0;
    }

    let cancelled = false;

    const schedule = (ms: number) => {
      window.clearTimeout(timerRef.current);
      timerRef.current = window.setTimeout(() => stepRef.current(), ms);
    };

    const finish = () => {
      indexRef.current = text.length;
      setShown(text);
      setDone(true);
      rushUntilRef.current = 0;
      onDoneRef.current?.();
    };

    const step = () => {
      if (cancelled) return;

      const rushing = rushUntilRef.current > 0;
      if (rushing && performance.now() >= rushUntilRef.current) {
        finish();
        return;
      }

      const next = indexRef.current + 1;
      indexRef.current = next;
      setShown(text.slice(0, next));

      if (next >= text.length) {
        setDone(true);
        rushUntilRef.current = 0;
        onDoneRef.current?.();
        return;
      }

      if (rushing) {
        const remaining = text.length - next;
        const left = Math.max(12, rushUntilRef.current - performance.now());
        schedule(left / Math.max(1, remaining));
        return;
      }

      const jitter = 0.65 + Math.random() * 0.7;
      const ch = text[next - 1];
      const pause = ch === "." || ch === "—" || ch === "," ? 90 : 0;
      schedule((1000 / cps) * jitter + pause);
    };

    stepRef.current = step;

    const initialDelay = indexRef.current === 0 ? delayMs : 0;
    schedule(initialDelay);

    return () => {
      cancelled = true;
      window.clearTimeout(timerRef.current);
    };
  }, [active, done, text, cps, delayMs]);

  const rush = useCallback(() => {
    if (done || indexRef.current >= text.length) return;
    if (!startedRef.current && !active) return;
    rushUntilRef.current = performance.now() + rushMs;
    window.clearTimeout(timerRef.current);
    stepRef.current();
  }, [active, done, text.length, rushMs]);

  const showCaret =
    showCursor && !done && (active || shown.length > 0) && shown.length < text.length;
  const canRush = !done && shown.length < text.length && (active || shown.length > 0);

  return (
    <Tag
      id={id}
      className={`stream-slot ${canRush ? "stream-rushable" : ""} ${className ?? ""}`}
      aria-label={text}
      title={canRush ? "Click to finish typing" : undefined}
      onClick={canRush ? rush : undefined}
      onKeyDown={
        canRush
          ? (e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                rush();
              }
            }
          : undefined
      }
      tabIndex={canRush ? 0 : undefined}
    >
      <span className="stream-measure" aria-hidden>
        {text}
      </span>
      <span className="stream-live" aria-hidden>
        {shown}
        {showCaret ? <span className="stream-cursor">▍</span> : null}
      </span>
    </Tag>
  );
}
