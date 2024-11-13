//go:build ignore

// Package schemagenerator
//
// Generates the schema for the config file
// Accepts the dir to generate the output in
package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/Ensono/taskctl/internal/config"
	"github.com/invopop/jsonschema"
)

func main() {
	var dir string
	flag.StringVar(&dir, "dir", ".", "Directory to use as base")
	flag.Parse()
	generateSchemaForTaskCtl(dir)
}

func generateSchemaForTaskCtl(dir string) {

	r := new(jsonschema.Reflector)
	if err := r.AddGoComments("github.com/Ensono/taskctl", "./"); err != nil {
		log.Fatal(err.Error())
	}
	s := r.Reflect(&config.ConfigDefinition{})
	// use 2 spaces for indentation
	out, err := json.MarshalIndent(s, "", `  `)
	if err != nil {
		log.Fatalf("failed to parse: %s", err)
	}
	schemaDir := filepath.Join(dir, "schemas")
	if err := os.MkdirAll(schemaDir, 0777); err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(schemaDir, "schema_v1.json"), out, 0777); err != nil {
		log.Fatalf("failed to write: %s", err)
	}
}
