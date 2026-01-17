package utility

import (
	"clip/modules"
	"slices"

	"bytes"
	"context"
	"fmt"
	"io"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	ansi "github.com/leaanthony/go-ansi-parser"
)

type WrappedReader struct {
	Channel chan string
	Close   bool
}

func WrapReaderToChannel(reader io.Reader) (ch chan string, free func()) {
	var doFree bool
	var outputString string
	buff := make([]byte, 1024)
	ch = make(chan string)
	free = func() { doFree = true }

	go func() {
		defer close(ch)
		for {
			n, _ := reader.Read(buff)
			if n == 0 {
				if doFree {
					return
				}

				continue
			}
			if runtime.GOOS == "windows" {
				reader := transform.NewReader(bytes.NewReader(buff[:n]), charmap.CodePage866.NewDecoder())
				bytes, _ := io.ReadAll(reader)
				outputString, _ = ansi.Cleanse(string(bytes))

			} else {
				outputString, _ = ansi.Cleanse(string(buff[:n]))
			}
			ch <- outputString

		}
	}()

	return
}

func NumberValidator(min, max int) func(string) error {
	return func(s string) error {
		i, e := strconv.Atoi(s)
		if e != nil {
			return e
		}
		if i < min {
			return fmt.Errorf("number must be greater or equal %d", min)
		}
		if i > max {
			return fmt.Errorf("number must be less or equal %d", max)
		}
		return nil
	}
}

func NewDropButton(icon fyne.Resource, canvas fyne.Canvas, menu *fyne.Menu) *widget.Button {
	popup := widget.NewPopUpMenu(menu, canvas)
	return widget.NewButtonWithIcon("", icon, func() {
		popup.Show()
	})
}

func IsCanceled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func EnumLines(output string) []string {
	divided := strings.Split(output, "\n")
	for i := 0; i < len(divided)-1; i++ {
		divided[i] = strconv.Itoa(i+1) + "  " + divided[i]
	}
	return divided
}

func GetQueue(m []*modules.Module) ([][]*modules.Module, error) {
	queueMap := make(map[int][]*modules.Module, 0)
	cases := make([]int, 0, 2)
	for i := range m {
		trimmedSpaces := strings.TrimSpace(m[i].Content)
		nextLine := strings.IndexFunc(trimmedSpaces, func(r rune) bool { return r == '\n' })
		if !strings.Contains(strings.ToLower(m[i].Content[:nextLine]), "queue") {
			return nil, fmt.Errorf("Queue is not declarated in module %s", m[i].Name)
		}
		j := 0
		for j < nextLine {
			step := string(trimmedSpaces[j])
			if step != ")" && step != "(" {
				j++
				continue
			}
			if step == "(" && len(cases) == 0 {
				cases[0] = j
			} else if step == ")" && len(cases) == 1 {
				cases[1] = j
			} else {
				return nil, fmt.Errorf("Queue is not declarated in module %s", m[i].Name)
			}
			j++
		}
		qNum, err := strconv.Atoi(trimmedSpaces[cases[0]:cases[1]])
		if err != nil {
			return nil, fmt.Errorf("Queue is not declarated in module %s", m[i].Name)
		}
		queueMap[qNum] = m
	}

	enumed := getSlice(queueMap)

	return enumed, nil
}

func getSlice(q map[int][]*modules.Module) (enumSlice [][]*modules.Module) {
	qSlice := []int{}
	for key := range q {
		qSlice = append(qSlice, key)
	}
	slices.Sort(qSlice)
	for _, k := range qSlice {
		enumSlice = append(enumSlice, q[k])
	}
	return enumSlice
}
