// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"strings"
	"unicode"
)

// FilterChecks applies selection to the available checks and returns their
// enabled state. Check names in the returned map are lower-case.
func FilterChecks(available, selection []string) map[string]bool {
	checks := make([]string, len(available))
	for i, check := range available {
		checks[i] = strings.ToLower(check)
	}

	enabled := map[string]bool{}
	for _, check := range selection {
		check = strings.ToLower(check)
		value := true
		if len(check) > 1 && check[0] == '-' {
			value = false
			check = check[1:]
		}

		switch {
		case check == "*" || check == "all":
			for _, available := range checks {
				enabled[available] = value
			}
		case strings.HasSuffix(check, "*"):
			prefix := strings.TrimSuffix(check, "*")
			isCategory := strings.IndexFunc(prefix, unicode.IsNumber) == -1
			for _, available := range checks {
				if isCategory {
					idx := strings.IndexFunc(available, unicode.IsNumber)
					if idx == -1 {
						idx = len(available)
					}
					if prefix == available[:idx] {
						enabled[available] = value
					}
				} else if strings.HasPrefix(available, prefix) {
					enabled[available] = value
				}
			}
		default:
			enabled[check] = value
		}
	}
	return enabled
}
