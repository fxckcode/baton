# Security

Baton v0.1 is read-only. It does not phone home, it does not refresh another tool's tokens, and it does not read OAuth files, browser cookies, or the macOS Keychain.

## Reporting

Open a private security advisory on GitHub for `fxckcode/baton` if you believe a build prints credentials, writes another tool's auth state, or can be driven into executing something other than the documented quota probes. Please include the command you ran and a redacted snippet of the output. Do not attach raw credential files.

## What a probe is allowed to do

`status`, `next`, and `doctor` may execute these commands when the binary is on `PATH`:

- `agy -p "/usage" --output-format json`
- `hermes usage --json`
- `amp usage`
- `kiro-cli chat --no-interactive "/usage"`

Those programs may use the network themselves. Baton does not. The Kiro command is a chat invocation and may create a session file. Baton does not delete it.

Claude quota is read from a status-line snapshot the user opted into. Codex quota is read from rollout JSONL. `auth.json` and `secrets.json` are not opened.

## Out of scope for v0.1

Calling provider APIs with another tool's access token is intentionally not implemented. Do not send patches that add those calls without an explicit opt-in and a review of what is printed.
