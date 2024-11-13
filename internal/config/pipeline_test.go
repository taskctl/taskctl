package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Ensono/taskctl/internal/config"
	"github.com/Ensono/taskctl/pkg/scheduler"
)

func TestBuildPipeline_Cyclical(t *testing.T) {

	var cyclicalYaml = `pipelines:
  pipeline1:    
    - task: task1
      name: task1
      depends_on: 
        - last-stage
      dir: "/root"
    - task: task2
      name: task2
      depends_on:
        - task1
      env: {}
    - task: task3
      name: last-stage
      depends_on: 
        - task2

tasks:
  task1:
    name: task1
  task2:
    name: task2
  task3:
    name: task3
`

	tmpDir, _ := os.MkdirTemp(os.TempDir(), "cyclical-pipeline")

	defer func() {
		os.RemoveAll(tmpDir)
	}()

	file := filepath.Join(tmpDir, "cyclical.yaml")
	_ = os.WriteFile(file, []byte(cyclicalYaml), 0777)

	cl := config.NewConfigLoader(config.NewConfig())
	_, err := cl.Load(file)
	if !errors.Is(err, scheduler.ErrCycleDetected) {
		t.Errorf("cycles detection failed")
	}
}

func TestBuildPipeline_Error(t *testing.T) {
	tmpDir, _ := os.MkdirTemp(os.TempDir(), "error-on-pipeline")
	defer func() {
		os.RemoveAll(tmpDir)
	}()

	t.Run("no such task", func(t *testing.T) {
		var errorYaml = `pipelines:
  pipeline1:    
    - task: task4
      name: task4
      depends_on:
        - last-stage
      dir: "/root"
tasks:
  task1:
    name: task1
  task2:
    name: task2
  task3:
    name: task3
`
		file := filepath.Join(tmpDir, "nosuch-task.yaml")
		_ = os.WriteFile(file, []byte(errorYaml), 0777)

		cl := config.NewConfigLoader(config.NewConfig())
		_, err := cl.Load(file)
		if err == nil || !strings.Contains(err.Error(), "no such task") {
			t.Error()
		}
	})
	t.Run("no such pipeline", func(t *testing.T) {
		var errorYaml = `pipelines:
  pipeline1:    
    - pipeline: task4
      name: task4
      depends_on:
        - last-stage
      dir: "/root"
tasks:
  task1:
    name: task1
  task2:
    name: task2
  task3:
    name: task3
`
		file := filepath.Join(tmpDir, "nosuch-pipeline.yaml")
		_ = os.WriteFile(file, []byte(errorYaml), 0777)

		cl := config.NewConfigLoader(config.NewConfig())
		_, err := cl.Load(file)
		if err == nil || !strings.Contains(err.Error(), "no such pipeline") {
			t.Error()
		}
	})
	t.Run("stage with same name", func(t *testing.T) {
		var errorYaml = `pipelines:
  pipeline1:    
    - task: task1
      name: task1
      depends_on:
        - last-stage
      dir: "/root"
    - task: task1
      name: task1
      depends_on:
        - last-stage
      dir: "/root"
tasks:
  task1:
    name: task1
  task2:
    name: task2
  task3:
    name: task3
`
		file := filepath.Join(tmpDir, "stage.yaml")
		_ = os.WriteFile(file, []byte(errorYaml), 0777)

		cl := config.NewConfigLoader(config.NewConfig())
		_, err := cl.Load(file)
		if err == nil || !strings.Contains(err.Error(), "stage with same name") {
			t.Error()
		}
	})
}
