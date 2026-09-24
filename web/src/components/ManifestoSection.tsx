"use client";

import { motion, useInView } from "framer-motion";
import { useEffect, useState, useRef } from "react";
import { useScrollState } from "@/lib/scroll-store";
import type { SectionId } from "@/lib/site";
import { StreamText } from "./StreamText";

type Props = {
  id: SectionId;
  kicker: string;
  line: string;
  sub: string;
  children?: React.ReactNode;
  align?: "left" | "right";
};

const ease = [0.16, 1, 0.3, 1] as const;

export function ManifestoSection({
  id,
  kicker,
  line,
  sub,
  children,
  align = "left",
}: Props) {
  const ref = useRef<HTMLElement>(null);
  const inView = useInView(ref, { amount: 0.4, margin: "-8% 0px -8% 0px" });
  const { setActive, flash } = useScrollState();
  const [lineDone, setLineDone] = useState(false);
  const [subDone, setSubDone] = useState(false);

  useEffect(() => {
    if (inView) setActive(id);
  }, [inView, id, setActive]);

  return (
    <section
      id={id}
      ref={ref}
      className={`section-shell ${align === "right" ? "text-right" : ""}`}
      data-flash={flash?.id === id ? String(flash.token) : undefined}
      aria-labelledby={`${id}-line`}
    >
      <div className="stage relative">
        <div
          className="pointer-events-none absolute -inset-x-8 -inset-y-10 -z-10 rounded-[2rem] bg-[radial-gradient(ellipse_at_center,rgba(10,12,15,0.88),transparent_70%)]"
          aria-hidden
        />

        <motion.p
          className="kicker mb-5"
          initial={{ opacity: 0, y: 14 }}
          animate={inView ? { opacity: 1, y: 0 } : { opacity: 0.15, y: 10 }}
          transition={{ duration: 0.9, ease }}
        >
          {kicker}
        </motion.p>

        <motion.div
          initial={{ opacity: 0, y: 28 }}
          animate={inView ? { opacity: 1, y: 0 } : { opacity: 0.2, y: 18 }}
          transition={{ duration: 1.05, delay: 0.08, ease }}
        >
          <StreamText
            id={`${id}-line`}
            as="h2"
            className="manifesto-line"
            text={line}
            active={inView}
            cps={32}
            delayMs={180}
            onDone={() => setLineDone(true)}
          />
        </motion.div>

        {/* Sub always reserves its final height; only fades in when line finishes */}
        <motion.div
          className={`mt-6 ${align === "right" ? "ml-auto" : ""}`}
          initial={false}
          animate={
            inView && lineDone
              ? { opacity: 1 }
              : { opacity: 0 }
          }
          transition={{ duration: 0.55, ease }}
          style={{ pointerEvents: inView && lineDone ? "auto" : "none" }}
        >
          <StreamText
            as="p"
            className={`subline ${align === "right" ? "ml-auto" : ""}`}
            text={sub}
            active={inView && lineDone}
            cps={48}
            delayMs={80}
            showCursor
            onDone={() => setSubDone(true)}
          />
        </motion.div>

        {children ? (
          <motion.div
            className={`mt-8 ${align === "right" ? "flex justify-end" : ""}`}
            initial={false}
            animate={
              inView && subDone
                ? { opacity: 1, y: 0 }
                : { opacity: 0, y: 0 }
            }
            transition={{ duration: 0.7, ease }}
            style={{ pointerEvents: inView && subDone ? "auto" : "none" }}
            aria-hidden={!(inView && subDone)}
          >
            {children}
          </motion.div>
        ) : null}
      </div>
    </section>
  );
}
