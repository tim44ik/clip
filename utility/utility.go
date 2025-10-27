package utility

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2/widget"
)

func Clamp(i, min, max int) int {
	if i < min {
		return min
	}
	if i > max {
		return max
	}
	return i
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

func EntryAutoexpand(entry *widget.Entry, minRows, maxRows int) func(string) {
	return func(s string) {
		entry.SetMinRowsVisible(Clamp(strings.Count(s, "\n")+1, minRows, maxRows))
	}
}

func IsCanceled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
