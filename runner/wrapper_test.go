package runner

import "testing"

func TestDockerWrapperWrap(t *testing.T) {
	tt := []struct {
		name    string
		spec    DockerSpec
		command string
		env     map[string]string
		dir     string
		want    string
	}{
		{
			name:    "run no env no dir",
			spec:    DockerSpec{Image: "alpine"},
			command: "echo hi",
			want:    "docker run --rm alpine sh -c 'echo hi'",
		},
		{
			name:    "run with env and dir",
			spec:    DockerSpec{Image: "alpine"},
			command: "echo hi",
			env:     map[string]string{"FOO": "bar baz", "QUOTE": "it's"},
			dir:     "/app",
			want:    `docker run --rm -e 'FOO=bar baz' -e "QUOTE=it's" -w /app alpine sh -c 'echo hi'`,
		},
		{
			name:    "run with host",
			spec:    DockerSpec{Image: "alpine", Host: "tcp://1.2.3.4:2375"},
			command: "echo hi",
			want:    "docker -H tcp://1.2.3.4:2375 run --rm alpine sh -c 'echo hi'",
		},
		{
			name:    "run with options",
			spec:    DockerSpec{Image: "alpine", Options: []string{"--network", "host"}},
			command: "echo hi",
			want:    "docker run --rm --network host alpine sh -c 'echo hi'",
		},
		{
			name:    "run with custom shell",
			spec:    DockerSpec{Image: "alpine", Shell: []string{"bash", "-lc"}},
			command: "echo hi",
			want:    "docker run --rm alpine bash -lc 'echo hi'",
		},
		{
			name:    "exec with container env and dir",
			spec:    DockerSpec{Container: "mycontainer"},
			command: "echo hi",
			env:     map[string]string{"FOO": "bar"},
			dir:     "/app",
			want:    "docker exec -e 'FOO=bar' -w /app mycontainer sh -c 'echo hi'",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			w := newDockerWrapper(tc.spec)
			got := w.wrap(tc.command, tc.env, tc.dir)
			if got != tc.want {
				t.Errorf("wrap() =\n  %q\nwant\n  %q", got, tc.want)
			}
		})
	}
}

func TestKubectlWrapperWrap(t *testing.T) {
	tt := []struct {
		name    string
		spec    KubernetesSpec
		command string
		env     map[string]string
		dir     string
		want    string
	}{
		{
			name:    "pod only",
			spec:    KubernetesSpec{Pod: "mypod"},
			command: "echo hi",
			want:    "kubectl exec mypod -- sh -c 'echo hi'",
		},
		{
			name: "namespace container kubeconfig context",
			spec: KubernetesSpec{
				Pod:         "mypod",
				Namespace:   "myns",
				Container:   "mycontainer",
				KubeContext: "mycluster",
				Kubeconfig:  "/path/to/kubeconfig",
			},
			command: "echo hi",
			want:    "kubectl --kubeconfig /path/to/kubeconfig --context mycluster -n myns exec mypod -c mycontainer -- sh -c 'echo hi'",
		},
		{
			name:    "with env and dir",
			spec:    KubernetesSpec{Pod: "mypod"},
			command: "echo hi",
			env:     map[string]string{"FOO": "bar baz", "QUOTE": "it's"},
			dir:     "/app",
			want:    `kubectl exec mypod -- sh -c "export FOO='bar baz' QUOTE=\"it's\"; cd /app && echo hi"`,
		},
		{
			name:    "with options",
			spec:    KubernetesSpec{Pod: "mypod", Options: []string{"--request-timeout", "30s"}},
			command: "echo hi",
			want:    "kubectl exec --request-timeout 30s mypod -- sh -c 'echo hi'",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			w := newKubectlWrapper(tc.spec)
			got := w.wrap(tc.command, tc.env, tc.dir)
			if got != tc.want {
				t.Errorf("wrap() =\n  %q\nwant\n  %q", got, tc.want)
			}
		})
	}
}

func TestSSHWrapperWrap(t *testing.T) {
	tt := []struct {
		name    string
		spec    SSHSpec
		command string
		env     map[string]string
		dir     string
		want    string
	}{
		{
			name:    "host only",
			spec:    SSHSpec{Host: "example.com"},
			command: "echo hi",
			want:    "ssh example.com 'echo hi'",
		},
		{
			name:    "user@host",
			spec:    SSHSpec{Host: "example.com", User: "deploy"},
			command: "echo hi",
			want:    "ssh deploy@example.com 'echo hi'",
		},
		{
			name:    "with port identity and options",
			spec:    SSHSpec{Host: "example.com", Port: 2222, IdentityFile: "/home/me/.ssh/id_rsa", Options: []string{"-o", "StrictHostKeyChecking=no"}},
			command: "echo hi",
			want:    "ssh -p 2222 -i /home/me/.ssh/id_rsa -o StrictHostKeyChecking=no example.com 'echo hi'",
		},
		{
			name:    "with env and dir",
			spec:    SSHSpec{Host: "example.com"},
			command: "echo hi",
			env:     map[string]string{"FOO": "bar baz", "QUOTE": "it's"},
			dir:     "/app",
			want:    `ssh example.com "export FOO='bar baz' QUOTE=\"it's\"; cd /app && echo hi"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			w := newSSHWrapper(tc.spec)
			got := w.wrap(tc.command, tc.env, tc.dir)
			if got != tc.want {
				t.Errorf("wrap() =\n  %q\nwant\n  %q", got, tc.want)
			}
		})
	}
}
