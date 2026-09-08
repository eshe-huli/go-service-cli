package project

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type File struct {
	Owner   string
	Content []byte
}
type Files map[string]File

type Loaded struct {
	Root     string
	Manifest Manifest
	State    State
}
type Change struct {
	Path         string `json:"path"`
	Owner        string `json:"owner"`
	Action       string `json:"action"`
	BeforeSHA256 string `json:"before_sha256,omitempty"`
	AfterSHA256  string `json:"after_sha256,omitempty"`
	Content      []byte `json:"-"`
	Before       []byte `json:"-"`
}
type Plan struct {
	ID      string   `json:"id"`
	Root    string   `json:"root"`
	Changes []Change `json:"changes"`
}

func Hash(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func ValidRelative(p string) bool {
	return p != "" && !filepath.IsAbs(p) && !strings.Contains(p, "\\") && filepath.ToSlash(filepath.Clean(p)) == p && p != "." && p != ".." && !strings.HasPrefix(p, "../")
}

// Root resolves existing ancestor symlinks (including macOS /tmp), then all
// project-relative accesses reject symlinks below this boundary. This protects
// normal CLI use, not adversarial concurrent filesystem replacement.
func Root(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	cur := abs
	tail := []string{}
	for {
		_, err = os.Lstat(cur)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", err
		}
		tail = append(tail, filepath.Base(cur))
		cur = parent
	}
	base, err := filepath.EvalSymlinks(cur)
	if err != nil {
		return "", err
	}
	for i := len(tail) - 1; i >= 0; i-- {
		base = filepath.Join(base, tail[i])
	}
	return base, nil
}
func FindRoot(start string) (string, error) {
	root, err := Root(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Lstat(filepath.Join(root, ManifestPath)); err == nil {
			return root, nil
		}
		parent := filepath.Dir(root)
		if parent == root {
			return "", Fail("PROJECT001", "no gsvc.json found", "Run inside a generated service, or pass --root.")
		}
		root = parent
	}
}
func SafePath(root, rel string) (string, error) {
	if !ValidRelative(rel) {
		return "", Fail("PATH001", "unsafe project-relative path: "+rel, "")
	}
	cur := root
	for _, part := range strings.Split(rel, "/") {
		cur = filepath.Join(cur, part)
		info, err := os.Lstat(cur)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", Fail("PATH002", "symlink inside project: "+rel, "Replace the link with a regular file or directory.")
		}
	}
	return filepath.Join(root, filepath.FromSlash(rel)), nil
}
func Read(root, rel string) ([]byte, bool, error) {
	p, err := SafePath(root, rel)
	if err != nil {
		return nil, false, err
	}
	info, err := os.Lstat(p)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if !info.Mode().IsRegular() {
		return nil, false, Fail("PATH003", "not a regular file: "+rel, "")
	}
	if info.Size() > 8<<20 {
		return nil, false, Fail("PATH004", "file exceeds the 8 MiB project metadata/source limit: "+rel, "")
	}
	b, err := os.ReadFile(p)
	return b, true, err
}
func Load(root string) (Loaded, error) {
	l := Loaded{Root: root}
	b, ok, err := Read(root, ManifestPath)
	if err != nil {
		return l, err
	}
	if !ok {
		return l, Fail("PROJECT001", "gsvc.json is missing", "")
	}
	if err = DecodeStrict(b, &l.Manifest); err != nil {
		return l, Fail("CONFIG005", err.Error(), "Fix malformed configuration; defaults are never silently substituted.")
	}
	if err = Validate(l.Manifest); err != nil {
		return l, err
	}
	sb, ok, err := Read(root, StatePath)
	if err != nil {
		return l, err
	}
	if !ok {
		return l, Fail("STATE001", "ownership state is missing", "Restore .gsvc/ownership.json from version control.")
	}
	if err = DecodeStrict(sb, &l.State); err != nil {
		return l, Fail("STATE001", err.Error(), "")
	}
	if l.State.SchemaVersion != SchemaVersion || l.State.ToolVersion != Version || l.State.ManifestSHA256 != Hash(b) {
		return l, Fail("STATE002", "manifest or ownership state was edited outside the CLI", "Restore both files from version control; use gsvc add to change the service contract.")
	}
	if len(l.State.Files) == 0 {
		return l, Fail("STATE003", "ownership state contains no files", "")
	}
	for p, o := range l.State.Files {
		if !ValidRelative(p) || p == ManifestPath || strings.HasPrefix(p, ".gsvc/") || (o.Owner != "generated" && o.Owner != "developer") || (o.Owner == "generated" && len(o.SHA256) != 64) {
			return l, Fail("STATE003", "invalid ownership entry: "+p, "")
		}
	}
	return l, nil
}
func BuildPlan(root string, m Manifest, old *State, files Files) (Plan, error) {
	Normalize(&m)
	if err := Validate(m); err != nil {
		return Plan{}, err
	}
	if _, exists, err := Read(root, ".gsvc/transaction.json"); err != nil {
		return Plan{}, err
	} else if exists {
		return Plan{}, Fail("TXN001", "an unfinished write transaction exists", "Run gsvc recover before making further changes.")
	}
	plan := Plan{Root: root, Changes: []Change{}}
	state := State{SchemaVersion: SchemaVersion, ToolVersion: Version, ManifestSHA256: Hash(JSON(m)), Files: map[string]Ownership{}}
	if old != nil {
		for p := range old.Files {
			if _, ok := files[p]; !ok {
				return plan, Fail("STATE004", "recorded file no longer belongs to the manifest: "+p, "Deletion and upgrades are not implicit.")
			}
		}
	}
	keys := make([]string, 0, len(files))
	for p := range files {
		keys = append(keys, p)
	}
	sort.Strings(keys)
	for _, p := range keys {
		f := files[p]
		before, exists, err := Read(root, p)
		if err != nil {
			return plan, err
		}
		previous, tracked := Ownership{}, false
		if old != nil {
			previous, tracked = old.Files[p]
		}
		if tracked && previous.Owner != f.Owner {
			return plan, Fail("OWN001", "file ownership changed: "+p, "")
		}
		if exists && !tracked {
			return plan, Fail("OWN002", "unowned file already exists: "+p, "Move or review it explicitly. The CLI never adopts or overwrites an unowned file.")
		}
		c := Change{Path: p, Owner: f.Owner, Action: "create", AfterSHA256: Hash(f.Content), Content: f.Content}
		if exists {
			c.Before = before
			c.BeforeSHA256 = Hash(before)
		}
		switch f.Owner {
		case "developer":
			if tracked && !exists {
				return plan, Fail("OWN003", "developer-owned file was deleted: "+p, "Restore it explicitly; scaffolding will not replace business code.")
			}
			if exists {
				c.Action = "preserve"
				c.Content = nil
				c.AfterSHA256 = c.BeforeSHA256
			}
			state.Files[p] = Ownership{Owner: "developer"}
		case "generated":
			if exists && previous.SHA256 != Hash(before) {
				return plan, Fail("OWN004", "generated file was modified: "+p, "Restore the generated file from version control; do not reset ownership hashes.")
			}
			if exists {
				c.Action = "update"
				if bytes.Equal(before, f.Content) {
					c.Action = "keep"
				}
			}
			state.Files[p] = Ownership{Owner: "generated", SHA256: Hash(f.Content)}
		default:
			return plan, Fail("OWN001", "unknown ownership: "+f.Owner, "")
		}
		plan.Changes = append(plan.Changes, c)
	}
	for _, meta := range []struct {
		p string
		b []byte
	}{{ManifestPath, JSON(m)}, {StatePath, JSON(state)}} {
		before, exists, err := Read(root, meta.p)
		if err != nil {
			return plan, err
		}
		if old == nil && exists {
			return plan, Fail("OWN002", "project metadata already exists: "+meta.p, "")
		}
		c := Change{Path: meta.p, Owner: "generated", Action: "create", Content: meta.b, AfterSHA256: Hash(meta.b)}
		if exists {
			c.Before = before
			c.BeforeSHA256 = Hash(before)
			c.Action = "update"
			if bytes.Equal(before, meta.b) {
				c.Action = "keep"
			}
		}
		plan.Changes = append(plan.Changes, c)
	}
	// Stable plan IDs omit the machine-specific project directory.
	plan.ID = Hash(JSON(plan.Changes))
	return plan, nil
}

