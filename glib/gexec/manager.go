package gexec

import (
	"context"
	"shopble/common/comerr"
	"io"
	"os"
	"os/exec"
)

type CustomCmd struct {
	Command string
	Args    []string
}

type StdoutLineTextGetter func(string) (skipThisLine bool, err error)
type StdoutStreamReader func(io.ReadCloser) error

func (c *CustomCmd) RunStream(ctx context.Context, streamReader StdoutStreamReader) (err error) {
	cmd := exec.CommandContext(ctx, c.Command, c.Args...)
	stdoutReader, err := cmd.StdoutPipe()
	if err != nil {
		err = comerr.WrapMessage(err, "failed to get stdout pipe")
		return
	}
	defer stdoutReader.Close()
	stderrReader, err := cmd.StderrPipe()
	if err != nil {
		err = comerr.WrapMessage(err, "failed to get stderr pipe")
		return
	}
	err = cmd.Start()
	if err != nil {
		err = comerr.WrapMessage(err, "failed to start command")
		return
	}
	// Read stderr and stdout concurrently
	go func() {
		defer stderrReader.Close()
		io.Copy(os.Stderr, stderrReader)
	}()
	if streamReader != nil {
		err = streamReader(stdoutReader)
		if err != nil {
			err = comerr.WrapStack(err, "operate stdout stream failed")
			return
		}
	}
	err = cmd.Wait()
	if err != nil {
		err = comerr.WrapMessage(err, "command execution failed")
		return
	}
	return
}
