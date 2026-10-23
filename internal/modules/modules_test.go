package modules

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadDiscoversPublicAndLocalModules(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "modules", "py", "module.json"),
		`{"id":"py","name":"Python","tracks":[{"id":"python","label":"Python","no_placement":true,"domains":["python_numpy"]}]}`)
	write(t, filepath.Join(root, "modules", "py", "concepts", "a.json"), `[]`)
	write(t, filepath.Join(root, "modules-local", "notes", "module.json"),
		`{"id":"notes","name":"My notes","tracks":[{"id":"python","label":"Python","domains":["python_mine"]}]}`)
	write(t, filepath.Join(root, "modules-local", "notes", "readings.json"),
		`{"c1":{"source":"Hoy 12.1","file":"hoy.pdf","page":494}}`)
	write(t, filepath.Join(root, "modules-local", "notes", "files", "hoy.pdf"), "%PDF")
	write(t, filepath.Join(root, "modules", "nomanifest", "concepts", "x.json"), `[]`)

	s, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Modules) != 2 || s.Modules[0].ID != "py" || !s.Modules[1].Local {
		t.Fatalf("modules: %+v", s.Modules)
	}
	if got := s.ConceptDirs(); len(got) != 1 {
		t.Errorf("concept dirs: %v", got)
	}
	tr := s.Tracks()
	if len(tr) != 2 || tr[0].ID != MathTrack || tr[1].ID != "python" || len(tr[1].Domains) != 2 {
		t.Fatalf("tracks: %+v", tr)
	}
	if s.TrackOf("python_mine") != "python" || s.TrackOf("calculus") != MathTrack {
		t.Error("TrackOf")
	}
	if s.Normalize("PYTHON") != "python" || s.Normalize("cs") != MathTrack || s.Normalize("") != MathTrack {
		t.Error("Normalize")
	}
	if s.HasPlacement("python") || !s.HasPlacement(MathTrack) {
		t.Error("HasPlacement")
	}
	r, ok := s.Reading("c1")
	if !ok || r.Module != "notes" || r.Page != 494 {
		t.Errorf("reading: %+v %v", r, ok)
	}
	if p := s.FilePath("notes", "hoy.pdf"); p == "" {
		t.Error("file inside files/ refused")
	}
	for _, bad := range []string{"../module.json", "/etc/passwd", "a/../../readings.json", ""} {
		if p := s.FilePath("notes", bad); p != "" {
			t.Errorf("FilePath(%q) = %q, want refused", bad, p)
		}
	}
}

func TestLocalConfigDisablesAModule(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "modules", "a", "module.json"), `{"id":"a"}`)
	write(t, filepath.Join(root, "modules", "b", "module.json"), `{"id":"b"}`)
	write(t, filepath.Join(root, "modules.local.json"), `{"disabled":["a"]}`)
	s, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Modules) != 1 || s.Modules[0].ID != "b" {
		t.Fatalf("modules: %+v", s.Modules)
	}
}

func TestConflictsAreErrors(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "modules", "a", "module.json"), `{"id":"same"}`)
	write(t, filepath.Join(root, "modules-local", "b", "module.json"), `{"id":"same"}`)
	if _, err := Load(root); err == nil {
		t.Error("duplicate module id accepted")
	}

	root = t.TempDir()
	write(t, filepath.Join(root, "modules", "a", "module.json"),
		`{"id":"a","tracks":[{"id":"x","label":"X","domains":["d"]}]}`)
	write(t, filepath.Join(root, "modules", "b", "module.json"),
		`{"id":"b","tracks":[{"id":"y","label":"Y","domains":["d"]}]}`)
	if _, err := Load(root); err == nil {
		t.Error("one domain on two tracks accepted")
	}

	root = t.TempDir()
	write(t, filepath.Join(root, "modules", "a", "module.json"),
		`{"id":"a","tracks":[{"id":"math","label":"Mine"}]}`)
	if _, err := Load(root); err == nil {
		t.Error("a module redefining the math track was accepted")
	}

	root = t.TempDir()
	write(t, filepath.Join(root, "modules", "a", "module.json"), `{"id":"a"}`)
	write(t, filepath.Join(root, "modules", "a", "readings.json"), `{"c":{"source":"x","file":"../../secret"}}`)
	if _, err := Load(root); err == nil {
		t.Error("a reading pointing outside files/ was accepted")
	}
}

// The repository's own modules must load.
func TestRepositoryModulesLoad(t *testing.T) {
	s, err := Load(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	off := map[string]bool{}
	for _, id := range s.Disabled {
		off[id] = true
	}
	for _, id := range []string{"python-dsh", "ma-econ"} {
		if s.Module(id) == nil && !off[id] {
			t.Errorf("committed module %s is missing", id)
		}
	}
}
