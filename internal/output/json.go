package output

import (
	"bytes"
	"encoding/json"
	"io"
	"sync"

	"github.com/taskctl/taskctl/task"
)

// RunStartedEvent is the first event emitted on an NDJSON run stream.
type RunStartedEvent struct {
	Event         string   `json:"event"`
	SchemaVersion int      `json:"schema_version"`
	Targets       []string `json:"targets"`
}

// TaskStartedEvent is emitted when a task begins execution.
type TaskStartedEvent struct {
	Event string `json:"event"`
	Task  string `json:"task"`
}

// TaskOutputEvent carries a single line of a task's stdout/stderr.
type TaskOutputEvent struct {
	Event  string `json:"event"`
	Task   string `json:"task"`
	Stream string `json:"stream"`
	Data   string `json:"data"`
}

// TaskFinishedEvent is emitted when a task completes.
type TaskFinishedEvent struct {
	Event      string `json:"event"`
	Task       string `json:"task"`
	Status     string `json:"status"`
	ExitCode   int    `json:"exit_code"`
	DurationMs int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

// TaskResult summarizes a single task's outcome within a run_finished event.
type TaskResult struct {
	Task       string `json:"task"`
	Status     string `json:"status"`
	ExitCode   int    `json:"exit_code"`
	DurationMs int64  `json:"duration_ms"`
}

// RunFinishedEvent is the last event emitted on an NDJSON run stream.
type RunFinishedEvent struct {
	Event      string       `json:"event"`
	Status     string       `json:"status"`
	DurationMs int64        `json:"duration_ms"`
	Tasks      []TaskResult `json:"tasks"`
}

// eventMu guards every NDJSON event write so that concurrent tasks writing
// to the same stdout stream never interleave a single event's bytes.
var eventMu sync.Mutex

// writeEvent marshals v and writes it as a single newline-terminated line to
// w. Marshaling happens outside the lock; only the write is guarded, keeping
// the critical section to the minimum needed for atomicity.
func writeEvent(w io.Writer, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	eventMu.Lock()
	defer eventMu.Unlock()

	if _, err = w.Write(data); err != nil {
		return err
	}
	_, err = w.Write(newline)
	return err
}

var newline = []byte{'\n'}

// EmitRunStarted writes the run_started event that opens an NDJSON run stream.
func EmitRunStarted(w io.Writer, targets []string) error {
	return writeEvent(w, RunStartedEvent{
		Event:         "run_started",
		SchemaVersion: 1,
		Targets:       targets,
	})
}

// EmitRunFinished writes the run_finished event that closes an NDJSON run stream.
func EmitRunFinished(w io.Writer, status string, durationMs int64, results []TaskResult) error {
	return writeEvent(w, RunFinishedEvent{
		Event:      "run_finished",
		Status:     status,
		DurationMs: durationMs,
		Tasks:      results,
	})
}

// TaskStatus maps a completed task's flags to the NDJSON status vocabulary
// ("done"/"skipped"/"failed") shared by the task_finished and run_finished events.
func TaskStatus(t *task.Task) string {
	switch {
	case t.Skipped:
		return "skipped"
	case t.Errored:
		return "failed"
	default:
		return "done"
	}
}

// jsonOutputWriter is the DecoratedOutputWriter that turns a task's output
// into NDJSON events. It line-buffers stdout and stderr independently (via
// StreamWriter) so a partial write on one stream never corrupts the other.
type jsonOutputWriter struct {
	t         *task.Task
	w         io.Writer
	mu        sync.Mutex
	bufStdout []byte
	bufStderr []byte
}

func newJSONOutputWriter(t *task.Task, w io.Writer) *jsonOutputWriter {
	return &jsonOutputWriter{t: t, w: w}
}

// buffer returns a pointer to the line buffer for the given stream, so both
// stdout and stderr are handled by one code path without a map lookup.
func (d *jsonOutputWriter) buffer(stream string) *[]byte {
	if stream == "stderr" {
		return &d.bufStderr
	}
	return &d.bufStdout
}

// WriteHeader emits the task_started event.
func (d *jsonOutputWriter) WriteHeader() error {
	return writeEvent(d.w, TaskStartedEvent{Event: "task_started", Task: d.t.Name})
}

// Write implements io.Writer, treating all writes as stdout. Callers that
// need stream attribution should use StreamWriter instead.
func (d *jsonOutputWriter) Write(p []byte) (int, error) {
	return d.writeStream("stdout", p)
}

// StreamWriter returns a facet of this decorator that attributes writes to
// the given stream ("stdout" or "stderr").
func (d *jsonOutputWriter) StreamWriter(stream string) io.Writer {
	return &jsonStreamWriter{d: d, stream: stream}
}

func (d *jsonOutputWriter) writeStream(stream string, p []byte) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	buf := d.buffer(stream)

	// Fast path: when nothing is buffered, scan p in place and only copy the
	// trailing partial line, avoiding a full copy of every write.
	data := p
	if len(*buf) > 0 {
		data = append(*buf, p...)
	}

	for {
		idx := bytes.IndexByte(data, '\n')
		if idx < 0 {
			break
		}

		line := bytes.TrimSuffix(data[:idx], []byte{'\r'})
		if err := d.emitOutput(stream, line); err != nil {
			return 0, err
		}

		data = data[idx+1:]
	}

	*buf = append((*buf)[:0], data...)
	return len(p), nil
}

func (d *jsonOutputWriter) emitOutput(stream string, data []byte) error {
	return writeEvent(d.w, TaskOutputEvent{
		Event:  "task_output",
		Task:   d.t.Name,
		Stream: stream,
		Data:   string(data),
	})
}

// WriteFooter flushes any buffered partial line on each stream, then emits
// the task_finished event.
func (d *jsonOutputWriter) WriteFooter() error {
	d.mu.Lock()
	for _, stream := range []string{"stdout", "stderr"} {
		buf := d.buffer(stream)
		if len(*buf) == 0 {
			continue
		}

		if err := d.emitOutput(stream, *buf); err != nil {
			d.mu.Unlock()
			return err
		}
		*buf = nil
	}
	d.mu.Unlock()

	status := TaskStatus(d.t)

	ev := TaskFinishedEvent{
		Event:      "task_finished",
		Task:       d.t.Name,
		Status:     status,
		ExitCode:   int(d.t.ExitCode),
		DurationMs: d.t.Duration().Milliseconds(),
	}
	if status == "failed" {
		ev.Error = d.t.ErrorMessage()
	}

	return writeEvent(d.w, ev)
}

// jsonStreamWriter is the io.Writer facet returned by jsonOutputWriter.StreamWriter.
type jsonStreamWriter struct {
	d      *jsonOutputWriter
	stream string
}

func (s *jsonStreamWriter) Write(p []byte) (int, error) {
	return s.d.writeStream(s.stream, p)
}
