// Offline bpmn.io rendering of the active pack, using installed dependencies.
const fs = require('node:fs');
const path = require('node:path');
const { chromium } = require('../web/node_modules/@playwright/test');

(async () => {
  const root = path.resolve(__dirname, '..');
  const pack = path.resolve(root, process.argv[2] || 'experiments/simple-erp/current-staff-v2');
  const output = path.resolve(root, 'outputs/diagram-review');
  fs.mkdirSync(output, { recursive: true });
  const browser = await chromium.launch({ channel: 'chrome', headless: true });
  const results = [];
  try {
    const page = await browser.newPage({ viewport: { width: 2600, height: 1200 } });
    await page.setContent('<html><body style="margin:0"><div id="canvas" style="width:2600px;height:1200px"></div></body></html>');
    await page.addScriptTag({ path: path.join(root, 'web/node_modules/bpmn-js/dist/bpmn-viewer.development.js') });
    await page.evaluate(() => { window.viewer = new window.BpmnJS({ container: '#canvas' }); });
    const folders = fs.statSync(pack).isFile() ? [...new Set(JSON.parse(fs.readFileSync(pack, 'utf8')).map(r => path.dirname(path.resolve(root, r.file))))] : fs.readdirSync(pack, { withFileTypes: true }).filter(e => e.isDirectory()).map(e => path.join(pack, e.name));
    for (const folder of folders) {
      for (const name of ['before.bpmn', 'after.bpmn']) {
        const file = path.join(folder, name);
        if (!fs.existsSync(file)) continue;
        const xml = fs.readFileSync(file, 'utf8');
        const result = await page.evaluate(async xml => {
          const { warnings } = await window.viewer.importXML(xml);
          window.viewer.get('canvas').zoom('fit-viewport');
          const elements = window.viewer.get('elementRegistry').getAll();
          const expected = [...xml.matchAll(/bpmnElement="([^"]+)"/g)].map(m => m[1]);
          const missing = expected.filter(id => !elements.some(e => e.id === id));
          const gateways = elements.filter(e => /Gateway$/.test(e.type)).map(e => ({
            id: e.id, type: e.type, incoming: e.incoming.length, outgoing: e.outgoing.length,
          }));
          const { svg } = await window.viewer.saveSVG();
          return { warnings: warnings.map(w => w.message), missing, gateways, svg };
        }, xml);
        const stem = `${path.relative(root, folder).replace(/[\\/]/g, "-")}-${name.replace('.bpmn', '')}`;
        fs.writeFileSync(path.join(output, `${stem}.svg`), result.svg, 'utf8');
        await page.screenshot({ path: path.join(output, `${stem}.png`) });
        delete result.svg;
        results.push({ file: path.relative(root, file), ...result });
      }
    }
  } finally { await browser.close(); }
  fs.writeFileSync(path.join(output, 'render-audit.json'), JSON.stringify(results, null, 2) + '\n');
  const failed = results.filter(r => r.warnings.length || r.missing.length);
  console.log(JSON.stringify({ rendered: results.length, failures: failed, output: path.relative(root, output) }));
  if (!results.length || failed.length) process.exitCode = 1;
})().catch(error => { console.error(error); process.exitCode = 1; });
