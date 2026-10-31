---
name: baton
description: Check remaining quota of coding-agent CLIs and pick the next one to use.
---

# Baton

Baton reports how much session or plan quota each installed coding-agent CLI has left, and which CLI to hand work to when the current one is spent. Run it yourself. Do not read the CLIs' credential files.

## When to run

1. Run `baton next --json` before a long task and again every few turns.
2. Use `recommendation.command` as the executable. Do not assume extra flags.
3. If the process exits with code 3, no agent in the chain is at or above the threshold. Tell the human. Do not invent a fallback.
4. If `confidence` is not `exact`, treat the number as a hint. `unknown` means there is no percentage. Never fill one in.
5. If `stale` is true, the number is an old local snapshot. Say so when you act on it.

## Do not

- Do not read `auth.json`, credential files, `secrets.json`, Keychain items, or browser cookies to help Baton.
- Do not print tokens, Authorization headers, or signed-in email addresses.
- Do not refresh another tool's tokens.

## Defaults

When no config file is present, the chain is `antigravity`, `cursor`, `claude`, `codex`, `hermes`, `amp`, `kiro` and the threshold is 15 percent remaining. The first chain entry that is installed, exact, and at or above the threshold wins. Cursor is in the chain, and v0.1 cannot read its quota, so Baton skips it.

Override the threshold for one run with `baton next --threshold 20`.

The config file is TOML at the OS config directory `baton/config.toml`. `BATON_CONFIG` overrides that path.

## Commands

- `baton list` shows what is installed.
- `baton status --json` prints quota for every known agent.
- `baton status --agent codex --json` limits that report to one agent.
- `baton next --json` prints the agent to use.
- `baton doctor` reports which data sources were found.
- `baton version` prints the version.

## Claude Code snapshot

Claude does not write rate limits to a file. Baton reads them only after a status-line command stores its JSON stdin at `$BATON_CACHE_DIR/claude-statusline.json`, or in the OS cache directory under `baton/claude-statusline.json`. Point Claude Code's `statusLine.command` at a command that tees that JSON. Baton does not start Claude to fetch it.

## Exit codes

- 0 means success, including a `next` choice.
- 1 means a runtime failure, such as a broken config file.
- 2 means a usage error.
- 3 means `next` found no agent at or above the threshold.
