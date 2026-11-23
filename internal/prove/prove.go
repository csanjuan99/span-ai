// Package prove runs the commands a project declared and classifies the result.
// A line with shell punctuation is not executed.
package prove

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"span/internal/statement"
)

// ErrProse means the configured line is not an argv command.
var ErrProse = errors.New("not a command")

// Checker starts a command. A non-nil error means it did not run.
type Checker interface {
	Run(ctx context.Context, command, dir string) (exitCode int, err error)
}

// Report is the band outcome the engine turns into a verdict.
type Report struct {
	Outcome  string // advance, hold, iterate
	Reason   string
	Commands []statement.Command
}

// Bands runs fast, then acceptance. A fast failure skips acceptance.
func Bands(ctx context.Context, check Checker, dir string, fast, acceptance []string) (Report, error) {
	fastReport, err := band(ctx, check, dir, "fast", fast)
	if err != nil {
		return Report{}, err
	}
	if fastReport.Outcome != statement.Advance {
		return fastReport, nil
	}
	acc, err := band(ctx, check, dir, "acceptance", acceptance)
	if err != nil {
		return Report{}, err
	}
	acc.Commands = append(fastReport.Commands, acc.Commands...)
	if acc.Outcome == statement.Advance && fastReport.Reason != "" {
		acc.Reason = "fast and acceptance passed"
	}
	return acc, nil
}

func band(ctx context.Context, check Checker, dir, name string, commands []string) (Report, error) {
	rep := Report{Outcome: statement.Advance, Reason: name + " passed"}
	if len(commands) == 0 {
		rep.Reason = name + " had no commands"
		return rep, nil
	}
	for _, command := range commands {
		if Prose(command) {
			rep.Outcome = statement.Hold
			rep.Reason = fmt.Sprintf("%s is not a command", command)
			rep.Commands = append(rep.Commands, statement.Command{Command: command, Band: name, Error: ErrProse.Error()})
			return rep, nil
		}
		code, err := check.Run(ctx, command, dir)
		item := statement.Command{Command: command, ExitCode: code, Band: name}
		if err != nil {
			item.Error = err.Error()
			rep.Commands = append(rep.Commands, item)
			rep.Outcome = statement.Hold
			rep.Reason = fmt.Sprintf("%s did not run: %s", command, err.Error())
			return rep, nil
		}
		rep.Commands = append(rep.Commands, item)
		if code != 0 {
			rep.Outcome = "iterate"
			rep.Reason = fmt.Sprintf("%s exited %d", command, code)
			return rep, nil
		}
	}
	return rep, nil
}

// Prose reports shell punctuation. Those lines are never executed.
func Prose(command string) bool {
	return strings.ContainsAny(command, "()<>;`|&")
}
