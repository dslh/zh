package testutil

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

var updateSnapshots = flag.Bool("update-snapshots", false, "update snapshot golden files")

// snapshotsDir returns the absolute path to the test/snapshots directory.
func snapshotsDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "test", "snapshots")
}

// AssertSnapshot compares actual output against a named golden file.
// If the -update-snapshots flag is set, it writes the actual output to the file instead.
// The name is used as the filename under test/snapshots/ (e.g. "root-help.txt").
func AssertSnapshot(t *testing.T, name string, actual string) {
	t.Helper()

	dir := snapshotsDir()
	path := filepath.Join(dir, name)

	if *updateSnapshots {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create snapshots dir: %v", err)
		}
		if err := os.WriteFile(path, []byte(actual), 0o644); err != nil {
			t.Fatalf("failed to write snapshot %q: %v", name, err)
		}
		t.Logf("updated snapshot: %s", name)
		return
	}

	expected, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("snapshot %q not found — run with -update-snapshots to create it: %v", name, err)
	}

	if string(expected) != actual {
		t.Errorf("snapshot %q mismatch\n\n--- expected ---\n%s\n--- actual ---\n%s", name, string(expected), actual)
	}
}
