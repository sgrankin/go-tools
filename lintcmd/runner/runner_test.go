// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runner

import (
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"
	"testing"

	"honnef.co/go/tools/config"
	"honnef.co/go/tools/go/loader"
	"honnef.co/go/tools/lintcmd/cache"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

func TestSelectAnalyzers(t *testing.T) {
	nilness := &analysis.Analyzer{Name: "nilness"}
	sa4023 := &analysis.Analyzer{Name: "SA4023", Requires: []*analysis.Analyzer{nilness}}
	s1000 := &analysis.Analyzer{Name: "S1000"}
	analyzers := []*analysis.Analyzer{s1000, sa4023}

	tests := []struct {
		name     string
		configs  [][]string
		override []string
		want     []string
	}{
		{
			name:     "disabled check omits dependency",
			configs:  [][]string{{"all", "-SA4023"}},
			override: []string{"inherit"},
			want:     []string{"S1000"},
		},
		{
			name:     "enabled check includes dependency",
			configs:  [][]string{{"SA4023"}},
			override: []string{"inherit"},
			want:     []string{"SA4023", "nilness"},
		},
		{
			name:     "initial packages use union",
			configs:  [][]string{{"all", "-SA4023"}, {"SA4023"}},
			override: []string{"inherit"},
			want:     []string{"S1000", "SA4023", "nilness"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkgs := make([]*loader.PackageSpec, len(tt.configs))
			for i, checks := range tt.configs {
				pkgs[i] = &loader.PackageSpec{Config: config.Config{Checks: checks}}
			}

			gotAnalyzers := allAnalyzers(selectAnalyzers(analyzers, pkgs, config.Config{Checks: tt.override}))
			got := make([]string, len(gotAnalyzers))
			for i, analyzer := range gotAnalyzers {
				got[i] = analyzer.Name
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("selected analyzers = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunnerOnlyExecutesSelectedAnalyzers(t *testing.T) {
	moduleDir := t.TempDir()
	writeFile := func(name, contents string) {
		t.Helper()
		path := filepath.Join(moduleDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeFile("go.mod", "module example.com/selection\n\ngo 1.26\n")
	writeFile("staticcheck.conf", "checks = [\"all\", \"-SLOW\"]\n")
	writeFile("pkg/pkg.go", "package pkg\n\nfunc F() {}\n")
	writeFile("pkg/pkg_test.go", "package pkg\n\nimport \"testing\"\n\nfunc TestF(t *testing.T) { F() }\n")

	var fastRuns atomic.Int64
	var slowRuns atomic.Int64
	var dependencyRuns atomic.Int64
	dependency := &analysis.Analyzer{
		Name: "dependency",
		Run: func(*analysis.Pass) (any, error) {
			dependencyRuns.Add(1)
			return nil, nil
		},
	}
	slow := &analysis.Analyzer{
		Name:     "SLOW",
		Requires: []*analysis.Analyzer{dependency},
		Run: func(*analysis.Pass) (any, error) {
			slowRuns.Add(1)
			return nil, nil
		},
	}
	fast := &analysis.Analyzer{
		Name: "FAST",
		Run: func(*analysis.Pass) (any, error) {
			fastRuns.Add(1)
			return nil, nil
		},
	}
	analyzers := []*analysis.Analyzer{fast, slow}

	c, err := cache.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	loadConfig := &packages.Config{Dir: moduleDir, Tests: true}
	r, err := New(config.Config{Checks: []string{"inherit"}}, c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run(loadConfig, analyzers, []string{"./pkg"}); err != nil {
		t.Fatal(err)
	}
	firstFastRuns := fastRuns.Load()
	if firstFastRuns == 0 {
		t.Fatal("enabled analyzer did not run")
	}
	if got := slowRuns.Load(); got != 0 {
		t.Fatalf("disabled analyzer ran %d times", got)
	}
	if got := dependencyRuns.Load(); got != 0 {
		t.Fatalf("dependency of disabled analyzer ran %d times", got)
	}

	r, err = New(config.Config{Checks: []string{"SLOW"}}, c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run(loadConfig, analyzers, []string{"./pkg"}); err != nil {
		t.Fatal(err)
	}
	if got := fastRuns.Load(); got != firstFastRuns {
		t.Fatalf("disabled analyzer ran %d additional times", got-firstFastRuns)
	}
	if got := slowRuns.Load(); got == 0 {
		t.Fatal("enabled analyzer did not run")
	}
	if got := dependencyRuns.Load(); got == 0 {
		t.Fatal("dependency of enabled analyzer did not run")
	}

	secondSlowRuns := slowRuns.Load()
	secondDependencyRuns := dependencyRuns.Load()
	r, err = New(config.Config{Checks: []string{"SLOW"}}, c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run(loadConfig, analyzers, []string{"./pkg"}); err != nil {
		t.Fatal(err)
	}
	if got := slowRuns.Load(); got != secondSlowRuns {
		t.Fatalf("cached analyzer ran %d additional times", got-secondSlowRuns)
	}
	if got := dependencyRuns.Load(); got != secondDependencyRuns {
		t.Fatalf("cached dependency ran %d additional times", got-secondDependencyRuns)
	}
}
