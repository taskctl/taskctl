package cmdutils_test

import (
	"bytes"
	"testing"

	"github.com/Ensono/taskctl/internal/cmdutils"
	"github.com/Ensono/taskctl/pkg/scheduler"
)

func TestPrintSummary(t *testing.T) {
	t.Run("no stages run", func(t *testing.T) {
		out := bytes.Buffer{}
		cmdutils.PrintSummary(&scheduler.ExecutionGraph{}, &out)
		if len(out.Bytes()) == 0 {
			t.Fatal("got 0, wanted bytes written")
		}
	})
	t.Run("one stage run", func(t *testing.T) {
		out := bytes.Buffer{}
		graph, _ := scheduler.NewExecutionGraph()
		graph.AddStage(&scheduler.Stage{
			Name:   "foo",
			Status: scheduler.StatusDone,
		})
		cmdutils.PrintSummary(graph, &out)
		if len(out.Bytes()) == 0 {
			t.Fatal("got 0, wanted bytes written")
		}
	})
}
