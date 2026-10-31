# Baton v0.1 implementation plan

Each task is one conventional commit. Production code for a task lands only after a test for that task has been run and has failed for the missing behavior. `gofmt` is applied before each commit. `go test ./...` is run before each commit that changes Go code.

Definition of done for the branch: every command in the design works, `go test -race ./...`, `go vet ./...`, and `staticcheck ./...` pass on Linux, GitHub Actions covers Linux, macOS, and Windows, `baton --skill` prints the embedded instructions, and the PR is a single draft into `main` with a “Not verified” section.

## Task 0 — design

Files: `docs/design.md`, `docs/plan.md`.

Done when both files are committed on `cursor/baton-v0.1-8fde` and no Go code is in that commit.

## Task 1 — schema

Files: `go.mod`, `internal/schema/schema.go`, `internal/schema/schema_test.go`, `internal/schema/testdata/report.golden.json`.

Steps:

1. Write tests for `baton/v1` JSON tags (`null` percents, empty windows, confidence values), minimum-of-windows, two-decimal rounding, and stale marking.
2. Run the test and confirm it fails.
3. Implement the types and helpers.
4. Add a golden file and compare `Marshal`.

Verify: `go test ./internal/schema`.

Done when a report round-trips to the golden JSON and a negative remaining percent (overage) is preserved.

## Task 2 — next

Files: `internal/next/select.go`, `internal/next/select_test.go`.

Cases: threshold boundary (`>=`), chain order beats a higher later percent, every agent below the threshold, unknown skipped, estimated skipped unless allowed, stale skipped unless allowed, not installed skipped, null remaining skipped, empty chain, agent missing from the input.

Verify: `go test ./internal/next`.

Done when those cases pass and selection does not read the clock or the filesystem.

## Task 3 — config

Files: `internal/config/config.go`, `internal/config/config_test.go`.

Cases: missing file returns defaults, `BATON_CONFIG` override, threshold and chain decode, invalid TOML errors, unknown keys ignored, `max_age` duration, bool flags.

Verify: `go test ./internal/config`.

Done when defaults match the design and a broken file returns a wrapped error.

## Task 4 — runner and detect

Files: `internal/run/run.go`, `internal/run/run_unix.go`, `internal/run/run_windows.go`, `internal/run/run_test.go`, `internal/agent/agent.go`, `internal/detect/detect.go`, `internal/detect/detect_test.go`, `internal/detect/paths.go`.

Steps:

1. Test that the real runner closes stdin, returns stdout, and times out.
2. Test detection with a fake runner and a temp home: binary on PATH, config dir only, neither, version timeout does not flip `installed` to false, Windows vs Unix config paths for one sample agent.
3. Implement.

Verify: `go test ./internal/run ./internal/detect ./internal/agent`.

Done when version probes use a 2s context and detection does not open files whose names look like credentials (`auth.json`, `secrets.json`).

## Task 5 — Codex adapter

Files: `internal/adapters/codex/parse.go`, `internal/adapters/codex/codex.go`, `internal/adapters/codex/parse_test.go`, `internal/adapters/codex/testdata/`.

Fixture: synthetic JSONL shaped like the persisted `token_count` event. One line carries `used_percent` 62 and 81, `plan_type` `plus`, and a fake API key in a sibling message line. Another file uses `resets_in_seconds`. A garbage line must not panic.

Verify: `go test ./internal/adapters/codex`.

Done when remaining is 38 and 19 (min 19), the API key is not in the result, and a directory with no rollout is unknown.

## Task 6 — Claude adapter

Files: `internal/adapters/claude/parse.go`, `internal/adapters/claude/claude.go`, `internal/adapters/claude/parse_test.go`, `internal/adapters/claude/testdata/`.

Fixture: the `rate_limits` object from the official status-line example, trimmed of unrelated fields, plus a file with no `rate_limits`.

Verify: `go test ./internal/adapters/claude`.

Done when `5h` remaining is 76.5, `observed_at` follows mtime, and a missing file is unknown with a note that names the snapshot path’s base name only in the note as a fixed phrase (the full path is returned separately for `doctor`, not embedded in free text from the file).

## Task 7 — Hermes adapter

Files: `internal/adapters/hermes/parse.go`, `internal/adapters/hermes/hermes.go`, `internal/adapters/hermes/parse_test.go`, `internal/adapters/hermes/testdata/`.

Fixture: the JSON document from the Hermes CLI reference. Extra cases: null `used_percent`, non-empty `unavailable_reason`, invalid JSON, command exit 1 via the fake runner.

Verify: `go test ./internal/adapters/hermes`.

Done when session remaining is 63 and weekly remaining is 87.5, and `unavailable_reason` text is not copied into `note`.

## Task 8 — Antigravity adapter

Files: `internal/adapters/antigravity/parse.go`, `internal/adapters/antigravity/antigravity.go`, `internal/adapters/antigravity/parse_test.go`, `internal/adapters/antigravity/testdata/`.

