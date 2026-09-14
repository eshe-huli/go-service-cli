package project

import "testing"

func TestModelValidation(t *testing.T) {
	for _, name := range []string{"../bad", "Payments", "payment_id", "type", "internal", "bootstrap", "main", "http", "httpx"} {
		if ValidateModuleName(name) == nil {
			t.Errorf("accepted module %q", name)
		}
	}
	for _, name := range []string{"payments", "billing2", "orders"} {
		if ValidateModuleName(name) != nil {
			t.Errorf("rejected module %q", name)
		}
	}
	for _, spec := range []string{"id:float64", "id:string,id:string", "validate:string", "bad-field:string", "incomplete", "id:string:optional"} {
		if _, err := ParseFields(spec); err == nil {
			t.Errorf("accepted fields %q", spec)
		}
	}
	for _, s := range []string{"example.com/service", "github.com/your-org/service/v2"} {
		if err := ValidateGoModule(s); err != nil {
			t.Fatal(err)
		}
	}
	for _, s := range []string{"../x", "example.com/a/../b", "x y", "example.com//x", "/abs/path"} {
		if ValidateGoModule(s) == nil {
			t.Errorf("accepted module path %q", s)
		}
	}
}

func TestCapabilityValidation(t *testing.T) {
	m := New("demo", "example.com/demo")
	m.Capabilities = []Capability{{ID: capabilityTransactionalInbox, Version: 1}}
	if err := Validate(m); err != nil {
		t.Fatal(err)
	}

	for _, capabilities := range [][]Capability{
		{{ID: "unknown.capability", Version: 1}},
		{{ID: capabilityTransactionalInbox, Version: 0}},
		{{ID: capabilityTransactionalInbox, Version: 2}},
		{{ID: capabilityTransactionalInbox, Version: 1}, {ID: capabilityTransactionalInbox, Version: 1}},
		{{ID: "projection.reconciliation-worker", Version: 1}},
	} {
		m.Capabilities = capabilities
		if Validate(m) == nil {
			t.Errorf("accepted capabilities %#v", capabilities)
		}
	}
	m.Capabilities = nil
	if Validate(m) == nil {
		t.Fatal("accepted an implicit capability set")
	}
}
func TestRoutesConflict(t *testing.T) {
	m := New("demo", "example.com/demo")
	m.Modules = []Module{{Name: "billing", Operations: []Operation{
		{Name: "find-one", Kind: "query", Input: []Field{{Name: "id", Type: "string"}}, Endpoint: &Endpoint{Method: "GET", Path: "/items/{id}"}},
		{Name: "find-two", Kind: "query", Input: []Field{{Name: "key", Type: "string"}}, Endpoint: &Endpoint{Method: "GET", Path: "/items/{key}"}},
	}}}
	if Validate(m) == nil {
		t.Fatal("overlapping patterns accepted")
	}
	m.Modules[0].Operations = m.Modules[0].Operations[:1]
	if err := Validate(m); err != nil {
		t.Fatal(err)
	}
	m.Modules[0].Operations[0].Endpoint.Path = "/healthz"
	if Validate(m) == nil {
		t.Fatal("health route accepted")
	}
}
func TestStrictJSON(t *testing.T) {
	for _, s := range []string{`{"unknown":1}`, `{} {}`, `{`} {
		var m Manifest
		if DecodeStrict([]byte(s), &m) == nil {
			t.Error(s)
		}
	}
}
func TestRelativePaths(t *testing.T) {
	for _, s := range []string{"../escape", "/absolute", "a/../b", "a\\b", ".", ""} {
		if ValidRelative(s) {
			t.Error(s)
		}
	}
}

func TestOperationFileNames(t *testing.T) {
	for _, name := range []string{"doc", "test", "gen", "create-test", "create-contract-gen"} {
		if ValidateOperationName(name) == nil {
			t.Error(name)
		}
	}
	if err := ValidateOperationName("create-test-record"); err != nil {
		t.Fatal(err)
	}
}
