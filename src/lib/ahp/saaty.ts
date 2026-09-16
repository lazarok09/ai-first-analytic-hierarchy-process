/** Saaty discrete intensities 1–9 (and reciprocals stored as value < 1). */
export const SAATY_INTENSITIES = [1, 2, 3, 4, 5, 6, 7, 8, 9] as const;

export type CriterionPair = {
  leftId: string;
  rightId: string;
};

/** Ordered unique unordered pairs (i < j by array index). */
export function criterionPairs(
  ids: string[],
): CriterionPair[] {
  const pairs: CriterionPair[] = [];
  for (let i = 0; i < ids.length; i += 1) {
    for (let j = i + 1; j < ids.length; j += 1) {
      pairs.push({ leftId: ids[i]!, rightId: ids[j]! });
    }
  }
  return pairs;
}

/**
 * Normalize stored Saaty value to UI: which side is preferred + intensity 1–9.
 * value > 1 means left preferred; value < 1 means right preferred (reciprocal).
 */
export function decodeSaaty(value: number): {
  preferLeft: boolean;
  intensity: number;
} {
  if (value >= 1) {
    return { preferLeft: true, intensity: clampIntensity(value) };
  }
  return { preferLeft: false, intensity: clampIntensity(1 / value) };
}

export function encodeSaaty(preferLeft: boolean, intensity: number): number {
  const i = clampIntensity(intensity);
  return preferLeft ? i : 1 / i;
}

function clampIntensity(n: number): number {
  const rounded = Math.round(n);
  if (rounded < 1) return 1;
  if (rounded > 9) return 9;
  return rounded;
}

/** Match judgment regardless of left/right orientation stored. */
export function findJudgmentForPair<
  T extends { leftId: string; rightId: string; value: number },
>(
  judgments: T[],
  leftId: string,
  rightId: string,
): { judgment: T; orientedValue: number } | null {
  const direct = judgments.find(
    (j) => j.leftId === leftId && j.rightId === rightId,
  );
  if (direct) {
    return { judgment: direct, orientedValue: direct.value };
  }
  const swapped = judgments.find(
    (j) => j.leftId === rightId && j.rightId === leftId,
  );
  if (swapped) {
    return { judgment: swapped, orientedValue: 1 / swapped.value };
  }
  return null;
}
