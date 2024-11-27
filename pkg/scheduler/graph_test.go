package scheduler_test

import (
	"errors"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/Ensono/taskctl/pkg/runner"
	"github.com/Ensono/taskctl/pkg/scheduler"
	"github.com/Ensono/taskctl/pkg/task"
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
	// - stage1
	// - stage4
	// - stage3
	// - stage2
	if bfs[2].Name != "stage3" {
		t.Errorf("penultimate node (%q) should be stage3", bfs[2].Name)
	}

	if bfs[3].Name != "stage2" {
		t.Errorf("last node (%q) should be stage2", bfs[3].Name)
	}

	// stage1 or stage4 are run in parallel and the first node can be either
	if !slices.Contains([]string{"stage1", "stage4"}, bfs[0].Name) {
		t.Errorf("first node (%q) should be either stage1 or stage4", bfs[0].Name)
	}
}

func TestExecutionGraph_BFS_Sorted(t *testing.T) {
	t.Parallel()
	stages := []*scheduler.Stage{
		scheduler.NewStage("stage1", func(s *scheduler.Stage) {
			s.Pipeline = &scheduler.ExecutionGraph{}
		}),
		scheduler.NewStage("stage2", func(s *scheduler.Stage) {
			s.DependsOn = []string{"stage3", "stage1"}
		}),
		scheduler.NewStage("stage3", func(s *scheduler.Stage) {
			s.DependsOn = []string{"stage1"}
		}),
		scheduler.NewStage("stage4"),
		scheduler.NewStage("stage5"),
	}

	g, err := scheduler.NewExecutionGraph("test_bfs", stages...)
	if err != nil {
		t.Fatal(err)
	}

	bfs := g.BFSNodesFlattened(scheduler.RootNodeName)
	sort.Sort(bfs)
	// after sorting it needs to look like this - always
	// - stage1
	// - stage4
	// - stage5
	// - stage3
	// - stage2
	for idx, v := range []string{"stage1", "stage4", "stage5", "stage3", "stage2"} {
		if bfs[idx].Name != v {
			t.Errorf("last node (%q) should be %s", bfs[idx].Name, v)
		}
	}
	
}

func TestExecutionGraph_Error(t *testing.T) {
	t.Parallel()
	s1, _ := scheduler.NewExecutionGraph("stage1")
	stages := []*scheduler.Stage{
		scheduler.NewStage("stage1", func(s *scheduler.Stage) {
			s1.AddStage(scheduler.NewStage("t:one", func(s *scheduler.Stage) {
				task := task.NewTask("t:one")
				task.Commands = []string{"true"}
				s.Task = task
			}))
			s.Pipeline = s1
		}),
		scheduler.NewStage("stage2", func(s *scheduler.Stage) {
			task := task.NewTask("s2:t2")
			task.Commands = []string{"false"}
			// task.AllowFailure = false
			s.Task = task
			s.DependsOn = []string{"stage3", "stage1"}
		}),
		scheduler.NewStage("stage3", func(s *scheduler.Stage) {
			task := task.NewTask("stage3")
			task.Commands = []string{"true"}
			s.Task = task
			s.DependsOn = []string{"stage1"}
		}),
		scheduler.NewStage("stage4", func(s *scheduler.Stage) {
			task := task.NewTask("s4:t1")
			task.Commands = []string{"false"}
			// task.AllowFailure = false
			s.Task = task
		}),
		scheduler.NewStage("stage5", func(s *scheduler.Stage) {
			task := task.NewTask("stage5")
			task.Commands = []string{"true"}
			s.Task = task
		}),
	}
	g, err := scheduler.NewExecutionGraph("test_bfs", stages...)
	if err != nil {
		t.Fatal(err)
	}
	tr, _ := runner.NewTaskRunner()
	s := scheduler.NewScheduler(tr)

	if err := s.Schedule(g); err == nil {
		t.Error("graph failed to error")
	}

	if !strings.Contains(g.Error().Error(), "stage2") {
		t.Errorf("incorrect error logged, got %s, wanted to include stage2\n", g.Error().Error())
	}
}
