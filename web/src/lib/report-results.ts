import type { Impact, Report } from "./types";

export type ResultCategory = "all" | "code" | "tests" | "docs";

export function independentChanges(report: Report) {
  const primaryIds = new Set(report.changes.filter(change => !change.part_of).map(change => change.new_element?.id ?? change.old_element?.id));
  return report.changes.map((change, index) => ({ change, index }))
    .filter(({ change }) => !change.part_of || !primaryIds.has(change.part_of));
}
export function resultCategory(impact: Impact): Exclude<ResultCategory, "all"> {
  const file = impact.file.replaceAll("\\", "/").toLowerCase();
  if (impact.layer === "documentation" || /(^|\/)(docs?|documentation)(\/|$)/.test(file) || /\.(md|mdx|rst|txt|bpmn)$/.test(file)) return "docs";
  if (impact.layer === "test" || /(^|\/)(__tests__|tests?|e2e|specs?)(\/|$)/.test(file) || /(?:_test\.go|[.-](test|spec)\.[^/]+)$/.test(file) || /(^|\/)test_[^/]+\.py$/.test(file)) return "tests";
  return "code";
}

export function matchesCategory(impact: Impact, category: ResultCategory) {
  return category === "all" || resultCategory(impact) === category;
}

export function groupResults(report: Report, category: ResultCategory) {
  const files = new Map<string, { file: string; entries: { changeIndex: number; rank: number; impact: Impact }[] }>();
  report.changes.forEach((change, changeIndex) => change.impacts.forEach((impact, index) => {
    if (!matchesCategory(impact, category)) return;
    const key = impact.file.replaceAll("\\", "/");
    const group = files.get(key) ?? { file: key, entries: [] };
    group.entries.push({ changeIndex, rank: index + 1, impact });
    files.set(key, group);
  }));
  // First appearance preserves report order. No invented score across changes.
  return [...files.values()];
}
