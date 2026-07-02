# AGENTS.md

## Cursor Cloud specific instructions

`devbox-cli` is a single Go module (`module devbox-cli`, `go 1.26.2`) that builds a one-shot
CLI binary named `devbox` for managing remote AWS dev boxes. It is not a long-running server —
each command performs an action and exits. Standard build/test/run commands are in `readme.md`
and `instructions.md`.

### Go toolchain gotcha (important)
- The VM's default `go` on `PATH` is older (1.22.x). The repo's `go.mod` pins `go 1.26.2`, so
  with `GOTOOLCHAIN=auto` (the default) `go build` / `go test` inside `/workspace` transparently
  download and use Go 1.26.2. No action needed for normal build/test.
- Dev tools (`golangci-lint`, `govulncheck`) must themselves be **built with Go 1.26.2**, otherwise
  they refuse to analyze this module (e.g. `the Go language version (go1.25) used to build
  golangci-lint is lower than the targeted Go version`). The update script installs them with
  `GOTOOLCHAIN=go1.26.2 go install ...`. Once installed, run them directly (no env var needed).
- Installed dev tools live in `$(go env GOPATH)/bin` (`~/go/bin`). This dir is added to `PATH`
  via `~/.bashrc`; if a tool is "command not found", invoke it as `~/go/bin/<tool>`.

### Common commands
- Build: `go build -o devbox .` (NOTE: a prebuilt `devbox` binary is committed at the repo root;
  building with `-o devbox` overwrites it — `git restore devbox` afterward, or build to another
  name, to avoid committing a rebuilt binary).
- Test: `go test ./...` (currently no test files).
- Lint: `golangci-lint run ./...` (CI pins `v2.12.2`).
- Security scan: `govulncheck ./...` (CI uses `v1.3.0`). Expect it to report Go **standard library**
  vulnerabilities that are only fixed in a newer Go patch (e.g. go1.26.3); these come from the
  toolchain, not this repo's code.

### Running locally without AWS
- The CLI has two modes (`devbox mode`): `local` (bring-your-own AWS key/secret) and `cloud`
  (managed, requires login to a backend server). Default/unset mode resolves to `local`.
- Local state lives in SQLite at `~/.devbox/devbox.db`; config/credentials in
  `~/.devbox/config.json` (mode 0600).
- Template management is fully local and needs no AWS: `devbox template new <name> "<startup cmd>"`,
  `devbox templates`, `devbox template delete <id>`. `devbox ls` lists local boxes.
- Actually creating/starting/ssh-ing into a box requires real AWS credentials (it provisions EC2),
  so those flows can't be exercised end-to-end without `awsAccessKey`/`awsSecret` configured via
  `devbox setup`.
- `-test` prefix (e.g. `devbox -test create demo`) runs a command's handler without real API/AWS
  calls — useful for smoke-testing wiring.
- Many interactive commands prompt on a TTY (mode picker, SSH-key generation). Pipe input to run
  non-interactively, e.g. `echo local | devbox mode`.
