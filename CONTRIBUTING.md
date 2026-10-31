# Contributing

v0.1 is intentionally small. A new agent is either detection-only or a quota adapter with a verified source.

## Detection only

Add one `Spec` in `internal/detect/catalog.go`: id, command name, binaries, and config directories. `internal/registry` wraps every spec that has no quota adapter. `status` then reports `confidence: "unknown"` and a null `remaining_pct`.

Cover the new id in `TestCatalogCoversDesignIDs` and `TestAllMatchesCatalogOnce` by extending the expected list only if you also change the order there. Those tests currently pin the v0.1 order.

## Quota adapter

Add an adapter only when you can point at a local file format or the agent's own command in upstream docs or source. Do not guess a JSON shape.

1. Create `internal/adapters/<id>/` with `Parse` covered by tests before the production parser, using fixtures under `testdata/`.
2. Add `testdata/README.md` that says where the fixture came from and whether it is synthetic.
3. Implement `agent.Adapter`. File readers must not open credential files. Command probes go through `agent.ProbeCommand` so stdin stays closed and the timeout is enforced.
4. On any unexpected payload, return `confidence: "unknown"` and a null remaining percent. Do not invent a number from token counts.
5. Register the adapter in the `quota` map in `internal/registry/registry.go`.
6. Keep notes free of emails, raw stdout, and anything that looks like a token. Add a test that a fake secret in the fixture does not appear in the marshaled agent.
7. Update the agent table in `README.md` and the verification note in `docs/design.md`.

`next` will start considering the agent only if its id is in the user's chain. The default chain is in `internal/config`.

## Checks

```sh
gofmt -l .
go test ./...
go vet ./...
staticcheck ./...
go test -race ./...
```

CI currently runs `staticcheck` v0.6.1 with Go 1.24. Use that version locally
when possible so local results match the GitHub Actions check.

## Commits

Use Conventional Commits. One behavior change per commit. Do not add a release workflow or a tag in a feature PR.
