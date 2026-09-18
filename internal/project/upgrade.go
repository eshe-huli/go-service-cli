package project

import (
	"encoding/json"
	"strings"
)

const legacyVersionV1 = "0.1.0"
const legacySchemaV1 = 1
const legacyPolicyV1 = "go-service/v1"

type legacyManifestV1 struct {
	SchemaVersion int      `json:"schema_version"`
	ToolVersion   string   `json:"tool_version"`
	Policy        string   `json:"policy"`
	Service       string   `json:"service"`
	GoModule      string   `json:"go_module"`
	Modules       []Module `json:"modules"`
}

type versionProbe struct {
	SchemaVersion int    `json:"schema_version"`
	ToolVersion   string `json:"tool_version"`
	Policy        string `json:"policy"`
}

// LoadForUpgrade accepts the current format or the one explicitly supported
// v0.1 format. It validates old ownership metadata before returning a v0.2
// manifest for the normal guarded Render -> BuildPlan -> Apply workflow.
func LoadForUpgrade(root string) (Loaded, bool, error) {
	b, ok, err := Read(root, ManifestPath)
	if err != nil {
		return Loaded{}, false, err
	}
	if !ok {
		return Loaded{}, false, Fail("PROJECT001", "gsvc.json is missing", "")
	}
	var probe versionProbe
	if err = json.Unmarshal(b, &probe); err != nil {
		return Loaded{}, false, Fail("CONFIG005", err.Error(), "Fix malformed configuration; defaults are never silently substituted.")
	}
	if probe.SchemaVersion == SchemaVersion && probe.ToolVersion == Version && probe.Policy == Policy {
		loaded, loadErr := Load(root)
		return loaded, true, loadErr
	}
	if probe.SchemaVersion != legacySchemaV1 || probe.ToolVersion != legacyVersionV1 || probe.Policy != legacyPolicyV1 {
		return Loaded{}, false, Fail("UPGRADE001", "unsupported project contract version", "This CLI upgrades only go-service/v1 from gsvc 0.1.0.")
	}

	var legacy legacyManifestV1
	if err = DecodeStrict(b, &legacy); err != nil {
		return Loaded{}, false, Fail("CONFIG005", err.Error(), "The v0.1 manifest must not contain undeclared fields.")
	}
	manifest := Manifest{
		SchemaVersion: SchemaVersion,
		ToolVersion:   Version,
		Policy:        Policy,
		Service:       legacy.Service,
		GoModule:      legacy.GoModule,
		Capabilities:  []Capability{},
		Modules:       legacy.Modules,
	}
	Normalize(&manifest)
	if err = Validate(manifest); err != nil {
		return Loaded{}, false, err
	}

	state, err := loadLegacyOwnershipState(root, b)
	if err != nil {
		return Loaded{}, false, err
	}
	return Loaded{Root: root, Manifest: manifest, State: state}, false, nil
}

func loadLegacyOwnershipState(root string, manifestBytes []byte) (State, error) {
	stateBytes, ok, err := Read(root, StatePath)
	if err != nil {
		return State{}, err
	}
	if !ok {
		return State{}, Fail("STATE001", "ownership state is missing", "Restore .gsvc/ownership.json from version control.")
	}
	var state State
	if err = DecodeStrict(stateBytes, &state); err != nil {
		return State{}, Fail("STATE001", err.Error(), "")
	}
	if state.SchemaVersion != legacySchemaV1 || state.ToolVersion != legacyVersionV1 || state.ManifestSHA256 != Hash(manifestBytes) {
		return State{}, Fail("STATE002", "v0.1 manifest or ownership state was edited outside the CLI", "Restore both files from version control before upgrading.")
	}
	if len(state.Files) == 0 {
		return State{}, Fail("STATE003", "ownership state contains no files", "")
	}
	for path, ownership := range state.Files {
		if !validLegacyOwnership(path, ownership) {
			return State{}, Fail("STATE003", "invalid ownership entry: "+path, "")
		}
	}
	return state, nil
}

func validLegacyOwnership(filePath string, ownership Ownership) bool {
	if !ValidRelative(filePath) || filePath == ManifestPath || strings.HasPrefix(filePath, ".gsvc/") {
		return false
	}
	if ownership.Owner != "generated" && ownership.Owner != "developer" {
		return false
	}
	return ownership.Owner != "generated" || len(ownership.SHA256) == 64
}
