// Package envutil provides helpers for working with environment variables.
package envutil

import (
	"bufio"
	"fmt"
	"os"
	"strings"
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

// ConvertToMapOfStrings converts map of interfaces to map of strings
func ConvertToMapOfStrings(m map[string]any) map[string]string {
	mdst := make(map[string]string)

	for k, v := range m {
		mdst[k] = v.(string)
	}
	return mdst
}

// ReadEnvFile reads env file in `k=v` format
func ReadEnvFile(filename string) (map[string]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

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
