package ui

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed assets/icon.svg
var iconData []byte

var appIcon = fyne.NewStaticResource("icon.svg", iconData)
