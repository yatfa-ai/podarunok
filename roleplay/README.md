# roleplay

Launch `claude` with a **role persona** — a markdown file that becomes the
session's identity, injected via `--append-system-prompt`. Like giving Claude Code
a job title and operating manual before it starts.

Personas are plain markdown in `~/.config/roleplay/roles/<role>.md`. Memory is
**Claude Code's native** machinery (`CLAUDE.md`, `~/.claude`) — roleplay
deliberately does not run its own memory store.

## Install

```sh
git clone <this repo> && cd roleplay
go build -o roleplay .
ln -sf "$PWD/roleplay" ~/.local/bin/roleplay   # or anywhere on your PATH
```

## Use

```sh
roleplay devops                       # run the "devops" persona, in the cwd
roleplay router ~/code/my-gateway     # cd into a workdir first
roleplay yatfa --model opus           # extra flags pass through to claude
roleplay list                         # list personas
roleplay new oncall                   # scaffold a new one
```

## Define a persona

Drop `~/.config/roleplay/roles/<name>.md`. Shape:

```markdown
# Role: <name>

One-line identity.

## Instincts
- how you approach the work

## Domain (what you know going in)
- facts about the environment

## Tools
- what you reach for

## Hard rules
- ...

## What you never do
- ...
```

The whole file is appended to claude's system prompt, so keep it high-signal
(opinionated and short beats exhaustive). Each persona is self-contained — roleplay
doesn't merge a shared context file.

## `--dangerously-skip-permissions`: per-host + overridable

Never hardcoded. Resolved as **flag > env > host > default**:

- `~/.config/roleplay/config.json`:
  ```json
  { "skip_permissions": false, "skip_hosts": ["my-laptop", "dev-box"] }
  ```
- override per invocation: `--skip-permissions` / `--no-skip-permissions`
- or env: `ROLEPLAY_SKIP=1` (or `0`)

On a host listed in `skip_hosts` (or when `skip_permissions: true`), the flag is
added automatically; elsewhere it's off unless overridden.

## Env knobs

- `ROLEPLAY_CONFIG=<dir>` — config/roles root (default `~/.config/roleplay`).
- `ROLEPLAY_DRY=1` — print the `claude` command instead of running it (for verifying
  personas / skip resolution).

## Why no memory store?

Claude Code already has memory: `CLAUDE.md` (project) and `~/.claude/.../memory/`
(user). roleplay injects the *persona* (system prompt) and lets claude's native
memory handle persistence. If you ever want role-scoped memory, point claude at a
role directory with `--add-dir` inside the persona's instructions — no need for
roleplay to know about it.
