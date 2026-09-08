package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gsvc.local/cli/internal/project"
)

func call(t *testing.T, want int, args ...string) map[string]any {
	t.Helper()
	var stdout, stderr bytes.Buffer
	args = append(args, "--json")
	got := Run(context.Background(), args, &stdout, &stderr)
	if stderr.Len() != 0 {
		t.Fatalf("JSON mode wrote stderr: %s", stderr.String())
	}
	var data map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &data); err != nil {
		t.Fatalf("non-JSON output: %s (%v)", stdout.String(), err)
	}
	if got != want {
		t.Fatalf("gsvc %v: exit %d want %d\n%s", args, got, want, stdout.String())
	}
	return data
}
func newService(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "demo")
	call(t, 0, "init", root, "--module", "example.com/demo")
	return root
}
func writeGo(t *testing.T, root, rel, code string) {
	t.Helper()
	b, err := format.Source([]byte(code))
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err = os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(p, b, 0644); err != nil {
		t.Fatal(err)
	}
}
func diagnosticCodes(data map[string]any) string { b, _ := json.Marshal(data); return string(b) }

func TestMachineProtocol(t *testing.T) {
	call(t, 0, "contract")
	call(t, 0, "version")
	call(t, 2, "nonsense")
	call(t, 2, "add", "command", "x")
	call(t, 2, "version", "--unknown")
	root := filepath.Join(t.TempDir(), "new-api")
	data := call(t, 0, "init", root, "--module", "example.com/new-api", "--dry-run")
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("dry-run created root")
	}
	id := data["data"].(map[string]any)["plan"].(map[string]any)["id"].(string)
	call(t, 3, "init", root, "--module", "example.com/new-api", "--expect", "wrong")
	call(t, 0, "init", root, "--module", "example.com/new-api", "--expect", id)
	call(t, 0, "check", "--root", root, "--strict")
}
func TestGeneratorCheckerAndGeneratedBuildAgree(t *testing.T) {
	root := newService(t)
	for _, name := range []string{"payments", "users"} {
		call(t, 0, "add", "module", name, "--root", root)
	}
	for _, tc := range []struct{ kind, name, in, out, path string }{
		{"command", "create-payment", "amount:money,currency:string,enabled:bool,count:int64", "payment_id:string", "/payments"},
		{"query", "get-payment", "id:string,page:int64,verbose:bool,amount:money", "payment_id:string,status:string", "/payments/{id}"},
		{"query", "list-payments", "", "", "/payments"},
		{"command", "refresh-payments", "", "", "/payments/refresh"},
	} {
		call(t, 0, "add", tc.kind, tc.name, "--module", "payments", "--in", tc.in, "--out", tc.out, "--root", root)
		call(t, 0, "add", "endpoint", tc.name, "--module", "payments", "--path", tc.path, "--root", root)
	}
	call(t, 0, "check", "--root", root, "--verify")
	data := call(t, 1, "check", "--root", root, "--strict")
	if !strings.Contains(diagnosticCodes(data), "WORK001") {
		t.Fatal("strict check missed scaffolds")
	}
	call(t, 0, "inspect", "--root", root)
}
func TestRerunPreservesBusinessAndConflictsBeforeWriting(t *testing.T) {
	root := newService(t)
	call(t, 0, "add", "module", "payments", "--root", root)
	call(t, 0, "add", "command", "create-payment", "--module", "payments", "--in", "amount:money", "--root", root)
	file := filepath.Join(root, "internal/payments/internal/app/create_payment.go")
	b, _ := os.ReadFile(file)
	b = append(b, []byte("\n// preserved sentinel\n")...)
	os.WriteFile(file, b, 0644)
	call(t, 0, "add", "command", "create-payment", "--module", "payments", "--in", "amount:money", "--root", root)
	after, _ := os.ReadFile(file)
	if !bytes.Equal(b, after) {
		t.Fatal("business file overwritten")
	}
	call(t, 3, "add", "command", "create-payment", "--module", "payments", "--in", "amount:money,currency:string", "--root", root)
	call(t, 0, "change", "operation", "create-payment", "--module", "payments", "--in", "amount:money,currency:string", "--root", root)
	after, _ = os.ReadFile(file)
	if !bytes.Equal(b, after) {
		t.Fatal("explicit contract change overwrote behavior")
	}
	managed := filepath.Join(root, "internal/bootstrap/routes_gen.go")
	orig, _ := os.ReadFile(managed)
	os.WriteFile(managed, append(orig, []byte("\n// tampered\n")...), 0644)
	call(t, 3, "add", "module", "other", "--root", root)
	if _, err := os.Stat(filepath.Join(root, "internal/other")); !os.IsNotExist(err) {
		t.Fatal("partial writes despite conflict")
	}
	data := call(t, 1, "check", "--root", root)
	if !strings.Contains(diagnosticCodes(data), "GEN001") {
		t.Fatal("managed drift missed")
	}
}
func TestArchitectureDrift(t *testing.T) {
	root := newService(t)
	call(t, 0, "add", "module", "payments", "--root", root)
	call(t, 0, "add", "module", "users", "--root", root)
	for _, tc := range []struct{ file, code, want string }{
		{"internal/payments/internal/domain/bad.go", "package domain\nimport _ \"net/http\"\n", "ARCH001"},
		{"internal/payments/internal/app/bad.go", "package app\nimport _ \"example.com/demo/internal/users\"\n", "ARCH001"},
		{"internal/payments/internal/app/bad.go", "package app\nfunc init() {}\n", "ARCH002"},
		{"internal/shared/bad.go", "package shared\n", "LAYOUT001"},
		{"internal/platform/fault/bad.go", "package fault\n", "LAYOUT003"},
	} {
		t.Run(tc.want+tc.file, func(t *testing.T) {
			writeGo(t, root, tc.file, tc.code)
			data := call(t, 1, "check", "--root", root)
			if !strings.Contains(diagnosticCodes(data), tc.want) {
				t.Fatal("missing " + tc.want)
			}
			os.Remove(filepath.Join(root, tc.file))
		})
	}
}
func TestStalePlanAndRouteChange(t *testing.T) {
	root := newService(t)
	call(t, 0, "add", "module", "greetings", "--root", root)
	data := call(t, 0, "add", "query", "greet-person", "--module", "greetings", "--in", "name:string", "--out", "message:string", "--root", root, "--dry-run")
	id := data["data"].(map[string]any)["plan"].(map[string]any)["id"].(string)
	// A change to any tracked developer file invalidates the reviewed plan.
	p := filepath.Join(root, "README.md")
	b, _ := os.ReadFile(p)
	os.WriteFile(p, append(b, '\n'), 0644)
	call(t, 3, "add", "query", "greet-person", "--module", "greetings", "--in", "name:string", "--out", "message:string", "--root", root, "--expect", id)
	call(t, 0, "add", "query", "greet-person", "--module", "greetings", "--in", "name:string", "--out", "message:string", "--root", root)
	call(t, 0, "add", "endpoint", "greet-person", "--module", "greetings", "--path", "/greetings/{name}", "--root", root)
	call(t, 0, "change", "endpoint", "greet-person", "--module", "greetings", "--path", "/hello/{name}", "--root", root)
	call(t, 0, "check", "--verify", "--root", root)
}
func TestImplementedOperationPassesStrictVerification(t *testing.T) {
	root := newService(t)
	call(t, 0, "add", "module", "greetings", "--root", root)
	call(t, 0, "add", "query", "greet-person", "--module", "greetings", "--in", "name:string", "--out", "message:string", "--root", root)
	call(t, 0, "add", "endpoint", "greet-person", "--module", "greetings", "--path", "/greetings/{name}", "--root", root)
	writeGo(t, root, "internal/greetings/internal/app/greet_person.go", `package app
import "context"
type GreetPersonDeps struct{}
type GreetPerson struct{deps GreetPersonDeps}
func NewGreetPerson(deps GreetPersonDeps)*GreetPerson{return &GreetPerson{deps:deps}}
func(h *GreetPerson)Execute(ctx context.Context,in GreetPersonInput)(GreetPersonOutput,error){if err:=ctx.Err();err!=nil{return GreetPersonOutput{},err};if err:=in.Validate();err!=nil{return GreetPersonOutput{},err};return GreetPersonOutput{Message:"Hello, "+in.Name+"!"},nil}
`)
	writeGo(t, root, "internal/greetings/internal/app/greet_person_test.go", `package app
import("context";"errors";"testing")
func TestGreeting(t *testing.T){got,err:=NewGreetPerson(GreetPersonDeps{}).Execute(context.Background(),GreetPersonInput{Name:"Ben"});if err!=nil||got.Message!="Hello, Ben!"{t.Fatalf("%+v %v",got,err)}}
func TestRequiredName(t *testing.T){_,err:=NewGreetPerson(GreetPersonDeps{}).Execute(context.Background(),GreetPersonInput{});if err==nil{t.Fatal("empty name accepted")}}
func TestCanceled(t *testing.T){ctx,cancel:=context.WithCancel(context.Background());cancel();_,err:=NewGreetPerson(GreetPersonDeps{}).Execute(ctx,GreetPersonInput{Name:"Ben"});if !errors.Is(err,context.Canceled){t.Fatal(err)}}
`)
	call(t, 0, "check", "--root", root, "--strict", "--verify")
}
func TestMalformedConfigDoesNotFallback(t *testing.T) {
	root := newService(t)
	os.WriteFile(filepath.Join(root, project.ManifestPath), []byte(`{"broken":`), 0644)
	data := call(t, 2, "inspect", "--root", root)
	if !strings.Contains(diagnosticCodes(data), "CONFIG005") {
		t.Fatal(fmt.Sprint(data))
	}
}
func TestMissingManagedOutputCanBeRestored(t *testing.T) {
	root := newService(t)
	p := filepath.Join(root, "internal/bootstrap/routes_gen.go")
	os.Remove(p)
	call(t, 1, "check", "--root", root)
	call(t, 0, "sync", "--root", root)
	call(t, 0, "check", "--root", root)
}

func TestVerificationFailureIsNotReportedAsSuccess(t *testing.T) {
	root := newService(t)
	writeGo(t, root, "internal/bootstrap/failure_test.go", `package bootstrap
import "testing"
func TestBusinessFailure(t *testing.T) { t.Fatal("intentional verification failure") }
`)
	data := call(t, 1, "check", "--root", root, "--verify")
	if data["ok"] != false {
		t.Fatal("failed tests reported success")
	}
	if !strings.Contains(diagnosticCodes(data), "intentional verification failure") {
		t.Fatal("missing test output")
	}
}
