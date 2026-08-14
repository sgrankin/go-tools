// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"maps"
	"testing"
)

func TestFilterChecks(t *testing.T) {
	available := []string{"S1000", "SA1000", "SA4023", "ST1000"}
	tests := []struct {
		name      string
		selection []string
		want      map[string]bool
	}{
		{
			name:      "all with exclusion",
			selection: []string{"all", "-SA4023"},
			want: map[string]bool{
				"s1000":  true,
				"sa1000": true,
				"sa4023": false,
				"st1000": true,
			},
		},
		{
			name:      "category",
			selection: []string{"S*"},
			want:      map[string]bool{"s1000": true},
		},
		{
			name:      "prefix",
			selection: []string{"SA1*"},
			want:      map[string]bool{"sa1000": true},
		},
		{
			name:      "case insensitive",
			selection: []string{"sa4023"},
			want:      map[string]bool{"sa4023": true},
		},
		{
			name:      "unknown literal",
			selection: []string{"X9999"},
			want:      map[string]bool{"x9999": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FilterChecks(available, tt.selection); !maps.Equal(got, tt.want) {
				t.Fatalf("FilterChecks() = %v, want %v", got, tt.want)
			}
		})
	}
}
