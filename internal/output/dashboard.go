package output

import (
	"bytes"
	"cmp"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/taskctl/taskctl/internal/tui"
	"github.com/taskctl/taskctl/task"
)

const (
	maxDashboardRows = 8
	// maxDashboardLineBuf caps the per-stream partial-line carry so a task
	// streaming output without line terminators can't grow the buffer without
	// bound; only the trailing bytes matter for a one-line display.
	maxDashboardLineBuf = 4096
)

// base is a process-wide singleton: all tasks in a run share one dashboard. The
// bubbletea program starts at most once (startOnce), and output.Close
// permanently closes closeCh to quit it — so the dashboard is one-shot and
// cannot be restarted within the process. taskctl's CLI runs a single dashboard
// session per invocation (run is single-shot; watch forces raw output), so this
// suffices. base is intentionally never reset to nil: a reset would let a new
// program start while the old one tears down, racing two bubbletea programs.
// baseMu guards lazy creation against the scheduler's concurrent task goroutines.
var (
	base   *baseDashboard
	baseMu sync.Mutex

	// rowID distinguishes concurrently running clones of the same task, which
	// share a Name; keying rows by a unique id keeps one clone's finish from
	// clearing another's elapsed time and output line.
	rowID atomic.Uint64
)

// baseDashboard is the live multi-task dashboard: a single bubbletea program that
// shows a spinner with the currently-running tasks and prints a "Finished" line
// as each completes. It is safe for concurrent add/remove from the scheduler's
// task goroutines. The dashboard lives here rather than in internal/tui because it
// is single-consumer (only output drives it) and shares its lifecycle with the
// decorator and singleton below; it borrows only the color palette from tui.
type baseDashboard struct {
	w         io.Writer
	mu        sync.Mutex // guards prog
	prog      *tea.Program
	startOnce sync.Once
	closeCh   chan bool
}

type dashboardOutputDecorator struct {
	b  *baseDashboard
	t  *task.Task
	id uint64

	mu        sync.Mutex // guards the line buffers; stdout and stderr write from different goroutines
	bufStdout []byte
	bufStderr []byte
}

type dashboardStreamWriter struct {
	d      *dashboardOutputDecorator
	stream string
}

func (s *dashboardStreamWriter) Write(p []byte) (int, error) {
	return s.d.writeStream(s.stream, p)
}

type taskStartedMsg struct {
	id      uint64
	name    string
	started time.Time
}
type taskFinishedMsg struct {
	id       uint64
	name     string
	errored  bool
	duration time.Duration
}

type taskOutputMsg struct {
	id   uint64
	line string
}

type taskRow struct {
	id       uint64
	name     string
	started  time.Time
	lastLine string
}

type dashboardModel struct {
	spin  spinner.Model
	rows  []taskRow
	width int
}

func (m dashboardModel) Init() tea.Cmd {
	return m.spin.Tick
}

func (m dashboardModel) rowIndex(id uint64) int {
	return slices.IndexFunc(m.rows, func(r taskRow) bool { return r.id == id })
}

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case taskStartedMsg:
		m.rows = append(m.rows, taskRow{id: msg.id, name: msg.name, started: time.Now()})
		slices.SortFunc(m.rows, func(a, b taskRow) int {
			if c := strings.Compare(a.name, b.name); c != 0 {
				return c
			}
			return cmp.Compare(a.id, b.id)
		})
		return m, nil
	case taskFinishedMsg:
		if i := m.rowIndex(msg.id); i != -1 {
			m.rows = slices.Delete(m.rows, i, i+1)
		}

		mark := tui.StyleSuccess.Render("✔")
		if msg.errored {
			mark = tui.StyleError.Render("✗")
		}
		line := fmt.Sprintf("%s Finished %s in %s", mark, tui.StyleBold.Render(msg.name), msg.duration)

		return m, tea.Println(line)
	case taskOutputMsg:
		// A finished task's row is gone; drop late output (e.g. from a shell
		// background job that outlived its task) instead of resurrecting it.
		if i := m.rowIndex(msg.id); i != -1 {
			m.rows[i].lastLine = msg.line
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m dashboardModel) View() tea.View {
	if len(m.rows) == 0 {
		return tea.NewView("")
	}

	w := m.width
	if w <= 0 {
		w = 80
	}

	visible := m.rows
	var overflow int
	if len(visible) > maxDashboardRows {
		overflow = len(visible) - maxDashboardRows
		visible = visible[:maxDashboardRows]
	}

	spin := m.spin.View()
	rows := make([]string, 0, len(visible)*2+1)
	for _, r := range visible {
		row := fmt.Sprintf("%s %s (%s)", spin, r.name, time.Since(r.started).Round(time.Second))
		rows = append(rows, ansi.Truncate(row, w, "…"))

		if r.lastLine != "" {
			rows = append(rows, tui.StyleFaint.Render(ansi.Truncate("  "+r.lastLine, w, "…")))
		}
	}

	if overflow > 0 {
		rows = append(rows, fmt.Sprintf("… and %d more", overflow))
	}

	return tea.NewView(strings.Join(rows, "\n"))
}

