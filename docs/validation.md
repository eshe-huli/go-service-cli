# Validation of gsvc v0.2.0

Date: September 14, 2026. Environment: macOS arm64, Go 1.27.1.

This records executed checks. It is not production certification, an exhaustive
correctness claim, or evidence that generated capability contracts have working
adapters.

## Completed checks

| Check | Result |
|---|---|
| `go test -count=1 -race -json ./...` on the CLI | Passed: 54 test/subtest events; no failed or skipped tests |
| `go vet ./...` on the CLI | Passed; no diagnostics |
| `go build -trimpath -o bin/gsvc ./cmd/gsvc` | Passed |
| Explicit `gsvc upgrade` of the checked v0.1 greeter fixture | Passed through dry-run, reviewed plan ID, application, and strict verification; developer-owned behavior/tests were preserved |
| `gsvc check --root examples/greeter --verify --strict --race --json` | Passed: 22 Go files checked, zero architecture errors or pending-work warnings; race tests and vet passed |
| Fresh `identity-provider-orchestrator` recipe dogfood | Passed init, dry-run, exact-plan application, capability inspection, idempotent rendering, 22-file strict source check, race tests, and vet; six `CAPABILITY004` warnings truthfully retain the unproved runtime work |
| Live `GET /healthz` | HTTP 200, `{"status":"ok"}` |
| Live `GET /greetings/Ben` | HTTP 200, `{"message":"Hello, Ben!"}` |
| SIGTERM of the built live example | Graceful exit, code 0 |

The 54 pass events include parent tests and subtests; they are not 54 independent
end-to-end scenarios. Packages without dedicated test files are not called skipped
tests here. Generation and checking are exercised together by CLI integration tests.

## Behaviors actually exercised

Tests create fresh services and business modules; generate empty and populated
commands/queries; compile/test complete generated services; bind literal and string
path-variable routes; and explicitly change contracts and endpoint paths.

Capability tests exercise immutable catalog copies, supported capability/version and
dependency validation, atomic and idempotent recipe composition, declared composition
derivation, fresh generated contract compilation/tests, checker-backed inspection, and
a failed recipe that performs no partial write. Policy tests prove that declared
consumer imports and narrow HTTP/Kafka/PostgreSQL/provider extension roots are allowed,
while undeclared platform paths and unapproved external SDKs are rejected. The
provider-neutral recipe declares service runtime, verified HTTP command context,
transactional inbox, operation journal, identity-provider port, and fenced
reconciliation contracts.

Generated contract tests also exercise immutable admitted payloads; typed principal
and organization-subject modes; rejection of `EXTERNAL_HOME_ORG` from the local subject
creation path; active reservation/write-fence boundaries; operation-journal claim CAS
tokens; stale/foreign/expired writer rejection before provider calls; and lease expiry
during claim or provider execution. These are contract-level proofs, not durable store
or live-provider proofs.

Upgrade tests accept only a verified v0.1 manifest plus matching ownership state,
produce an explicit v0.2 plan, preserve developer-owned behavior, and reject malformed,
unsupported, or edited legacy metadata. Other version migrations are intentionally not
implemented.

Negative tests cover invalid names and fields, reserved file names, conflicting
routes, malformed configuration, unknown flags and capabilities, stale plans, edited
generated output, unowned/managed ownership conflicts, domain I/O imports, direct peer
imports, init() wiring, files outside the approved layout, and unauthorized extra
platform helpers.

Writer tests cover preservation of developer edits, conflict detection before target
writes, cooperative writer locking, symlink rejection, rollback-journal recovery,
and refusal to overwrite unrelated post-interruption edits. Recovery is tested with
constructed interrupted-transaction state, not a power-loss experiment.

Runtime tests cover required JSON fields (including false/zero), duplicate keys,
case-alias/unknown keys, nulls, multiple JSON documents, size limits, query validation,
exact decimal-string syntax, panic handling, and not leaking unexpected error details
to HTTP responses. A deliberately failing generated service test correctly causes
verification to return a nonzero exit status.

A separately implemented greeting operation passes the strict gate. Pending operation
scaffolds fail strict checks rather than being reported as finished work.

## Evidence files

- `test-results.jsonl`: full final CLI race-test event stream.
- `vet.log`: CLI vet output (empty because there were no diagnostics).
- `example-check.json`: strict example check with standard tests and vet.
- `example-race-check.json`: strict example check with race tests and vet.
- `identity-provider-recipe-capabilities.json`: declared capability statuses and
  explicit implementation gaps for the fresh recipe dogfood.
- `identity-provider-recipe-check.json`: strict race-test and vet result for that
  generated recipe service.
- `live-smoke.json`: actual HTTP responses and graceful process exit.
- `toolchain.txt`: local toolchain version.
- `cli-contract.json`: the CLI's machine-readable command, catalog, recipe, and
  policy contract.

No live PostgreSQL, broker, identity provider, authorization service, deployment, or
production load test was performed. The recipe generates compile-tested contracts but
does not install those systems, write provider state, or prove an authorized actor
flow. Source checks, remote CI, release artifacts, deployment, runtime revision, and
actor proof remain separate gates.
