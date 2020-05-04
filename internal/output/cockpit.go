package output

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/logrusorgru/aurora"

	"github.com/taskctl/taskctl/internal/task"
)

var cockpitDecorator *CockpitOutputDecorator

type CockpitOutputDecorator struct {
	w       io.Writer
	tasks   []*task.Task
	mu      sync.Mutex
	spinner *spinner.Spinner
	charSet int
	closeCh chan bool
}

func NewCockpitOutputWriter(w io.Writer, closeCh chan bool) *CockpitOutputDecorator {
	if cockpitDecorator == nil {
		cockpitDecorator = &CockpitOutputDecorator{
			charSet: 14,
			w:       w,
			tasks:   make([]*task.Task, 0),
			closeCh: closeCh,
		}
	}

	return cockpitDecorator
}

func (d *CockpitOutputDecorator) Write(p []byte) (int, error) {
	return len(p), nil
}

func (d *CockpitOutputDecorator) WriteHeader(t *task.Task) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.tasks = append(d.tasks, t)

	if d.spinner == nil {
		d.spinner = d.startSpinner()
		go func() {
			<-d.closeCh
			d.spinner.Stop()
		}()
	}

	return nil
}

func (d *CockpitOutputDecorator) WriteFooter(t *task.Task) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for i := 0; i < len(d.tasks); i++ {
		if d.tasks[i] == t {
			d.tasks = append(d.tasks[:i], d.tasks[i+1:]...)
		}
	}

	var mark = aurora.Green("✔")
	if t.Errored {
		mark = aurora.Red("✗")
	}

	d.spinner.FinalMSG = fmt.Sprintf("%s Finished %s in %s\r\n", mark, aurora.Bold(t.Name), t.Duration())
	d.spinner.Restart()
	d.spinner.FinalMSG = ""
	return nil
}

func (d *CockpitOutputDecorator) ForTask(t *task.Task) DecoratedOutputWriter {
	return d
}

func (d *CockpitOutputDecorator) startSpinner() *spinner.Spinner {
	s := spinner.New(spinner.CharSets[d.charSet], 100*time.Millisecond, spinner.WithColor("yellow"))
	s.Writer = d.w
	s.PreUpdate = func(s *spinner.Spinner) {
		tasks := make([]string, 0)
		for _, v := range d.tasks {
			tasks = append(tasks, v.Name)
		}
		sort.Strings(tasks)
		s.Suffix = " Running: " + strings.Join(tasks, ", ")
	}
	s.Start()

	return s
}
