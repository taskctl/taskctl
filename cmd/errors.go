package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/taskctl/taskctl/internal/tui"
)

// usageError marks an arg/flag/usage error: the user invoked a command wrong.
// present prints the message together with the command's usage and exits 2.
type usageError struct{ err error }

func (e usageError) Error() string { return e.err.Error() }
func (e usageError) Unwrap() error { return e.err }

// reportedError marks an error whose details were already shown to the user
// (the end-of-run summary, the JSON run_finished event, or validate's own ✗
// line), so present exits non-zero without printing it a second time.
type reportedError struct{ err error }

func (e reportedError) Error() string { return e.err.Error() }
func (e reportedError) Unwrap() error { return e.err }

// exitError carries the process exit code from Run back to main.
type exitError struct{ code int }

func (e exitError) Error() string { return fmt.Sprintf("exit status %d", e.code) }

// ExitCode returns the process exit code an error from Run maps to: the code
// carried by an exitError, 1 for any other non-nil error, or 0 for nil.
func ExitCode(err error) int {
	if ee, ok := errors.AsType[exitError](err); ok {
		return ee.code
	}
	if err != nil {
		return 1
	}
	return 0
}

// present writes the appropriate rendering of err (if any) to cmd's error
// stream and returns the process exit code. cmd is the command that failed, as
// returned by cobra's ExecuteC, so usage errors can print the right usage.
func present(cmd *cobra.Command, err error) int {
	if err == nil {
		return 0
	}

	w := cmd.ErrOrStderr()

	if _, ok := errors.AsType[reportedError](err); ok {
		return 1
	}

	tui.Println(w, tui.StyleError.Render("Error:")+" "+err.Error())

	if _, ok := errors.AsType[usageError](err); ok {
		tui.Println(w, "")
		_ = cmd.Usage()
		return 2
	}
	return 1
}

// markUsageErrors wraps every command's positional-arg validator so an
// arg error becomes a usageError, letting present print usage for it. A nil
// validator is replaced with cobra.ArbitraryArgs, which never errors.
func markUsageErrors(cmd *cobra.Command) {
	inner := cmd.Args
	if inner == nil {
		inner = cobra.ArbitraryArgs
	}
	cmd.Args = func(c *cobra.Command, args []string) error {
		if err := inner(c, args); err != nil {
			return usageError{err}
		}
		return nil
	}

	for _, sub := range cmd.Commands() {
		markUsageErrors(sub)
	}
}

// exactArgs requires exactly n positional args, returning msg instead of
// cobra's terse "accepts N arg(s), received M".
func exactArgs(n int, msg string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) != n {
			return errors.New(msg)
		}
		return nil
	}
}

// minArgs requires at least n positional args, returning msg on shortfall.
func minArgs(n int, msg string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) < n {
			return errors.New(msg)
		}
		return nil
	}
}
