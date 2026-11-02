import { readdirSync, readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("../", import.meta.url));
const decoder = new TextDecoder("utf-8", { fatal: true });
const mojibake = /(?:\u00c3[\u0080-\u00ff\u2000-\u20ff]|\u00c2[\u0080-\u00bf]|\u00e2[\u20ac\u2020\u2013\u02c6]|\ufffd)/u;
let failures = 0;
function visit(directory) {
  for (const entry of readdirSync(directory, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
    const file = path.join(directory, entry.name);
    if (entry.isDirectory()) { visit(file); continue; }
    if (!/\.(tsx?|css)$/.test(file)) continue;
    let text;
    try { text = decoder.decode(readFileSync(file)); }
    catch { console.error(`${path.relative(root, file)}: invalid UTF-8`); failures++; continue; }
    text.split(/\r?\n/).forEach((line, i) => {
      if (mojibake.test(line)) { console.error(`${path.relative(root, file)}:${i + 1}: suspicious mojibake sequence`); failures++; }
    });
  }
}
visit(path.join(root, "web", "src"));
if (!failures) console.log("UI text encoding passed (UTF-8; no known mojibake sequences).");
process.exitCode = failures ? 1 : 0;
