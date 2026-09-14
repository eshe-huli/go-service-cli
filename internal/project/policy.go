package project

// These decisions are compiled into this policy version, not configurable per
// developer or agent. To change them, release and review a new CLI policy.
func PolicyDescription() map[string]any {
	return map[string]any{
		"id":            Policy,
		"layout":        "cmd/api; internal/bootstrap; internal/<business>/internal/{domain,app,adapter/{httpapi,postgres}}",
		"transport":     "net/http; GET queries; POST commands; JSON; explicit generated registration",
		"composition":   "ordinary constructors, operation-specific dependency structs, bootstrap.Wire",
		"persistence":   "postgres adapter; pgx/v5 import family allowed; adapter and SQL generation not yet supplied",
		"domain":        "permitted standard-library packages only; no framework, transport, database, filesystem, or peer business imports",
		"application":   "own domain/application packages and platform fault/validate; no transport, persistence, filesystem, or peer business imports",
		"cross_module":  "consumer-owned interfaces injected at composition; no direct peer module imports",
		"global_checks": []string{"no init() wiring", "no unsafe, cgo, or plugin imports", "generated content must match canonical output", "no nested go.mod", "gofmt", "strict mode rejects pending scaffolds and test skips"},
		"ownership":     "CLI owns contracts, routes, generated wiring, runtime and agent instructions; developers own behavior, ports, adapters and business tests",
		"capabilities":  "versioned manifest declarations rendered from the canonical catalog; declared means source structure only, while runtime proof remains external",
		"recipes":       "deterministic capability compositions applied through the ordinary dry-run, plan, expect and ownership writer; recipe names are derived, not persisted",
		"limitations":   []string{"structural checks do not prove business correctness, authorization, transactional safety, or concurrency safety", "not a sandbox or defense against deliberate checker/CI edits", "generated API is unauthenticated and loopback-only by default", "explicit contract regeneration is supported; business behavior is never automatically migrated", "only the documented gsvc 0.1.0 go-service/v1 project upgrade is supported; other versions are rejected"},
	}
}
