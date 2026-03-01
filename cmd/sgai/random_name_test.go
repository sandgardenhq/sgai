package main

import (
	"regexp"
	"testing"
)

func TestGenerateRandomForkName(t *testing.T) {
	t.Run("matchesExpectedFormat", func(t *testing.T) {
		pattern := regexp.MustCompile(`^[a-z]+-[a-z]+-[0-9aeiou]{4}$`)
		for range 100 {
			name := generateRandomForkName()
			if !pattern.MatchString(name) {
				t.Errorf("generated name %q does not match expected format adjective-color-XXXX", name)
			}
		}
	})

	t.Run("usesKnownAdjectives", func(t *testing.T) {
		adjectiveSet := make(map[string]bool)
		for _, adj := range forkNameAdjectives {
			adjectiveSet[adj] = true
		}
		for range 100 {
			name := generateRandomForkName()
			parts := splitForkName(name)
			if len(parts) < 3 {
				t.Fatalf("name %q has fewer than 3 parts", name)
			}
			if !adjectiveSet[parts[0]] {
				t.Errorf("adjective %q not in known adjective set", parts[0])
			}
		}
	})

	t.Run("usesKnownColors", func(t *testing.T) {
		colorSet := make(map[string]bool)
		for _, c := range forkNameColors {
			colorSet[c] = true
		}
		for range 100 {
			name := generateRandomForkName()
			parts := splitForkName(name)
			if len(parts) < 3 {
				t.Fatalf("name %q has fewer than 3 parts", name)
			}
			if !colorSet[parts[1]] {
				t.Errorf("color %q not in known color set", parts[1])
			}
		}
	})

	t.Run("producesUniqueNames", func(t *testing.T) {
		seen := make(map[string]bool)
		collisions := 0
		for range 1000 {
			name := generateRandomForkName()
			if seen[name] {
				collisions++
			}
			seen[name] = true
		}
		if collisions > 10 {
			t.Errorf("too many collisions: %d out of 1000", collisions)
		}
	})

	t.Run("passesWorkspaceNameValidation", func(t *testing.T) {
		for range 50 {
			name := generateRandomForkName()
			if errMsg := validateWorkspaceName(name); errMsg != "" {
				t.Errorf("name %q failed validation: %s", name, errMsg)
			}
		}
	})

	t.Run("suffixContainsOnlyValidChars", func(t *testing.T) {
		validChars := map[byte]bool{
			'0': true, '1': true, '2': true, '3': true, '4': true,
			'5': true, '6': true, '7': true, '8': true, '9': true,
			'a': true, 'e': true, 'i': true, 'o': true, 'u': true,
		}
		for range 100 {
			name := generateRandomForkName()
			parts := splitForkName(name)
			suffix := parts[len(parts)-1]
			if len(suffix) != 4 {
				t.Errorf("suffix %q length = %d; want 4", suffix, len(suffix))
			}
			for _, ch := range []byte(suffix) {
				if !validChars[ch] {
					t.Errorf("suffix %q contains invalid char %q", suffix, string(ch))
				}
			}
		}
	})
}

func splitForkName(name string) []string {
	parts := make([]string, 0, 3)
	lastDash := -1
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '-' {
			if lastDash == -1 {
				parts = append([]string{name[i+1:]}, parts...)
				lastDash = i
			} else {
				parts = append([]string{name[i+1 : lastDash]}, parts...)
				parts = append([]string{name[:i]}, parts...)
				return parts
			}
		}
	}
	if lastDash != -1 {
		parts = append([]string{name[:lastDash]}, parts...)
	}
	return parts
}
