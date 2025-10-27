package shell

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"smartpentestutility/utility"
	"strings"
)

type Runtime struct {
	Variables map[string]string
	Output    bytes.Buffer
}

func NewRuntime() *Runtime {
	return &Runtime{
		Variables: map[string]string{},
	}
}

func (r *Runtime) Execute(code string, ctx context.Context) error {
	if utility.IsCanceled(ctx) {
		return fmt.Errorf("Отменено")
	}
	i := 0
	cmd := exec.CommandContext(ctx, "bash")
	stdIn, _ := cmd.StdinPipe()
	stdOut, _ := cmd.StdoutPipe()
	stdErr, _ := cmd.StderrPipe()

	e := cmd.Start()
	if e != nil {
		return e
	}

	reader := bufio.NewReader(stdOut)
	errReader := bufio.NewReader(stdErr)

	run := func(command string) {
		stdIn.Write([]byte(command + "\n"))
		stdIn.Write([]byte("echo __END__\n"))
		for {
			l, _ := reader.ReadString('\n')
			if strings.Contains(l, "__END__") {
				break
			}
			r.Output.Write([]byte(l))
		}
		for errReader.Buffered() > 0 {
			l, _ := errReader.ReadString('\n')
			r.Output.Write([]byte(l))
		}
	}

	for line := range strings.SplitSeq(code, "\n") {
		i++
		// time.Sleep(10 * time.Second)
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
		run(line)

	}
	return nil
}
