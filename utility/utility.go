package utility

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// func Clamp(i, min, max int) int {
// 	if i < min {
// 		return min
// 	}
// 	if i > max {
// 		return max
// 	}
// 	return i
// }

type WrappedReader struct {
	Channel chan string
	Close   bool
}

func WrapReaderToChannel(reader io.Reader) (ch chan string, free func()) {
	var doFree bool
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
				time.Sleep(time.Second / 25)
				continue
			}
			ch <- string(buff[:n])
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
