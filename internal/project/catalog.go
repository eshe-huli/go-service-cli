package project

import (
	"fmt"
	"sort"
	"strings"
)

const CapabilityMaturity = "contract-scaffold"

const (
	capabilityTransactionalInbox   = "eventing.transactional-inbox"
	capabilityOperationJournal     = "projection.operation-journal"
	capabilityReconciliationWorker = "projection.reconciliation-worker"
)

type CapabilityOutput struct {
	Path     string `json:"path"`
	Template string `json:"template"`
}

// CapabilityExtension is a narrow developer-owned implementation boundary.
// ExternalImports is an allowlist of exact import families, not a plugin hook.
type CapabilityExtension struct {
	Root            string   `json:"root"`
	Description     string   `json:"description"`
	ExternalImports []string `json:"external_imports"`
}

type CapabilityDefinition struct {
	Capability
	Description            string                `json:"description"`
	Maturity               string                `json:"maturity"`
	Requires               []Capability          `json:"requires"`
	ConsumerImport         string                `json:"consumer_import"`
	Outputs                []CapabilityOutput    `json:"outputs"`
	Extensions             []CapabilityExtension `json:"extensions"`
	ImplementationRequired []string              `json:"implementation_required"`
}

type RecipeDefinition struct {
	Name         string       `json:"name"`
	Version      int          `json:"version"`
	Description  string       `json:"description"`
	Capabilities []Capability `json:"capabilities"`
}

type CapabilityAssessment struct {
	Capability
	Description            string                `json:"description"`
	Maturity               string                `json:"maturity"`
	Status                 string                `json:"status"`
	Requires               []Capability          `json:"requires"`
	ConsumerImport         string                `json:"consumer_import"`
	Outputs                []CapabilityOutput    `json:"outputs"`
	Extensions             []CapabilityExtension `json:"extensions"`
	ImplementationRequired []string              `json:"implementation_required"`
}

type RecipeApplication struct {
	Name         string       `json:"name"`
	Version      int          `json:"version"`
	Capabilities []Capability `json:"capabilities"`
}

