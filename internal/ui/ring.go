package ui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"easy-alarms/internal/alarm"
)

const snoozeFor = 5 * time.Minute

// Ring is the scheduler callback: play sound, notify, pop the window with a
// stop/snooze dialog. It runs on the scheduler goroutine, so everything is
// deferred to the Fyne main thread — alarms are only ever touched there.
func (u *UI) Ring(a *alarm.Alarm) {
	fyne.Do(func() {
		// With several alarms ringing at once the dialogs stack; the first
		// sound keeps playing and stops when the last dialog is dismissed.
		if u.ringing == 0 {
			u.player.Play(a.Sound)
		}
		u.ringing++

		title := rowTitle(a)
		u.app.SendNotification(fyne.NewNotification("Easy Alarms", title))

		// Settle post-fire state before rescheduling: one-shots disarm,
		// repeating clocks pick up their next occurrence naturally.
		switch {
		case a.Kind == alarm.KindTimer:
			a.FiresAt = time.Time{}
		case !a.Repeats():
			a.Enabled = false
		}
		u.commit()

		u.win.Show()
		u.win.RequestFocus()
		big := canvas.NewText("⏰", theme.Color(theme.ColorNameForeground))
		big.TextSize = 48
		big.Alignment = fyne.TextAlignCenter
		msg := widget.NewLabel(fmt.Sprintf("%s\n%s", title, time.Now().Format("15:04")))
		msg.Alignment = fyne.TextAlignCenter
		content := container.NewVBox(big, msg)
		dialog.ShowCustomConfirm("¡Alarma!", "🔕 Parar", fmt.Sprintf("😴 Posponer %s", compactDuration(snoozeFor)), content,
			func(stop bool) {
				u.ringing--
				if u.ringing == 0 {
					u.player.Stop()
				}
				if !stop {
					u.sched.Snooze(a.ID, snoozeFor)
				}
				u.rebuildList()
			}, u.win)
	})
}
