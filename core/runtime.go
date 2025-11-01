package core

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"smartpentestutility/utility"
	"strings"
	"time"
)

type Runtime struct {
	Module    *Module
	Variables map[string]string
	Directory string
}

func NewRuntime(m *Module) *Runtime {
	m.Output = ""

	return &Runtime{
		Module:    m,
		Variables: map[string]string{},
	}
}

func (r *Runtime) Execute(code string, ctx context.Context, outputter func(string)) error {
	stdOutR, stdOutW, e := os.Pipe()
	if e != nil {
		return e
	}
	defer stdOutR.Close()
	defer stdOutW.Close()
	stdErrR, stdErrW, e := os.Pipe()
	if e != nil {
		return e
	}
	defer stdErrR.Close()
	defer stdErrW.Close()
	stdInR, stdInW, e := os.Pipe()
	if e != nil {
		return e
	}
	defer stdInR.Close()
	defer stdInW.Close()

	writeStdIn := func(s string) {
		stdInW.Write([]byte(s))
	}

	proc, e := os.StartProcess("/bin/bash", nil, &os.ProcAttr{
		Files: []*os.File{stdInR, stdOutW, stdErrW},
	})
	if e != nil {
		return e
	}

	ctx, ctxCancel := context.WithCancel(ctx)
	defer ctxCancel()
	stdOutCh, stdOutCancel := utility.WrapReaderToChannel(stdOutR)
	stdErrCh, stdErrCancel := utility.WrapReaderToChannel(stdErrR)
	defer stdOutCancel()
	defer stdErrCancel()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case str := <-stdOutCh:
				outputter(str)
			case str := <-stdErrCh:
				outputter(str)

			}
		}
	}()

	for line := range strings.SplitSeq(code, "\n") {
		if utility.IsCanceled(ctx) {
			return fmt.Errorf("Отменено")
		}

		line = strings.Trim(line, " \t\r")
		if line == "" {
			continue
		}

		if line[0] == '%' && strings.Contains(line, "=") {
			kvp := strings.SplitN(line, "=", 2)
			r.Variables[strings.Trim(kvp[0], " \t")] = strings.Trim(kvp[1], " \t")
			continue
		}

		re := regexp.MustCompile(`%\w+`)
		line = re.ReplaceAllStringFunc(line, func(s string) string {
			if val, ok := r.Variables[s]; ok {
				return val
			}
			return s
		})
		writeStdIn("echo ] " + strings.ReplaceAll(line, ">", "]") + "\n")
		writeStdIn(line + "\n")
	}
	writeStdIn("exit\n")
	waitCh := make(chan struct{}, 1)
	defer close(waitCh)
	go func() {
		proc.Wait()
		waitCh <- struct{}{}
	}()
	select {
	case <-waitCh:
		return nil
	case <-ctx.Done():
		proc.Kill()
	}

	time.Sleep(time.Second * 1)

	return nil
}
