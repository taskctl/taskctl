package config

import (
	"bytes"
	"io/ioutil"
	"testing"

	"gopkg.in/yaml.v2"
)

var testConfig, _ = ioutil.ReadFile("testdata/tasks.yaml")

func TestConfig_decode(t *testing.T) {
	loader := NewConfigLoader()

	var cm = make(map[string]interface{})
	var dec = yaml.NewDecoder(bytes.NewReader(testConfig))
	dec.SetStrict(true)

	err := dec.Decode(cm)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := loader.decode(cm)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := cfg.Tasks["test-task"]; !ok {
		t.Fatal("tasks parsing error")
	}
}
