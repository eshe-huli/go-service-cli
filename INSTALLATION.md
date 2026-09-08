# Laptop installation — gsvc 0.1.0

Installed and exercised on 2026-09-08, macOS arm64, Go 1.27.1. This is the supplied
standalone Go Service CLI, not a replacement for the NestJS CLI or older Go CLI.

- Source: `/Users/macbook/development/command/go-service-cli`
- Executable: `/Users/macbook/development/command/go-service-cli/bin/gsvc`
- PATH entry: `/Users/macbook/development/command/gsvc` (symlink to that binary).
  The existing login-shell PATH already contains `Development/command`; both
  spellings resolve here on this Mac. No shell profile was edited.
- Original archive: `/Users/macbook/Downloads/go-service-cli-v0.1.0.zip`
- Archive SHA256: `a33656300a2e015150e08f1a66ba27a8a5e8f1362a916871bb84f59b67a596e0`
- Binary SHA256: `7afc5beac95e78ac2a03dad3cced9d48d8f43286c08be281e4d638f008c3a228`

All 78 supplied manifest entries matched after extracting the 79-file archive.
No original source, templates, generated example or distribution instructions
were changed. The local Git workspace preserves an initial empty review anchor
and the supplied-source installation snapshot. Source has not been published,
and no remote repository or license was selected.

## Use it

```sh
gsvc version
gsvc contract --json
gsvc init /absolute/new-service --module example.com/new-service --dry-run --json
# Review data.plan.id, then repeat the exact init arguments with --expect <id>.
# In the new service, use add/change and implement developer-owned behavior.
gsvc check --root /absolute/new-service --verify --strict --json
```

Do not use `gsvc init` on an existing PaySwitch service. The full workflow and
supported grammar are in README.md and `gsvc contract --json`.

For the implemented example:

```sh
gsvc check --root /Users/macbook/development/command/go-service-cli/examples/greeter --verify --strict --race --json
```

To rebuild after a reviewed source change:

```sh
cd /Users/macbook/development/command/go-service-cli
go test -count=1 -race ./...
go vet ./...
go build -trimpath -o bin/gsvc ./cmd/gsvc
gsvc check --root examples/greeter --verify --strict --race --json
```

## Executed macOS proof

- Fresh-shell `command -v gsvc` and `gsvc version`: resolves the installed binary,
  version 0.1.0, policy go-service/v1, schema 1.
- CLI `go test -count=1 -race ./...`, `go vet ./...`, native arm64 build: pass.
- Strict example with race tests and vet: 22 Go files, zero errors/warnings,
  both verification subprocesses exit 0, no truncated output.
- New disposable service: dry-run created no directory; reviewed plan ID applied;
  adding a module/query succeeded. Strict check returned exit 1 with WORK001,
  WORK002 and WORK003 for unfinished implementation/test scaffolds, as required.
- Real example bound only to 127.0.0.1:18783: `/healthz` and `/greetings/Ben`
  returned HTTP 200 and the expected JSON. SIGTERM exited 0; listener removed.
- Local secret scan of the distribution: no findings.
- Local Sonar: 18 original source/test files analyzed, 34 inherited
  maintainability findings; **not a clean quality gate**. No mapped server
  project. Details: `docs/sonar-baseline-2026-09-08.md`.

Grok was explicitly attempted for a bounded no-shell, read-only ownership and
recovery review. Authentication/preflight passed with installed Grok 1.0.13,
but inference did not start: its sandbox rejected the Docker runtime socket
symlink. Protections were not weakened; no verdict or patch was produced.
The temporary credential home is absent after exit cleanup, and the owner CLI
reports the exact attempted session absent. Independent review remains open.

## Boundaries

This is a usable local development foundation, not production certification.
Generated HTTP is unauthenticated and loopback-first. Identity/authorization,
database/sqlc, durable messaging/outbox, deployment and policy upgrades are not
implemented by this version. No PaySwitch service, cluster, permission, money
flow or published artifact was changed by installation. The existing PaySwitch
Joonalabs readiness goal remains separately active.
