package output

import (
	"bytes"
	"strings"
	"testing"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/taskctl/taskctl/task"
)

func newTestDashboardDecorator() *dashboardOutputDecorator {
	return newDashboardOutputWriter(&task.Task{Name: "task1"}, bytes.NewBuffer(nil), make(chan bool))
}

func Test_dashboardOutputDecorator(t *testing.T) {
	// WriteHeader starts the process-wide bubbletea program; it quits only when
	// closeCh closes, so drive base's channel here and close it to tear down —
	// otherwise a later Close() (e.g. TestNewTaskOutput) blocks in p.Wait().
	closeCh = make(chan bool)
	dec := newDashboardOutputWriter(&task.Task{Name: "task1"}, bytes.NewBuffer(nil), closeCh)
	if err := dec.WriteHeader(); err != nil {
		t.Fatal(err)
	}
	if _, err := dec.Write([]byte("lorem ipsum")); err != nil {
		t.Fatal(err)
	}
	if err := dec.WriteFooter(); err != nil {
		t.Fatal(err)
	}
	close(closeCh)
}

func Test_dashboardOutputDecorator_Write_partialLineCarry(t *testing.T) {
	dec := newTestDashboardDecorator()

	if _, err := dec.Write([]byte("par")); err != nil {
		t.Fatal(err)
	}
	if _, err := dec.Write([]byte("tial\nfull\nrest")); err != nil {
		t.Fatal(err)
	}

	if got := string(dec.bufStdout); got != "rest" {
		t.Errorf("bufStdout = %q, want %q", got, "rest")
	}
}

// Both '\n' and '\r' terminate a line, so a carriage-return progress bar
// surfaces its latest state and no raw '\r' survives into the carry buffer.
func Test_dashboardOutputDecorator_scanLine_carriageReturn(t *testing.T) {
	dec := newTestDashboardDecorator()

	dec.mu.Lock()
	last := dec.scanLine("stdout", []byte("10%\r20%\r30%"))
	dec.mu.Unlock()

	if last != "20%" {
		t.Errorf("last = %q, want %q", last, "20%")
	}
	if got := string(dec.bufStdout); got != "30%" {
		t.Errorf("bufStdout = %q, want %q", got, "30%")
	}
}

// A partial stdout write must not merge with a stderr line into one garbled
// dashboard line — each stream carries its own buffer.
func Test_dashboardOutputDecorator_streamsDoNotMerge(t *testing.T) {
	dec := newTestDashboardDecorator()
	stderr := dec.StreamWriter("stderr")

	if _, err := dec.Write([]byte("hello")); err != nil { // partial stdout, no newline
		t.Fatal(err)
	}
	if _, err := stderr.Write([]byte("world\n")); err != nil {
		t.Fatal(err)
	}

	if got := string(dec.bufStdout); got != "hello" {
		t.Errorf("bufStdout = %q, want %q", got, "hello")
	}
	if got := string(dec.bufStderr); got != "" {
		t.Errorf("bufStderr = %q, want empty", got)
	}
}

// Output without line terminators (e.g. a carriage-return progress bar) must not
// grow the carry buffer without bound.
func Test_dashboardOutputDecorator_Write_boundsBuffer(t *testing.T) {
	dec := newTestDashboardDecorator()

	for range 100 {
		if _, err := dec.Write(bytes.Repeat([]byte("x"), 1000)); err != nil {
			t.Fatal(err)
		}
	}

	if got := len(dec.bufStdout); got > maxDashboardLineBuf {
		t.Errorf("bufStdout len = %d, want <= %d", got, maxDashboardLineBuf)
	}
}

func update(m dashboardModel, msg tea.Msg) dashboardModel {
	next, _ := m.Update(msg)
	return next.(dashboardModel)
}

func newTestDashboardModel() dashboardModel {
	return dashboardModel{spin: spinner.New()}
}

func Test_dashboardModel_Update(t *testing.T) {
	m := newTestDashboardModel()
	m = update(m, taskStartedMsg{id: 1, name: "x"})
	m = update(m, taskOutputMsg{id: 1, line: "hello"})

	if i := m.rowIndex(1); i == -1 || m.rows[i].lastLine != "hello" {
		t.Fatalf("row 1 lastLine = %q, want %q", m.rows[i].lastLine, "hello")
	}

	m = update(m, taskFinishedMsg{id: 1, name: "x"})
	if m.rowIndex(1) != -1 {
		t.Error("row 1 should have been removed on finish")
	}
}

// Two concurrent clones of one task share a Name but not an id; finishing one
// must not clear the other's row.
func Test_dashboardModel_Update_sameNameConcurrent(t *testing.T) {
	m := newTestDashboardModel()
	m = update(m, taskStartedMsg{id: 1, name: "build"})
	m = update(m, taskStartedMsg{id: 2, name: "build"})
	m = update(m, taskOutputMsg{id: 2, line: "still going"})

	m = update(m, taskFinishedMsg{id: 1, name: "build"})

	i := m.rowIndex(2)
	if i == -1 {
		t.Fatal("row 2 should still be running after row 1 finished")
	}
	if m.rows[i].lastLine != "still going" {
		t.Errorf("row 2 lastLine = %q, want %q", m.rows[i].lastLine, "still going")
	}
}

// Output arriving after a task finished (e.g. a shell background job that
// outlived its task) must be dropped, not resurrect the row.
func Test_dashboardModel_Update_lateOutputDropped(t *testing.T) {
	m := newTestDashboardModel()
	m = update(m, taskStartedMsg{id: 1, name: "x"})
	m = update(m, taskFinishedMsg{id: 1, name: "x"})
	m = update(m, taskOutputMsg{id: 1, line: "late"})

	if len(m.rows) != 0 {
		t.Errorf("rows = %d, want 0 (late output must not re-add a row)", len(m.rows))
	}
}

func Test_dashboardModel_View_truncatesToWidth(t *testing.T) {
	m := newTestDashboardModel()
	m = update(m, taskStartedMsg{id: 1, name: "a-very-long-task-name-that-exceeds-the-terminal-width-on-its-own"})
	m = update(m, taskOutputMsg{id: 1, line: "this is a very long line of task output that should be truncated"})
	m = update(m, tea.WindowSizeMsg{Width: 20})

	view := m.View()
	for line := range strings.SplitSeq(view.Content, "\n") {
		if w := ansi.StringWidth(line); w > 20 {
			t.Errorf("line %q has width %d, want <= 20", line, w)
		}
	}
}
