package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// tappable wraps content and invokes onDouble on a double-click, so a whole
// row area can act as an "edit" target alongside the explicit pencil button.
type tappable struct {
	widget.BaseWidget
	content  fyne.CanvasObject
	onDouble func()
}

func newTappable(content fyne.CanvasObject, onDouble func()) *tappable {
	t := &tappable{content: content, onDouble: onDouble}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tappable) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.content)
}

// Tapped is a no-op but required so the canvas routes pointer events here,
// which is what lets DoubleTapped fire.
func (t *tappable) Tapped(*fyne.PointEvent) {}

func (t *tappable) DoubleTapped(*fyne.PointEvent) {
	if t.onDouble != nil {
		t.onDouble()
	}
}
