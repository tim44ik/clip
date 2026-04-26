package core

import (
	appErrors "clip/errors"
	"clip/locales"
	"errors"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

const (
	errDataFormat        appErrors.Code = "data_format_error"
	errScenarioInProcess appErrors.Code = "scenario_already_running"
	errNotStarted        appErrors.Code = "scenario_not_started"
)

func ShowError(lang string, err error, window fyne.Window) {
	var e *appErrors.Error
	if errors.As(err, &e) {
		place := locales.T(lang, string(e.Place))
		var msg string
		if place != "" {
			msg = locales.T(lang, string(e.Code)) + " " + place
		} else if e.Place != "" {
			msg = locales.T(lang, string(e.Code)) + " " + string(e.Place)
		} else {
			msg = locales.T(lang, string(e.Code))
		}
		dialog.ShowError(errors.New(msg), window)
		return
	}
	dialog.ShowError(err, window)
}
