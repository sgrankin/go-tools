// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package loader

import (
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestConfigDir(t *testing.T) {
	pkgDir := t.TempDir()
	sourceDir := t.TempDir()
	tests := []struct {
		name string
		pkg  *packages.Package
		want string
	}{
		{
			name: "source directory",
			pkg: &packages.Package{
				Dir:     pkgDir,
				GoFiles: []string{filepath.Join(sourceDir, "file.go")},
			},
			want: sourceDir,
		},
		{
			name: "generated package directory",
			pkg:  &packages.Package{Dir: pkgDir},
			want: pkgDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := packageConfigDir(tt.pkg)
			if got != tt.want {
				t.Fatalf("config directory = %q, want %q", got, tt.want)
			}
		})
	}
}
