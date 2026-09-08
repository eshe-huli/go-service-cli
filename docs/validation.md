# Validation of gsvc v0.1.0

Date: September 8, 2026. Environment: Linux amd64, Go 1.23.2.

This is a record of executed checks, not a claim of production certification,
exhaustive correctness, or measured reduction in agent drift.

## Completed checks

| Check | Result |
|---|---|
| `go test -count=1 -race -json ./...` on the CLI | Passed: 26 top-level tests; 40 passing test/subtest events; no failed or skipped tests |
| `go vet ./...` on the CLI | Passed; no diagnostics |
| `go build -trimpath -o bin/gsvc ./cmd/gsvc` | Passed |
| `gsvc check --root examples/greeter --verify --strict --race --json` | Passed: 22 Go files checked, zero architecture errors or pending-work warnings; generated service tests with the race detector and vet passed |
| Live `GET /healthz` | HTTP 200, `{"status":"ok"}` |
| Live `GET /greetings/Ben` | HTTP 200, `{"message":"Hello, Ben!"}` |
| Termination of the live example process | Graceful exit, code 0 |

The 40 test/subtest events include parent tests; they are not 40 independent
end-to-end scenarios. Packages without dedicated test files are not called skipped
tests here. Generation and checking are exercised together by CLI integration tests.

## Behaviors actually exercised

Tests create fresh services and business modules; generate empty and populated
commands/queries; compile/test complete generated services; bind literal and string
path-variable routes; and explicitly change contracts and endpoint paths.

Negative tests cover invalid names and fields, reserved file names, conflicting
routes, malformed configuration, unknown flags, stale plans, edited generated output,
unowned/managed ownership conflicts, domain I/O imports, direct peer imports, init()
wiring, files outside the approved layout, and unauthorized extra platform helpers.

Writer tests cover preservation of developer edits, conflict detection before target
writes, cooperative writer locking, symlink rejection, rollback-journal recovery,
and refusal to overwrite unrelated post-interruption edits. Recovery is tested with
constructed interrupted-transaction state, not a power-loss experiment.

Runtime tests cover required JSON fields (including false/zero), duplicate keys,
case-alias/unknown keys, nulls, multiple JSON documents, size limits, query validation,
exact decimal-string syntax, panic handling, and not leaking unexpected error details
to HTTP responses. A deliberately failing generated service test correctly causes
verification to return a nonzero exit status.

A separately implemented greeting operation passes the strict gate. Pending
operation scaffolds fail strict checks rather than being reported as finished work.

## Evidence files

- `test-results.jsonl`: full final CLI test event stream.
- `vet.log`: CLI vet output (empty because there were no diagnostics).
- `example-check.json`: strict example check with standard tests/vet.
- `example-race-check.json`: strict example check with race-instrumented tests/vet.
- `live-smoke.json`: actual HTTP responses and process exit code.
- `toolchain.txt`: local toolchain version.
- `cli-contract.json`: the CLI's own machine-readable command and policy contract.

The local compiler was Go 1.23.2 because that was the available runtime. Go 1.23 is the
source language floor, not a deployment recommendation. Use a supported Go toolchain
for real releases. The included GitHub Actions configuration selects a stable Go
release but was not run on a remote repository. macOS/Windows execution and supported
newer-toolchain execution were not verified in this environment.

No live PostgreSQL, broker, identity provider, authorization service, or production
load test was performed; those capabilities are not implemented in this foundation.
