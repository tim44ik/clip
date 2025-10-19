package core

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2/widget"
)

func clamp(i, min, max int) int {
	if i < min {
		return min
	}
	if i > max {
		return max
	}
	return i
}

func numberValidator(min, max int) func(string) error {
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

func entryAutoexpand(entry *widget.Entry, minRows, maxRows int) func(string) {
	return func(s string) {
		entry.SetMinRowsVisible(clamp(strings.Count(s, "\n")+1, minRows, maxRows))
	}
}
