package utils

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

const bashName = "bash"

func emptyCancel() {
}

func createBatchCommand(ctx context.Context, cmds []string, timeout int) (*exec.Cmd, context.CancelFunc, error) {
	if len(cmds) == 0 {
		return nil, nil, errors.New("empty commands")
	}
	var cmd *exec.Cmd
	var fn context.CancelFunc = emptyCancel
	if len(cmds) == 1 {
		if timeout == 0 {
			cmd = exec.CommandContext(ctx, bashName, "-c", cmds[0])
		} else {
			ctx1, cancel := context.WithTimeout(ctx, time.Second*time.Duration(timeout))
			fn = cancel
			cmd = exec.CommandContext(ctx1, bashName, "-c", cmds[0])
		}
	} else {
		if timeout == 0 {
			cmd = exec.CommandContext(ctx, bashName)
		} else {
			ctx1, cancel := context.WithTimeout(ctx, time.Second*time.Duration(timeout))
			fn = cancel
			cmd = exec.CommandContext(ctx1, bashName)
		}
		cmdBuffer := bytes.NewBuffer(nil)
		for _, v := range cmds {
			cmdBuffer.WriteString(v)
			cmdBuffer.WriteByte('\n')
		}
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return nil, nil, err
		}
		go func() {
			io.Copy(stdin, cmdBuffer)
			stdin.Close()
		}()
	}
	return cmd, fn, nil
}

func BatchOutputCmd(ctx context.Context, cmdStr string, timeout int) ([]byte, error) {
	return BatchOutput(ctx, []string{cmdStr}, timeout)
}

func BatchOutput(ctx context.Context, cmds []string, timeout int) ([]byte, error) {
	cmd, fn, err := createBatchCommand(ctx, cmds, timeout)
	if err != nil {
		return nil, err
	}
	defer fn()
	return cmd.Output()
}

func BatchRun(ctx context.Context, cmds []string, timeout int) error {
	cmd, fn, err := createBatchCommand(ctx, cmds, timeout)
	if err != nil {
		return err
	}
	defer fn()
	return cmd.Run()
}

func BatchOutErr(ctx context.Context, cmds []string, timeout int) (string, string, error) {
	cmd, fn, err := createBatchCommand(ctx, cmds, timeout)
	if err != nil {
		return "", "", err
	}
	defer fn()

	var out bytes.Buffer
	cmd.Stdout = &out
	var errout bytes.Buffer
	cmd.Stderr = &errout

	err = cmd.Run()
	return strings.Trim(out.String(), "\n"), errout.String(), err
}

func BatchRunStdout(ctx context.Context, cmds []string, timeout int) error {
	cmd, fn, err := createBatchCommand(ctx, cmds, timeout)
	if err != nil {
		return err
	}
	defer fn()
	outReader, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	fmt.Println(strings.Join(cmds, "\n"))

	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(outReader)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}

	return cmd.Wait()
}
