package config_test

import (
	"errors"
	"fmt"
	"os"
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

	file, cleanUp := configLoaderTestHelper(t, cyclicalYaml)
	defer cleanUp()
	cl := config.NewConfigLoader(config.NewConfig())
	_, err := cl.Load(file)
	if !errors.Is(err, scheduler.ErrCycleDetected) {
		t.Errorf("cycles detection failed")
	}
}

func TestBuildPipeline_Error(t *testing.T) {
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

		file, cleanUp := configLoaderTestHelper(t, errorYaml)
		defer cleanUp()
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

		file, cleanUp := configLoaderTestHelper(t, errorYaml)
		defer cleanUp()
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
		file, cleanUp := configLoaderTestHelper(t, errorYaml)
		defer cleanUp()
		cl := config.NewConfigLoader(config.NewConfig())
		_, err := cl.Load(file)
		if err == nil || !strings.Contains(err.Error(), "stage with same name") {
			t.Error()
		}
	})
}

func TestConfig_TaskLoader(t *testing.T) {
	t.Run("task correctly built from config using envfile as well as env keys", func(t *testing.T) {
		tmpEnv, _ := os.CreateTemp("", "*.env")
		defer os.Remove(tmpEnv.Name())
		_, _ = tmpEnv.Write([]byte(`FOO=taskX
ANOTHER_VAR=moo`))

		yamlTasks := fmt.Sprintf(`tasks:
  task-p2:1:
    command:
      - |
        echo "hello, p2 ${FOO} env: ${ENV_NAME:-unknown}"
    context: podman
    env:
      FOO: task1
      GLOBAL_VAR: overwritteninTask
    envfile:
      path: %s

  task-p2:2:
    command:
      - |
        for i in $(seq 1 5); do
          echo "hello, p2 ${FOO} - env: ${ENV_NAME:-unknown} - iteration $i"
          sleep 0
        done
    env:
      FOO: task2`, tmpEnv.Name())
		file, cleanUp := configLoaderTestHelper(t, yamlTasks)
		defer cleanUp()
		cl := config.NewConfigLoader(config.NewConfig())
		taskctlCfg, err := cl.Load(file)
		if err != nil {
			t.Error()
		}
		val, ok := taskctlCfg.Tasks["task-p2:1"]
		if !ok {
			t.Error("failed to add task to config")
		}
		if val.EnvFile == nil {
			t.Fatal("failed to read the env file")
		}
		if val.EnvFile.PathValue != tmpEnv.Name() {
			t.Error("incorrect env file name")
		}
	})
}

func configLoaderTestHelper(t *testing.T, configInput string) (file string, cleanUp func()) {
	t.Helper()
	tmpfile, _ := os.CreateTemp(os.TempDir(), "config-pipeline-*.yml")

	_ = os.WriteFile(tmpfile.Name(), []byte(configInput), 0777)

	return tmpfile.Name(), func() {
		os.Remove(tmpfile.Name())
	}
}
