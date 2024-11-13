package output_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Ensono/taskctl/pkg/output"
	"github.com/Ensono/taskctl/pkg/task"
	"github.com/sirupsen/logrus"
)

func TestOutput_prefixedOutputDecorator(t *testing.T) {
	ttests := map[string]struct {
		input  []byte
		expect string
	}{
		"new line added": {
			input:  []byte("lorem ipsum"),
			expect: "\x1b[36mtask1\x1b[0m: lorem ipsum\r\n",
		},
		"contains new lines": {
			input: []byte(`lorem ipsum
multiline stuff`),
			expect: "\x1b[36mtask1\x1b[0m: lorem ipsum\r\n\x1b[36mtask1\x1b[0m: multiline stuff\r\n",
		},
		"contains new lines with trailing newline": {
			input: []byte(`lorem ipsum
multiline stuff
`),
			expect: "\x1b[36mtask1\x1b[0m: lorem ipsum\r\n\x1b[36mtask1\x1b[0m: multiline stuff\r\n",
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
