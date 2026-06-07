package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"

	"easy-alarms/internal/autostart"
)

// setupTray installs the StatusNotifierItem icon. Note GNOME's appindicator
// extension routes every click to the menu, so "open window" must be a menu
// item rather than a primary-click action.
func (u *UI) setupTray() {
	desk, ok := u.app.(desktop.App)
	if !ok {
		return
	}

	auto := fyne.NewMenuItem("Arrancar al iniciar sesión", nil)
	auto.Checked = autostart.Enabled()

	menu := fyne.NewMenu("Easy Alarms",
		fyne.NewMenuItem("Abrir Easy Alarms", func() {
			u.win.Show()
			u.win.RequestFocus()
		}),
		fyne.NewMenuItemSeparator(),
		auto,
	)

	auto.Action = func() {
		var err error
		if auto.Checked {
			err = autostart.Disable()
		} else {
			err = autostart.Enable()
		}
		if err != nil {
			dialog.ShowError(err, u.win)
			return
		}
		auto.Checked = !auto.Checked
		menu.Refresh()
	}

	desk.SetSystemTrayMenu(menu)
	desk.SetSystemTrayIcon(appIcon)
}
