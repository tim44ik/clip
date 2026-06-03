package utility

import (
	"clip/errors"
	"clip/models/modules"
	"slices"

	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type WrappedReader struct {
	Channel chan string
	Close   bool
}

func NewDropButton(icon fyne.Resource, canvas fyne.Canvas, menu *fyne.Menu) *widget.Button {
	popup := widget.NewPopUpMenu(menu, canvas)

	return widget.NewButtonWithIcon("", icon, func() {
		popup.Show()
	})
}

func GetQueue(m []*modules.Module) ([][]*modules.Module, error) {
	queueMap := make(map[int][]*modules.Module, 0)

	for i := range m {
		trimmedSpaces := strings.TrimSpace(m[i].Content)
		nextLine := strings.IndexFunc(trimmedSpaces, func(r rune) bool { return r == '\n' })

		if nextLine == -1 && !strings.Contains(strings.ToLower(m[i].Content), "queue") {
			return nil, errors.NewWithPlace(errQueueNotDeclarated, errors.Place(m[i].Name))
		}
		if nextLine != -1 && !strings.Contains(strings.ToLower(m[i].Content[:nextLine]), "queue") {
			return nil, errors.NewWithPlace(errQueueNotDeclarated, errors.Place(m[i].Name))
		}
		if nextLine == -1 && strings.Contains(strings.ToLower(m[i].Content), "queue") {
			return nil, errors.NewWithPlace(errNoCommandsGiven, errors.Place(m[i].Name))
		}

		j := 0
		cases := make([]int, 0, 2)

		for j < nextLine {
			if len(cases) == 2 {
				break
			}
			step := string(trimmedSpaces[j])
			if step != ")" && step != "(" {
				j++
				continue
			}
			if step == "(" && len(cases) == 0 {
				cases = append(cases, j)
			} else if step == ")" && len(cases) == 1 {
				cases = append(cases, j)
			} else {
				return nil, errors.NewWithPlace(errQueueNotDeclarated, errors.Place(m[i].Name))
			}

			j++
		}
		if len(cases) != 2 || j != nextLine {
			return nil, errors.NewWithPlace(errQueueNotDeclarated, errors.Place(m[i].Name))
		}

		qNum, err := strconv.Atoi(trimmedSpaces[cases[0]+1 : cases[1]])
		if err != nil {
			return nil, errors.NewWithPlace(errQueueNotDeclarated, errors.Place(m[i].Name))
		}
		m[i].Content = m[i].Content[nextLine:]
		queueMap[qNum] = append(queueMap[qNum], m[i])
	}

	return sendSlice(queueMap), nil
}

func sendSlice(q map[int][]*modules.Module) (enumSlice [][]*modules.Module) {
	qSlice := make([]int, 0, len(q))

	for key := range q {
		qSlice = append(qSlice, key)
	}

	slices.Sort(qSlice)

	for _, k := range qSlice {
		enumSlice = append(enumSlice, q[k])
	}
	return enumSlice
}