var capabilityDefinitions = []CapabilityDefinition{
	{
		Capability:     Capability{ID: capabilityTransactionalInbox, Version: 1},
		Description:    "Durable, idempotent event admission contracts shared by broker and authenticated HTTP adapters.",
		Maturity:       CapabilityMaturity,
		ConsumerImport: "internal/platform/inbox",
		Outputs: []CapabilityOutput{
			{Path: "internal/platform/inbox/contracts_gen.go", Template: "capability_transactional_inbox.go"},
			{Path: "internal/platform/inbox/contracts_gen_test.go", Template: "capability_transactional_inbox_test.go"},
		},
		Extensions: []CapabilityExtension{
			{Root: "internal/platform/inbox/adapter/kafka", Description: "Kafka transport adapter; admission remains authenticated and schema allowlisted.", ExternalImports: []string{"github.com/twmb/franz-go/pkg/kgo"}},
			{Root: "internal/platform/inbox/adapter/http", Description: "Authenticated internal HTTP admission adapter using the standard library.", ExternalImports: []string{}},
			{Root: "internal/platform/inbox/adapter/postgres", Description: "PostgreSQL transactional inbox adapter.", ExternalImports: []string{"github.com/jackc/pgx/v5"}},
		},
		ImplementationRequired: []string{
			"implement an allowlisted, schema-versioned broker adapter and authenticated HTTP adapter",
			"commit the inbox receipt and desired-state mutation atomically in a real database transaction",
			"prove duplicate, out-of-order, poison-message, retry, dead-letter, and restart behavior",
		},
	},
	{
		Capability:     Capability{ID: "identity-provider.port", Version: 1},
		Description:    "Provider-neutral organization, subject, role-snapshot, and service-boundary grant contracts.",
		Maturity:       CapabilityMaturity,
		ConsumerImport: "internal/platform/identityprovider",
		Outputs: []CapabilityOutput{
			{Path: "internal/platform/identityprovider/contracts_gen.go", Template: "capability_identity_provider.go"},
			{Path: "internal/platform/identityprovider/contracts_gen_test.go", Template: "capability_identity_provider_test.go"},
		},
		Extensions: []CapabilityExtension{
			{Root: "internal/platform/identityprovider/adapter", Description: "Provider adapter implemented against the neutral port; external SDKs require a reviewed capability version.", ExternalImports: []string{}},
		},
		ImplementationRequired: []string{
			"implement a provider adapter outside generated code and authenticate it with a reviewed machine credential",
			"use fenced canonical subject reservations; never correlate or merge subjects by email or phone",
			"route external-home access through existing-subject authorization; never create a target-organization subject for EXTERNAL_HOME_ORG",
			"prove organization, subject, role snapshot, service-boundary grant, revocation, collision, and read-after-write behavior",
		},
	},
	{
		Capability:     Capability{ID: "platform.http-command-ingress", Version: 1},
		Description:    "Verified principal, provenance, idempotency, and request-context contracts for internal commands.",
		Maturity:       CapabilityMaturity,
		ConsumerImport: "internal/platform/commanding",
		Outputs: []CapabilityOutput{
			{Path: "internal/platform/commanding/context_gen.go", Template: "capability_http_command_ingress.go"},
			{Path: "internal/platform/commanding/context_gen_test.go", Template: "capability_http_command_ingress_test.go"},
		},
		Extensions: []CapabilityExtension{
			{Root: "internal/platform/commanding/adapter", Description: "Authenticated HTTP context adapter; credential verification remains developer-owned.", ExternalImports: []string{}},
		},
		ImplementationRequired: []string{
			"authenticate the caller before constructing a verified command context",
			"authorize service and action scope through the owning platform authorities",
			"prove replay, missing provenance, wrong audience, cancellation, timeout, and denial behavior",
		},
	},
	{
		Capability:     Capability{ID: "platform.service-runtime", Version: 1},
		Description:    "A generated service manifest that declares contract capabilities without asserting runtime readiness.",
		Maturity:       CapabilityMaturity,
		ConsumerImport: "internal/platform/serviceinfo",
		Outputs: []CapabilityOutput{
			{Path: "internal/platform/serviceinfo/manifest_gen.go", Template: "capability_service_runtime.go"},
			{Path: "internal/platform/serviceinfo/manifest_gen_test.go", Template: "capability_service_runtime_test.go"},
		},
		Extensions: []CapabilityExtension{
			{Root: "internal/platform/serviceinfo/adapter", Description: "Service catalog and dependency-readiness adapter.", ExternalImports: []string{}},
		},
		ImplementationRequired: []string{
			"wire truthful liveness and dependency-aware readiness",
			"register the immutable revision and declared capabilities through authenticated Service Access",
			"keep source, CI, artifact, deployment, runtime, and authorized actor proof separate",
		},
	},
	{
		Capability:     Capability{ID: capabilityOperationJournal, Version: 1},
		Description:    "Durable desired-state operation, attempt, supersession, and terminal-outcome contracts.",
		Maturity:       CapabilityMaturity,
		ConsumerImport: "internal/platform/operationjournal",
		Outputs: []CapabilityOutput{
			{Path: "internal/platform/operationjournal/contracts_gen.go", Template: "capability_operation_journal.go"},
			{Path: "internal/platform/operationjournal/contracts_gen_test.go", Template: "capability_operation_journal_test.go"},
		},
		Extensions: []CapabilityExtension{
			{Root: "internal/platform/operationjournal/adapter/postgres", Description: "PostgreSQL operation-journal adapter.", ExternalImports: []string{"github.com/jackc/pgx/v5"}},
		},
		ImplementationRequired: []string{
			"implement durable compare-and-set transitions and deterministic operation keys",
			"persist provider receipts without storing credentials or raw sensitive tokens",
			"prove duplicate, superseded, retryable, terminal, crash-between-write-and-commit, and manual-repair states",
		},
	},
	{
		Capability:     Capability{ID: capabilityReconciliationWorker, Version: 1},
		Description:    "Leased, fenced, bounded reconciliation worker contracts for one active provider writer.",
		Maturity:       CapabilityMaturity,
		Requires:       []Capability{{ID: capabilityOperationJournal, Version: 1}},
		ConsumerImport: "internal/platform/reconciliation",
		Outputs: []CapabilityOutput{
			{Path: "internal/platform/reconciliation/contracts_gen.go", Template: "capability_reconciliation_worker.go"},
			{Path: "internal/platform/reconciliation/contracts_gen_test.go", Template: "capability_reconciliation_worker_test.go"},
		},
		Extensions: []CapabilityExtension{
			{Root: "internal/platform/reconciliation/adapter", Description: "Worker scheduling, lease, and execution adapters.", ExternalImports: []string{}},
		},
		ImplementationRequired: []string{
			"implement leases or fencing that prevent concurrent provider mutation",
			"bound retries and distinguish retryable, terminal, superseded, and operator-required outcomes",
			"prove cancellation, lease expiry, failover, provider-success/local-commit-failure, and restart reuse",
		},
	},
}

