package cli

import (
	"fmt"
	"sort"
	"strings"

	"gsvc.local/cli/internal/project"
)

type Flag struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required,omitempty"`
}
type Command struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	Argument         string `json:"argument,omitempty"`
	OptionalArgument bool   `json:"optional_argument,omitempty"`
	Mutation         bool   `json:"mutation"`
	Flags            []Flag `json:"flags"`
}

func flags(extra ...Flag) []Flag {
	return append([]Flag{{Name: "json", Type: "boolean", Description: "Return one JSON envelope on stdout"}}, extra...)
}
func rootFlag() Flag {
	return Flag{Name: "root", Type: "string", Description: "Project directory (default: find from current directory)"}
}
func mutationFlags(extra ...Flag) []Flag {
	out := []Flag{rootFlag(), {Name: "dry-run", Type: "boolean", Description: "Plan without writing"}, {Name: "expect", Type: "string", Description: "Require the exact reviewed plan ID"}}
	return flags(append(out, extra...)...)
}
func Commands() []Command {
	return []Command{
		{Name: "version", Description: "Show the CLI and policy version", Flags: flags()},
		{Name: "contract", Description: "Describe supported commands, flags, invariants, and exit codes", Flags: flags()},
		{Name: "init", Description: "Create a standalone Go service in an empty directory", Argument: "directory", Mutation: true, Flags: flags(Flag{Name: "module", Type: "string", Required: true, Description: "Go import path, e.g. example.com/payments-api"}, Flag{Name: "name", Type: "string", Description: "Service name (default: directory basename)"}, Flag{Name: "dry-run", Type: "boolean", Description: "Plan without writing"}, Flag{Name: "expect", Type: "string", Description: "Require the reviewed plan ID"})},
		{Name: "add module", Description: "Create one business module with enforced internal boundaries", Argument: "name", Mutation: true, Flags: mutationFlags()},
		{Name: "add command", Description: "Add a typed write operation, dependency struct, contract and tests", Argument: "name", Mutation: true, Flags: mutationFlags(Flag{Name: "module", Type: "string", Required: true, Description: "Existing business module"}, Flag{Name: "in", Type: "string", Description: "Required inputs: amount:money,currency:string"}, Flag{Name: "out", Type: "string", Description: "Outputs: payment_id:string"})},
		{Name: "add query", Description: "Add a typed read operation, dependency struct, contract and tests", Argument: "name", Mutation: true, Flags: mutationFlags(Flag{Name: "module", Type: "string", Required: true, Description: "Existing business module"}, Flag{Name: "in", Type: "string", Description: "Required inputs: id:string"}, Flag{Name: "out", Type: "string", Description: "Outputs: id:string,status:string"})},
		{Name: "add endpoint", Description: "Bind an existing operation (POST command / GET query)", Argument: "operation", Mutation: true, Flags: mutationFlags(Flag{Name: "module", Type: "string", Required: true, Description: "Existing business module"}, Flag{Name: "path", Type: "string", Required: true, Description: "HTTP path, e.g. /payments or /payments/{id}"})},
		{Name: "change operation", Description: "Explicitly change operation fields; preserve business implementations and tests", Argument: "name", Mutation: true, Flags: mutationFlags(Flag{Name: "module", Type: "string", Required: true, Description: "Existing business module"}, Flag{Name: "in", Type: "string", Description: "Replace inputs; omit to preserve, empty string to clear"}, Flag{Name: "out", Type: "string", Description: "Replace outputs; omit to preserve, empty string to clear"})},
		{Name: "change endpoint", Description: "Explicitly change an existing route path; preserve operation behavior", Argument: "operation", Mutation: true, Flags: mutationFlags(Flag{Name: "module", Type: "string", Required: true, Description: "Existing business module"}, Flag{Name: "path", Type: "string", Required: true, Description: "New endpoint path"})},
		{Name: "upgrade", Description: "Explicitly migrate a supported older project contract to this CLI version", Mutation: true, Flags: mutationFlags()},
		{Name: "sync", Description: "Reconcile managed output without overwriting developer-owned code", Mutation: true, Flags: mutationFlags()},
		{Name: "recipe", Description: "List or apply a typed capability recipe through the guarded project plan", Argument: "name", OptionalArgument: true, Mutation: true, Flags: mutationFlags()},
		{Name: "capabilities", Description: "Inspect declared capability contracts and their scaffold status", Flags: flags(rootFlag())},
		{Name: "inspect", Description: "Expose the live project model, ownership, policy and diagnostics", Flags: flags(rootFlag())},
		{Name: "check", Description: "Check generated integrity, layout, imports, formatting and pending work", Flags: flags(rootFlag(), Flag{Name: "strict", Type: "boolean", Description: "Fail on pending scaffolds and test skips"}, Flag{Name: "verify", Type: "boolean", Description: "Also execute go test ./... and go vet ./..."}, Flag{Name: "race", Type: "boolean", Description: "Use go test -race; requires --verify"})},
		{Name: "recover", Description: "Roll back an interrupted CLI write without overwriting unrelated edits", Mutation: true, Flags: flags(rootFlag(), Flag{Name: "dry-run", Type: "boolean", Description: "List recovery targets without writing"})},
	}
}

