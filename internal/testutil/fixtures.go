package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// fixturesDir returns the absolute path to the test/fixtures directory.
func fixturesDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "test", "fixtures")
}

// LoadFixture reads a fixture file from test/fixtures/ and returns its content.
func LoadFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join(fixturesDir(), name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to load fixture %q: %v", name, err)
	}
	return data
}

// LoadFixtureString reads a fixture file and returns it as a string.
func LoadFixtureString(t *testing.T, name string) string {
	return string(LoadFixture(t, name))
}
