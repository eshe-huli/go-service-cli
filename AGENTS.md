# Working on the gsvc CLI itself

This repository is the standalone generator/checker, not a generated service. Do not
run `gsvc inspect` or `gsvc check` at this repository root; those commands require a
service manifest. Use them against `examples/greeter` or a temporary generated fixture.

Read README.md and docs/architecture.md. Preserve one source of truth for the service
model and architecture policy in internal/project. Both generation and checking must
continue to agree. The command grammar is declared in internal/cli/spec.go and reused
by parsing, help and machine-readable discovery.

Capabilities are versioned project contracts declared in `gsvc.json`. Their catalog,
recipe compositions, generated outputs, validation, and upgrade rules live under
`internal/project`; do not create a second recipe state file. Recipes may only union
typed capabilities and must use the ordinary dry-run/plan/expect writer. Concrete
provider, broker, database, credential, and business-policy implementations remain
developer-owned and must never be implied by a declared contract.

Treat developer-owned content as non-regenerable. Do not introduce a --force path that
overwrites business files, resets hashes, silently adopts files, weakens checks, or
swallows invalid configuration. Exercise failures and complete generated applications,
not merely template snapshots.

Runtime helpers are real source under internal/assets/runtime and embedded into
managed generated output. Update their tests before changing behavior. A runtime,
capability, or template change requires recreating the example with the current
generator, preserving its separately authored application implementation/test files,
and rerunning its gate.

Use standard Go tooling, explicit constructors and errors. No third-party dependency
is required for this CLI. Do not add runtime DI, a general plugin system, silent
dependency installation, new database stacks, company-specific identity rules or
copied features without a concrete approved requirement. Supported business-operation
changes are different from policy changes. Capability output must say what remains
unimplemented and unproven.

Before completing a CLI change:

    gofmt -w cmd internal
    go test -race ./...
    go vet ./...
    go build -o bin/gsvc ./cmd/gsvc
    bin/gsvc check --root examples/greeter --verify --strict

Do not invent test results, claim all Go semantics are checked, or call this a sandbox.
A Go 1.23 language floor is not a recommendation to deploy an unsupported Go release.
Use a supported toolchain for releases. Do not publish a remote repository or select
a public license without the owner's instruction.
