# gsvc — Go Service CLI

**One supported service-development workflow, for humans and coding agents.**

This is a new, standalone Go project. It is not a NestJS port, a TypeScript wrapper,
or a change to `nestjs-ddd-cli`. The working binary name is **gsvc**.

The CLI owns structural decisions. Developers and agents own business behavior.
Generated services are ordinary Go: explicit constructors, typed application calls,
standard HTTP interfaces, and no reflection-based dependency injection container.

**Status: v0.2.0 foundation.** Scaffolding, explicit contract updates, ownership-safe
writes, typed capability recipes, machine-readable discovery, drift checks, and a
working example are implemented. Capability recipes generate contracts, not working
authentication, authorization, database, broker, or provider adapters. No claim is
made that the foundation is ready to expose to a public network.

## Install from this source distribution

Use a currently supported Go toolchain. The language floor is Go 1.23.

```sh
cd go-service-cli
go test ./...
go build -o ./bin/gsvc ./cmd/gsvc
export PATH="$(pwd)/bin:$PATH"
gsvc version
```

For a persistent installation, run `go install ./cmd/gsvc` from this directory and
make sure your Go binary directory is on PATH. No public module has been published:
`gsvc.local/cli` is deliberately a local module identity, not a fictitious remote URL.
The CLI and initial generated services have no third-party Go dependencies.

## Create a service

```sh
gsvc init ../payments-api --module example.com/payments-api
cd ../payments-api

gsvc add module payments
gsvc add command create-payment --module payments \
  --in 'amount:money,currency:string' --out 'payment_id:string'
gsvc add endpoint create-payment --module payments --path /payments

gsvc add query get-payment --module payments \
  --in 'id:string' --out 'payment_id:string,status:string'
gsvc add endpoint get-payment --module payments --path '/payments/{id}'

gsvc check --verify
go run ./cmd/api
```

`/healthz` returns process liveness. New operations return HTTP 501 until their
business implementations exist. Business test scaffolds are explicitly pending;
normal checks report warnings, while the **strict completion gate fails** until
those implementations and tests have been completed.

The service listens on `127.0.0.1:8080` by default. Setting `HTTP_ADDR` can change the
binding. The initial API is **unauthenticated**: do not expose it to an untrusted
network without the required security and infrastructure work.

## Give an agent a constrained workflow

Every generated service includes a managed `AGENTS.md` and a `CLAUDE.md` pointer.
Tell the agent:

> Read AGENTS.md. Run `gsvc contract --json`, `gsvc inspect --json`, and
> `gsvc check --json`. Use the CLI for structural changes; implement only the
> requested behavior in developer-owned files. Finish with
> `gsvc check --verify --strict --json`. Do not weaken the checks to make them pass.

The discovery commands expose the actual CLI grammar, field types, architecture
policy, live module/operation inventory, exact ownership paths, and current diagnostics.
They do not require an agent to guess from a static prompt.

```sh
# Review the exact file actions without changing anything.
gsvc add query list-payments --module payments --dry-run --json

# Rerun the same command with --expect <plan-id> to bind it to that review.
# (Read the ID from data.plan.id in the JSON response.)
gsvc add query list-payments --module payments --expect '<reviewed-plan-id>'

# Final gate after implementing behavior and real tests.
gsvc check --verify --strict --json
```

All commands are non-interactive. JSON failures use the same envelope as successes.
Generation does not execute shell scripts or fetch dependencies. Explicit `--verify`
executes project tests and Go tooling; it is not a sandbox and Go may fetch declared
dependencies according to the environment's normal settings.

## Apply typed capability recipes

Recipes are deterministic, versioned compositions over the same guarded project model.
They never run plugins, install dependencies, or persist a second recipe-state file.

```sh
# List the catalog without requiring a service directory.
gsvc recipe --json

# Review the exact manifest and generated contract changes.
gsvc recipe identity-provider-orchestrator --root ../identity-provider \
  --dry-run --json

# Apply only the reviewed plan.
gsvc recipe identity-provider-orchestrator --root ../identity-provider \
  --expect '<reviewed-plan-id>' --json

# Inspect declarations and the work still required for runtime truth.
gsvc capabilities --root ../identity-provider --json
```

The initial `identity-provider-orchestrator` recipe is provider-neutral. It declares
service-runtime, verified-command-context, transactional-inbox, operation-journal,
identity-provider-port, and fenced-reconciliation contracts. A concrete ZITADEL,
Keycloak, or other adapter remains developer-owned. Kafka and PostgreSQL libraries,
credentials, migrations, workers, authorization decisions, and provider writes are
not installed or claimed by the recipe. Status `declared` means only that the
capability is in the manifest; the command's separate `contract_check` verifies the
canonical files and source integrity, then lists the remaining operational evidence.

