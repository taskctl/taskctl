package output

import (
	"fmt"
	"io"
	"io/ioutil"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/logrusorgru/aurora"

	"github.com/taskctl/taskctl/pkg/task"
)

type CockpitOutputDecorator struct {
	w       io.Writer
	tasks   map[uint32]string
	mu      sync.Mutex
	spinner *spinner.Spinner
	charSet int
}

func NewCockpitOutputWriter() *CockpitOutputDecorator {
	return &CockpitOutputDecorator{
		charSet: 14,
		w:       ioutil.Discard,
		tasks:   make(map[uint32]string),
	}
}

func (d *CockpitOutputDecorator) WithWriter(w io.Writer) {
	d.w = w
}
func (d *CockpitOutputDecorator) Write(t *task.Task, b []byte) error {
	return nil
}

func (d *CockpitOutputDecorator) WriteHeader(t *task.Task) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.tasks[t.Index] = t.Name

	if d.spinner == nil {
		d.spinner = d.startSpinner()
	}

	return nil
}

func (d *CockpitOutputDecorator) WriteFooter(t *task.Task) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.tasks, t.Index)

	var mark = aurora.Green("✔")
	if t.Errored {
		mark = aurora.Red("✗")
	}
	d.spinner.FinalMSG = fmt.Sprintf("%s Finished %s in %s\r\n", mark, aurora.Bold(t.Name), t.Duration())
	d.spinner.Restart()
	d.spinner.FinalMSG = ""
	return nil
}

func (d *CockpitOutputDecorator) startSpinner() *spinner.Spinner {
	s := spinner.New(spinner.CharSets[d.charSet], 100*time.Millisecond, spinner.WithColor("yellow"))
	s.Writer = d.w
	s.PreUpdate = func(s *spinner.Spinner) {
		tasks := make([]string, 0)
		for _, v := range d.tasks {
			tasks = append(tasks, v)
		}
		sort.Strings(tasks)
		s.Suffix = " Running: " + strings.Join(tasks, ", ")
	}
	s.Start()

	return s
}

func (d *CockpitOutputDecorator) Close() {
	d.spinner.Stop()
}
