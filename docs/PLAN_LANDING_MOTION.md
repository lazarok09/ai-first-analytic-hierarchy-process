# Plan: Landing page motion

**Status:** shipped. The first section (`hook`) still types. Prove it, Yours, Contract, Report, and Install fade in together with the full sentence already visible.  
**Date:** 2026-10-16  
**Surface:** `web/` only. Do not touch `internal/render` or `output/report.html` in the same change.

---

## Problem

Headlines and paragraphs on the landing page are typed one character at a time. The paragraph and the buttons stay invisible until that typing finishes.

`web/src/components/StreamText.tsx` starts with an empty visible string. The full sentence sits in a `visibility: hidden` measure span. `web/src/components/ManifestoSection.tsx` then gates the rest:

1. Kicker and headline fade in only after the section is 40% in view. Before that they sit near opacity 0.
2. The paragraph stays at opacity 0 until `lineDone`.
3. Install and GitHub buttons stay at opacity 0 until `subDone`.

A blocked script, a slow hydration, or a section that never counts as in view leaves a dark page with no sentence to read. The typewriter also makes the sentence harder to read while it arrives.

## Goals

1. The full headline is in the HTML on first paint.
2. One short entrance per section. Nothing waits on another line of text.
3. Motion that hides copy is gone.
4. The 3D scene stays the motion on the page.

### Non-goals

- Rewriting the section copy in `web/src/lib/site.ts`.
- Changing Lenis, the table of contents, or `DeckEngine`.
- A new animation library.
- Per-word, scramble, or typewriter effects.

## What to remove

- `StreamText` on every `line` and `sub`.
- `lineDone` and `subDone`, and the opacity gates that depend on them.
- The resting pose that leaves copy at opacity 0.15 or 0.2 until intersection fires.
- `.stream-*` rules in `web/src/app/globals.css` once nothing imports `StreamText`.
- `web/src/components/StreamText.tsx` if it has no remaining imports.

## What to put in its place

Render the kicker, headline, paragraph, and buttons as normal text.

When a section enters the viewport, fade and rise the whole block once:

- duration about 400ms
- ease-out
- kicker, headline, paragraph, and buttons move together

If `prefers-reduced-motion: reduce`, skip the transform and show full opacity immediately.

Do not animate individual words.

## Files

| File | Change |
|------|--------|
| `web/src/components/ManifestoSection.tsx` | Drop `StreamText` and the done-flags. One entrance for the block. |
| `web/src/components/StreamText.tsx` | Delete when unused. |
| `web/src/app/globals.css` | Delete `.stream-slot`, `.stream-measure`, `.stream-live`, `.stream-cursor`, `.stream-rushable`. |

## Done when

- The opening section still types its headline and paragraph.
- Prove it, Yours, Contract, Report, and Install show the full sentence and fade in together. Their buttons do not wait on typing.
- `prefers-reduced-motion: reduce` shows the opening section as static text too.
- `npm run build` in `web/` still writes the static export.
