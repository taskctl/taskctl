# Artifacts

> Removed the scrape of stdout as a default Output storage after every task

Any task can assign outputs in a form of files or a special dotenv format output which is then added to the tascktl runner and available to all the subsequent tasks.

> Currently onle dotenv is available as an artifact output

The artifacts are only usefil inside pipelines, ensuring you provide a depends on will make sure the variables are ready.

The outputs are available across contexts.

Example below uses the default context to generate some stored artifact and then it is used in a container context.

> after commands are not stored as artifacts

```yaml
summary: true
debug: true
output: prefixed

contexts:
  newdocker:ctx:
    container:
      name: alpine:latest 
    envfile:
      exclude:
        # these will be excluded by default
        - PATH
        - HOME
        - TMPDIR

pipelines:
  tester:
    - task: task:one
    - task: task:two
      depends_on:
        - task:one

tasks:
  task:one:
    env:
      ORIGINAL_VAR: foo123
    command:
      - echo "task one ORIGINAL_VAR => ${ORIGINAL_VAR} should be foo123"
      - echo ORIGINAL_VAR=foo333 > .artifact.env
    after:
      # should run in a new context
      - echo ORIGINAL_VAR=shouldNOTBEUSED > .artifact.env
    artifacts:
      name: test_env_from_task_one
      path: .artifact.env
      type: dotenv

  task:two:
    command:
      - echo "task:two ORIGINAL_VAR => ${ORIGINAL_VAR} should be foo333"
    context: newdocker:ctx
```