package project

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestCapabilityCatalogIsValid(t *testing.T) {
	capabilities := CapabilityCatalog()
	if len(capabilities) != 6 {
		t.Fatalf("got %d capabilities", len(capabilities))
	}
	seen := map[string]bool{}
	outputs := map[string]string{}
	extensionRoots := map[string]string{}
	for _, definition := range capabilities {
		validateCapabilityHeader(t, definition, seen)
		recordCapabilityOutputs(t, definition, outputs)
		recordCapabilityExtensions(t, definition, extensionRoots)
	}
}

func validateCapabilityHeader(t *testing.T, definition CapabilityDefinition, seen map[string]bool) {
	t.Helper()
	if seen[definition.ID] || definition.Version < 1 || definition.Maturity != CapabilityMaturity {
		t.Fatalf("invalid capability %#v", definition)
	}
	seen[definition.ID] = true
	if len(definition.Outputs) == 0 || len(definition.ImplementationRequired) == 0 {
		t.Fatalf("capability lacks output or proof boundary: %#v", definition)
	}
	if !ValidRelative(definition.ConsumerImport) || !strings.HasPrefix(definition.ConsumerImport, "internal/platform/") {
		t.Fatalf("invalid consumer import: %#v", definition)
	}
	for _, requirement := range definition.Requires {
		if err := ValidateCapability(requirement); err != nil {
			t.Fatalf("capability %s has invalid requirement: %v", definition.ID, err)
		}
	}
}

func recordCapabilityOutputs(t *testing.T, definition CapabilityDefinition, outputs map[string]string) {
	t.Helper()
	for _, output := range definition.Outputs {
		if !ValidRelative(output.Path) || output.Template == "" {
			t.Fatalf("invalid capability output %#v", output)
		}
		if owner, exists := outputs[output.Path]; exists {
			t.Fatalf("output %s overlaps %s and %s", output.Path, owner, definition.ID)
		}
		outputs[output.Path] = definition.ID
	}
}

func recordCapabilityExtensions(t *testing.T, definition CapabilityDefinition, roots map[string]string) {
	t.Helper()
	for _, extension := range definition.Extensions {
		validateCapabilityExtension(t, definition, extension)
		for root, owner := range roots {
			if extension.Root == root || strings.HasPrefix(extension.Root, root+"/") || strings.HasPrefix(root, extension.Root+"/") {
				t.Fatalf("extension roots %s (%s) and %s (%s) overlap", root, owner, extension.Root, definition.ID)
			}
		}
		roots[extension.Root] = definition.ID
	}
}

func validateCapabilityExtension(t *testing.T, definition CapabilityDefinition, extension CapabilityExtension) {
	t.Helper()
	if !ValidRelative(extension.Root) || !strings.HasPrefix(extension.Root, definition.ConsumerImport+"/") || extension.Description == "" {
		t.Fatalf("invalid capability extension %#v", extension)
	}
	for _, importPath := range extension.ExternalImports {
		if strings.TrimSpace(importPath) != importPath || !strings.Contains(strings.Split(importPath, "/")[0], ".") {
			t.Fatalf("invalid external import family %q", importPath)
		}
	}
}

func TestRecipeCatalogIsValid(t *testing.T) {
	recipes := RecipeCatalog()
	if len(recipes) != 2 {
		t.Fatalf("got %d recipes", len(recipes))
	}
	for _, recipe := range recipes {
		if recipe.Name == "" || recipe.Version < 1 || len(recipe.Capabilities) == 0 {
			t.Fatalf("invalid recipe %#v", recipe)
		}
		for _, capability := range recipe.Capabilities {
			if err := ValidateCapability(capability); err != nil {
				t.Fatalf("recipe %s: %v", recipe.Name, err)
			}
		}
	}
}

func TestCatalogAccessorsReturnDeepCopies(t *testing.T) {
	capabilities := CapabilityCatalog()
	recipes := RecipeCatalog()
	capabilities[0].Outputs[0].Path = "mutated"
	capabilities[len(capabilities)-1].Requires[0].ID = "mutated"
	capabilities[0].Extensions[0].ExternalImports[0] = "mutated"
	capabilities[0].ImplementationRequired[0] = "mutated"
	recipes[0].Capabilities[0].ID = "mutated"

	freshCapabilities := CapabilityCatalog()
	freshRecipes := RecipeCatalog()
	if freshCapabilities[0].Outputs[0].Path == "mutated" || freshCapabilities[0].Extensions[0].ExternalImports[0] == "mutated" {
		t.Fatal("catalog accessor exposed mutable output or extension state")
	}
	if freshCapabilities[0].ImplementationRequired[0] == "mutated" || freshCapabilities[len(capabilities)-1].Requires[0].ID == "mutated" {
		t.Fatal("catalog accessor exposed mutable proof or dependency state")
	}
	if freshRecipes[0].Capabilities[0].ID == "mutated" {
		t.Fatal("recipe accessor exposed mutable state")
	}
}

func TestCatalogJSONUsesExplicitArrays(t *testing.T) {
	encoded := string(JSON(CapabilityCatalog()))
	if strings.Contains(encoded, `"external_imports": null`) {
		t.Fatal("catalog emitted an implicit external-import allowlist")
	}
	if strings.Contains(encoded, `"requires": null`) {
		t.Fatal("catalog emitted implicit capability dependencies")
	}
}

func TestApplyRecipeIsAtomicIdempotentAndDerived(t *testing.T) {
	manifest := New("identity-provider", "example.com/identity-provider")
	applied, application, err := ApplyRecipe(manifest, "identity-provider-orchestrator")
	if err != nil {
		t.Fatal(err)
	}
	if len(application.Capabilities) != 6 || len(applied.Capabilities) != 6 {
		t.Fatalf("unexpected application %#v", application)
	}
	if len(applied.Modules) != 0 {
		t.Fatal("recipe encoded business modules instead of capability-owned output")
	}
	if len(manifest.Capabilities) != 0 {
		t.Fatal("recipe mutated its input")
	}
	again, _, err := ApplyRecipe(applied, "identity-provider-orchestrator")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(applied, again) {
		t.Fatal("identical recipe application changed the manifest")
	}
	if got := DeclaredRecipeCompositions(applied); len(got) != 2 {
		t.Fatalf("expected composite and subset recipe declarations, got %#v", got)
	}
	for _, assessment := range AssessCapabilities(applied) {
		if assessment.Status != "declared" {
			t.Fatalf("capability %s was overstated or missing: %s", assessment.ID, assessment.Status)
		}
	}
}

func TestApplyRecipeRejectsUnknownAndVersionMismatchWithoutMutation(t *testing.T) {
	manifest := New("demo", "example.com/demo")
	for _, tc := range []struct {
		name string
		code string
	}{
		{name: "missing", code: "RECIPE001"},
	} {
		_, _, err := ApplyRecipe(manifest, tc.name)
		var failure *Error
		if !errors.As(err, &failure) || failure.Code != tc.code {
			t.Fatalf("got %v, want %s", err, tc.code)
		}
		if len(manifest.Capabilities) != 0 {
			t.Fatal("failed recipe mutated its input")
		}
	}

	manifest.Capabilities = []Capability{{ID: "eventing.transactional-inbox", Version: 2}}
	before := cloneManifest(manifest)
	_, _, err := ApplyRecipe(manifest, "projection-worker")
	var failure *Error
	if !errors.As(err, &failure) || failure.Code != "RECIPE002" {
		t.Fatalf("got %v, want RECIPE002", err)
	}
	if !reflect.DeepEqual(manifest, before) {
		t.Fatal("version conflict mutated its input")
	}
}
