package output

import (
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"

	"github.com/taskctl/taskctl/internal/task"
)

var base *baseCockpit

type baseCockpit struct {
	w       io.Writer
	tasks   []*task.Task
	mu      sync.Mutex
	spinner *spinner.Spinner
	charSet int
	closeCh chan bool
}

type CockpitOutputDecorator struct {
	b *baseCockpit
	t *task.Task
}

func NewCockpitOutputWriter(t *task.Task, w io.Writer) *CockpitOutputDecorator {
	if base == nil {
		base = &baseCockpit{
			charSet: 14,
			w:       w,
			tasks:   make([]*task.Task, 0),
			closeCh: closeCh,
		}
	}

	return &CockpitOutputDecorator{
		t: t,
		b: base,
	}
}

func (d *CockpitOutputDecorator) Write(p []byte) (int, error) {
	return len(p), nil
}

func (d *CockpitOutputDecorator) WriteHeader() error {
	d.b.Add(d.t)
	return nil
}

func (d *CockpitOutputDecorator) WriteFooter() error {
	d.b.Remove(d.t)
	return nil
}

func (b *baseCockpit) start() *spinner.Spinner {
	if b.spinner != nil {
		return b.spinner
	}

	s := spinner.New(spinner.CharSets[b.charSet], 100*time.Millisecond, spinner.WithColor("yellow"))
	s.Writer = b.w
	s.PreUpdate = func(s *spinner.Spinner) {
		tasks := make([]string, 0)
		for _, v := range b.tasks {
			tasks = append(tasks, v.Name)
		}
		sort.Strings(tasks)
		s.Suffix = " Running: " + strings.Join(tasks, ", ")
	}
	s.Start()

	return s
}

func (b *baseCockpit) Add(t *task.Task) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.tasks = append(b.tasks, t)

	if b.spinner == nil {
		b.spinner = b.start()
		go func() {
			<-b.closeCh
			b.spinner.Stop()
		}()
	}
}

func (b *baseCockpit) Remove(t *task.Task) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.tasks = append(b.tasks, t)

	if b.spinner == nil {
		b.spinner = b.start()
		go func() {
			<-b.closeCh
			b.spinner.Stop()
		}()
	}
}
