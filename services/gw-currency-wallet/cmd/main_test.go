package main

import (
	"path/filepath"
	"testing"
)

func TestCLIFlags(t *testing.T) {
	if err := run([]string{"-h"}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"-unknown"}, {"unexpected"}, {"-c"}, {"-c", filepath.Join(t.TempDir(), "missing.env")}} {
		if err := run(args); err == nil {
			t.Errorf("expected error for %v", args)
		}
	}
}
