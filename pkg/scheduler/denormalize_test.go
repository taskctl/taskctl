package scheduler_test

import (
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/Ensono/taskctl/internal/config"
	"github.com/Ensono/taskctl/pkg/scheduler"
	"github.com/Ensono/taskctl/pkg/task"
	"github.com/Ensono/taskctl/pkg/variables"
)

func TestStageFrom_originalToNew(t *testing.T) {
	oldStage := scheduler.NewStage("old-stage", func(s *scheduler.Stage) {
		s.DependsOn = []string{"task1"}
		s.Task = task.NewTask("task2")
		s.WithEnv(variables.FromMap(map[string]string{"foo": "bar", "original": "oldVal"}))
		s.WithVariables(variables.FromMap(map[string]string{"var1": "bar", "var2": "oldVal"}))
	})

	g, _ := scheduler.NewExecutionGraph("test-merge", oldStage)
	g.Env = map[string]string{"global": "global-stuff"}
	newStage := scheduler.NewStage("new-stage")
	newStage.FromStage(oldStage, g, []string{"test-merge"})

	if len(newStage.Env().Map()) == 0 {
		t.Fatal("not merged env")
	}
	for k, v := range newStage.Env().Map() {
		fmt.Printf("%s = %v\n", k, v)
	}
}

func TestExecutionGraph_Flatten(t *testing.T) {
	t.Parallel()

	g := helperGraph(t, "graph:pipeline1")
	if g == nil {
		t.Fatal("graph not found")
	}
	if len(g.Nodes()) != 8 {
		t.Errorf("top level graph does not have correct number of top level jobs, got %v wanted %v", len(g.Nodes()), 8)
	}

	if len(g.Children(scheduler.RootNodeName)) != 1 {
		t.Errorf("root job incorrect got %v wanted %v", len(g.Children(scheduler.RootNodeName)), 1)
	}

	if len(g.Children("graph:task3")) != 2 {
		t.Errorf("graph:task3 children incorrect got %v wanted %v", len(g.Children("graph:task3")), 2)
	}

	flattenedStages := map[string]*scheduler.Stage{}

	g.Flatten(scheduler.RootNodeName, []string{g.Name()}, flattenedStages)
	if len(flattenedStages) != 13 {
		t.Errorf("stages incorrectly flattened: got %v wanted %v\n", len(flattenedStages), 13)
	}
	gotStages := []string{}
	for k, v := range flattenedStages {
		if k != v.Name {
			t.Errorf("key should be the same as name got key (%s) and name (%s)\n", k, v.Name)
		}
		gotStages = append(gotStages, k)
	}
	// keep the list a bit wet to ensure changes are maintained
	nodeList := []string{"graph:pipeline1->graph:pipeline3->graph:task3", "graph:pipeline1->dev->task-p2:1",
		"graph:pipeline1->prod->task-p2:1", "graph:pipeline1->graph:pipeline3->graph:task2",
		"graph:pipeline1->graph:task2", "graph:pipeline1->graph:task4", "graph:pipeline1->graph:pipeline3",
		"graph:pipeline1->graph:task3", "graph:pipeline1->dev", "graph:pipeline1->dev->task-p2:2",
		"graph:pipeline1->prod", "graph:pipeline1->graph:task1", "graph:pipeline1->prod->task-p2:2"}
	for _, v := range nodeList {
		if !slices.Contains(gotStages, v) {
			t.Errorf("stage (%s) not found in %q\n", v, gotStages)
		}
	}
}

func TestStageTable_ops(t *testing.T) {
	g := helperGraph(t, "graph:pipeline1")
	flattenedStages := map[string]*scheduler.Stage{}
	g.Flatten(scheduler.RootNodeName, []string{g.Name()}, flattenedStages)
	// add the root stage just for testing
	flattenedStages["graph:pipeline1"] = &scheduler.Stage{Name: "graph:pipeline1"}

	st := scheduler.StageTable(flattenedStages)
	t.Run("parents", func(t *testing.T) {

		t2 := st.RecurseParents("graph:pipeline1->graph:pipeline3->graph:task2")
		if len(t2) != 2 {
			t.Errorf("%v is not the required length 2", len(t2))
		}
		if t2[0].Name != "graph:pipeline1" {
			t.Errorf("incorrectly reversed order in slice")
		}
		if t2[1].Name != "graph:pipeline1->graph:pipeline3" {
			t.Errorf("incorrectly reversed order in slice")
		}

		tp1 := st.RecurseParents("graph:pipeline1->prod->task-p2:2")

		if len(tp1) != 2 {
			t.Errorf("%v is not the required length 2", len(t2))
		}
		if tp1[0].Name != "graph:pipeline1" {
			t.Errorf("incorrectly reversed order in slice")
		}
		if tp1[1].Name != "graph:pipeline1->prod" {
			t.Errorf("incorrectly reversed order in slice")
		}
	})

	t.Run("nth children", func(t *testing.T) {
		prod := st.NthLevelChildren("graph:pipeline1->prod", 1)
		if len(prod) != 2 {
			t.Error("wrong number of children")
		}

		gp1 := st.NthLevelChildren("graph:pipeline1", 2)
		if len(gp1) != 6 {
			t.Error("wrong number of children at that level")
		}

		prod2 := st.NthLevelChildren("graph:pipeline1->prod", 2)
		if len(prod2) != 0 {
			t.Error("wrong number of children")
		}
	})
}

