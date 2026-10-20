import type { Report } from "./types";

// Compare file presence per change, not a repo-wide union of unrelated changes.
export function methodDifferences(reports: Report[]) {
  if (reports.length < 2 || reports.some(r => !r.input_fingerprint) || new Set(reports.map(r => r.input_fingerprint)).size !== 1) return null;
  const rows = new Map<string, { change: string; file: string; methods: string[] }>();
  reports.forEach(report => report.changes.forEach(change => {
    const element = change.new_element ?? change.old_element;
    const id = `${change.change_type} · ${element?.id ?? "unresolved"}`;
    new Set(change.impacts.map(impact => impact.file.replaceAll("\\", "/"))).forEach(file => {
      const key = `${id}|${file}`;
      const row = rows.get(key) ?? { change: id, file, methods: [] };
      if (!row.methods.includes(report.method)) row.methods.push(report.method);
      rows.set(key, row);
    });
  }));
  return [...rows.values()];
}
