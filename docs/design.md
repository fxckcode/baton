# Baton v0.1 design

Baton is a single-shot Go CLI (`baton`, module `github.com/fxckcode/baton`) that tells an AI coding agent how much quota each installed coding-agent CLI has left, and which one to hand work to when the current one is spent. It does not run a daemon, does not phone home, and does not call provider APIs itself.

This document records decisions made before implementation. Source verification was done on 31 Oct 2026 by reading the pages and files linked below. Nothing here was checked against a logged-in account on this machine.

## Decisions

1. **Language: Go, stdlib first.** The work is file parsing, process execution, and JSON. One third-party library is allowed: [`github.com/BurntSushi/toml`](https://github.com/BurntSushi/toml) to load the config file. Hand-rolling TOML would mis-handle escapes and arrays. No CLI framework, no logging library, no HTTP client in v0.1.
2. **Offline by default means Baton opens no sockets.** Adapters that shell out to an agent’s own usage command (`agy`, `hermes`, `amp`, `kiro-cli`) may cause *that* program to use the network. Baton does not read OAuth tokens, refresh them, or call `api.anthropic.com`, `chatgpt.com`, `copilot_internal`, `cloudcode-pa`, or similar. Those stay future opt-in and are not implemented.
3. **Never invent a percent.** If a parser cannot find a provider-reported number in a shape documented below, the agent is `confidence: "unknown"` and `remaining_pct` is JSON `null`.
4. **`remaining_pct` is the minimum across that agent’s windows.** The tightest window is the one that blocks the agent. Antigravity model families are separate pools; v0.1 still applies the minimum and lists every window so the caller can see which pool is tight. Doing otherwise would special-case one agent inside the stable schema.
5. **`next` walks a priority chain. It does not pick the largest remaining percent.** The first chain entry that is eligible wins. Ties are chain order.
6. **Eligibility.** Installed, `confidence` is `exact` (or `estimated` only when `allow_estimated` is true), `remaining_pct` is non-null and `>= threshold`, and not stale unless `allow_stale` is true. `unknown` is never eligible. Default threshold is 15. Default `allow_estimated` is false. Default `allow_stale` is true, because local snapshots are often hours old and are still the only exact number we have; `stale: true` stays visible in the output.
7. **Stale.** `stale` is true when `observed_at` is set and `now - observed_at > max_age`. Default `max_age` is `6h`. A fresh CLI probe sets `observed_at` to the probe time, so it is not stale. Missing data is not stale; it is unknown.
8. **Read-only.** Baton never writes config into another tool, never refreshes tokens, and never deletes session files. The Kiro probe is the exception that is documented under that adapter: the upstream command is a chat invocation and may create a session file of its own. Baton does not clean that up.
9. **Secrets.** Parsers copy only the fields named in this document. Emails, raw stdout, `Authorization` values, and token-shaped strings are not placed in notes, JSON, or logs. There is no log file.
10. **Command runner and clock are injected.** Tests do not need the real CLIs. There is no mutable process-global registry.
11. **Config format: TOML only**, at `os.UserConfigDir()/baton/config.toml` (`%AppData%\baton` on Windows, `~/Library/Application Support/baton` on macOS, `$XDG_CONFIG_HOME/baton` or `~/.config/baton` on Linux). `BATON_CONFIG` overrides the path. A missing file uses defaults. A present but invalid file is a hard error (exit 1), not a silent fallback.
12. **Schema is `baton/v1`.** Adding fields later is allowed. Renaming or retyping the fields in this document is not.

## Architecture

```
cmd/baton/main.go                 parse os.Args, call cli.Run, exit
internal/cli                      flag parsing, human and JSON output, exit codes
internal/schema                   baton/v1 types, min remaining, stale, JSON
internal/next                     pure chain selection
internal/config                   TOML load and defaults
internal/agent                    Adapter interface, Env, catalog metadata
internal/detect                   PATH lookup, config-dir existence, --version
internal/run                      Runner implementation (timeouts, no stdin)
internal/adapters/claude          statusline snapshot file
internal/adapters/codex           rollout JSONL
internal/adapters/antigravity     agy -p "/usage" --output-format json
internal/adapters/hermes          hermes usage --json
internal/adapters/amp             amp usage text
internal/adapters/kiro            kiro-cli chat --no-interactive "/usage"
internal/adapters/catalog         detection-only agents
internal/skill                    go:embed SKILL.md
```

`Adapter`:

```go
type Adapter interface {
    ID() string
    Command() string
    Detect(ctx context.Context, env Env) Detection
    Status(ctx context.Context, env Env, det Detection) schema.Agent
}
```

`Status` does not return an error for a bad payload or a failed probe. It returns `confidence: unknown` and a fixed note. `Detect` does not fail the process when a binary is missing.

`list` only detects. `status` detects and probes every catalog agent. `next` detects the chain and probes only chain entries that have a quota adapter. `doctor` reports which source path or command was available, without payloads.

Exit codes: `0` success, `1` runtime failure (bad config, internal error), `2` usage error, `3` `next` found no eligible agent. `next --json` still prints the report on exit 3.

## JSON schema `baton/v1`

```json
{
  "schema": "baton/v1",
  "generated_at": "2026-10-31T00:00:00Z",
  "agents": [
    {
      "agent": "codex",
      "installed": true,
      "version": "0.80.0",
      "plan": "plus",
      "windows": [
        {"name": "5h", "remaining_pct": 38.0, "reset_at": "2026-10-31T01:12:00Z"}
      ],
      "remaining_pct": 38.0,
      "confidence": "exact",
      "source": "local:codex-rollout",
      "observed_at": "2026-10-30T22:41:10Z",
      "stale": false,
      "note": ""
    }
  ],
  "recommendation": {
    "agent": "codex",
    "command": "codex",
    "reason": "first in chain with remaining_pct >= 15"
  }
}
```

| Field | Rule |
|---|---|
| `schema` | Always `baton/v1`. |
| `plan`, `remaining_pct`, `observed_at`, window `remaining_pct`, window `reset_at` | JSON `null` when unknown. |
| `windows` | Empty array when there are none. |
| `confidence` | `exact`, `estimated`, or `unknown`. v0.1 emits `estimated` only if a future adapter sets it. No v0.1 adapter estimates a percent from token counts. |
| `source` | Short stable tag, see adapters. |
| `note` | Present when it helps. Omitted when empty. Never a secret and never raw command output. |
| `recommendation` | Present only on `next` when an agent was selected. |

`version` is omitted when detection could not read one. Percents are rounded to two decimal places.

## Config defaults

```toml
chain = ["antigravity", "cursor", "claude", "codex", "hermes", "amp", "kiro"]
threshold = 15.0
max_age = "6h"
allow_estimated = false
allow_stale = true
```

`--threshold` overrides `threshold` for that invocation. Unknown keys are ignored so a newer config still loads.

Cursor is in the default chain on purpose. v0.1 cannot read its quota, so `next` skips it and continues. That matches the handover example (Antigravity, then Cursor, then Claude) without pretending Cursor has a number.

## Agents

Quota adapters are implemented only where a local file or the agent’s own command has a documented shape. Everyone else is detected and returned as `unknown`.

Detection is `LookPath` of the binary **or** an existing config directory. Version is `<bin> --version` with a 2 second timeout when a binary exists. Version text longer than one short line is truncated. A version failure does not clear `installed`.

| ID | Binary | Config signal | Quota in v0.1 |
|---|---|---|---|
| `claude` | `claude` | `$CLAUDE_CONFIG_DIR` or `~/.claude` | yes, local snapshot |
| `codex` | `codex` | `$CODEX_HOME` or `~/.codex` | yes, local rollout |
| `antigravity` | `agy` | `~/.gemini/antigravity-cli` | yes, `agy` command |
| `hermes` | `hermes` | `$HERMES_HOME` or `~/.hermes` | yes, `hermes` command |
| `amp` | `amp` | `~/.local/share/amp` (and the OS user-share equivalent) | yes, `amp` command |
| `kiro` | `kiro-cli` | `~/.kiro` | yes, `kiro-cli` command |
| `cursor` | `agent`, also `cursor-agent` | `~/.cursor` | unknown |
| `gemini` | `gemini` | `$GEMINI_CLI_HOME` or `~/.gemini` | unknown |
| `copilot` | `copilot` | `~/.copilot` | unknown |
| `opencode` | `opencode` | `~/.local/share/opencode` or OS user-share `opencode` | unknown |
| `pi` | `pi` | `~/.pi/agent` | unknown |
| `droid` | `droid` | `~/.factory` | unknown |
| `kimi` | `kimi` | `~/.kimi` | unknown |
| `qwen` | `qwen` | `~/.qwen` | unknown |
| `crush` | `crush` | `~/.local/share/crush` or OS user-share `crush` | unknown |
| `goose` | `goose` | `~/.config/goose` | unknown |
| `aider` | `aider` | `~/.aider.conf.yml` | unknown |
| `cline` | `cline` | `~/.cline` | unknown |
| `warp` | `warp`, `oz` | none required | unknown |
| `grok` | `grok` | `~/.grok` | unknown |
| `kilo` | `kilo` | `~/.local/share/kilo` or OS user-share `kilo` | unknown |
| `zed` | `zed` | none required | unknown |
| `auggie` | `auggie` | none required | unknown |
| `devin` | `devin` | none required | unknown |
| `codebuff` | `codebuff` | none required | unknown |
| `junie` | `junie` | none required | unknown |
| `openclaw` | `openclaw` | none required | unknown |

The unknown note for those agents is a fixed sentence: quota is not read in v0.1 because the only known sources are interactive UI or non-public APIs. Paths above are install conventions used for detection, not quota sources.

### Claude Code — verified command shape, local cache is ours

**Verification: (a) field names.** [Claude Code status line docs](https://code.claude.com/docs/en/statusline) document `rate_limits.five_hour.used_percentage`, `rate_limits.five_hour.resets_at`, and the same pair for `seven_day` and `spend_limit`. `used_percentage` is 0–100 consumed. `resets_at` is Unix seconds. Each window may be absent. The object exists for claude.ai Pro, Max, and Ultra, or a Claude apps gateway spend limit, and only after the first API response. The docs’ example JSON is the fixture source.

**Not on disk by itself.** Claude passes that JSON to the status-line process on stdin. It does not write a quota file we can read. v0.1 therefore reads a snapshot at `$BATON_CACHE_DIR/claude-statusline.json` or `os.UserCacheDir()/baton/claude-statusline.json` **only if something else has written it**. `doctor` and `SKILL.md` tell the user to point Claude Code’s `statusLine.command` at a `tee` of that path. Baton does not register the hook and does not start Claude.

- Source tag: `local:claude-statusline`.
- `observed_at`: file modification time.
- `plan`: null. The snapshot does not include a plan name.
- Windows: `5h`, `7d`, and `spend` when those objects exist. Remaining = `100 - used_percentage`.
- Missing file, invalid JSON, or no usable window: `unknown`, `remaining_pct` null. Note says the snapshot is missing or had no rate-limit windows.
- Context-window `used_percentage` is **not** quota and is ignored.

### Codex — verified local rollout

**Verification: (a).** [`codex-rs/codex-api/src/rate_limits.rs`](https://github.com/openai/codex/blob/main/codex-rs/codex-api/src/rate_limits.rs) parses server headers into `used_percent`, `window_minutes`, and `resets_at`. [openai/codex#35746](https://github.com/openai/codex/issues/35746) shows the persisted `TokenCount` event carrying `RateLimitSnapshot` with those fields. Public descriptions of `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl` agree on lines shaped as `type: event_msg`, `payload.type: token_count`, `payload.rate_limits.{primary,secondary,plan_type}`.

Reader:

- Directory: `$CODEX_HOME/sessions` or `~/.codex/sessions`.
- Consider the newest `rollout-*.jsonl` files, up to 8, by modification time. Skip `*.jsonl.zst` (not decompressed in v0.1).
- Within a file, the last `token_count` event that contains a primary or secondary window wins. Files larger than 4 MiB contribute only their last 1 MiB.
- `used_percent` is consumed. Remaining = `100 - used_percent`.
- Window name: `5h` when `window_minutes` is 300, `weekly` when 10080, otherwise `{n}m` or `primary` / `secondary` if minutes are absent.
- `resets_at` Unix seconds becomes `reset_at`. If `resets_at` is absent and `resets_in_seconds` is present, `reset_at = event timestamp + resets_in_seconds`. That legacy field is **(b)**: current `rate_limits.rs` writes `resets_at`; older write-ups mention `resets_in_seconds`. Absence of the legacy field is fine.
- `plan` from `plan_type` when it is a string.
- `observed_at` from the event `timestamp`.
- Source tag: `local:codex-rollout`.
- Lines that are not JSON or not this event are skipped. A transcript line that merely contains the words `rate_limits` does not become a quota.

Baton does not read `auth.json`.

### Antigravity — command verified, JSON shape inferred

**Verification: (a) command and cost.** [Release 1.1.11](https://github.com/google-antigravity/antigravity-cli/releases/tag/1.1.11) says `-p "/usage"`, `/quota`, and `/credits` emit a structured payload under `--output-format json` and do not start an agent turn, spend quota, or leave a conversation.

**Verification: (b) JSON fields.** The release notes do not include a sample object. Independent parsers describe the same envelope: `status`, `command.name`, `command.data.groups[].buckets[]` with `remaining_fraction` (0–1) or `remainingFraction`, and `reset_time` / `resetTime`. See [usage-monitor-cli `antigravity.rs`](https://docs.rs/usage-monitor-cli/latest/src/usage_monitor_cli/provider/antigravity.rs.html) and [pi-antigravity `usage.ts`](https://cdn.jsdelivr.net/npm/@tian.zuo/pi-antigravity@0.11.3/lib/usage.ts). The fixture is synthetic and labeled as such. A live capture is still required.

Probe: `agy -p "/usage" --output-format json`, timeout 15s, stdin closed. No PTY.

- Accept `status` `SUCCESS`, `OK`, or empty. Any other status is unknown.
- Accept `command.name` `/usage`, `usage`, or empty. Any other name is unknown.
- A bucket counts only when a remaining fraction is in `[0, 1]`. Values above 1 are not treated as percents.
- `remaining_pct = fraction * 100`.
- Window name: `"{group} {window}"` using `window`, then bucket `name`, then `id`.
- `reset_time` must be RFC3339. Anything else leaves `reset_at` null.
- Disabled buckets are skipped.
- Source tag: `cli:agy-usage`.
- `observed_at` is the probe time.
- Non-JSON stdout, timeout, or non-zero exit: unknown. The note is `command exited N`, `command timed out`, or `output was not a usage payload`. Stdout and stderr are not copied.

### Hermes — verified JSON

**Verification: (a).** [CLI reference](https://hermes-agent.nousresearch.com/docs/reference/cli-commands) and [`hermes_cli/subcommands/usage.py`](https://github.com/NousResearch/hermes-agent/blob/main/hermes_cli/subcommands/usage.py) define `hermes usage --json`. Keys: `provider`, `source`, `plan`, `fetched_at`, `windows[].{label,used_percent,resets_at,detail}`, `unavailable_reason`. `used_percent` and `resets_at` may be null. Exit 1 means nothing was fetched; stdout is empty.

Probe: `hermes usage --json`, timeout 15s. This asks Hermes to fetch. Baton does not see the credential.

- Non-zero exit or invalid JSON: unknown, note `command exited N` or `output was not a usage payload`.
- `unavailable_reason` non-empty: unknown, note `provider reported usage unavailable`. The reason string is not copied; it is not documented as secret-free.
- A window with null `used_percent` is dropped. If none remain, unknown.
- Remaining = `100 - used_percent`. Name is `label` (`Session`, `Weekly`, …).
- `plan` from `plan`. `observed_at` from `fetched_at`.
- Source tag: `cli:hermes-usage`.
- Dollar balances stuffed into `detail` are not converted into a percent.

### Amp — command verified, text shape from public sanitized captures

**Verification: (a) command.** [Amp pricing](https://ampcode.com/docs/pricing) says the balance is available by running `amp usage`. There is no `--json` flag ([CodexBar#2435](https://github.com/steipete/CodexBar/issues/2435)).

**Verification: (b) text.** The same issue quotes a sanitized subscription line. [CodexBar#3050](https://github.com/steipete/CodexBar/issues/3050) quotes a later line and a credits line. Both are the fixtures, with the example address already used upstream. They were not captured on this machine.

Supported lines:

- `Subscription <plan>: <n>% other usage and <n>% orb usage remaining - resets upon renewal in <d> days`
- `Amp <plan> Subscription: <n>% other usage and <n>% orb usage remaining - resets upon renewal in <d> days|month|months`

`other` and `orb` are remaining percents, not consumed percents. A day count sets `reset_at` to probe time plus that many 24 hour periods. `month` / `months` does **not** invent a timestamp. Dollar lines (`Individual credits: $4.35 remaining`, `Workspace …: $… remaining`, `Amp Free: $… / $…`) do not become percents. `Amp Free: <n>% remaining` is a remaining percent named `free`.

- If at least one percent window parsed: `exact`, source `cli:amp-usage`, `observed_at` = probe time.
- If only dollar lines parsed: unknown, note `Amp reported a credit balance, not a percentage`.
- The signed-in email line is ignored and must not appear in output.

### Kiro — command verified, text shape from public reports

**Verification: (a) invocation.** [Kiro CLI commands](https://kiro.dev/docs/reference/cli-commands/) document `kiro-cli chat --no-interactive` and a positional prompt. [Slash commands](https://kiro.dev/docs/reference/slash-commands/) say `/usage` shows billing and credits, and that non-interactive runs print the full breakdown.

**Verification: (b) text, and a side effect.** [aws/amazon-q-developer-cli#3867](https://github.com/aws/amazon-q-developer-cli/issues/3867) shows:

```
Estimated Usage | resets on 2026-08-01 | KIRO POWER
Credits (15312.32 of 10000 covered in plan)
... 153.1%
```

The percent there matches `used / limit`. Remaining = `(limit - used) / limit * 100`, which may be negative when the account is in overage. `resets on YYYY-MM-DD` is `reset_at` at UTC midnight. `resets on MM/DD` uses the next occurrence of that month and day in UTC. A plan token matching `KIRO …` after `|` is `plan`.

[CodexBar’s Kiro doc](https://github.com/steipete/CodexBar/blob/main/docs/kiro.md) describes an older box drawing (`Monthly credits`, `(0.00 of 50 covered in plan)`, `Bonus credits: X/Y credits used`). v0.1 also accepts:

- `(<used> of <limit> covered in plan)` as the plan window named `monthly`
- `Bonus credits` followed by `<used>/<limit> credits used` as window `bonus`

If the bar percent and the `used of limit` ratio disagree by more than 1 point, the ratio wins and the bar is ignored. The ratio is the explicit count. If only a bar percent is present and no counts, the result is unknown rather than guessing whether the bar is used or remaining.

ANSI sequences are stripped before matching. Source tag: `cli:kiro-usage`. Timeout 20s.

**Side effect:** this runs `chat`. CodexBar reports that Kiro may save a session file for the probe. Baton still runs it because the documented non-interactive usage path is this command, and it does not delete those files. `doctor` says so. There is no documented `kiro-cli usage --json` ([kirodotdev/Kiro#8477](https://github.com/kirodotdev/Kiro/issues/8477) is still a feature request).

### Explicitly not in v0.1

Network or credential reads for Claude OAuth usage, Codex `wham/usage`, Gemini `retrieveUserQuota`, Cursor web usage, Copilot internal API, OpenCode Go, Cline, Warp, Windsurf, Grok, Kilo, Zed, and bridges to CodexBar or tokscale. Token-count estimates in the style of ccusage are not a remaining percent: the cap is not in the local transcript.

## Security

- No telemetry, no auto-update, no browser cookies, no macOS Keychain.
- Child processes get the inherited environment (the other CLI needs `HOME`) and a closed stdin. Timeouts are enforced with `CommandContext`. On Unix the process group is killed on cancel so a hung probe does not outlive the timeout.
- `doctor` prints paths and fixed notes, not file contents.
- Tests include fixture secrets (`sk-ant-…`, `Bearer …`, an email) and assert they are absent from JSON and human output.

## Comparison

| Tool | What it already does | What v0.1 does not copy |
|---|---|---|
| [CodexBar](https://github.com/steipete/CodexBar) | Quota for a large provider set, JSON CLI, macOS app | Browser cookies, Keychain, internal APIs, per-provider reimplementation |
| [tokscale](https://github.com/junhoyeo/tokscale) | Quota and token totals | Its provider list and leaderboard |
| [ccusage](https://github.com/ccusage/ccusage) | Local token and cost accounting | Historical usage as if it were remaining quota |
| `hermes usage` | One configured provider | Baton only shells out; it does not reimplement Hermes’s fetchers |

Baton’s job is the handover: `next`, a stable `confidence`, a skill file, and the same binary on Linux, macOS, and Windows.

## Implementation notes

The adapter registry lives in `internal/registry`, not `internal/agent`. Adapters import `internal/agent` for the interface. Putting the constructor in `internal/agent` would import the adapters and cycle.

`baton/v1` also has optional `binary` and `config_path`. They are omitted when empty. They are detection results, not quota. `list --json` uses the same document with `source: "detect"` and a null remaining percent so callers do not learn a second schema.
