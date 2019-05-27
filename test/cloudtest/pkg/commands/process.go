package commands

import (
	"bufio"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"os/exec"
)

type proc struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func ExecRead( ctx context.Context, args []string) (string, error) {
	proc, error := ExecProc(ctx, args)
	if error != nil {
		return "", error
	}
	output := ""
	reader := bufio.NewReader(proc.stdout)
	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		output += s + "\n"
	}
	return output, nil
}

func ExecProc(ctx context.Context, args []string) (proc, error) {
	if len(args) == 0 {
		return proc{}, fmt.Errorf("missing command to run")
	}

	ctx, cancel := context.WithCancel(ctx)
	p := proc{
		cmd:    exec.CommandContext(ctx, args[0], args[1:]...),
		cancel: cancel,
	}
	logrus.Debugf("exec: %s", p.cmd.Args)
	var err error
	p.stdout, err = p.cmd.StdoutPipe()
	if err != nil {
		return p, err
	}
	p.stderr, err = p.cmd.StderrPipe()
	if err != nil {
		return p, err
	}
	err = p.cmd.Start()
	if err == nil {
		logrus.Debugf("go test pid: %d", p.cmd.Process.Pid)
	}
	return p, err
}