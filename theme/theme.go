package theme

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type BlackTextTheme struct{}

func (b BlackTextTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameForeground:
		return color.Black
	case theme.ColorNameInputBackground:
		return color.White
	case theme.ColorNameButton:
		return color.RGBA{R: 196, G: 196, B: 196, A: 255}
	case theme.ColorNameDisabled:
		return color.Black
	case theme.ColorNameInputBorder:
		return color.Black
	default:
		return theme.LightTheme().Color(name, variant)
	}
}

func (b BlackTextTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.LightTheme().Font(style)
}

func (b BlackTextTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.LightTheme().Icon(name)
}

func (b BlackTextTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.LightTheme().Size(name)
}
