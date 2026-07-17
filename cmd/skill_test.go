package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The real SKILL.md is embedded and injected by package main, which isn't
// linked into this test binary, so inject a representative template here to
// make the install round-trip assertions meaningful.
func init() {
	SetSkillTemplate("---\nname: taskctl\n---\n\n# taskctl test skill\n")
}

func TestInstallSkill_FreshInstall(t *testing.T) {
	dir := t.TempDir()

	path, err := installSkill(dir, false)
	if err != nil {
		t.Fatal(err)
	}

	wantPath := filepath.Join(dir, ".claude", "skills", "taskctl", "SKILL.md")
	if path != wantPath {
		t.Errorf("expected returned path %q, got %q", wantPath, path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != skillTemplate {
		t.Errorf("installed content does not match embedded SKILL.md template")
	}
}

func TestInstallSkill_ExistingFileErrorsWithoutForce(t *testing.T) {
	dir := t.TempDir()

	if _, err := installSkill(dir, false); err != nil {
		t.Fatal(err)
	}

	_, err := installSkill(dir, false)
	if err == nil {
		t.Fatal("expected error on second install without --force, got nil")
	}

	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("expected error to mention --force, got %q", err.Error())
	}
}

func TestInstallSkill_ForceOverwrites(t *testing.T) {
	dir := t.TempDir()

	path, err := installSkill(dir, false)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte("stale content"), 0o644); err != nil {
		t.Fatal(err)
	}

	newPath, err := installSkill(dir, true)
	if err != nil {
		t.Fatal(err)
	}

	if newPath != path {
		t.Errorf("expected path %q, got %q", path, newPath)
	}

	content, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != skillTemplate {
		t.Errorf("expected --force to overwrite with the embedded template")
	}
}
