# Codex fixtures

`rollout.jsonl` is synthetic. The `token_count` / `rate_limits` object matches the fields in
`codex-rs/codex-api/src/rate_limits.rs` (`used_percent`, `window_minutes`, `resets_at`, `plan_type`)
and the persisted event shape shown in openai/codex#35746. It is not a capture from a signed-in Codex CLI.

The sibling message line carries fake secret strings so tests can prove the parser does not copy transcript text.

`legacy.jsonl` adds `resets_in_seconds`, which current `rate_limits.rs` does not write. Older public write-ups mention it. The fixture is synthetic and marked as such.