Each capability advertises one exact application import plus narrow developer-owned
extension roots. Only external import families explicitly listed for that root pass
the checker. For example, the inbox capability permits franz-go only below its Kafka
adapter root, standard-library HTTP only below its HTTP root, and pgx only below its
PostgreSQL root; declaring a capability does not open the general platform tree or
allow an arbitrary SDK. Capability dependencies are explicit and validated:
reconciliation requires the operation journal, so a hand-written incomplete manifest
is rejected before rendering.

`gsvc check --verify --strict` is a source-structure gate. Declared capabilities emit
`CAPABILITY004` warnings even when that gate passes because adapter integration,
adverse-state, deployed-revision, and authorized-actor proof are external facts. A
green process exit must not be reported as operational capability completion.

`gsvc upgrade` provides the explicit, ownership-checked migration from the v0.1 project
contract to v0.2. Review it with `--dry-run --json`, bind the result with `--expect`,
and commit the manifest, ownership metadata, and regenerated managed output together.
Other source versions are rejected instead of guessed.

## Change a contract explicitly

`add` is idempotent for identical requests and rejects accidental changes to existing
contracts. Use `change` to deliberately evolve fields or a route.

```sh
gsvc change operation create-payment --module payments \
  --in 'amount:money,currency:string,reference:string' --dry-run --json

# After reviewing the plan:
gsvc change operation create-payment --module payments \
  --in 'amount:money,currency:string,reference:string'

gsvc change endpoint get-payment --module payments --path '/v1/payments/{id}'
```

Omit `--in` or `--out` to preserve that side. Pass an empty value to clear it.
Existing implementation and business test files are preserved. Contract updates may
require you or the agent to update behavior and tests; this is not an automatic
business migration. Routes are validated against the new input contract before any
write. Removal of modules/operations and upgrades other than the documented v0.1 to
v0.2 project migration are not implemented.

## What is enforced

The checker parses Go syntax and imports, then applies the same model used by the
generator. It checks inward layer dependencies, forbidden peer-module imports,
approved adapter locations, prohibited `init()` wiring, registered Execute methods,
context signatures, nested Go modules, formatting, and generated-file integrity.

The domain has a small permitted standard-library surface. The application cannot
import HTTP, a database driver, the filesystem, or another business module. Shared
runtime files are managed output, not a place to invent miscellaneous helpers.
The only external application-source import family currently permitted is pgx/v5,
and only at the PostgreSQL/composition boundary. This permission is not a supplied
or verified PostgreSQL implementation.

Strict mode also rejects scaffold markers, not-implemented sentinels, and test skip
calls. Actual Go compilation, tests, and vet are performed with `--verify`.

**This reduces accidental drift, not deliberate evasion.** Require the gate in CI,
review policy/manifest/ownership/CI changes, and keep checker permissions separate
from routine feature work. Static checks do not prove authorization, tenant isolation,
transaction correctness, test quality, performance, or the absence of races.

## Ownership and recovery

| Owner | Examples | Behavior on rerun |
|---|---|---|
| CLI | Contracts, HTTP bindings, generated registration, runtime helpers, agent instructions | Must match canonical output; manual edits are rejected |
| Developer | Operations, ports/dependency structs, domain behavior, adapters, main/wiring, business tests | Preserved byte-for-byte |

An unowned existing target is never overwritten or silently adopted. Writes use a
cooperative exclusive lock, complete preflight, per-file atomic replacement, and a
rollback journal. A reviewed plan becomes stale when tracked files change.

`gsvc sync` can restore missing managed output. It will not bless manually edited
managed files or reconstruct missing business implementations. Restore those from
version control. `gsvc recover --root PATH --dry-run` inspects an interrupted write;
`gsvc recover --root PATH` rolls it back only when no unrelated edits would be lost.
See `docs/recovery.md` for stale-lock handling and the filesystem guarantees.

## Try the complete example

`examples/greeter` was created with the CLI and then its greeting operation and tests
were implemented. It is a complete local demonstration, not a fake financial service.

```sh
cd examples/greeter
gsvc check --verify --strict
go run ./cmd/api
# In another terminal:
curl http://127.0.0.1:8080/greetings/Ben
# {"message":"Hello, Ben!"}
```

## Maintain this CLI

```sh
go test -race ./...
go vet ./...
go build -o bin/gsvc ./cmd/gsvc
```

Integration tests generate fresh multi-module services, build/test them, inject
architectural drift, exercise stale plans and ownership conflicts, and demonstrate
that a real implemented operation passes the strict gate. Runtime helpers also have
independent tests. `docs/validation.md` records this distribution's actual checks.

The canonical source remote is `github.com/eshe-huli/go-service-cli`, while the Go
module identity remains deliberately local until a public module path and license are
approved. Source CI is included. No tagged release, published Go module, or required
branch-protection rule is claimed by this source tree.
