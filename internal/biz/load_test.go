package biz

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCandidates_Empty(t *testing.T) {
	dir := t.TempDir()
	cs, err := LoadCandidates(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 0 {
		t.Fatalf("got %d, want 0", len(cs))
	}
}

func TestLoadCandidates_SkipsNonDirs(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(
		filepath.Join(dir, "readme.txt"),
		[]byte("hi"), 0644)
	cs, err := LoadCandidates(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 0 {
		t.Fatalf("got %d, want 0", len(cs))
	}
}

func TestLoadCandidates_SkipsDirWithoutData(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "empty-biz"), 0755)
	cs, err := LoadCandidates(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 0 {
		t.Fatalf("got %d, want 0", len(cs))
	}
}

func TestLoadCandidates_LoadsCandidate(t *testing.T) {
	dir := t.TempDir()
	slug := "test-biz"
	bizDir := filepath.Join(dir, slug)
	os.MkdirAll(bizDir, 0755)
	data := `{"name":"Test","asking_price":100000}`
	os.WriteFile(
		filepath.Join(bizDir, "data.json"),
		[]byte(data), 0644)

	cs, err := LoadCandidates(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 1 {
		t.Fatalf("got %d candidates", len(cs))
	}
	if cs[0].Slug != slug {
		t.Fatalf("slug: got %q, want %q",
			cs[0].Slug, slug)
	}
	if cs[0].Biz.Name != "Test" {
		t.Fatalf("name: got %q", cs[0].Biz.Name)
	}
	if cs[0].Biz.AskingPrice != 100000 {
		t.Fatalf("price: got %v", cs[0].Biz.AskingPrice)
	}
}

func TestLoadCandidates_BadJSON(t *testing.T) {
	dir := t.TempDir()
	bizDir := filepath.Join(dir, "bad")
	os.MkdirAll(bizDir, 0755)
	os.WriteFile(
		filepath.Join(bizDir, "data.json"),
		[]byte("{not json"), 0644)

	_, err := LoadCandidates(dir)
	if err == nil {
		t.Fatal("expected error for bad JSON")
	}
}
