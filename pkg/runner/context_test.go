package runner

import (
	"context"
	"github.com/trntv/wilson/pkg/builder"
	"github.com/trntv/wilson/pkg/util"
	"testing"
)

func TestContext_BuildContext(t *testing.T) {
	wcfg := &builder.WilsonConfigDefinition{
		Docker: util.Executable{
			Bin: "/opt/docker",
		},
	}
	c, _ := BuildContext(builder.ContextDefinition{
		Type: "local",
		Env:  map[string]string{"TEST_VAR": "TEST_VAL"},
	}, wcfg)

	cmd, _ := c.createCommand(context.Background(), "echo ${TEST_VAR}")
	if cmd.String() != "/bin/sh -c echo ${TEST_VAR}" {
		t.Errorf("local build failed %s", cmd.String())
	}

	if !util.InArray(cmd.Env, "TEST_VAR=TEST_VAL") {
		t.Error("env not found")
	}

	c, _ = BuildContext(builder.ContextDefinition{
		Type: "container",
		Container: builder.ContainerDefinition{
			Provider: "docker",
			Image:    "alpine:latest",
			Exec:     false,
			Env:      map[string]string{"TEST_VAR": "TEST_VAL"},
		},
	}, wcfg)

	cmd, _ = c.createCommand(context.Background(), "echo ${TEST_VAR}")
	if cmd.String() != "/opt/docker run --rm -e TEST_VAR=TEST_VAL alpine:latest /bin/sh -c echo ${TEST_VAR}" {
		t.Errorf("docker build failed %s", cmd.String())
	}

	c, _ = BuildContext(builder.ContextDefinition{
		Type: "container",
		Container: builder.ContainerDefinition{
			Provider: "docker-compose",
			Name:     "alpine",
			Exec:     true,
			Options:  []string{"--user=root"},
			Env:      map[string]string{"TEST_VAR": "TEST_VAL"},
			Executable: util.Executable{
				Args: []string{"--file=example/docker-compose.yaml"},
			},
		},
		Up: []string{"docker-compose up -d alpine"},
	}, wcfg)

	cmd, _ = c.createCommand(context.Background(), "echo ${TEST_VAR}")
	if cmd.String() != "/usr/local/bin/docker-compose --file=example/docker-compose.yaml exec -T --user=root -e TEST_VAR=TEST_VAL alpine /bin/sh -c echo ${TEST_VAR}" {
		t.Errorf("docker-compose build failed %s", cmd.String())
	}

	c, _ = BuildContext(builder.ContextDefinition{
		Type: "container",
		Container: builder.ContainerDefinition{
			Provider: "kubectl",
			Name:     "deployment/geocoder",
			Options:  nil,
			Env:      map[string]string{"TEST_VAR": "TEST_VAL"},
			Executable: util.Executable{
				Bin: "/usr/bin/kubectl",
			},
		},
	}, wcfg)

	cmd, _ = c.createCommand(context.Background(), "echo ${TEST_VAR}")
	if cmd.String() != "/usr/bin/kubectl exec deployment/geocoder -- /bin/sh -c TEST_VAR=TEST_VAL echo ${TEST_VAR}" {
		t.Errorf("kubectl build failed %s", cmd.String())
	}

	c, _ = BuildContext(builder.ContextDefinition{
		Type: "remote",
		SSH: builder.SSHConfigDefinition{
			Options: []string{"-6", "-C"},
			User:    "root",
			Host:    "host",
		},
	}, wcfg)

	cmd, _ = c.createCommand(context.Background(), "echo ${TEST_VAR}")
	if cmd.String() != "/usr/bin/ssh -6 -C -T root@host /bin/sh -c echo ${TEST_VAR}" {
		t.Errorf("ssh build failed %s", cmd.String())
	}
}