func TestExecutionGraph_Denormalize(t *testing.T) {
	t.Parallel()
	g := helperGraph(t, "graph:pipeline1")

	t.Run("check sample graph", func(t *testing.T) {
		got, err := g.Denormalize()
		if err != nil {
			t.Error(err.Error())
		}
		if got == nil {
			t.Error("got nil, wanted a denormalized graph")
		}

		// "graph:pipeline1->prod->task-p2:1"
		prodPipeline, err := got.Node("graph:pipeline1->prod")
		if err != nil || prodPipeline == nil {
			t.Error("incorrectly built denormalized graph")
		}

		if prodPipeline.Pipeline == nil {
			t.Error("incorrectly built denormalized graph")
		}

		tp21, err := prodPipeline.Pipeline.Node("graph:pipeline1->prod->task-p2:1")
		if err != nil {
			t.Error("incorrectly built denormalized graph")
		}

		val, ok := tp21.Env().Map()["ENV_NAME"]
		if !ok {
			t.Error("incorrectly built denormalized graph")
		}
		if val != "prod" {
			t.Errorf("incorrectly inherited env across stages, got %s, wanted prod", val)
		}

		// test Context < Pipeline < Task precedence
		// Context => GLOBAL_VAR: this is it
		// Pipeline => GLOBAL_VAR: prodPipeline
		// Task => GLOBAL_VAR: overwritteninTask
		valGlobal, okGlobal := tp21.Env().Map()["GLOBAL_VAR"]
		if !okGlobal {
			t.Error("incorrectly built denormalized graph")
		}
		if valGlobal != "overwritteninTask" {
			t.Errorf("incorrectly inherited env across stages, got %s, wanted overwritteninTask", valGlobal)
		}
	})
}

var ymlInputTester = []byte(`
output: prefixed
contexts:
  podman:
    container:
      name: alpine:latest
    env: 
      GLOBAL_VAR: this is it
    envfile:
      exclude:
        - HOME

ci_meta:
  targetOpts:
    github:
      "on": 
        push:
          branches:
            - gfooo

pipelines:
  prod:
    - pipeline: graph:pipeline2
      env:
        ENV_NAME: prod
        GLOBAL_VAR: prodPipeline
  graph:pipeline1:
    - task: graph:task2
      depends_on: 
        - graph:task1
    - task: graph:task3
      depends_on: [graph:task1]
    - name: dev
      pipeline: graph:pipeline2
      depends_on: [graph:task3]
      env:
        ENV_NAME: dev
    - pipeline: prod
      depends_on: [graph:task3]
    - task: graph:task4
      depends_on:
        - graph:task2
    - task: graph:task1
    - pipeline: graph:pipeline3
      depends_on:
        - graph:task4

  graph:pipeline2:
    - task: task-p2:2
    - task: task-p2:1
      depends_on:
        - task-p2:2

  graph:pipeline3:
    - task: graph:task2
    - task: graph:task3

tasks:
  graph:task1:
    command: |
      for i in $(seq 1 5); do
        echo "hello task 1 in env ${ENV_NAME} - iteration $i"
        sleep 0
      done
    context: podman

  graph:task2:
    command: |
      echo "hello task 2"
      echo "another line1"
      echo "another line2"
      echo "another line3"
    context: podman

  graph:task3:
    command: 
      - echo "hello, task3 in env ${ENV_NAME}"
    env:
      FOO: bar

  graph:task4:
    command: | 
      echo "hello, task4 in env ${ENV_NAME}"
    context: podman
    env:
      FOO: bar

  task-p2:1:
    command:
      - |
        echo "hello, p2 ${FOO} env: ${ENV_NAME:-unknown}"
    context: podman
    env:
      FOO: task1
      GLOBAL_VAR: overwritteninTask

  task-p2:2:
    command:
      - |
        for i in $(seq 1 5); do
          echo "hello, p2 ${FOO} - env: ${ENV_NAME:-unknown} - iteration $i"
          sleep 0
        done
    env:
      FOO: task2
`)

func helperGraph(t *testing.T, name string) *scheduler.ExecutionGraph {
	t.Helper()

	tf, err := os.CreateTemp("", "graph-*.yml")
	if err != nil {
		t.Fatal("failed to create a temp file")
	}
	defer os.Remove(tf.Name())
	if _, err := tf.Write(ymlInputTester); err != nil {
		t.Fatal(err)
	}

	cl := config.NewConfigLoader(config.NewConfig())
	cfg, err := cl.Load(tf.Name())
	return cfg.Pipelines[name]
}
