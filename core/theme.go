package core

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type blackTextTheme struct{}

func (b blackTextTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameForeground:
		return color.Black
	case theme.ColorNameInputBackground:
		return color.White
	case theme.ColorNameButton:
		return color.RGBA{R: 196, G: 196, B: 196, A: 255}
	default:
		return theme.LightTheme().Color(name, variant)
	}
}

func (b blackTextTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.LightTheme().Font(style)
}

func (b blackTextTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.LightTheme().Icon(name)
}

func (b blackTextTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.LightTheme().Size(name)
}
