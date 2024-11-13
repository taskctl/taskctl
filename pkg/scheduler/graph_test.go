package scheduler_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Ensono/taskctl/pkg/scheduler"
)

func TestExecutionGraph_AddStage(t *testing.T) {
	t.Parallel()
	g, err := scheduler.NewExecutionGraph("test")
	if err != nil {
		t.Fatal(err)
	}

	err = g.AddStage(scheduler.NewStage("stage1", func(s *scheduler.Stage) {
		s.DependsOn = []string{"stage2"}
	}))
	if err != nil {
		t.Fatal()
	}
	err = g.AddStage(scheduler.NewStage("stage2", func(s *scheduler.Stage) {
		s.DependsOn = []string{"stage1"}
	}))
	if err == nil {
		t.Fatal("add stage cycle detection failed\n")
	}
	if err != nil && !errors.Is(err, scheduler.ErrCycleDetected) {
		t.Fatalf("incorrect error (%q), wanted: %q\n", err, scheduler.ErrCycleDetected)
	}
}

func TestExecutionGraph_AddStagesAtOnce(t *testing.T) {
	t.Parallel()

	s1 := scheduler.NewStage("stage1", func(s *scheduler.Stage) {
		s.DependsOn = []string{"stage2"}
	})

	s2 := scheduler.NewStage("stage2", func(s *scheduler.Stage) {
		s.DependsOn = []string{"stage1"}
	})

	_, err := scheduler.NewExecutionGraph("test", s1, s2)

	if err == nil {
		t.Fatal("add stage cycle detection failed\n")
	}
	if err != nil && !errors.Is(err, scheduler.ErrCycleDetected) {
		t.Fatalf("incorrect error (%q), wanted: %q\n", err, scheduler.ErrCycleDetected)
	}
}

func TestExecutionGraph_Nodes(t *testing.T) {
	ttests := map[string]struct {
		stages     []*scheduler.Stage
		checkNodes []string
		errTyp     error
	}{
		"all nodes exist": {
			stages: []*scheduler.Stage{
				scheduler.NewStage("stage1"),
				scheduler.NewStage("stage2"),
				scheduler.NewStage("stage3"),
				scheduler.NewStage("stage4"),
			},
			checkNodes: []string{"stage1", "stage2", "stage3", "stage4"},
			errTyp:     nil,
		},
		"node not found error": {
			stages: []*scheduler.Stage{
				scheduler.NewStage("stage1"),
				scheduler.NewStage("stage2"),
				scheduler.NewStage("stage3"),
				scheduler.NewStage("stage4"),
			},
			checkNodes: []string{"stage7"},
			errTyp:     scheduler.ErrNodeNotFound,
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			g, err := scheduler.NewExecutionGraph(name, tt.stages...)
			if err != nil {
				if !errors.Is(err, tt.errTyp) {
					t.Fatalf("different error expected, %q, want: %q", err.Error(), tt.errTyp)
				}
			}
			for _, v := range tt.checkNodes {
				if _, err := g.Node(v); err != nil {
					if !errors.Is(err, tt.errTyp) {
						t.Fatalf("different error expected, %q, want: %q", err.Error(), tt.errTyp)
					}
				}
			}
			for _, v := range g.Children(scheduler.RootNodeName) {
				if !slices.Contains(tt.stages, v) {
					t.Fatal("children not contain expected node")
				}
			}
		})
	}
}

func TestExecutionGraph_TreeWalk_BFS(t *testing.T) {
	t.Parallel()
	stages := []*scheduler.Stage{
		scheduler.NewStage("stage1", func(s *scheduler.Stage) {
			s.Pipeline = &scheduler.ExecutionGraph{}
		}),
		scheduler.NewStage("stage2", func(s *scheduler.Stage) {
			s.DependsOn = []string{"stage3"}
		}),
		scheduler.NewStage("stage3", func(s *scheduler.Stage) {
			s.DependsOn = []string{"stage1"}
		}),
		scheduler.NewStage("stage4"),
	}
	g, err := scheduler.NewExecutionGraph("test_bfs", stages...)
	if err != nil {
		t.Fatal(err)
	}

	bfs := g.BFSNodesFlattened(scheduler.RootNodeName)

	if bfs[3].Name != "stage2" {
		t.Errorf("last node (%q) should be stage2", bfs[3].Name)
	}

	if bfs[2].Name != "stage3" {
		t.Errorf("penultimate node (%q) should be stage3", bfs[2].Name)
	}
	// stage1 or stage4 are run in parallel and the first node can be either
	if !slices.Contains([]string{"stage1", "stage4"}, bfs[0].Name) {
		t.Errorf("first node (%q) should be either stage1 or stage4", bfs[0].Name)
	}
}
