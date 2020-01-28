package task

import (
	"github.com/trntv/wilson/pkg/builder"
	"github.com/trntv/wilson/pkg/util"
	"io"
	"sync"
	"time"
)

type Task struct {
	Command      []string
	Context      string
	Env          []string
	Variations   []map[string]string
	Dir          string
	Timeout      *time.Duration
	AllowFailure bool

	Name        string
	Description string
	Start       time.Time
	End         time.Time

	Stdout io.ReadCloser
	Stderr io.ReadCloser

	log struct {
		sync.Mutex
		data []byte
	}
}

func BuildTask(def *builder.TaskDefinition) *Task {
	t := &Task{
		Name:         def.Name,
		Description:  def.Description,
		Command:      def.Command,
		Env:          make([]string, 0),
		Variations:   def.Variations,
		Dir:          def.Dir,
		Timeout:      def.Timeout,
		AllowFailure: def.AllowFailure,
	}

	t.Context = def.Context
	t.Env = util.ConvertEnv(def.Env)
	if t.Context == "" {
		t.Context = "local"
	}

	return t
}

func (t *Task) Duration() time.Duration {
	return t.End.Sub(t.Start)
}

func (t *Task) WriteLog(l []byte) {
	t.log.Lock()
	t.log.data = l
	t.log.Unlock()
}

func (t *Task) ReadLog() []byte {
	return t.log.data
}

func (t *Task) SetStdout(stdout io.ReadCloser) {
	t.Stdout = stdout
}

func (t *Task) SetStderr(stderr io.ReadCloser) {
	t.Stderr = stderr
}
