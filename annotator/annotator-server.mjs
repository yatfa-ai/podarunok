// Tiny zero-dep Node server for the Annotator (generic, app-agnostic).
//   GET  /                            → annotator page
//   GET  /shots/<file>                → element screenshots (user-provided)
//   GET  /datasets                    → { names:[...] } list of saved datasets
//   GET  /datasets/<name>             → that dataset JSON
//   POST /datasets/<name> (body=JSON) → save a dataset
//   POST /submit      (body=JSON)     → annotations → submissions/latest.json (+ timestamped copy)
//   GET  /latest                      → last submission
// Run:  node annotator/annotator-server.mjs   (PORT=xxxx to override)
import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = path.dirname(fileURLToPath(import.meta.url));
const SUBMISSIONS = path.join(ROOT, 'submissions');
const DATASETS = path.join(ROOT, 'datasets');
[SUBMISSIONS, DATASETS].forEach(d => fs.mkdirSync(d, { recursive: true }));

const PORT = process.env.PORT || 4567;
const MIME = {
  '.html': 'text/html; charset=utf-8',
  '.png': 'image/png', '.jpg': 'image/jpeg', '.jpeg': 'image/jpeg',
  '.json': 'application/json', '.js': 'text/javascript',
  '.css': 'text/css', '.svg': 'image/svg+xml',
};

function serveFile(res, filePath){
  fs.readFile(filePath, (err, data) => {
    if (err) { res.writeHead(404); res.end('not found'); return; }
    res.writeHead(200, { 'Content-Type': MIME[path.extname(filePath).toLowerCase()] || 'application/octet-stream' });
    res.end(data);
  });
}
function safeStatic(res, relRoot, relPath){
  const base = path.join(ROOT, relRoot);
  const fp = path.join(base, path.normalize(relPath).replace(/^([/\\])+/, ''));
  if (!fp.startsWith(base)) { res.writeHead(403); res.end('forbidden'); return; }
  serveFile(res, fp);
}
function readBody(req, cb){
  let body = '';
  req.on('data', c => (body += c));
  req.on('end', () => cb(body));
}
// dataset names: a-z A-Z 0-9 _ - only (no traversal)
function safeName(s){ return /^[A-Za-z0-9_-]+$/.test(s) ? s : null; }

const server = http.createServer((req, res) => {
  const u = new URL(req.url, `http://localhost:${PORT}`);
  const p = u.pathname;

  if (req.method === 'GET' && (p === '/' || p === '/annotator.html' || p === '/dashboard-annotator.html')) {
    return serveFile(res, path.join(ROOT, 'annotator.html'));
  }
  if (req.method === 'GET' && p.startsWith('/shots/')) {
    return safeStatic(res, 'shots', p.slice('/shots/'.length));
  }
  if (req.method === 'GET' && p === '/datasets') {
    fs.readdir(DATASETS, (err, files) => {
      const names = (files||[]).filter(f => f.endsWith('.json')).map(f => f.slice(0, -5));
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ names }));
    });
    return;
  }
  if (req.method === 'GET' && p.startsWith('/datasets/')) {
    const name = safeName(decodeURIComponent(p.slice('/datasets/'.length).replace(/\.json$/, '')));
    if (!name) { res.writeHead(400); res.end('bad name'); return; }
    return serveFile(res, path.join(DATASETS, name + '.json'));
  }
  if (req.method === 'POST' && p.startsWith('/datasets/')) {
    const name = safeName(decodeURIComponent(p.slice('/datasets/'.length).replace(/\.json$/, '')));
    if (!name) { res.writeHead(400); res.end('bad name'); return; }
    return readBody(req, body => {
      try { JSON.parse(body); } catch (e) { res.writeHead(400); res.end('bad json'); return; }
      fs.writeFile(path.join(DATASETS, name + '.json'), body, () => {
        console.log(`[${new Date().toISOString()}] dataset saved: ${name}`);
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ ok: true, name }));
      });
    });
  }
  if (req.method === 'GET' && p === '/latest') {
    const f = path.join(SUBMISSIONS, 'latest.json');
    if (fs.existsSync(f)) return serveFile(res, f);
    res.writeHead(404); res.end('no submissions yet'); return;
  }
  if (req.method === 'POST' && p === '/submit') {
    return readBody(req, body => {
      try {
        JSON.parse(body);
        const ts = new Date().toISOString().replace(/[:.]/g, '-');
        fs.writeFileSync(path.join(SUBMISSIONS, `annotations-${ts}.json`), body);
        fs.writeFileSync(path.join(SUBMISSIONS, 'latest.json'), body);
        const j = JSON.parse(body);
        const n = j.elements?.length ?? 0;
        console.log(`[${new Date().toISOString()}] submission saved — ${n} elements (${j.name||"untitled"})`);
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ ok: true }));
      } catch (e) {
        console.error('submit error:', e.message);
        res.writeHead(400); res.end('bad json');
      }
    });
  }
  res.writeHead(404); res.end('not found');
});

server.listen(PORT, () =>
  console.log(`annotator server: http://localhost:${PORT}/   (datasets + POST /submit → submissions/)`)
);
