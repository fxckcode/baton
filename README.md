# Baton

Baton is a single-shot terminal CLI that tells an AI coding agent how much quota each installed coding-agent CLI has left, and which one to hand work to when the current one runs out.

The binary is `baton`. The module path is `github.com/fxckcode/baton`. v0.1 is installed from source. There is no daemon, no telemetry, and no network call made by Baton itself.

CodexBar, tokscale, ccusage, and `hermes usage` already report usage. Baton does not reimplement them. It adds a stable `baton/v1` JSON document with a confidence level, a priority chain (`next`), a skill file for agents, and the same commands on Linux, macOS, and Windows.

## Install

Go 1.22 or newer:

```sh
git clone https://github.com/fxckcode/baton.git
cd baton
go build -o baton ./cmd/baton
./baton version
```

`go install github.com/fxckcode/baton/cmd/baton@latest` works only after a module version is published. v0.1 does not publish one.

## Commands

```sh
baton list
baton list --json
baton status
baton status --json --agent codex
baton next
baton next --json --threshold 20
baton doctor
baton skill
baton --skill
baton version
```

`list` only detects binaries and config directories. `status` and `doctor` also read quota. `next` probes the configured chain and prints the first eligible agent:

```text
agent: codex
command: codex
reason: first in chain with remaining_pct >= 15
```

Exit codes: `0` success, `1` runtime failure (including a broken config file), `2` usage error, `3` `next` found no agent at or above the threshold. `next --json` still prints the report on exit 3, without a `recommendation`.

`baton --skill` prints instructions an agent can follow. It tells the agent to run `baton next --json`, to stop on exit 3, and not to read other tools' credentials.

## Config

TOML, optional. The default path is `os.UserConfigDir()/baton/config.toml` (`%AppData%\baton\config.toml` on Windows, `~/Library/Application Support/baton/config.toml` on macOS, `$XDG_CONFIG_HOME/baton/config.toml` or `~/.config/baton/config.toml` on Linux). `BATON_CONFIG` overrides it. A missing file uses the defaults. An explicit path that is missing or invalid is an error.

```toml
chain = ["antigravity", "cursor", "claude", "codex", "hermes", "amp", "kiro"]
threshold = 15.0
max_age = "6h"
allow_estimated = false
allow_stale = true
```

`next` walks `chain` in order. The first installed agent with `confidence: "exact"`, a non-null `remaining_pct`, and `remaining_pct >= threshold` wins. A later agent with more quota left does not jump the queue. `unknown` is never eligible. Estimated rows are skipped unless `allow_estimated` is true. Stale rows are eligible by default, and the JSON marks them `stale: true`.

Cursor is in the default chain because the handover example is Antigravity, then Cursor, then Claude. v0.1 cannot read Cursor's quota, so `next` skips it.

`--threshold` overrides `threshold` for one run.

## JSON

`list --json`, `status --json`, and `next --json` print schema `baton/v1`. The fields and the rule for the minimum window are in [docs/design.md](docs/design.md). A percent is present only when a parser read one. Otherwise `remaining_pct` is `null` and `confidence` is `"unknown"`.

Optional fields `binary` and `config_path` are set when detection found them. They are omitted when empty.

## Agents

