// Copyright (c) 2024 Project Illium
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package repo

import (
	"os"
	"path"
	"path/filepath"
	"testing"
)

func TestCreateDefaultConfigFile(t *testing.T) {
	// Setup a temporary directory
	tmpDir := path.Join(os.TempDir(), "ilxd")
	testpath := filepath.Join(tmpDir, "test.conf")

	// Clean-up
	defer func() {
		os.Remove(testpath)
		os.Remove(tmpDir)
	}()

	err := createDefaultConfigFile(testpath, false)
	if err != nil {
		t.Fatalf("Failed to create a default config file: %v", err)
	}

	_, err = os.ReadFile(testpath)
	if err != nil {
		t.Fatalf("Failed to read generated default config file: %v", err)
	}
}