type parsed struct {
	Spec   Command
	Arg    string
	Values map[string]string
	Help   bool
}

func parse(args []string) (parsed, error) {
	p := parsed{Values: map[string]string{}}
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		p.Help = true
		return p, nil
	}
	if args[0] == "--version" {
		args = append([]string{"version"}, args[1:]...)
	}
	name := args[0]
	used := 1
	if name == "add" || name == "change" {
		if len(args) < 2 {
			return p, project.Fail("USAGE001", name+" requires a subcommand", "Run gsvc contract --json.")
		}
		name += " " + args[1]
		used = 2
	}
	found := false
	for _, s := range Commands() {
		if s.Name == name {
			p.Spec = s
			found = true
			break
		}
	}
	if !found {
		return p, project.Fail("USAGE001", "unknown command: "+name, "Run gsvc help or gsvc contract --json.")
	}
	specs := map[string]Flag{}
	for _, f := range p.Spec.Flags {
		specs[f.Name] = f
	}
	positional := []string{}
	for i := used; i < len(args); i++ {
		arg := args[i]
		if arg == "--help" || arg == "-h" {
			p.Help = true
			continue
		}
		if !strings.HasPrefix(arg, "--") {
			if strings.HasPrefix(arg, "-") {
				return p, project.Fail("USAGE002", "unsupported short flag: "+arg, "")
			}
			positional = append(positional, arg)
			continue
		}
		key, value, equals := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
		f, ok := specs[key]
		if !ok {
			return p, project.Fail("USAGE002", "unknown flag: --"+key, "Run gsvc "+name+" --help.")
		}
		if _, exists := p.Values[key]; exists {
			return p, project.Fail("USAGE002", "duplicate flag: --"+key, "")
		}
		if f.Type == "boolean" {
			if !equals {
				value = "true"
			}
			if value != "true" && value != "false" {
				return p, project.Fail("USAGE002", "boolean flag requires true or false: --"+key, "")
			}
		} else if !equals {
			i++
			if i >= len(args) || strings.HasPrefix(args[i], "--") {
				return p, project.Fail("USAGE002", "missing value for --"+key, "")
			}
			value = args[i]
		}
		p.Values[key] = value
	}
	if p.Help {
		return p, nil
	}
	minimum, maximum := 0, 0
	if p.Spec.Argument != "" {
		maximum = 1
		if !p.Spec.OptionalArgument {
			minimum = 1
		}
	}
	if len(positional) < minimum || len(positional) > maximum {
		return p, project.Fail("USAGE003", fmt.Sprintf("%s expects %d to %d positional argument(s)", name, minimum, maximum), "Flags may precede or follow the argument.")
	}
	if len(positional) == 1 {
		p.Arg = positional[0]
	}
	for _, f := range p.Spec.Flags {
		if f.Required && strings.TrimSpace(p.Values[f.Name]) == "" {
			return p, project.Fail("USAGE004", "required flag: --"+f.Name, "")
		}
	}
	if p.Values["race"] == "true" && p.Values["verify"] != "true" {
		return p, project.Fail("USAGE005", "--race requires --verify", "")
	}
	return p, nil
}
func contract() any {
	return map[string]any{"schema_version": 1, "tool": "gsvc", "version": project.Version, "project_schema_version": project.SchemaVersion, "policy": project.PolicyDescription(), "commands": Commands(), "field_types": []string{"string", "money", "bool", "int64"}, "capabilities": project.CapabilityCatalog(), "recipes": project.RecipeCatalog(), "capability_maturity": project.CapabilityMaturity, "exit_codes": map[string]string{"0": "success", "1": "check or execution failure", "2": "invalid command or configuration", "3": "ownership, plan, lock, or recovery conflict"}, "guarantees": []string{"no interactive prompts", "no shell execution or downloads during generation", "JSON mode emits one envelope, including failures", "dry-run does not create files", "identical structural commands and recipe applications are idempotent", "recipes only declare versioned capabilities; the renderer owns their contract files", "declared capabilities are contract scaffolds, not proof of runtime behavior", "add rejects changed existing contracts; change performs explicit contract regeneration without migrating business behavior"}}
}
func help(p parsed) string {
	var b strings.Builder
	if p.Spec.Name == "" {
		fmt.Fprintf(&b, "gsvc %s — convention-driven Go services for humans and agents\n\n", project.Version)
		for _, c := range Commands() {
			fmt.Fprintf(&b, "  %-16s %s\n", c.Name, c.Description)
		}
		b.WriteString("\nUse gsvc <command> --help or gsvc contract --json.\n")
		return b.String()
	}
	fmt.Fprintf(&b, "gsvc %s %s\n%s\n\nFlags:\n", p.Spec.Name, p.Spec.Argument, p.Spec.Description)
	fs := append([]Flag(nil), p.Spec.Flags...)
	sort.Slice(fs, func(i, j int) bool { return fs[i].Name < fs[j].Name })
	for _, f := range fs {
		required := ""
		if f.Required {
			required = " (required)"
		}
		fmt.Fprintf(&b, "  --%-12s %-8s %s%s\n", f.Name, f.Type, f.Description, required)
	}
	return b.String()
}
