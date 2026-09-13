package remarkable

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectChiappaUsesMachinePath(t *testing.T) {
	dir := t.TempDir()
	machine := filepath.Join(dir, "machine")
	if err := os.WriteFile(machine, []byte(chiappaMachine+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := detectChiappa(machine, filepath.Join(dir, "missing-model"))
	if err != nil {
		t.Fatal(err)
	}
	if got != chiappaMachine {
		t.Fatalf("device = %q, want %q", got, chiappaMachine)
	}
}

func TestDetectChiappaFallsBackToDeviceTree(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "model")
	if err := os.WriteFile(model, append([]byte(chiappaMachine), 0), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := detectChiappa(filepath.Join(dir, "missing-machine"), model)
	if err != nil {
		t.Fatal(err)
	}
	if got != chiappaMachine {
		t.Fatalf("device = %q, want %q", got, chiappaMachine)
	}
}

func TestDetectChiappaRejectsOtherDevices(t *testing.T) {
	dir := t.TempDir()
	machine := filepath.Join(dir, "machine")
	if err := os.WriteFile(machine, []byte("reMarkable Ferrari"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := detectChiappa(machine, filepath.Join(dir, "model")); err == nil {
		t.Fatal("expected unsupported-device error")
	}
}
