// Package project defines the versioned service contract. Generation and checking
// both consume this model; there is no second, independently configured layout.
package project

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/token"
	"io"
	"net/http"
	"path"
	"regexp"
	"sort"
	"strings"
)

const Version = "0.2.0"
const SchemaVersion = 2
const Policy = "go-service/v2"
const ManifestPath = "gsvc.json"
const StatePath = ".gsvc/ownership.json"

// Error is a stable, machine-readable failure, not a panic or a prompt.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

func (e *Error) Error() string              { return e.Code + ": " + e.Message }
func Fail(code, message, hint string) error { return &Error{code, message, hint} }

type Manifest struct {
	SchemaVersion int          `json:"schema_version"`
	ToolVersion   string       `json:"tool_version"`
	Policy        string       `json:"policy"`
	Service       string       `json:"service"`
	GoModule      string       `json:"go_module"`
	Capabilities  []Capability `json:"capabilities"`
	Modules       []Module     `json:"modules"`
}
type Capability struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
}
type Module struct {
	Name       string      `json:"name"`
	Operations []Operation `json:"operations"`
}
type Operation struct {
	Name     string    `json:"name"`
	Kind     string    `json:"kind"`
	Input    []Field   `json:"input"`
	Output   []Field   `json:"output"`
	Endpoint *Endpoint `json:"endpoint,omitempty"`
}
type Field struct {
	Name string `json:"name"`
	Type string `json:"type"`
}
type Endpoint struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}
type Ownership struct {
	Owner  string `json:"owner"`
	SHA256 string `json:"sha256,omitempty"`
}
type State struct {
	SchemaVersion  int                  `json:"schema_version"`
	ToolVersion    string               `json:"tool_version"`
	ManifestSHA256 string               `json:"manifest_sha256"`
	Files          map[string]Ownership `json:"files"`
}

