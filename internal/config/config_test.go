package config

import (
	"gopkg.in/yaml.v2"
	"testing"
)

var testConfig = `
contexts:
  local: # will be created automatically if not set
    type: local
    bin: /bin/bash
    args:
      - -c
    env:
      VAR_NAME: VAR_VALUE
    before: SOME COMMAND TO RUN BEFORE EVERY TASK
    after: SOME COMMAND TO RUN WHEN TASK FINISHED SUCCESSFULLY

  docker-context-name:
    type: container
    container:
      provider: docker
      image: alpine:latest
      options:
        - -v /folder:/folder
      exec: false
      env:
        VAR_NAME: VAR_VALUE
    env:
      VAR_NAME: VAR_VALUE # eg. "DOCKER_HOST"
    before: SOME COMMAND TO RUN BEFORE TASK
    after: SOME COMMAND TO RUN WHEN TASK FINISHED SUCCESSFULLY

  docker-compose-context-name:
    type: container
    container:
      provider: docker-compose
      name: api
      exec: true
      args:
        - --file=docker-compose.yaml
      options:
        - --user=root
      env:
        VAR_NAME: VAR_VALUE
    up: docker-compose up -d --build --force-recreate api # Executes once before first context usage
    down: docker-compose down api # Executes once when all tasks done
    env:
      VAR_NAME: VAR_VALUE # eg."COMPOSE_FILE"
    before: SOME COMMAND TO RUN BEFORE TASK
    after: SOME COMMAND TO RUN WHEN TASK FINISHED SUCCESSFULLY

  ssh-context-name:
    type: remote
    ssh:
      user: root
      host: some-server
      options:
        - -6
        - -C
    env:
      VAR_NAME: VAR_VALUE # eg. "DOCKER_HOST"
    before: SOME COMMAND TO RUN BEFORE TASK
    after: SOME COMMAND TO RUN WHEN TASK FINISHED SUCCESSFULLY

pipelines:
  pipeline1:
    - task: task1
    - name: some-stage-name
      task: task2
      depends_on: task1
    - task: task3
      depends_on: task1 # task2 and task3 will run in parallel when task1 finished
    - task: task4
      depends_on: [task1, some-stage-name]
      env:
        VAR_NAME: VAR_VALUE # overrides task env

tasks:
  task1:
    context: local # optional. "local" is context by default
    command:
      - echo ${ARGS} # ARGS is populated by arguments passed to task. eg. wilson run task task1 -- arg1 arg2
      - echo "My name is task1"
    env:
      VAR_NAME: VAR_VALUE
    dir: /task/working/dir # current directory by default
    timeout: 10s

  task2:
    context: docker-context-name
    command:
      - echo "Hello from container"
    env:
      VAR_NAME: VAR_VALUE

  task3:
    context: docker-compose-context-name # local is default context
    command:
      - echo "Hello from container created by docker-compose"
    env:
      - VAR_NAME=VAR_VALUE

  task-to-be-triggered-by-watcher:
    command:
      - echo ${EVENT_NAME} ${EVENT_PATH}

watchers:
  watcher1:
    watch: ["README.*", "pkg/**/*.go"]
    events: [create, write, remove, rename, chmod]
    task: task1

# Default values for different providers
shell:
  bin: bin
  args: [arg1, arg2]

docker:
  bin: bin
  args: [arg1, arg2]

docker-compose:
  bin: bin
  args: [arg1, arg2]

kubectl:
  bin: bin
  args: [arg1, arg2]

ssh:
  bin: bin
  args: [arg1, arg2]
`

func TestConfig_UnmarshalYAML(t *testing.T) {
	cfg := &Config{}
	err := yaml.Unmarshal([]byte(testConfig), cfg)
	if err != nil {
		t.Fatal(err)
	}

	var ok bool
	if _, ok = cfg.Pipelines["pipeline1"]; !ok {
		t.Fatal("pipelines parsing error")
	}

	if cfg.Pipelines["pipeline1"][0].Name == "" {
		t.Errorf("stage parsing error")
	}

	if _, ok = cfg.Tasks["task1"]; !ok {
		t.Fatal("tasks parsing error")
	}

	if v, ok := cfg.Tasks["task3"].Env["VAR_NAME"]; !ok || v != "VAR_VALUE" {
		t.Fatal("tasks env parsing error")
	}

	if _, ok = cfg.Contexts["docker-compose-context-name"]; !ok {
		t.Fatal("contexts parsing error")
	}

	if _, ok = cfg.Watchers["watcher1"]; !ok {
		t.Fatal("watchers parsing error")
	}

	// todo: assertions
}
