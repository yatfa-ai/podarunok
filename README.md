# annotator

A small, generic, **app-agnostic** screenshot annotation tool. Walk a set of elements, attach a screenshot and a verdict to each, then export or submit the whole review as JSON.

Zero dependencies — just Node. Nothing in this tool is tied to any particular app, page, or domain. **A review is data, not code** (a JSON dataset); the tool loads and edits it.

---

## What it does

- Define **elements** (name + group), attach a **screenshot**, a **verdict**, and a **note**.
- **Verdicts are configurable** per dataset (default: `rework` / `keep` / `kill` / `unsure`).
- A **dataset** is plain JSON: `{ name, verdicts, global, elements[] }`. Load one, edit, and:
  - **Export** it as a JSON file,
  - **Save dataset** back to the server (`/datasets/<name>`),
  - **Submit** the verdicts to `/submit` (written to `submissions/` with a timestamped copy).
- State is also kept in `localStorage`, keyed per dataset name, so multiple reviews don't collide.

## Run

```bash
node annotator/annotator-server.mjs
# or
npm start
```

Open http://localhost:4567/ (override the port with `PORT=xxxx`).

- Start empty and **+ element**, or
- **Import** a dataset JSON, or
- Pick a server-saved dataset from the **load** dropdown, or
- Open `http://localhost:4567/?d=<name>` to load dataset `<name>` directly.

## Dataset format

```json
{
  "name": "my-review",
  "verdicts": [["rework","Rework"],["keep","Keep"],["kill","Kill"],["unsure","Unsure"]],
  "global": "overall notes",
  "elements": [
    { "id":"e1", "group":"Header", "name":"Logo", "shot":"logo", "verdict":"keep", "note":"", "img":"" }
  ]
}
```

- `shot` — a slug resolved to `shots/<slug>.png` (relative to the server). Provide your own files.
- `img` — a pasted image (data URL); overrides `shot` if set.
- `id` — optional; generated if absent.

## Screenshots

Drop your screenshots into `annotator/shots/` (filename without extension = the `shot` slug), or paste/load an image onto any element in the UI. `shots/` is **gitignored** — it holds your captures, not part of the tool.

### Capturing many screenshots (optional workflow)

For an existing web app, [agent-browser](https://www.npmjs.com/package/agent-browser) captures a screenshot per URL in one command. Multiple separate `agent-browser` calls in one shell command collide — run them as a single `batch`, and use **absolute paths** (batch writes relative to its daemon's cwd, not the shell's):

```bash
SHOTS=/abs/path/to/annotator/shots
mkdir -p "$SHOTS"; find "$SHOTS" -name '*.png' -delete
agent-browser batch \
  'open "http://your-app.example/page-a"'   "screenshot $SHOTS/page-a.png" \
  'open "http://your-app.example/page-b"'   "screenshot $SHOTS/page-b.png" \
  ...
```

Then reference those slugs (`page-a`, `page-b`, …) in your dataset.

## Layout

```
annotator/
  annotator.html           # the tool (generic, no app data)
  annotator-server.mjs     # zero-dep Node server
  shots/                   # your screenshots (gitignored)
  datasets/                # your datasets (gitignored, except example.json)
    example.json           # a tiny demo dataset
  submissions/             # POST /submit output (gitignored)
```
