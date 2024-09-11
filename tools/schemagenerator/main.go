// package schemagenerator
//
// Generates the schema for the config file
package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/Ensono/taskctl/internal/config"
	"github.com/invopop/jsonschema"
)

func main() {
	r := new(jsonschema.Reflector)
	if err := r.AddGoComments("github.com/Ensono/taskctl", "./"); err != nil {
		log.Fatal(err.Error())
	}
	s := r.Reflect(&config.ConfigDefinition{})
	out, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		log.Fatalf("failed to parse: %s", err)
	}
	if err := os.MkdirAll(filepath.Join(".", "schemas"), 0777); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("./schemas/schema_v1.json", out, 0777); err != nil {
		log.Fatalf("failed to write: %s", err)
	}
}