func (b *baseDashboard) start() {
	b.startOnce.Do(func() {
		sp := spinner.New()
		sp.Spinner = spinner.Dot
		sp.Style = tui.StyleSpinner

		b.mu.Lock()
		b.prog = tea.NewProgram(dashboardModel{spin: sp}, tea.WithOutput(b.w), tea.WithInput(nil), tea.WithoutSignalHandler())
		p := b.prog
		b.mu.Unlock()

		go func() { _, _ = p.Run() }()

		go func() {
			<-b.closeCh
			p.Quit()
		}()
	})
}

func (b *baseDashboard) program() *tea.Program {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.prog
}

func (b *baseDashboard) send(msg tea.Msg) {
	if p := b.program(); p != nil {
		p.Send(msg)
	}
}

// wait blocks until the dashboard's program has fully shut down (after Quit),
// so final output is flushed and the terminal restored before the caller proceeds.
func (b *baseDashboard) wait() {
	if p := b.program(); p != nil {
		p.Wait()
	}
}

func newDashboardOutputWriter(t *task.Task, w io.Writer, close chan bool) *dashboardOutputDecorator {
	baseMu.Lock()
	if base == nil {
		base = &baseDashboard{
			w:       w,
			closeCh: close,
		}
	}
	b := base
	baseMu.Unlock()

	return &dashboardOutputDecorator{
		t:  t,
		b:  b,
		id: rowID.Add(1),
	}
}

func (d *dashboardOutputDecorator) buffer(stream string) *[]byte {
	if stream == "stderr" {
		return &d.bufStderr
	}
	return &d.bufStdout
}

// Write treats all writes as stdout; callers needing stream attribution use
// StreamWriter (kept separate so a partial stdout write never merges with stderr).
func (d *dashboardOutputDecorator) Write(p []byte) (int, error) {
	return d.writeStream("stdout", p)
}

func (d *dashboardOutputDecorator) StreamWriter(stream string) io.Writer {
	return &dashboardStreamWriter{d: d, stream: stream}
}

// writeStream forwards only the latest non-blank complete line of this write to
// the dashboard — one Send per write, not per line, since the view shows a
// single latest line per task.
func (d *dashboardOutputDecorator) writeStream(stream string, p []byte) (int, error) {
	d.mu.Lock()
	last := d.scanLine(stream, p)
	d.mu.Unlock()

	if last != "" {
		d.b.send(taskOutputMsg{id: d.id, line: last})
	}
	return len(p), nil
}

// scanLine consumes p into the stream's carry buffer and returns the latest
// non-blank line this write completed. Both '\n' and '\r' terminate a line, so
// carriage-return progress bars surface their latest state instead of
// accumulating invisibly (and no raw '\r' ever reaches the renderer, where it
// would overdraw the row). Callers must hold d.mu.
func (d *dashboardOutputDecorator) scanLine(stream string, p []byte) string {
	buf := d.buffer(stream)

	data := p
	if len(*buf) > 0 {
		data = append(*buf, p...)
	}

	var last []byte
	for {
		idx := bytes.IndexAny(data, "\r\n")
		if idx < 0 {
			break
		}
		if line := data[:idx]; len(bytes.TrimSpace(line)) > 0 {
			last = line
		}
		data = data[idx+1:]
	}
	// Copy before the carry is rewritten below — last may alias its backing array.
	line := string(last)

	if len(data) > maxDashboardLineBuf {
		data = data[len(data)-maxDashboardLineBuf:]
	}
	if cap(*buf) > maxDashboardLineBuf {
		*buf = make([]byte, 0, maxDashboardLineBuf)
	}
	*buf = append((*buf)[:0], data...)

	return line
}

func (d *dashboardOutputDecorator) WriteHeader() error {
	d.b.start()
	d.b.send(taskStartedMsg{id: d.id, name: d.t.Name})
	return nil
}

func (d *dashboardOutputDecorator) WriteFooter() error {
	d.b.send(taskFinishedMsg{id: d.id, name: d.t.Name, errored: d.t.Errored, duration: d.t.Duration()})
	return nil
}
