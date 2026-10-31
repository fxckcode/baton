# Baton

CLI en Go, de una sola ejecución, para que un agente de IA vea cuánta cuota le queda a cada CLI de programación instalada y elija a cuál pasar el trabajo.

La documentación completa está en inglés en [README.md](README.md). El diseño y las fuentes verificadas están en [docs/design.md](docs/design.md).

```sh
go build -o baton ./cmd/baton
./baton list
./baton status --json
./baton next --json
./baton --skill
```

`next` sale con código 0 si hay un agente por encima del umbral y con código 3 si ninguno llega. Sin archivo de configuración, la cadena es Antigravity, Cursor, Claude, Codex, Hermes, Amp y Kiro, con umbral del 15 %. Cursor se detecta, pero v0.1 no lee su cuota.

Baton no abre la red por su cuenta, no lee tokens de otras herramientas y no inventa un porcentaje: si no lo sabe, `confidence` es `unknown` y `remaining_pct` es `null`.

Licencia MIT.
