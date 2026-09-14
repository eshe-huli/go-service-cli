// Package check enforces import-aware AST rules and canonical generated output.
// It does not pretend that static layout checks prove business correctness.
package check

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path"
	"sort"
	"strconv"
	"strings"

	"gsvc.local/cli/internal/generate"
	"gsvc.local/cli/internal/project"
)

type Diagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Message  string `json:"message"`
	Hint     string `json:"hint,omitempty"`
}
type Report struct {
	OK           bool         `json:"ok"`
	FilesChecked int          `json:"files_checked"`
	Errors       int          `json:"errors"`
	Warnings     int          `json:"warnings"`
	Diagnostics  []Diagnostic `json:"diagnostics"`
}

func (r *Report) Add(code, severity, file string, line int, message, hint string) {
	r.Diagnostics = append(r.Diagnostics, Diagnostic{code, severity, file, line, message, hint})
}
func (r *Report) Finish() {
	sort.SliceStable(r.Diagnostics, func(i, j int) bool {
		a, b := r.Diagnostics[i], r.Diagnostics[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Code < b.Code
	})
	r.Errors, r.Warnings = 0, 0
	for _, d := range r.Diagnostics {
		if d.Severity == "error" {
			r.Errors++
		} else {
			r.Warnings++
		}
	}
	r.OK = r.Errors == 0
}

type location struct{ layer, module, adapter, source string }

const layerCapabilityExtension = "capability-extension"

func locate(p string, m project.Manifest) (location, bool) {
	dir := path.Dir(p)
	if dir == "cmd/api" {
		return location{layer: "entry", source: p}, true
	}
	if dir == "internal/bootstrap" {
		return location{layer: "bootstrap", source: p}, true
	}
	if _, ok := project.CapabilityExtensionForPath(m, p); ok {
		return location{layer: layerCapabilityExtension, source: p}, true
	}
	if strings.HasPrefix(dir, "internal/platform/") {
		return location{layer: "platform", source: p}, true
	}
	for _, mod := range m.Modules {
		root := "internal/" + mod.Name
		if dir == root {
			return location{layer: "facade", module: mod.Name, source: p}, true
		}
		for _, layer := range []string{"app", "domain"} {
			prefix := root + "/internal/" + layer
			if dir == prefix || strings.HasPrefix(dir, prefix+"/") {
				return location{layer: layer, module: mod.Name, source: p}, true
			}
		}
		for _, adapter := range []string{"httpapi", "postgres"} {
			prefix := root + "/internal/adapter/" + adapter
			if dir == prefix || strings.HasPrefix(dir, prefix+"/") {
				return location{layer: "adapter", module: mod.Name, adapter: adapter, source: p}, true
			}
		}
	}
	return location{}, false
}

var pure = map[string]bool{
	"bytes": true, "cmp": true, "errors": true, "fmt": true, "math": true, "math/big": true, "math/bits": true,
	"regexp": true, "slices": true, "sort": true, "strconv": true, "strings": true, "time": true,
	"unicode": true, "unicode/utf8": true, "unicode/utf16": true, "net/url": true,
	"encoding/hex": true, "encoding/base64": true,
}

func importAllowed(imp string, loc location, m project.Manifest, test bool) (bool, string) {
	if imp == "unsafe" || imp == "C" || imp == "plugin" {
		return false, "unsafe, cgo, and plugin imports are outside this policy"
	}
	if !strings.Contains(strings.Split(imp, "/")[0], ".") {
		// The Go compiler verifies the package actually exists during --verify.
		if loc.layer == "domain" || loc.layer == "app" {
			if pure[imp] || (loc.layer == "app" && imp == "context") || (test && (imp == "testing" || imp == "testing/quick")) {
				return true, ""
			}
			return false, loc.layer + " cannot depend on standard-library I/O or operational package " + imp
		}
		return true, ""
	}
	own := m.GoModule + "/"
	if !strings.HasPrefix(imp, own) {
		pgx := imp == "github.com/jackc/pgx/v5" || strings.HasPrefix(imp, "github.com/jackc/pgx/v5/")
		if pgx && ((loc.layer == "adapter" && loc.adapter == "postgres") || loc.layer == "bootstrap" || loc.layer == "facade") {
			return true, ""
		}
		if loc.layer == layerCapabilityExtension && project.CapabilityExternalImportAllowed(m, loc.source, imp) {
			return true, ""
		}
		return false, "unapproved external import: " + imp
	}
	rel := strings.TrimPrefix(imp, own)
	if loc.layer == "platform" || loc.layer == layerCapabilityExtension {
		return strings.HasPrefix(rel, "internal/platform/"), "platform helpers cannot depend on business code"
	}
	if loc.layer == "entry" {
		return rel == "internal/bootstrap", "entrypoint may import only the bootstrap project package"
	}
	if loc.layer == "bootstrap" {
		if strings.HasPrefix(rel, "internal/platform/") {
			return true, ""
		}
		for _, mod := range m.Modules {
			if rel == "internal/"+mod.Name {
				return true, ""
			}
		}
		return false, "bootstrap must use module facades, not module internals"
	}
	if rel == "internal/platform/fault" || rel == "internal/platform/validate" {
		return loc.layer != "domain", "domain does not import framework/application helpers"
	}
	if rel == "internal/platform/httpx" {
		return loc.layer == "adapter" && loc.adapter == "httpapi", "HTTP helpers belong only in the HTTP adapter"
	}
	if loc.layer == "app" && project.CapabilityConsumerImportAllowed(m, rel) {
		return true, ""
	}
	root := "internal/" + loc.module + "/internal/"
	if !strings.HasPrefix(rel, root) {
		return false, "direct peer-module or shared-package dependency; inject a consumer-owned capability instead"
	}
	target := strings.TrimPrefix(rel, root)
	if loc.layer == "facade" {
		return true, ""
	}
	is := func(prefix string) bool { return target == prefix || strings.HasPrefix(target, prefix+"/") }
	switch loc.layer {
	case "domain":
		return is("domain"), "domain cannot depend on application or adapters"
	case "app":
		return is("domain") || is("app"), "application cannot depend on an adapter"
	case "adapter":
		return is("domain") || is("app") || is("adapter/"+loc.adapter), "one adapter cannot import another adapter"
	}
	return false, "unsupported dependency"
}
func Run(l project.Loaded, strict bool) (Report, error) {
	r := Report{Diagnostics: []Diagnostic{}}
	desired, err := generate.Render(l.Manifest)
	if err != nil {
		return r, err
	}
	if _, exists, err := project.Read(l.Root, ".gsvc/transaction.json"); err != nil {
		return r, err
	} else if exists {
		r.Add("TXN001", "error", ".gsvc/transaction.json", 0, "unfinished write transaction", "Run gsvc recover.")
	}
	for p, f := range desired {
		b, exists, err := project.Read(l.Root, p)
		if err != nil {
			return r, err
		}
		o, tracked := l.State.Files[p]
		if !tracked || o.Owner != f.Owner {
			r.Add("OWN001", "error", p, 0, "ownership does not match the canonical model", "Restore ownership metadata from version control.")
		}
		if !exists {
			r.Add("OWN003", "error", p, 0, "required file is missing", "Restore developer-owned files; gsvc sync can restore missing managed output.")
			continue
		}
		if f.Owner == "generated" {
			if !bytes.Equal(b, f.Content) {
				r.Add("GEN001", "error", p, 0, "generated content differs from canonical output", "Restore this managed file; implement behavior in developer-owned code.")
			}
			if tracked && o.SHA256 != project.Hash(b) {
				r.Add("OWN004", "error", p, 0, "generated file hash differs from recorded ownership", "Do not reset hashes to accept changes.")
			}
		}
	}
	for p := range l.State.Files {
		if _, ok := desired[p]; !ok {
			r.Add("OWN005", "error", p, 0, "ownership entry is not in the model", "")
		}
	}
	for _, definition := range project.DeclaredCapabilityDefinitions(l.Manifest) {
		r.Add(
			"CAPABILITY004",
			"warning",
			project.ManifestPath,
			0,
			fmt.Sprintf("%s@%d is structurally declared; runtime proof remains outside the gsvc source gate", definition.ID, definition.Version),
			strings.Join(definition.ImplementationRequired, "; "),
		)
	}
	paths, err := project.SourceFiles(l.Root)
	if err != nil {
		return r, err
	}
	foundExec := map[string]bool{}
	for _, p := range paths {
		if p == "go.mod" {
			b, _, err := project.Read(l.Root, p)
			if err != nil {
				return r, err
			}
			match := false
			for _, line := range strings.Split(string(b), "\n") {
				f := strings.Fields(line)
				if len(f) >= 2 && f[0] == "module" {
					v := f[1]
					if u, err := strconv.Unquote(v); err == nil {
						v = u
					}
					match = v == l.Manifest.GoModule
				}
				if len(f) > 0 && f[0] == "replace" {
					r.Add("DEP002", "error", p, 0, "module replacements require a reviewed policy extension", "")
				}
			}
			if !match {
				r.Add("CONFIG006", "error", p, 0, "go.mod module differs from gsvc.json", "")
			}
			continue
		}
		if strings.HasSuffix(p, "/go.mod") {
			r.Add("LAYOUT002", "error", p, 0, "nested Go modules are not allowed", "")
			continue
		}
		r.FilesChecked++
		b, _, err := project.Read(l.Root, p)
		if err != nil {
			return r, err
		}
		loc, ok := locate(p, l.Manifest)
		if !ok {
			r.Add("LAYOUT001", "error", p, 0, "Go source is outside the approved layout", "Create business modules with gsvc add module.")
			continue
		}
		file, managed := desired[p]
		managed = managed && file.Owner == "generated"
		if loc.layer == "platform" && !managed {
			r.Add("LAYOUT003", "error", p, 0, "new shared runtime code requires a CLI extension", "Keep one versioned runtime, not per-service utility variants.")
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, p, b, parser.AllErrors|parser.ParseComments)
		if err != nil {
			r.Add("GO001", "error", p, 0, err.Error(), "Fix syntax before continuing.")
			continue
		}
		formatted, err := format.Source(b)
		if err == nil && !bytes.Equal(b, formatted) {
			r.Add("FMT001", "error", p, 0, "source is not gofmt formatted", "Run gofmt on developer-owned files.")
		}
		test := strings.HasSuffix(p, "_test.go")
		aliases := map[string]string{}
		for _, im := range f.Imports {
			imp, err := strconv.Unquote(im.Path.Value)
			if err != nil {
				continue
			}
			alias := path.Base(imp)
			if im.Name != nil {
				alias = im.Name.Name
			}
			aliases[alias] = imp
			if alias == "." {
				r.Add("ARCH004", "error", p, fset.Position(im.Pos()).Line, "dot imports obscure dependency ownership", "")
			}
			if yes, reason := importAllowed(imp, loc, l.Manifest, test); !yes {
				r.Add("ARCH001", "error", p, fset.Position(im.Pos()).Line, reason, "Follow inward dependencies and consumer-owned interfaces.")
			}
		}
		if strings.Contains(string(b), "gsvc:pending") {
			severity := "warning"
			if strict {
				severity = "error"
			}
			r.Add("WORK001", severity, p, 0, "pending scaffold behavior or tests", "Implement the behavior and real business assertions; do not merely delete the marker.")
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.FuncDecl:
				if node.Name.Name == "init" && node.Recv == nil {
					r.Add("ARCH002", "error", p, fset.Position(node.Pos()).Line, "implicit init() wiring is prohibited", "Use explicit constructors and bootstrap.Wire.")
				}
				if loc.layer == "app" && !test && node.Name.Name == "Execute" && node.Recv != nil {
					receiver := node.Recv.List[0].Type
					if star, ok := receiver.(*ast.StarExpr); ok {
						receiver = star.X
					}
					id, ok := receiver.(*ast.Ident)
					if !ok {
						break
					}
					registered := false
					mod := l.Manifest.FindModule(loc.module)
					for _, op := range mod.Operations {
						if project.Export(op.Name) == id.Name {
							registered = true
							foundExec[loc.module+"/"+op.Name] = true
						}
					}
					if !registered {
						r.Add("ARCH003", "error", p, fset.Position(node.Pos()).Line, "Execute receiver is not a registered operation", "Use gsvc add command or gsvc add query.")
					}
					valid := false
					if len(node.Type.Params.List) > 0 {
						if sel, ok := node.Type.Params.List[0].Type.(*ast.SelectorExpr); ok {
							if pkg, ok := sel.X.(*ast.Ident); ok {
								valid = aliases[pkg.Name] == "context" && sel.Sel.Name == "Context"
							}
						}
					}
					if !valid {
						r.Add("ARCH005", "error", p, fset.Position(node.Pos()).Line, "Execute requires context.Context as its first parameter", "")
					}
				}
			case *ast.CallExpr:
				if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
					if test && (sel.Sel.Name == "Skip" || sel.Sel.Name == "Skipf" || sel.Sel.Name == "SkipNow") {
						severity := "warning"
						if strict {
							severity = "error"
						}
						r.Add("WORK002", severity, p, fset.Position(node.Pos()).Line, "test skip call prevents a complete verification gate", "Replace scaffold skips with assertions. This strict profile does not permit skipped tests.")
					}
					if id, ok := sel.X.(*ast.Ident); ok && aliases[id.Name] == "net/http" && sel.Sel.Name == "NewServeMux" && !managed && !test {
						r.Add("ARCH006", "error", p, fset.Position(node.Pos()).Line, "route registries are generated, not manually created", "Use gsvc add endpoint.")
					}
				}
			case *ast.SelectorExpr:
				if loc.layer == "app" && !test && node.Sel.Name == "ErrNotImplemented" {
					if id, ok := node.X.(*ast.Ident); ok && aliases[id.Name] == l.Manifest.GoModule+"/internal/platform/fault" {
						severity := "warning"
						if strict {
							severity = "error"
						}
						r.Add("WORK003", severity, p, fset.Position(node.Pos()).Line, "operation still returns the not-implemented sentinel", "Implement the operation before declaring it complete.")
					}
				}
			}
			return true
		})
	}
	for _, mod := range l.Manifest.Modules {
		for _, op := range mod.Operations {
			if !foundExec[mod.Name+"/"+op.Name] {
				r.Add("ARCH007", "error", "internal/"+mod.Name+"/internal/app/"+project.Stem(op.Name)+".go", 0, fmt.Sprintf("registered operation %s has no Execute method", op.Name), "")
			}
		}
	}
	r.Finish()
	return r, nil
}
