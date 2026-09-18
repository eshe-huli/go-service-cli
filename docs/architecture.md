# Architecture and scope

## Product boundary

The product is an independent CLI that standardizes how a developer or agent creates
and evolves a Go service. Familiar workflow, ordinary Go output. It is not a Go clone
of Nest decorators, providers, command buses, exceptions, ORM entities, or runtime
module discovery. Company/platform adapters can be separately added when their real
contracts are available and approved.

## The implementation

`internal/project` defines the service model, typed capability and recipe catalogs,
explicit v0.1-to-v0.2 migration, fixed policy description, strict validation,
ownership model, plan IDs, guarded writer, and recovery journal.

`internal/generate` embeds templates, renders the model, formats Go source, and labels
each output as developer-owned or managed. It does not write files.

`internal/check` renders canonical output again, compares managed content, parses
source files with go/parser, inspects import-aware ASTs, and produces structured
diagnostics. It is intentionally described as an AST checker, not a whole-program
semantic proof. `--verify` adds Go compilation/tests/vet.

`internal/cli` declares the command grammar once, parses flags without prompts,
produces a JSON envelope, coordinates plans/writes, and optionally executes Go tools.

`internal/assets/runtime` is independently compiled/tested helper code. It is embedded
into each generated service as version-owned output. This is a bootstrap distribution
choice, not an unmaintained copy-paste model: changes are managed by the CLI policy.
A separately published shared runtime can replace this distribution mechanism in a
future reviewed version, without importing a framework into business entities.

## Generated layout

```
cmd/api/main.go                         developer: process lifecycle
internal/bootstrap/wire.go             developer: explicit adapter composition
internal/bootstrap/routes_gen.go       managed: service registration
internal/payments/module.go            developer: module factories
internal/payments/module_gen.go        managed: typed operation wiring
internal/payments/internal/domain/     developer: business invariants
internal/payments/internal/app/        developer: operations, ports, tests
  create_payment_contract_gen.go       managed: operation input/output contract
internal/payments/internal/adapter/
  httpapi/routes_gen.go                managed: HTTP binding
  postgres/                           developer: approved persistence boundary
internal/platform/{fault,httpx,validate}/ managed helpers and their tests
internal/platform/<capability>/          managed capability contracts and tests
gsvc.json                             managed live model
.gsvc/ownership.json                   managed ownership metadata
AGENTS.md                             managed agent contract
```

Nested internal packages provide Go import boundaries between business modules.
Layer policy further restricts imports inside each module. Application code uses
consumer-owned interfaces injected through operation-specific dependency structs.
The module exposes aliases for those dependency shapes so bootstrap can wire them
without importing private application packages. No runtime DI container is needed.

## Intentional limits

The HTTP profile supports POST commands, GET queries, clean literal paths and
single-segment string path variables. It does not support wildcards, arbitrary route
plugins, streaming, uploads, gRPC, GraphQL or WebSockets. Inputs are required scalar
string/money/bool/int64 fields; optional/nested/list fields require a schema extension.

Money is a decimal string. Syntax is validated on input; currency, rounding, precision,
arithmetic and business meaning are not implemented by the scaffold.

No authentication, authorization, database client, migration engine, outbox, broker,
tenant context or identity client is silently faked. A capability recipe may generate
typed contracts for one of these boundaries, but status `declared` never means the
adapter, credential, durable store, worker, deployment, or actor flow exists. The empty
PostgreSQL package marks the approved adapter location, not functioning persistence.

The generated HTTP process sets server timeouts and supports graceful shutdown. This
is not a production readiness claim, an authentication substitute, or a guarantee that
application operations honor cancellation. Health reports liveness only.

## Agent drift controls

Discovery (`contract`, `inspect`, `capabilities`, and no-argument `recipe`) makes the
supported workflow explicit. A recipe only unions versioned capabilities into the
manifest; rendering derives their files, so recipe names are not a second source of
state. Ownership makes application idempotent and safe to rerun. The checker reports
drift. Capability declarations open only catalog-owned consumer imports, extension
roots, and exact external import families; undeclared paths remain rejected. Strict
verification distinguishes operation scaffolds from completed business work, while
`CAPABILITY004` keeps its source-structure result visibly separate from runtime proof.
Catalog-declared capability dependencies are also validated before rendering, so a
composition cannot silently generate source that imports an undeclared primitive.
Required CI and review of the CLI, policy, manifest, ownership metadata, adapters, and
external operational evidence are the enforcement layer.

An agent with permission to change everything can evade any local tool. Plan IDs and
hashes detect stale state and content changes, not the authority of whoever changed
the state. This project makes no tamper-proof or security-sandbox claim.

## Extension order

The v0.2 catalog deliberately starts with six narrow contracts and two compositions,
not a general plugin system. The next useful additions remain real PostgreSQL and
broker vertical slices with transaction, restart, and integration tests; recipe
catalog growth should follow proven service needs. Versioned API schema/export,
contract removal, further migrations, and concrete security/platform adapters remain
future work, not features implied by a capability declaration.

## Primary technical references

- Go server/module organization: https://go.dev/doc/modules/layout
- Standard HTTP router: https://pkg.go.dev/net/http
- Source parsing: https://pkg.go.dev/go/parser
- Abstract syntax trees: https://pkg.go.dev/go/ast
- Supported release history: https://go.dev/doc/devel/release

These inform the underlying Go mechanisms. The particular business-module policy is
this project's opinion, not a claim that the Go team mandates DDD or this layout.
