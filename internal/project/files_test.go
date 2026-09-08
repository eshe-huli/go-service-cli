package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriterPreservationAndPreflight(t *testing.T) {
	root := filepath.Join(t.TempDir(), "service")
	m := New("service", "example.com/service")
	files := Files{"business.go": {Owner: "developer", Content: []byte("business")}, "registry.go": {Owner: "generated", Content: []byte("registry")}}
	p, err := BuildPlan(root, m, nil, files)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("planning wrote a directory")
	}
	if err = Apply(p, "bad-id"); err == nil {
		t.Fatal("stale ID accepted")
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("stale ID wrote a directory")
	}
	if err = Apply(p, p.ID); err != nil {
		t.Fatal(err)
	}
	l, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "business.go"), []byte("edited business"), 0644); err != nil {
		t.Fatal(err)
	}
	p, err = BuildPlan(root, m, &l.State, files)
	if err != nil {
		t.Fatal(err)
	}
	if err = Apply(p, p.ID); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(root, "business.go"))
	if string(b) != "edited business" {
		t.Fatal("overwrote developer content")
	}
	files["new.go"] = File{Owner: "developer", Content: []byte("new")}
	p, err = BuildPlan(root, m, &l.State, files)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "registry.go"), []byte("edited registry"), 0644); err != nil {
		t.Fatal(err)
	}
	if err = Apply(p, p.ID); err == nil {
		t.Fatal("accepted changed file after plan")
	}
	if _, err := os.Stat(filepath.Join(root, "new.go")); !os.IsNotExist(err) {
		t.Fatal("partial write before conflict detection")
	}
	if _, err = BuildPlan(root, m, &l.State, files); err == nil {
		t.Fatal("accepted edited managed output")
	}
}
func TestSymlinkRejected(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "internal")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := SafePath(root, "internal/escape.go"); err == nil {
		t.Fatal("symlink accepted")
	}
	entries, _ := os.ReadDir(outside)
	if len(entries) != 0 {
		t.Fatal("outside changed")
	}
}
func TestRecovery(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".gsvc"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "old.txt"), []byte("after"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "new.txt"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	j := journal{SchemaVersion: 1, Files: []journalFile{{Path: "old.txt", Before: []byte("before"), BeforeExists: true, AfterSHA256: Hash([]byte("after"))}, {Path: "new.txt", AfterSHA256: Hash([]byte("new"))}}}
	if err := os.WriteFile(filepath.Join(root, ".gsvc/transaction.json"), JSON(j), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Recover(root, true); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(root, "old.txt"))
	if string(b) != "after" {
		t.Fatal("dry recovery wrote")
	}
	if _, err := Recover(root, false); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(filepath.Join(root, "old.txt"))
	if string(b) != "before" {
		t.Fatal("not restored")
	}
	if _, err := os.Stat(filepath.Join(root, "new.txt")); !os.IsNotExist(err) {
		t.Fatal("new file retained")
	}
}
func TestRecoveryRefusesUnrelatedEdit(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".gsvc"), 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "x"), []byte("unrelated"), 0644)
	j := journal{SchemaVersion: 1, Files: []journalFile{{Path: "x", Before: []byte("old"), BeforeExists: true, AfterSHA256: Hash([]byte("after"))}}}
	os.WriteFile(filepath.Join(root, ".gsvc/transaction.json"), JSON(j), 0644)
	if _, err := Recover(root, false); err == nil {
		t.Fatal("unrelated edit overwritten")
	}
}
func TestWriterLock(t *testing.T) {
	root := t.TempDir()
	unlock, err := acquire(root)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	if _, err = acquire(root); err == nil {
		t.Fatal("second writer acquired lock")
	}
}
