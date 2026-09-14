// Package cli implements a non-interactive command protocol suitable for agents.
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"gsvc.local/cli/internal/check"
	"gsvc.local/cli/internal/generate"
	"gsvc.local/cli/internal/project"
)

type envelope struct {
	SchemaVersion int            `json:"schema_version"`
	Tool          string         `json:"tool"`
	Version       string         `json:"version"`
	OK            bool           `json:"ok"`
	Command       string         `json:"command"`
	Data          any            `json:"data,omitempty"`
	Error         *project.Error `json:"error,omitempty"`
}

func wantsJSON(args []string) bool {
	for _, a := range args {
		if a == "--json" || a == "--json=true" {
			return true
		}
	}
	return false
}
func Run(ctx context.Context, args []string, out, errout io.Writer) int {
	p, err := parse(args)
	jsonMode := wantsJSON(args)
	if err == nil && p.Help && !jsonMode {
		fmt.Fprint(out, help(p))
		return 0
	}
	var value any
	exit := 0
	if err == nil {
		if p.Help {
			value = contract()
		} else {
			value, exit, err = execute(ctx, p)
		}
	}
	var failure *project.Error
	if err != nil {
		if !errors.As(err, &failure) {
			failure = &project.Error{Code: "EXEC001", Message: err.Error()}
		}
		exit = 2
		if strings.HasPrefix(failure.Code, "EXEC") {
			exit = 1
		}
		for _, prefix := range []string{"OWN", "STATE", "PLAN", "LOCK", "TXN", "PATH", "PROJECT002"} {
			if strings.HasPrefix(failure.Code, prefix) {
				exit = 3
			}
		}
		if failure.Code == "RECIPE002" {
			exit = 3
		}
	}
	if jsonMode {
		e := envelope{SchemaVersion: 1, Tool: "gsvc", Version: project.Version, OK: exit == 0, Command: p.Spec.Name, Data: value, Error: failure}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if enc.Encode(e) != nil {
			return 1
		}
		return exit
	}
	if failure != nil {
		fmt.Fprintf(errout, "%s: %s\n", failure.Code, failure.Message)
		if failure.Hint != "" {
			fmt.Fprintln(errout, failure.Hint)
		}
		return exit
	}
	switch v := value.(type) {
	case mutationResult:
		action := "Applied"
		if v.DryRun {
			action = "Planned"
		}
		fmt.Fprintf(out, "%s %s\nPlan %s\n", action, v.Plan.Root, v.Plan.ID)
		if v.Recipe != nil {
			fmt.Fprintf(out, "Recipe %s@%d (%d capability contracts)\n", v.Recipe.Name, v.Recipe.Version, len(v.Recipe.Capabilities))
		}
		count := 0
		for _, c := range v.Plan.Changes {
			if c.Action == "create" || c.Action == "update" {
				count++
				fmt.Fprintf(out, "  %-7s %-9s %s\n", c.Action, c.Owner, c.Path)
			}
		}
		if count == 0 {
			fmt.Fprintln(out, "No changes. Developer-owned implementation preserved.")
		}
	case verificationReport:
		for _, d := range v.Check.Diagnostics {
			fmt.Fprintf(out, "%s %s %s:%d %s\n", d.Severity, d.Code, d.File, d.Line, d.Message)
		}
		fmt.Fprintf(out, "%d files checked; %d errors; %d warnings\n", v.Check.FilesChecked, v.Check.Errors, v.Check.Warnings)
		for _, run := range v.Verification {
			fmt.Fprintf(out, "%s: exit %d\n%s", strings.Join(run.Command, " "), run.ExitCode, run.Output)
		}
	default:
		b, _ := json.MarshalIndent(v, "", "  ")
		fmt.Fprintln(out, string(b))
	}
	return exit
}

type mutationResult struct {
	DryRun bool                       `json:"dry_run"`
	Plan   project.Plan               `json:"plan"`
	Recipe *project.RecipeApplication `json:"recipe,omitempty"`
}
type execution struct {
	Command   []string `json:"command"`
	ExitCode  int      `json:"exit_code"`
	Output    string   `json:"output"`
	Truncated bool     `json:"truncated"`
}
type verificationReport struct {
	Check        check.Report `json:"check"`
	Verification []execution  `json:"verification"`
}

