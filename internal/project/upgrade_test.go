package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadForUpgradeAcceptsOnlyVerifiedV1OrCurrent(t *testing.T) {
	root := t.TempDir()
	legacy := legacyManifestV1{
		SchemaVersion: legacySchemaV1,
		ToolVersion:   legacyVersionV1,
		Policy:        legacyPolicyV1,
		Service:       "demo",
		GoModule:      "example.com/demo",
		Modules:       []Module{},
	}
	manifestBytes := JSON(legacy)
	state := State{
		SchemaVersion:  legacySchemaV1,
		ToolVersion:    legacyVersionV1,
		ManifestSHA256: Hash(manifestBytes),
		Files:          map[string]Ownership{"README.md": {Owner: "developer"}},
	}
	if err := os.MkdirAll(filepath.Join(root, ".gsvc"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ManifestPath), manifestBytes, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, StatePath), JSON(state), 0644); err != nil {
		t.Fatal(err)
	}
	loaded, current, err := LoadForUpgrade(root)
	if err != nil {
		t.Fatal(err)
	}
	if current || loaded.Manifest.SchemaVersion != SchemaVersion || loaded.Manifest.Capabilities == nil {
		t.Fatalf("unexpected upgrade load %#v current=%v", loaded.Manifest, current)
	}

	state.ManifestSHA256 = "tampered"
	if err = os.WriteFile(filepath.Join(root, StatePath), JSON(state), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err = LoadForUpgrade(root); err == nil {
		t.Fatal("accepted mismatched legacy ownership metadata")
	}
}
