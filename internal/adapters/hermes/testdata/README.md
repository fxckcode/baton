# Hermes fixtures

`usage.json` is the `--json` document from the Hermes CLI reference
(https://hermes-agent.nousresearch.com/docs/reference/cli-commands), which matches
`usage_snapshot_document` in `hermes_cli/subcommands/usage.py`. It is the documented
example, not a capture from a logged-in Hermes install.

`unavailable.json` and `null-window.json` are synthetic variants of that schema:
one sets `unavailable_reason`, the other leaves `used_percent` null.
