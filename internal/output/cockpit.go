package output

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/taskctl/taskctl/internal/tui"
	"github.com/taskctl/taskctl/task"
)

// base is a process-wide singleton: all tasks in a run share one cockpit
// dashboard. The bubbletea program starts at most once (startOnce), and
// output.Close permanently closes closeCh to quit it — so the cockpit is
// one-shot and cannot be restarted within the process. taskctl's CLI runs a
// single cockpit session per invocation (run is single-shot; watch forces raw
// output), so this suffices. base is intentionally never reset to nil: a reset
// would let a new program start while the old one tears down, racing two
// bubbletea programs. baseMu guards lazy creation against the scheduler's
// concurrent task goroutines.
var (
	base   *baseCockpit
	baseMu sync.Mutex
)

// baseCockpit is the live multi-task dashboard: a single bubbletea program that
// shows a spinner with the currently-running tasks and prints a "Finished" line
// as each completes. It is safe for concurrent add/remove from the scheduler's
// task goroutines. The cockpit lives here rather than in internal/tui because it
// is single-consumer (only output drives it) and shares its lifecycle with the
// decorator and singleton below; it borrows only the color palette from tui.
type baseCockpit struct {
	w         io.Writer
	mu        sync.Mutex // guards prog
	prog      *tea.Program
	startOnce sync.Once
	closeCh   chan bool
}

type cockpitOutputDecorator struct {
	b *baseCockpit
	t *task.Task
}

// taskStartedMsg is sent when a task's output begins.
type taskStartedMsg struct {
	name string
}

// taskFinishedMsg is sent when a task's output completes.
type taskFinishedMsg struct {
	name     string
	errored  bool
	duration time.Duration
}

type cockpitModel struct {
	spin  spinner.Model
	tasks []string
}

func (m cockpitModel) Init() tea.Cmd {
	return m.spin.Tick
}

func (m cockpitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case taskStartedMsg:
		m.tasks = append(m.tasks, msg.name)
		slices.Sort(m.tasks)
		return m, nil
	case taskFinishedMsg:
		if i := slices.Index(m.tasks, msg.name); i != -1 {
			m.tasks = slices.Delete(m.tasks, i, i+1)
		}

		mark := tui.StyleSuccess.Render("✔")
		if msg.errored {
			mark = tui.StyleError.Render("✗")
		}
		line := fmt.Sprintf("%s Finished %s in %s", mark, tui.StyleBold.Render(msg.name), msg.duration)

		return m, tea.Println(line)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m cockpitModel) View() tea.View {
	if len(m.tasks) == 0 {
		return tea.NewView("")
	}

	return tea.NewView(m.spin.View() + " Running: " + strings.Join(m.tasks, ", "))
}

func (b *baseCockpit) start() {
	b.startOnce.Do(func() {
		sp := spinner.New()
		sp.Spinner = spinner.Dot
		sp.Style = tui.StyleSpinner

		m := cockpitModel{spin: sp}

		b.mu.Lock()
		b.prog = tea.NewProgram(m, tea.WithOutput(b.w), tea.WithInput(nil), tea.WithoutSignalHandler())
		p := b.prog
		b.mu.Unlock()

		go func() { _, _ = p.Run() }()

		go func() {
			<-b.closeCh
			p.Quit()
		}()
	})
}

func (b *baseCockpit) add(t *task.Task) {
	b.start()

	b.mu.Lock()
	p := b.prog
	b.mu.Unlock()

	if p == nil {
		return
	}
	p.Send(taskStartedMsg{name: t.Name})
}

func (b *baseCockpit) remove(t *task.Task) {
	b.mu.Lock()
	p := b.prog
	b.mu.Unlock()

	if p == nil {
		return
	}
	p.Send(taskFinishedMsg{name: t.Name, errored: t.Errored, duration: t.Duration()})
}

// wait blocks until the cockpit's program has fully shut down (after Quit),
// so final output is flushed and the terminal restored before the caller proceeds.
func (b *baseCockpit) wait() {
	b.mu.Lock()
	p := b.prog
	b.mu.Unlock()

	if p == nil {
		return
	}
	p.Wait()
}

func newCockpitOutputWriter(t *task.Task, w io.Writer, close chan bool) *cockpitOutputDecorator {
	baseMu.Lock()
	if base == nil {
		base = &baseCockpit{
			w:       w,
			closeCh: close,
		}
	}
	b := base
	baseMu.Unlock()

	return &cockpitOutputDecorator{
		t: t,
		b: b,
	}
}

func (d *cockpitOutputDecorator) Write(p []byte) (int, error) {
	return len(p), nil
}

func (d *cockpitOutputDecorator) WriteHeader() error {
	d.b.add(d.t)
	return nil
}

func (d *cockpitOutputDecorator) WriteFooter() error {
	d.b.remove(d.t)
	return nil
}