| Agent | Detected by | Quota in v0.1 | Verification |
|---|---|---|---|
| Claude Code (`claude`) | `claude`, `~/.claude` or `$CLAUDE_CONFIG_DIR` | Status-line snapshot in the baton cache, if you store it | Fields: [status line docs](https://code.claude.com/docs/en/statusline). Claude does not write this file itself |
| Codex (`codex`) | `codex`, `~/.codex` or `$CODEX_HOME` | Latest `token_count` rate limit in `sessions/**/rollout-*.jsonl` | [rate_limits.rs](https://github.com/openai/codex/blob/main/codex-rs/codex-api/src/rate_limits.rs), [openai/codex#35746](https://github.com/openai/codex/issues/35746) |
| Antigravity (`agy`) | `agy`, `~/.gemini/antigravity-cli` | `agy -p "/usage" --output-format json` | Command and "does not spend quota": [release 1.1.11](https://github.com/google-antigravity/antigravity-cli/releases/tag/1.1.11). JSON field names are inferred from third-party parsers, not from a live capture |
| Hermes (`hermes`) | `hermes`, `~/.hermes` or `$HERMES_HOME` | `hermes usage --json` | [CLI reference](https://hermes-agent.nousresearch.com/docs/reference/cli-commands) |
| Amp (`amp`) | `amp`, user data dir `amp` | `amp usage` text | Command: [Amp pricing](https://ampcode.com/docs/pricing). Line shapes: [CodexBar#2435](https://github.com/steipete/CodexBar/issues/2435), [CodexBar#3050](https://github.com/steipete/CodexBar/issues/3050) |
| Kiro (`kiro-cli`) | `kiro-cli`, `~/.kiro` | `kiro-cli chat --no-interactive "/usage"` | Command: [Kiro CLI](https://kiro.dev/docs/reference/cli-commands/) and [slash commands](https://kiro.dev/docs/reference/slash-commands/). Text shape: [amazon-q-developer-cli#3867](https://github.com/aws/amazon-q-developer-cli/issues/3867). The chat command may create a session file |
| Cursor, Gemini CLI, Copilot CLI, OpenCode, Pi, Droid, Kimi, Qwen, Crush, Goose, Aider, Cline, Warp, Grok, Kilo, Zed, Auggie, Devin, Codebuff, Junie, OpenClaw | Binary on `PATH` and/or a conventional config path | Not read. `confidence` is `unknown` | Detection only. The known quota sources are interactive UI or non-public APIs |

`remaining_pct` for an agent is the minimum across its windows. Dollar credit balances (Amp) are not converted into a percent.

### Claude snapshot

Baton reads `claude-statusline.json` from `$BATON_CACHE_DIR` or the OS cache directory `baton/`. Claude Code will create that file if its status-line command tees the JSON it already passes on stdin. Baton does not launch Claude to fetch quota.

## Security

- Baton opens no sockets. The four commands above may, because they are those CLIs.
- Credential files are not read. Tokens are not refreshed and not logged.
- Parsers copy named fields only. A bad payload becomes `unknown`.
- There is no telemetry and no auto-update.

See [SECURITY.md](SECURITY.md).

## Comparison

| Project | What it does | What Baton does not copy |
|---|---|---|
| [CodexBar](https://github.com/steipete/CodexBar) | Quota for a large set of providers, plus a JSON CLI | Browser cookies, Keychain, internal APIs, per-provider reimplementation |
| [tokscale](https://github.com/junhoyeo/tokscale) | Quota and token totals | Its provider list |
| [ccusage](https://github.com/ccusage/ccusage) | Local token and cost history | Treating historical usage as remaining quota |
| `hermes usage` | One configured provider | Baton only runs the command. It does not reimplement the fetch |

## Not verified on a logged-in account

The parsers are tested against fixtures. Those fixtures are the official examples, sanitized public reports, or synthetic JSON shaped like third-party parsers. This repository was not built against a signed-in Claude, Codex, Antigravity, Hermes, Amp, or Kiro account. See each `testdata/README.md` and [docs/design.md](docs/design.md).

macOS and Windows are covered by the GitHub Actions matrix. Development here ran on Linux.

## Local smoke verification

The repository's deterministic checks passed locally on October 31, 2026:

```sh
go test ./...
go vet ./...
go test -race ./...
```

The installed CLI smoke check also verified version reporting for Claude Code,
Codex, Hermes, Antigravity, Kiro, Cursor Agent, Gemini CLI, GitHub Copilot CLI,
OpenCode, Pi, and Kilo. Live quota probes succeeded for `hermes usage --json`
and `agy -p "/usage" --output-format json`. The Kiro probe reached its own
browser-authentication path and exited non-zero because this environment was not
authenticated; Baton correctly reported Kiro as `unknown` rather than inventing
a percentage. Version detection and quota parsing are separate guarantees: a
CLI being installed does not mean Baton has a verified quota source for it.

To reproduce the Baton-level smoke check after building:

```sh
go build -o baton ./cmd/baton
./baton list --json
./baton status --json
./baton next --json
```

## Layout

`cmd/baton` is the entrypoint. `internal/cli` parses flags. `internal/next` is the pure chain picker. `internal/schema` is `baton/v1`. Adapters live in `internal/adapters`. The registry is `internal/registry` so adapters can import the `agent` interface without a cycle.

## License

[MIT](LICENSE). A short Spanish overview is in [README_es.md](README_es.md).