type journalFile struct {
	Path         string `json:"path"`
	Before       []byte `json:"before"`
	BeforeExists bool   `json:"before_exists"`
	AfterSHA256  string `json:"after_sha256"`
}
type journal struct {
	SchemaVersion int           `json:"schema_version"`
	Files         []journalFile `json:"files"`
}

func pending(c Change) bool {
	return c.Action == "create" || c.Action == "update" || c.Action == "delete"
}
func acquire(root string) (func(), error) {
	p, err := SafePath(root, ".gsvc/write.lock")
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if os.IsExist(err) {
		return nil, Fail("LOCK001", "another writer or a stale write lock exists", "Check that no gsvc process is running before manually removing .gsvc/write.lock.")
	}
	if err != nil {
		return nil, err
	}
	_, werr := fmt.Fprintf(f, "pid=%d\n", os.Getpid())
	cerr := f.Close()
	if werr != nil || cerr != nil {
		os.Remove(p)
		return nil, fmt.Errorf("write lock: %v %v", werr, cerr)
	}
	return func() { _ = os.Remove(p) }, nil
}
func atomicWrite(root, rel string, b []byte) error {
	p, err := SafePath(root, rel)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(p), ".gsvc-write-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err = f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Chmod(tmp, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
func preflight(plan Plan) error {
	for _, c := range plan.Changes {
		b, ok, err := Read(plan.Root, c.Path)
		if err != nil {
			return err
		}
		h := ""
		if ok {
			h = Hash(b)
		}
		if h != c.BeforeSHA256 {
			return Fail("PLAN001", "file changed after planning: "+c.Path, "Generate a fresh plan and review it before applying.")
		}
	}
	return nil
}

// Apply uses an exclusive cooperative lock, complete preflight, per-file atomic
// replacement, and a rollback journal. It is not a multi-file filesystem transaction
// or a power-loss durability guarantee. Recovery is explicit after interruption.
func Apply(plan Plan, expected string) error {
	if expected != "" && expected != plan.ID {
		return Fail("PLAN001", "plan ID no longer matches", "Rerun with --dry-run --json and review the new plan.")
	}
	unlock, err := acquire(plan.Root)
	if err != nil {
		return err
	}
	defer unlock()
	if _, exists, err := Read(plan.Root, ".gsvc/transaction.json"); err != nil {
		return err
	} else if exists {
		return Fail("TXN001", "unfinished transaction exists", "Run gsvc recover.")
	}
	if err = preflight(plan); err != nil {
		return err
	}
	j := journal{SchemaVersion: SchemaVersion, Files: []journalFile{}}
	for _, c := range plan.Changes {
		if pending(c) {
			j.Files = append(j.Files, journalFile{c.Path, c.Before, c.BeforeSHA256 != "", c.AfterSHA256})
		}
	}
	if len(j.Files) == 0 {
		return nil
	}
	if err = atomicWrite(plan.Root, ".gsvc/transaction.json", JSON(j)); err != nil {
		return err
	}
	for _, c := range plan.Changes {
		if !pending(c) {
			continue
		}
		if err = atomicWrite(plan.Root, c.Path, c.Content); err != nil {
			if rerr := restore(plan.Root, j); rerr != nil {
				return Fail("TXN002", fmt.Sprintf("write failed: %v; rollback failed: %v", err, rerr), "Run gsvc recover after fixing the filesystem problem.")
			}
			_ = os.Remove(filepath.Join(plan.Root, ".gsvc/transaction.json"))
			return err
		}
	}
	return os.Remove(filepath.Join(plan.Root, ".gsvc/transaction.json"))
}
func restore(root string, j journal) error {
	// Do not roll back over an unrelated edit made after an interruption.
	for _, f := range j.Files {
		b, ok, err := Read(root, f.Path)
		if err != nil {
			return err
		}
		if !ok {
			if !f.BeforeExists {
				continue
			}
			return Fail("TXN003", "previously existing file is now missing: "+f.Path, "")
		}
		if Hash(b) != f.AfterSHA256 && (!f.BeforeExists || Hash(b) != Hash(f.Before)) {
			return Fail("TXN003", "file changed outside the interrupted transaction: "+f.Path, "Preserve this edit and reconcile it manually before recovery.")
		}
	}
	for i := len(j.Files) - 1; i >= 0; i-- {
		f := j.Files[i]
		if f.BeforeExists {
			if err := atomicWrite(root, f.Path, f.Before); err != nil {
				return err
			}
		} else {
			p, err := SafePath(root, f.Path)
			if err != nil {
				return err
			}
			if err = os.Remove(p); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}
func Recover(root string, dry bool) (any, error) {
	b, ok, err := Read(root, ".gsvc/transaction.json")
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, Fail("TXN004", "no interrupted transaction to recover", "")
	}
	var j journal
	if err = DecodeStrict(b, &j); err != nil {
		return nil, err
	}
	if j.SchemaVersion != SchemaVersion {
		return nil, Fail("TXN004", "unsupported recovery journal", "")
	}
	for _, f := range j.Files {
		if !ValidRelative(f.Path) || f.Path == ".gsvc/write.lock" || f.Path == ".gsvc/transaction.json" {
			return nil, Fail("PATH001", "unsafe recovery journal", "")
		}
	}
	paths := []string{}
	for _, f := range j.Files {
		paths = append(paths, f.Path)
	}
	result := map[string]any{"dry_run": dry, "restore_paths": paths}
	if dry {
		return result, nil
	}
	unlock, err := acquire(root)
	if err != nil {
		return nil, err
	}
	defer unlock()
	if err = restore(root, j); err != nil {
		return nil, err
	}
	err = os.Remove(filepath.Join(root, ".gsvc/transaction.json"))
	return result, err
}
func EmptyForInit(root string) error {
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Name() != ".git" {
			return Fail("PROJECT002", "init requires an empty directory (a .git directory is allowed)", "Choose a new service directory; existing projects are not rewritten.")
		}
	}
	return nil
}

// SourceFiles returns Go files and nested modules without following symlinks.
func SourceFiles(root string) ([]string, error) {
	out := []string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == root {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "vendor") {
			return filepath.SkipDir
		}
		if d.Type()&os.ModeSymlink != 0 {
			return Fail("PATH002", "symlink inside project: "+rel, "")
		}
		if !d.IsDir() && (strings.HasSuffix(p, ".go") || d.Name() == "go.mod") {
			out = append(out, rel)
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}
