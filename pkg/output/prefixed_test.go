package output_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Ensono/taskctl/pkg/output"
	"github.com/Ensono/taskctl/pkg/task"
)

func TestOutput_prefixedOutputDecorator(t *testing.T) {
	ttests := map[string]struct {
		input  []byte
		expect string
	}{
		"new line added": {
			input:  []byte("lorem ipsum"),
			expect: "\x1b[36mtask1\x1b[0m: lorem ipsum\n",
		},
		"contains new line": {
			input: []byte(`lorem ipsum
multiline stuff`),
			expect: "\x1b[36mtask1\x1b[0m: lorem ipsum\n\x1b[36mtask1\x1b[0m: multiline stuff\n",
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			l, b := &bytes.Buffer{}, &bytes.Buffer{}
			logrus.SetOutput(l)

			dec := output.NewPrefixedOutputWriter(&task.Task{Name: "task1"}, b)
			err := dec.WriteHeader()
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(l.String(), "Running task task1...") {
				t.Fatal()
			}

			n, err := dec.Write(tt.input)
			if err != nil && n == 0 {
				t.Fatal()
			}
			if !strings.EqualFold(b.String(), tt.expect) {
				t.Fatalf("got: %s\nwanted: %s\n", b.String(), tt.expect)
			}

			err = dec.WriteFooter()
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(l.String(), "task1 finished") {
				t.Fatal()
			}
		})
	}
}
