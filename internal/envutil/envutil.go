// Package envutil provides helpers for working with environment variables.
package envutil

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/taskctl/taskctl/internal/iox"
)

// ConvertEnv converts map representing the environment to array of strings in the form "key=value"
func ConvertEnv(env map[string]string) []string {
	var i int
	enva := make([]string, len(env))
	for k, v := range env {
		enva[i] = fmt.Sprintf("%s=%s", k, v)
		i++
	}

	return enva
}

// OverlayEnviron merges overlay onto base (base in "key=value" form, overlay as a map).
// For exact key matches, it drops shadowed base entries before appending overlay,
// making precedence explicit. The result may still contain duplicates from base;
// callers may still pass the list through a normalization/dedup step (e.g. Windows case-folding).
func OverlayEnviron(base []string, overlay map[string]string) []string {
	merged := make([]string, 0, len(base)+len(overlay))
	for _, kv := range base {
		k, _, ok := strings.Cut(kv, "=")
		if ok {
			if _, shadowed := overlay[k]; shadowed {
				continue
			}
		}
		merged = append(merged, kv)
	}

	return append(merged, ConvertEnv(overlay)...)
}

// ConvertToMapOfStrings converts map of interfaces to map of strings
func ConvertToMapOfStrings(m map[string]any) map[string]string {
	mdst := make(map[string]string)

	for k, v := range m {
		mdst[k] = v.(string)
	}
	return mdst
}

// SanitizeEnviron filters out entries with invalid variable names, such as
// Cygwin's "!::=::\" or Windows' hidden "=C:=C:\dir" drive entries.
//
// Only the first character is checked (letter or underscore) rather than the
// full POSIX name (e.g. syntax.ValidName): this deliberately keeps legitimate
// Windows variables like "ProgramFiles(x86)" that child processes may need,
// while still dropping the OS-generated junk above.
func SanitizeEnviron(environ []string) []string {
	sanitized := make([]string, 0, len(environ))
	for _, entry := range environ {
		key, _, found := strings.Cut(entry, "=")
		if !found || key == "" || !isNameStart(key[0]) {
			continue
		}
		sanitized = append(sanitized, entry)
	}

	return sanitized
}

// isNameStart reports whether b is a valid first character of an environment
// variable name (an ASCII letter or underscore).
func isNameStart(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// ReadEnvFile reads env file in `k=v` format
func ReadEnvFile(filename string) (map[string]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer iox.Close(f)

	envs := make(map[string]string)
	envscanner := bufio.NewScanner(f)
	for envscanner.Scan() {
		k, v, found := strings.Cut(envscanner.Text(), "=")
		if !found {
			continue
		}
		envs[k] = v
	}

	if err := envscanner.Err(); err != nil {
		return nil, err
	}

	return envs, nil
}
