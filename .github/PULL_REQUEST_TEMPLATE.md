## Summary

Describe the change and the user-visible or maintainer-visible outcome.

## Verification

- [ ] `gofmt -l .` is clean
- [ ] `go test ./...`
- [ ] `go vet ./...`
- [ ] `staticcheck ./...`
- [ ] `go test -race ./...` (when applicable)
- [ ] README or design documentation updated when behavior or support changes

## Safety and privacy

- [ ] No credentials, tokens, raw account output, or personal data are included
- [ ] Quota parsing remains fail-closed: unknown data produces `remaining_pct: null`
- [ ] The change does not read another tool's credential files

## Additional context

Add fixture provenance, upstream references, compatibility notes, or screenshots when useful.
