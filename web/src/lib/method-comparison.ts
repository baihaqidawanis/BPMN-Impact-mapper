import { independentChanges } from "./report-results";
import type { Change, Report } from "./types";

export function comparisonChanges(report: Report): Change[] {
  return independentChanges(report).map(({ change }) => change);
}

export function unresolvedComparisonChanges(report: Report): number {
  return comparisonChanges(report).filter(change => change.impacts.length === 0 || change.unresolved_impacts.length > 0).length;
}

export function candidateKeys(report: Report): Set<string> {
  return new Set(comparisonChanges(report).flatMap(change => change.impacts.map(impact =>
    `${changeKey(change)}|${impact.file.replaceAll("\\", "/")}`)));
}

export function changeKey(change: Change): string {
  return `${change.change_type}|${change.new_element?.id ?? change.old_element?.id ?? "unresolved"}`;
}

// Jaccard overlap compares returned file/change sets, not correctness.
export function overlapPercent(left: Set<string>, right: Set<string>): number {
  const union = new Set([...left, ...right]);
  if (union.size === 0) return 100;
  let shared = 0;
  for (const item of left) if (right.has(item)) shared++;
  return Math.round((shared / union.size) * 100);
}