var recipeDefinitions = []RecipeDefinition{
	{
		Name:        "identity-provider-orchestrator",
		Version:     1,
		Description: "Provider-neutral identity projection control-plane contracts; concrete provider and Kafka adapters remain developer-owned.",
		Capabilities: []Capability{
			{ID: "platform.service-runtime", Version: 1},
			{ID: "platform.http-command-ingress", Version: 1},
			{ID: capabilityTransactionalInbox, Version: 1},
			{ID: capabilityOperationJournal, Version: 1},
			{ID: "identity-provider.port", Version: 1},
			{ID: capabilityReconciliationWorker, Version: 1},
		},
	},
	{
		Name:        "projection-worker",
		Version:     1,
		Description: "Durable event intake, operation journal, and single-writer reconciliation contracts.",
		Capabilities: []Capability{
			{ID: capabilityTransactionalInbox, Version: 1},
			{ID: capabilityOperationJournal, Version: 1},
			{ID: capabilityReconciliationWorker, Version: 1},
		},
	},
}

func CapabilityCatalog() []CapabilityDefinition {
	out := make([]CapabilityDefinition, 0, len(capabilityDefinitions))
	for _, definition := range capabilityDefinitions {
		out = append(out, cloneCapabilityDefinition(definition))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func RecipeCatalog() []RecipeDefinition {
	out := make([]RecipeDefinition, 0, len(recipeDefinitions))
	for _, recipe := range recipeDefinitions {
		clone := recipe
		clone.Capabilities = append([]Capability(nil), recipe.Capabilities...)
		out = append(out, clone)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func CapabilityByID(id string) (CapabilityDefinition, bool) {
	for _, definition := range capabilityDefinitions {
		if definition.ID == id {
			return cloneCapabilityDefinition(definition), true
		}
	}
	return CapabilityDefinition{}, false
}

func cloneCapabilityDefinition(definition CapabilityDefinition) CapabilityDefinition {
	clone := definition
	clone.Requires = append([]Capability{}, definition.Requires...)
	clone.Outputs = append([]CapabilityOutput(nil), definition.Outputs...)
	clone.Extensions = make([]CapabilityExtension, 0, len(definition.Extensions))
	for _, extension := range definition.Extensions {
		extensionClone := extension
		extensionClone.ExternalImports = append([]string{}, extension.ExternalImports...)
		clone.Extensions = append(clone.Extensions, extensionClone)
	}
	clone.ImplementationRequired = append([]string(nil), definition.ImplementationRequired...)
	return clone
}

func DeclaredCapabilityDefinitions(manifest Manifest) []CapabilityDefinition {
	definitions := make([]CapabilityDefinition, 0, len(manifest.Capabilities))
	for _, capability := range manifest.Capabilities {
		definition, ok := CapabilityByID(capability.ID)
		if ok && capability.Version == definition.Version {
			definitions = append(definitions, definition)
		}
	}
	return definitions
}

func CapabilityExtensionForPath(manifest Manifest, filePath string) (CapabilityExtension, bool) {
	directory := pathDirectory(filePath)
	for _, definition := range DeclaredCapabilityDefinitions(manifest) {
		for _, extension := range definition.Extensions {
			if directory == extension.Root || strings.HasPrefix(directory, extension.Root+"/") {
				clone := extension
				clone.ExternalImports = append([]string{}, extension.ExternalImports...)
				return clone, true
			}
		}
	}
	return CapabilityExtension{}, false
}

func CapabilityConsumerImportAllowed(manifest Manifest, relativeImport string) bool {
	for _, definition := range DeclaredCapabilityDefinitions(manifest) {
		if relativeImport == definition.ConsumerImport {
			return true
		}
	}
	return false
}

func CapabilityExternalImportAllowed(manifest Manifest, filePath, importPath string) bool {
	extension, ok := CapabilityExtensionForPath(manifest, filePath)
	if !ok {
		return false
	}
	for _, allowed := range extension.ExternalImports {
		if importPath == allowed || strings.HasPrefix(importPath, allowed+"/") {
			return true
		}
	}
	return false
}

func pathDirectory(filePath string) string {
	if index := strings.LastIndex(filePath, "/"); index >= 0 {
		return filePath[:index]
	}
	return "."
}

func ValidateCapability(capability Capability) error {
	definition, ok := CapabilityByID(capability.ID)
	if !ok {
		return Fail("CAPABILITY001", "unsupported capability: "+capability.ID, "Use gsvc capabilities or gsvc recipe to inspect the typed catalog.")
	}
	if capability.Version != definition.Version {
		return Fail("CAPABILITY002", fmt.Sprintf("unsupported capability version %s@%d", capability.ID, capability.Version), fmt.Sprintf("This CLI supports %s@%d.", definition.ID, definition.Version))
	}
	return nil
}

func ValidateCapabilityDependencies(capabilities []Capability) error {
	declared := make(map[string]int, len(capabilities))
	for _, capability := range capabilities {
		declared[capability.ID] = capability.Version
	}
	for _, capability := range capabilities {
		definition, ok := CapabilityByID(capability.ID)
		if !ok || definition.Version != capability.Version {
			continue
		}
		for _, requirement := range definition.Requires {
			if declared[requirement.ID] != requirement.Version {
				return Fail(
					"CAPABILITY005",
					fmt.Sprintf("%s@%d requires %s@%d", capability.ID, capability.Version, requirement.ID, requirement.Version),
					"Apply a catalog recipe so capability dependencies are declared atomically.",
				)
			}
		}
	}
	return nil
}

func ApplyRecipe(manifest Manifest, name string) (Manifest, RecipeApplication, error) {
	var selected *RecipeDefinition
	for _, recipe := range RecipeCatalog() {
		if recipe.Name == name {
			copy := recipe
			selected = &copy
			break
		}
	}
	if selected == nil {
		return Manifest{}, RecipeApplication{}, Fail("RECIPE001", "unknown recipe: "+name, "Available recipes: "+strings.Join(recipeNames(), ", ")+".")
	}

	clone := cloneManifest(manifest)
	versions := map[string]int{}
	for _, capability := range clone.Capabilities {
		versions[capability.ID] = capability.Version
	}
	for _, capability := range selected.Capabilities {
		if err := ValidateCapability(capability); err != nil {
			return Manifest{}, RecipeApplication{}, err
		}
		if version, exists := versions[capability.ID]; exists && version != capability.Version {
			return Manifest{}, RecipeApplication{}, Fail("RECIPE002", fmt.Sprintf("capability %s is declared at version %d, recipe requires version %d", capability.ID, version, capability.Version), "Capability upgrades require an explicit migration.")
		}
	}
	for _, capability := range selected.Capabilities {
		if _, exists := versions[capability.ID]; !exists {
			clone.Capabilities = append(clone.Capabilities, capability)
			versions[capability.ID] = capability.Version
		}
	}
	Normalize(&clone)
	if err := Validate(clone); err != nil {
		return Manifest{}, RecipeApplication{}, err
	}
	application := RecipeApplication{Name: selected.Name, Version: selected.Version, Capabilities: append([]Capability(nil), selected.Capabilities...)}
	return clone, application, nil
}

func AssessCapabilities(manifest Manifest) []CapabilityAssessment {
	declared := map[string]int{}
	for _, capability := range manifest.Capabilities {
		declared[capability.ID] = capability.Version
	}
	results := make([]CapabilityAssessment, 0, len(capabilityDefinitions))
	for _, definition := range CapabilityCatalog() {
		status := "available"
		if version, ok := declared[definition.ID]; ok && version == definition.Version {
			status = "declared"
		} else if ok {
			status = "version-mismatch"
		}
		results = append(results, CapabilityAssessment{
			Capability: definition.Capability, Description: definition.Description, Maturity: definition.Maturity,
			Status: status, Requires: append([]Capability{}, definition.Requires...), ConsumerImport: definition.ConsumerImport,
			Outputs:                append([]CapabilityOutput(nil), definition.Outputs...),
			Extensions:             cloneCapabilityDefinition(definition).Extensions,
			ImplementationRequired: append([]string(nil), definition.ImplementationRequired...),
		})
	}
	return results
}

func DeclaredRecipeCompositions(manifest Manifest) []RecipeDefinition {
	declared := map[string]int{}
	for _, capability := range manifest.Capabilities {
		declared[capability.ID] = capability.Version
	}
	compositions := []RecipeDefinition{}
	for _, recipe := range RecipeCatalog() {
		complete := true
		for _, capability := range recipe.Capabilities {
			if declared[capability.ID] != capability.Version {
				complete = false
				break
			}
		}
		if complete {
			compositions = append(compositions, recipe)
		}
	}
	return compositions
}

func cloneManifest(manifest Manifest) Manifest {
	clone := manifest
	clone.Capabilities = append([]Capability(nil), manifest.Capabilities...)
	clone.Modules = make([]Module, 0, len(manifest.Modules))
	for _, module := range manifest.Modules {
		moduleClone := Module{Name: module.Name, Operations: make([]Operation, 0, len(module.Operations))}
		for _, operation := range module.Operations {
			operationClone := operation
			operationClone.Input = append([]Field(nil), operation.Input...)
			operationClone.Output = append([]Field(nil), operation.Output...)
			if operation.Endpoint != nil {
				endpoint := *operation.Endpoint
				operationClone.Endpoint = &endpoint
			}
			moduleClone.Operations = append(moduleClone.Operations, operationClone)
		}
		clone.Modules = append(clone.Modules, moduleClone)
	}
	return clone
}

func recipeNames() []string {
	names := make([]string, 0, len(recipeDefinitions))
	for _, recipe := range recipeDefinitions {
		names = append(names, recipe.Name)
	}
	sort.Strings(names)
	return names
}