func execute(ctx context.Context, p parsed) (any, int, error) {
	if err := ctx.Err(); err != nil {
		return nil, 1, err
	}
	switch p.Spec.Name {
	case "version":
		return map[string]any{"version": project.Version, "policy": project.Policy, "schema_version": project.SchemaVersion}, 0, nil
	case "contract":
		return contract(), 0, nil
	case "recipe":
		if p.Arg == "" {
			return map[string]any{"recipes": project.RecipeCatalog(), "capabilities": project.CapabilityCatalog(), "maturity": project.CapabilityMaturity}, 0, nil
		}
	case "init":
		root, err := project.Root(p.Arg)
		if err != nil {
			return nil, 2, err
		}
		if err = project.EmptyForInit(root); err != nil {
			return nil, 3, err
		}
		name := p.Values["name"]
		if name == "" {
			name = filepath.Base(root)
		}
		m := project.New(name, p.Values["module"])
		return mutate(root, m, nil, p)
	case "recover":
		root := p.Values["root"]
		if root == "" {
			root = "."
		}
		root, err := project.Root(root)
		if err != nil {
			return nil, 2, err
		}
		v, err := project.Recover(root, p.Values["dry-run"] == "true")
		return v, 0, err
	}
	root := p.Values["root"]
	var err error
	if root == "" {
		root, err = project.FindRoot(".")
	} else {
		root, err = project.Root(root)
	}
	if err != nil {
		return nil, 2, err
	}
	if p.Spec.Name == "upgrade" {
		l, _, err := project.LoadForUpgrade(root)
		if err != nil {
			return nil, 2, err
		}
		return mutate(root, l.Manifest, &l.State, p)
	}
	l, err := project.Load(root)
	if err != nil {
		return nil, 2, err
	}
	switch p.Spec.Name {
	case "capabilities":
		contractCheck, err := check.Run(l, false)
		if err != nil {
			return nil, 1, err
		}
		exit := 0
		if !contractCheck.OK {
			exit = 1
		}
		return map[string]any{"root": root, "declared": l.Manifest.Capabilities, "assessments": project.AssessCapabilities(l.Manifest), "declared_recipe_compositions": project.DeclaredRecipeCompositions(l.Manifest), "maturity": project.CapabilityMaturity, "contract_check": contractCheck}, exit, nil
	case "inspect":
		r, err := check.Run(l, false)
		if err != nil {
			return nil, 1, err
		}
		files, err := project.SourceFiles(root)
		if err != nil {
			return nil, 1, err
		}
		return map[string]any{"root": root, "project": l.Manifest, "capabilities": project.AssessCapabilities(l.Manifest), "declared_recipe_compositions": project.DeclaredRecipeCompositions(l.Manifest), "ownership": l.State.Files, "source_files": files, "policy": project.PolicyDescription(), "check": r, "next_commands": []string{"gsvc recipe --json", "gsvc recipe NAME --dry-run --json", "gsvc add command NAME --module MODULE --in ... --out ... --dry-run --json", "gsvc check --verify --strict --json"}}, 0, nil
	case "check":
		report, err := check.Run(l, p.Values["strict"] == "true")
		if err != nil {
			return nil, 1, err
		}
		value := verificationReport{Check: report, Verification: []execution{}}
		if !report.OK {
			return value, 1, nil
		}
		if p.Values["verify"] == "true" {
			commands := [][]string{{"go", "test", "./..."}, {"go", "vet", "./..."}}
			if p.Values["race"] == "true" {
				commands[0] = []string{"go", "test", "-race", "./..."}
			}
			for _, command := range commands {
				run := verify(ctx, root, command)
				value.Verification = append(value.Verification, run)
				if run.ExitCode != 0 {
					return value, 1, nil
				}
			}
		}
		return value, 0, nil
	case "add module":
		if err = project.ValidateModuleName(p.Arg); err != nil {
			return nil, 2, err
		}
		if l.Manifest.FindModule(p.Arg) == nil {
			l.Manifest.Modules = append(l.Manifest.Modules, project.Module{Name: p.Arg, Operations: []project.Operation{}})
		}
	case "add command", "add query":
		if err = project.ValidateOperationName(p.Arg); err != nil {
			return nil, 2, err
		}
		mod := l.Manifest.FindModule(p.Values["module"])
		if mod == nil {
			return nil, 2, project.Fail("CONFIG007", "unknown module: "+p.Values["module"], "Create it with gsvc add module.")
		}
		input, err := project.ParseFields(p.Values["in"])
		if err != nil {
			return nil, 2, err
		}
		output, err := project.ParseFields(p.Values["out"])
		if err != nil {
			return nil, 2, err
		}
		op := project.Operation{Name: p.Arg, Kind: strings.TrimPrefix(p.Spec.Name, "add "), Input: input, Output: output}
		if old := mod.FindOperation(p.Arg); old != nil {
			copy := *old
			copy.Endpoint = nil
			if !reflect.DeepEqual(copy, op) {
				return nil, 3, project.Fail("OWN006", "existing operation has a different contract", "Use gsvc change operation with a reviewed --dry-run plan; behavior is never automatically migrated.")
			}
		} else {
			mod.Operations = append(mod.Operations, op)
		}
	case "add endpoint":
		mod := l.Manifest.FindModule(p.Values["module"])
		if mod == nil {
			return nil, 2, project.Fail("CONFIG007", "unknown business module", "")
		}
		op := mod.FindOperation(p.Arg)
		if op == nil {
			return nil, 2, project.Fail("CONFIG008", "unknown operation: "+p.Arg, "Create the operation before binding a route.")
		}
		method := "POST"
		if op.Kind == "query" {
			method = "GET"
		}
		endpoint := &project.Endpoint{Method: method, Path: p.Values["path"]}
		if op.Endpoint != nil && !reflect.DeepEqual(op.Endpoint, endpoint) {
			return nil, 3, project.Fail("OWN006", "operation already has a different endpoint", "Use gsvc change endpoint with a reviewed --dry-run plan.")
		}
		op.Endpoint = endpoint
	case "change operation", "change endpoint":
		mod := l.Manifest.FindModule(p.Values["module"])
		if mod == nil {
			return nil, 2, project.Fail("CONFIG007", "unknown business module", "")
		}
		op := mod.FindOperation(p.Arg)
		if op == nil {
			return nil, 2, project.Fail("CONFIG008", "unknown operation: "+p.Arg, "")
		}
		if p.Spec.Name == "change endpoint" {
			if op.Endpoint == nil {
				return nil, 2, project.Fail("CONFIG008", "operation has no endpoint", "Use gsvc add endpoint.")
			}
			op.Endpoint.Path = p.Values["path"]
		} else {
			_, hasIn := p.Values["in"]
			_, hasOut := p.Values["out"]
			if !hasIn && !hasOut {
				return nil, 2, project.Fail("USAGE004", "change operation requires --in or --out", "Omit a flag to preserve that side of the contract.")
			}
			if hasIn {
				op.Input, err = project.ParseFields(p.Values["in"])
				if err != nil {
					return nil, 2, err
				}
			}
			if hasOut {
				op.Output, err = project.ParseFields(p.Values["out"])
				if err != nil {
					return nil, 2, err
				}
			}
		}
	case "sync":
	case "recipe":
		manifest, application, err := project.ApplyRecipe(l.Manifest, p.Arg)
		if err != nil {
			return nil, 2, err
		}
		value, exit, err := mutate(root, manifest, &l.State, p)
		if err != nil {
			return value, exit, err
		}
		result := value.(mutationResult)
		result.Recipe = &application
		return result, exit, nil
	default:
		return nil, 2, project.Fail("USAGE001", "unsupported command", "")
	}
	return mutate(root, l.Manifest, &l.State, p)
}
func mutate(root string, m project.Manifest, old *project.State, p parsed) (any, int, error) {
	files, err := generate.Render(m)
	if err != nil {
		return nil, 2, err
	}
	plan, err := project.BuildPlan(root, m, old, files)
	if err != nil {
		return nil, 3, err
	}
	dry := p.Values["dry-run"] == "true"
	if expected := p.Values["expect"]; expected != "" && expected != plan.ID {
		return nil, 3, project.Fail("PLAN001", "reviewed plan no longer matches", "")
	}
	if !dry {
		if err = project.Apply(plan, p.Values["expect"]); err != nil {
			return nil, 3, err
		}
	}
	return mutationResult{DryRun: dry, Plan: plan}, 0, nil
}

// cappedBuffer still consumes all subprocess output without unbounded memory.
type cappedBuffer struct {
	b         bytes.Buffer
	truncated bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	left := (128 << 10) - b.b.Len()
	if left > 0 {
		end := n
		if end > left {
			end = left
		}
		_, _ = b.b.Write(p[:end])
	}
	if n > left {
		b.truncated = true
	}
	return n, nil
}
func verify(parent context.Context, root string, command []string) execution {
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Dir = root
	// Honor the selected toolchain and dependency policy. Verification is explicit:
	// Go may execute project tests and fetch declared dependencies if configured.
	cmd.Env = append(os.Environ(), "GOWORK=off")
	var output cappedBuffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	code := 0
	if err != nil {
		code = 1
		var e *exec.ExitError
		if errors.As(err, &e) {
			code = e.ExitCode()
		}
		if ctx.Err() != nil {
			_, _ = fmt.Fprintf(&output, "\nverification interrupted: %v\n", ctx.Err())
		} else if code == 1 && output.b.Len() == 0 {
			_, _ = fmt.Fprintf(&output, "%v\n", err)
		}
	}
	return execution{Command: command, ExitCode: code, Output: output.b.String(), Truncated: output.truncated}
}