var kebab = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
var pkgName = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
var fieldName = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)
var modulePath = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._~/-]*$`)
var literalSegment = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func New(name, module string) Manifest {
	return Manifest{SchemaVersion: SchemaVersion, ToolVersion: Version, Policy: Policy, Service: name, GoModule: module, Capabilities: []Capability{}, Modules: []Module{}}
}
func DecodeStrict(data []byte, out any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return fmt.Errorf("expected exactly one JSON document")
	}
	return nil
}
func JSON(v any) []byte {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return append(b, '\n')
}
func Normalize(m *Manifest) {
	if m.Capabilities == nil {
		m.Capabilities = []Capability{}
	}
	sort.Slice(m.Capabilities, func(i, j int) bool { return m.Capabilities[i].ID < m.Capabilities[j].ID })
	if m.Modules == nil {
		m.Modules = []Module{}
	}
	sort.Slice(m.Modules, func(i, j int) bool { return m.Modules[i].Name < m.Modules[j].Name })
	for i := range m.Modules {
		if m.Modules[i].Operations == nil {
			m.Modules[i].Operations = []Operation{}
		}
		sort.Slice(m.Modules[i].Operations, func(a, b int) bool { return m.Modules[i].Operations[a].Name < m.Modules[i].Operations[b].Name })
		for j := range m.Modules[i].Operations {
			op := &m.Modules[i].Operations[j]
			if op.Input == nil {
				op.Input = []Field{}
			}
			if op.Output == nil {
				op.Output = []Field{}
			}
		}
	}
}
func ValidateName(name string) error {
	if !kebab.MatchString(name) {
		return Fail("INPUT001", "name must be lower-kebab-case: "+name, "Example: create-payment")
	}
	return nil
}
func ValidateOperationName(name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	if name == "doc" || name == "test" || name == "gen" || strings.HasSuffix(name, "-test") || strings.HasSuffix(name, "-gen") {
		return Fail("INPUT006", "operation name collides with reserved Go file suffixes: "+name, "Do not use doc or names ending in -test or -gen.")
	}
	return nil
}
func ValidateModuleName(name string) error {
	if !pkgName.MatchString(name) || token.Lookup(name).IsKeyword() || name == "init" || name == "main" || name == "internal" || name == "bootstrap" || name == "platform" || name == "vendor" || name == "testdata" || name == "http" || name == "httpx" {
		return Fail("INPUT002", "invalid or reserved business module name: "+name, "Use one lowercase Go package name, such as payments.")
	}
	return nil
}
func ValidateGoModule(s string) error {
	if !modulePath.MatchString(s) || !strings.Contains(strings.Split(s, "/")[0], ".") {
		return Fail("INPUT003", "invalid Go module path: "+s, "Example: github.com/your-org/payments-api")
	}
	for _, p := range strings.Split(s, "/") {
		if p == "" || p == "." || p == ".." || strings.HasSuffix(p, ".") {
			return Fail("INPUT003", "invalid Go module path: "+s, "")
		}
	}
	return nil
}
func Validate(m Manifest) (err error) {
	if m.SchemaVersion != SchemaVersion || m.ToolVersion != Version || m.Policy != Policy {
		return Fail("CONFIG001", "unsupported schema, CLI version, or architecture policy", "Use the CLI version pinned in gsvc.json; upgrades are explicit.")
	}
	if err = ValidateName(m.Service); err != nil {
		return err
	}
	if err = ValidateGoModule(m.GoModule); err != nil {
		return err
	}
	if m.Capabilities == nil {
		return Fail("CAPABILITY001", "capabilities must be an explicit array", "Upgrade the project with gsvc upgrade before using this CLI version.")
	}
	capabilities := map[string]bool{}
	for _, capability := range m.Capabilities {
		if capabilities[capability.ID] {
			return Fail("CAPABILITY003", "duplicate capability: "+capability.ID, "")
		}
		if err = ValidateCapability(capability); err != nil {
			return err
		}
		capabilities[capability.ID] = true
	}
	if err = ValidateCapabilityDependencies(m.Capabilities); err != nil {
		return err
	}
	seen := map[string]bool{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(http.ResponseWriter, *http.Request) {})
	defer func() {
		if r := recover(); r != nil {
			err = Fail("ROUTE001", fmt.Sprint(r), "Choose a non-conflicting endpoint path.")
		}
	}()
	for _, mod := range m.Modules {
		if err = ValidateModuleName(mod.Name); err != nil {
			return err
		}
		if seen[mod.Name] {
			return Fail("CONFIG002", "duplicate module: "+mod.Name, "")
		}
		seen[mod.Name] = true
		names := map[string]bool{}
		for _, op := range mod.Operations {
			if err = ValidateOperationName(op.Name); err != nil {
				return err
			}
			if names[Export(op.Name)] {
				return Fail("CONFIG003", "duplicate operation or Go name collision: "+op.Name, "")
			}
			names[Export(op.Name)] = true
			if op.Kind != "command" && op.Kind != "query" {
				return Fail("CONFIG004", "operation kind must be command or query", "")
			}
			if err = ValidateFields(op.Input); err != nil {
				return err
			}
			if err = ValidateFields(op.Output); err != nil {
				return err
			}
			if op.Endpoint != nil {
				want := "POST"
				if op.Kind == "query" {
					want = "GET"
				}
				if op.Endpoint.Method != want {
					return Fail("ROUTE002", op.Kind+" endpoints use "+want, "This profile supports POST commands and GET queries only.")
				}
				if err = ValidateRoute(op); err != nil {
					return err
				}
				mux.HandleFunc(op.Endpoint.Method+" "+op.Endpoint.Path, func(http.ResponseWriter, *http.Request) {})
			}
		}
	}
	return nil
}
func ValidateFields(fields []Field) error {
	seen := map[string]bool{}
	for _, f := range fields {
		if !fieldName.MatchString(f.Name) {
			return Fail("INPUT004", "field names must be snake_case: "+f.Name, "")
		}
		goName := Export(f.Name)
		if goName == "Validate" || seen[goName] {
			return Fail("INPUT004", "duplicate or reserved field name: "+f.Name, "")
		}
		seen[goName] = true
		switch f.Type {
		case "string", "money", "bool", "int64":
		default:
			return Fail("INPUT005", "unsupported field type: "+f.Type, "Supported types: string, money (decimal string), bool, int64.")
		}
	}
	return nil
}
func ParseFields(s string) ([]Field, error) {
	fields := []Field{}
	if strings.TrimSpace(s) == "" {
		return fields, nil
	}
	for _, spec := range strings.Split(s, ",") {
		parts := strings.Split(strings.TrimSpace(spec), ":")
		if len(parts) != 2 {
			return nil, Fail("INPUT004", "invalid field: "+spec, "Use name:type,name:type.")
		}
		fields = append(fields, Field{parts[0], parts[1]})
	}
	return fields, ValidateFields(fields)
}
func ValidateRoute(op Operation) error {
	p := op.Endpoint.Path
	if p == "/healthz" {
		return Fail("ROUTE003", "the /healthz path is reserved", "")
	}
	if !strings.HasPrefix(p, "/") || p == "/" || strings.HasSuffix(p, "/") || path.Clean(p) != p {
		return Fail("ROUTE003", "endpoint must be a clean absolute path without a trailing slash: "+p, "")
	}
	inputs := map[string]Field{}
	for _, f := range op.Input {
		inputs[f.Name] = f
	}
	vars := map[string]bool{}
	for _, s := range strings.Split(p[1:], "/") {
		if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
			n := strings.TrimSuffix(strings.TrimPrefix(s, "{"), "}")
			f, ok := inputs[n]
			if op.Kind != "query" || !ok || f.Type != "string" || vars[n] {
				return Fail("ROUTE004", "path variables require unique string inputs on a query: "+s, "")
			}
			vars[n] = true
		} else if !literalSegment.MatchString(s) {
			return Fail("ROUTE003", "unsupported route segment: "+s, "Use literal segments and query string variables like /payments/{id}.")
		}
	}
	return nil
}
func Export(s string) string {
	var b strings.Builder
	for _, p := range strings.FieldsFunc(s, func(r rune) bool { return r == '-' || r == '_' }) {
		switch p {
		case "id", "api", "http", "url", "uuid", "json", "sql":
			b.WriteString(strings.ToUpper(p))
		default:
			b.WriteString(strings.ToUpper(p[:1]))
			b.WriteString(p[1:])
		}
	}
	return b.String()
}
func Stem(s string) string { return strings.ReplaceAll(s, "-", "_") }
func GoType(t string) string {
	if t == "money" {
		return "string"
	}
	return t
}
func (m *Manifest) FindModule(name string) *Module {
	for i := range m.Modules {
		if m.Modules[i].Name == name {
			return &m.Modules[i]
		}
	}
	return nil
}
func (m *Module) FindOperation(name string) *Operation {
	for i := range m.Operations {
		if m.Operations[i].Name == name {
			return &m.Operations[i]
		}
	}
	return nil
}
