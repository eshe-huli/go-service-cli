# Supplied v0.1.0 Sonar baseline — 2026-09-08

Scope: 18 original Go files under `cmd/` and `internal/`, including embedded runtime source and tests; no Go source was modified during laptop installation. This is a local read-only analysis, not a server quality gate. No matching `gsvc` Sonar project was found. Generated example copies and Go text templates were not separately analyzed as Go source. The entire extracted distribution passed the local secret scan.

The analyzer returned 34 pre-existing maintainability findings, all classified CRITICAL by its maintainability profile. This is **not** a clean Sonar result, a security certification, or 34 confirmed runtime defects. No findings were suppressed. Refactoring these files is a separate reviewed change with compatibility and regeneration proof.

| File | Line | Rule | Finding |
| --- | --- | --- | --- |
| `internal/check/check.go` | 72 | go:S1192 | Define a constant instead of duplicating this literal "internal/platform/" 3 times. |
| `internal/check/check.go` | 76 | go:S1192 | Define a constant instead of duplicating this literal "internal/" 4 times. |
| `internal/check/check.go` | 64 | go:S3776 | Refactor this method to reduce its Cognitive Complexity from 18 to the 15 allowed. |
| `internal/check/check.go` | 103 | go:S3776 | Refactor this method to reduce its Cognitive Complexity from 40 to the 15 allowed. |
| `internal/check/check.go` | 168 | go:S3776 | Refactor this method to reduce its Cognitive Complexity from 194 to the 15 allowed. |
| `internal/project/model.go` | 165 | go:S1186 | Add a nested comment explaining why this function is empty or complete the implementation. |
| `internal/project/model.go` | 208 | go:S1186 | Add a nested comment explaining why this function is empty or complete the implementation. |
| `internal/project/model.go` | 153 | go:S3776 | Refactor this method to reduce its Cognitive Complexity from 44 to the 15 allowed. |
| `internal/project/files_test.go` | 87 | go:S1192 | Define a constant instead of duplicating this literal "new.txt" 3 times. |
| `internal/project/files_test.go` | 12 | go:S1192 | Define a constant instead of duplicating this literal "business.go" 3 times. |
| `internal/project/files_test.go` | 84 | go:S1192 | Define a constant instead of duplicating this literal "old.txt" 4 times. |
| `internal/project/files.go` | 180 | go:S1192 | Define a constant instead of duplicating this literal ".gsvc/transaction.json" 8 times. |
| `internal/project/files.go` | 137 | go:S3776 | Refactor this method to reduce its Cognitive Complexity from 16 to the 15 allowed. |
| `internal/project/files.go` | 175 | go:S3776 | Refactor this method to reduce its Cognitive Complexity from 54 to the 15 allowed. |
| `internal/project/files.go` | 361 | go:S3776 | Refactor this method to reduce its Cognitive Complexity from 19 to the 15 allowed. |
| `internal/project/files.go` | 404 | go:S3776 | Refactor this method to reduce its Cognitive Complexity from 26 to the 15 allowed. |
| `internal/cli/cli_test.go` | 70 | go:S1192 | Define a constant instead of duplicating this literal "--strict" 3 times. |
| `internal/cli/cli_test.go` | 86 | go:S1192 | Define a constant instead of duplicating this literal "--verify" 4 times. |
| `internal/cli/cli_test.go` | 63 | go:S1192 | Define a constant instead of duplicating this literal "example.com/new-api" 3 times. |
| `internal/cli/cli_test.go` | 148 | go:S1192 | Define a constant instead of duplicating this literal "greet-person" 7 times. |
| `internal/cli/cli_test.go` | 148 | go:S1192 | Define a constant instead of duplicating this literal "message:string" 4 times. |
| `internal/cli/cli_test.go` | 84 | go:S1192 | Define a constant instead of duplicating this literal "--path" 4 times. |
| `internal/cli/cli_test.go` | 68 | go:S1192 | Define a constant instead of duplicating this literal "--expect" 3 times. |
| `internal/cli/cli_test.go` | 37 | go:S1192 | Define a constant instead of duplicating this literal "--module" 17 times. |
| `internal/cli/cli_test.go` | 78 | go:S1192 | Define a constant instead of duplicating this literal "create-payment" 5 times. |
| `internal/cli/cli_test.go` | 148 | go:S1192 | Define a constant instead of duplicating this literal "name:string" 4 times. |
| `internal/cli/cli_test.go` | 70 | go:S1192 | Define a constant instead of duplicating this literal "--root" 33 times. |
| `internal/cli/cli.go` | 41 | go:S3776 | Refactor this method to reduce its Cognitive Complexity from 37 to the 15 allowed. |
| `internal/cli/cli.go` | 135 | go:S3776 | Refactor this method to reduce its Cognitive Complexity from 92 to the 15 allowed. |
| `internal/cli/spec.go` | 41 | go:S1192 | Define a constant instead of duplicating this literal "Existing business module" 5 times. |
| `internal/cli/spec.go` | 32 | go:S1192 | Define a constant instead of duplicating this literal "dry-run" 3 times. |
| `internal/cli/spec.go` | 60 | go:S3776 | Refactor this method to reduce its Cognitive Complexity from 49 to the 15 allowed. |
| `internal/generate/generate.go` | 129 | go:S3776 | Refactor this method to reduce its Cognitive Complexity from 60 to the 15 allowed. |
| `internal/assets/runtime/httpx/httpx.go` | 25 | go:S3776 | Refactor this method to reduce its Cognitive Complexity from 24 to the 15 allowed. |

The archive hash and actual macOS checks are recorded in `INSTALLATION.md`. Preserve the original archive as the comparison baseline when making a correction; do not adjust ownership hashes or completion checks to silence failures.
