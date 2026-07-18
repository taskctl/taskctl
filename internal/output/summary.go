package output

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/taskctl/taskctl/internal/tui"
	"github.com/taskctl/taskctl/task"
)

const logTailLines = 10

// statusMarks drives both the counts header and the per-stage lines, in
// display order. An unknown status (future typo or addition) falls back to
// the last row rather than disappearing.
var statusMarks = []struct {
	status, sym, label string
	style              *lipgloss.Style
}{
	{"done", "✔", "succeeded", &tui.StyleSuccess},
	{"failed", "✗", "failed", &tui.StyleError},
	{"skipped", "⊘", "skipped", &tui.StyleFaint},
	{"canceled", "⊗", "canceled", &tui.StyleFaint},
}

// StageSummary is one row of the end-of-run summary — a task or pipeline stage
// with its final status and captured-output stats.
type StageSummary struct {
	Name        string
	Status      string
	Start       time.Time
	Duration    time.Duration
	ExitCode    int16
	OutputBytes int
	ErrMessage  string
	LogTail     []string
}

// SummarizeTask reads t's final state into a StageSummary without draining its
// output buffers, so a later reader of t.Log still sees the full output.
func SummarizeTask(t *task.Task) StageSummary {
	s := StageSummary{
		Name:        t.Name,
		Status:      TaskStatus(t),
		Start:       t.Start,
		Duration:    t.Duration(),
		ExitCode:    t.ExitCode,
		OutputBytes: t.Log.Stdout.Len() + t.Log.Stderr.Len(),
	}

	// A task that "succeeded" without ever starting actually failed before
	// execution (context, hook, or compile error) — those paths return an
	// error without setting Errored or Start.
	if s.Status == "done" && t.Start.IsZero() {
		s.Status = "failed"
	}

	if t.Errored {
		if t.Error != nil {
			s.ErrMessage = strings.TrimSpace(t.Error.Error())
		}
		s.LogTail = lastLines(&t.Log.Stderr, logTailLines)
		if len(s.LogTail) == 0 {
			s.LogTail = lastLines(&t.Log.Stdout, logTailLines)
		}
	}

	return s
}

// SummarizeTasks maps SummarizeTask over tasks, preserving their order.
func SummarizeTasks(tasks []*task.Task) []StageSummary {
	out := make([]StageSummary, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, SummarizeTask(t))
	}
	return out
}

// PrintRunSummary writes the human end-of-run summary to w, ordering stages by
// their start time regardless of the order items are passed in; stages that
// never started (zero Start: skipped/canceled) sort after the ones that ran.
func PrintRunSummary(w io.Writer, items []StageSummary, total time.Duration) {
	items = slices.Clone(items)
	slices.SortStableFunc(items, func(a, b StageSummary) int {
		switch {
		case a.Start.IsZero() && b.Start.IsZero():
			return 0
		case a.Start.IsZero():
			return 1
		case b.Start.IsZero():
			return -1
		}
		return a.Start.Compare(b.Start)
	})

	tui.Println(w, summaryHeader(items, total))

	nameWidth := 0
	for _, it := range items {
		nameWidth = max(nameWidth, lipgloss.Width(it.Name))
	}

	for _, it := range items {
		tui.Println(w, summaryLine(it, nameWidth))
		printFailureDetail(w, it)
	}
}

// statusIndex resolves a status to its statusMarks row, falling back to the
// last (canceled) row for unknown statuses so they are still counted and
// rendered.
func statusIndex(status string) int {
	for i, m := range statusMarks {
		if m.status == status {
			return i
		}
	}
	return len(statusMarks) - 1
}

func summaryHeader(items []StageSummary, total time.Duration) string {
	counts := make([]int, len(statusMarks))
	for _, it := range items {
		counts[statusIndex(it.Status)]++
	}

	var parts []string
	for i, m := range statusMarks {
		if n := counts[i]; n > 0 {
			parts = append(parts, m.style.Render(fmt.Sprintf("%s %d %s", m.sym, n, m.label)))
		}
	}
	parts = append(parts, tui.StyleBold.Render(formatDuration(total)+" total"))

	return strings.Join(parts, tui.StyleFaint.Render(" · "))
}

func summaryLine(it StageSummary, nameWidth int) string {
	m := statusMarks[statusIndex(it.Status)]
	pad := strings.Repeat(" ", max(0, nameWidth-lipgloss.Width(it.Name)))
	line := m.style.Render(m.sym+" "+it.Name) + pad

	switch it.Status {
	case "skipped", "canceled":
		return line + "  " + tui.StyleFaint.Render(it.Status)
	}

	line += "  " + formatDuration(it.Duration)
	if it.Status == "failed" {
		if it.ExitCode > 0 {
			line += tui.StyleError.Render(fmt.Sprintf("  exit %d", it.ExitCode))
		}
		if it.OutputBytes > 0 {
			line += tui.StyleFaint.Render(fmt.Sprintf("  (%s output)", humanizeBytes(it.OutputBytes)))
		}
	}
	return line
}

func formatDuration(d time.Duration) string {
	switch {
	case d >= time.Second:
		return d.Round(10 * time.Millisecond).String()
	case d >= time.Millisecond:
		return d.Round(100 * time.Microsecond).String()
	default:
		return d.Round(time.Microsecond).String()
	}
}

func printFailureDetail(w io.Writer, it StageSummary) {
	if it.Status != "failed" {
		return
	}
	if it.ErrMessage != "" {
		tui.Println(w, tui.StyleFaint.Render("    "+it.ErrMessage))
	}
	for _, l := range it.LogTail {
		tui.Println(w, tui.StyleFaint.Render("    "+l))
	}
}

// lastLines extracts the final n lines without copying the whole buffer: it
// scans backwards for line boundaries and converts only the bounded tail,
// keeping the cost independent of how large the captured log is.
func lastLines(buf *bytes.Buffer, n int) []string {
	b := bytes.TrimRight(buf.Bytes(), "\r\n")
	if len(b) == 0 {
		return nil
	}

	start := len(b)
	for range n {
		nl := bytes.LastIndexByte(b[:start], '\n')
		if nl < 0 {
			start = 0
			break
		}
		start = nl
	}
	if start > 0 {
		start++ // step past the newline preceding the tail
	}

	lines := strings.Split(string(b[start:]), "\n")
	for i, l := range lines {
		// Keep what the terminal would have displayed: drop the CR of CRLF
		// endings, and for self-overwriting progress output keep only the
		// segment after the last carriage return — raw CRs would re-home the
		// cursor and overwrite the indented tail block.
		l = strings.TrimRight(l, "\r")
		if j := strings.LastIndexByte(l, '\r'); j >= 0 {
			l = l[j+1:]
		}
		lines[i] = l
	}
	return lines
}

func humanizeBytes(n int) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}

	div, exp := int64(unit), 0
	for i := int64(n) / unit; i >= unit; i /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGT"[exp])
}