Fixture: synthetic envelope marked in `testdata/README.md`. Cases: fraction out of range ignored, `status` `ERROR`, camelCase `remainingFraction`, command failure.

Verify: `go test ./internal/adapters/antigravity`.

Done when the probe argv is exactly `agy -p /usage --output-format json` with the prompt as one argument `"/usage"`.

## Task 9 — Amp adapter

Files: `internal/adapters/amp/parse.go`, `internal/adapters/amp/amp.go`, `internal/adapters/amp/parse_test.go`, `internal/adapters/amp/testdata/`.

Fixtures: the two sanitized public lines (subscription word order changed). Dollar-only output stays unknown. Email must not appear.

Verify: `go test ./internal/adapters/amp`.

Done when both line shapes produce `other` and `orb` windows and `in 29 days` sets a reset while `in 1 month` does not.

## Task 10 — Kiro adapter

Files: `internal/adapters/kiro/parse.go`, `internal/adapters/kiro/kiro.go`, `internal/adapters/kiro/parse_test.go`, `internal/adapters/kiro/testdata/`.

Fixtures: the overage sample from the public issue, a bonus-credit sample based on the CodexBar doc, ANSI-wrapped text, and a bar-only sample that must stay unknown.

Verify: `go test ./internal/adapters/kiro`.

Done when argv is `kiro-cli chat --no-interactive /usage` with prompt `"/usage"`, and overage remaining is negative.

## Task 11 — catalog and wiring

Files: `internal/adapters/catalog/catalog.go`, `internal/adapters/catalog/catalog_test.go`, `internal/agent/registry.go`.

The registry returns quota adapters plus one detection-only adapter per remaining agent. Detection-only `Status` sets `unknown` and the fixed note. It does not run the binary except for the version probe done in `Detect`.

Verify: `go test ./internal/adapters/catalog ./internal/agent`.

Done when every id in the design table is present once.

## Task 12 — skill

Files: `internal/skill/SKILL.md`, `internal/skill/skill.go`, `internal/skill/skill_test.go`.

The test lists the rules the skill must contain: front matter `name: baton`, `baton next --json`, exit code 3, `confidence`, do not read credentials, default chain, threshold.

Verify: `go test ./internal/skill`.

Done when the text is embedded with `go:embed` and contains no token examples that look real.

## Task 13 — CLI

Files: `internal/cli/cli.go`, `internal/cli/cli_test.go`, `cmd/baton/main.go`.

Cases:

- `version` and `--version` print the version.
- `--skill` and `skill` print the embedded skill and nothing else.
- `list` and `list --json` show an installed temp binary and a missing agent.
- `status --json` is schema `baton/v1`.
- `status --agent nope` exits 1 with a fixed message.
- `next --json --threshold 20` selects the first eligible agent.
- `next` exits 3 when none qualify, and `--json` still prints a report without `recommendation`.
- `doctor` mentions a missing Claude snapshot and does not print fixture secrets.
- A Codex home containing `sk-ant-test-secret` and `Bearer test-token` in a rollout produces status JSON and human text that do not contain those strings.

Verify: `go test ./internal/cli ./cmd/baton`.

Done when exit codes match the design and `cmd/baton` is only a main wrapper.

## Task 14 — docs, CI, release config

Files: `README.md`, `README_es.md`, `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `LICENSE`, `.gitignore`, `.github/workflows/ci.yml`, `.goreleaser.yaml`, `.golangci.yml` only if staticcheck is not enough. Prefer invoking `staticcheck` directly so the CI config stays small.

CI: `permissions: contents: read`, matrix `ubuntu-latest`, `macos-latest`, `windows-latest`, `go test ./...`, `go vet ./...`, `staticcheck ./...`. Linux also runs `go test -race ./...` if the full matrix with `-race` is too heavy; if the matrix is fast enough, `-race` runs on all three. No release workflow and no tag.

Verify: `gofmt -l`, `go test ./...`, `go vet ./...`, `staticcheck ./...`, `goreleaser check` when the binary is installable.

Done when the README table matches the verification levels in `docs/design.md` and states that live logged-in output was not captured.

## Task 15 — self-review

Read the diff for dead code, duplicated parsers, notes that echo raw output, and comments that restate the code. Fix with a follow-up commit. Re-run `go build -o /tmp/baton ./cmd/baton`, `go test -race ./...`, `go vet ./...`, and staticcheck. Paste those commands and their results into the draft PR body, plus the not-verified list:

- No live `agy -p "/usage" --output-format json` capture.
- No live `amp usage` capture beyond the public sanitized quotes.
- No live `kiro-cli chat --no-interactive "/usage"` capture.
- No live Codex rollout from a signed-in CLI.
- No live Claude status-line payload beyond the official documentation example.
- No live `hermes usage --json` capture beyond the documented example.
- macOS and Windows not executed here; CI is configured to run them.
