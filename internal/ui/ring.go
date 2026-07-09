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

// ringSilenceAfter caps how long an unattended alarm keeps making noise; the
// ring dialog stays up so it is still visible when the user comes back.
const ringSilenceAfter = 3 * time.Minute

var snoozeOptions = []time.Duration{5 * time.Minute, 10 * time.Minute, 15 * time.Minute}

// Ring is the scheduler callback: play sound, notify, pop the window with a
// stop/snooze dialog. It runs on the scheduler goroutine, so everything is
// deferred to the Fyne main thread — alarms are only ever touched there.
func (u *UI) Ring(a *alarm.Alarm) {
	fyne.Do(func() {
		// With several alarms ringing at once the dialogs stack; the first
		// sound keeps playing and stops when the last dialog is dismissed.
		u.startRingSound(a.Sound)
		u.ringDialogs++

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

		var d dialog.Dialog
		snoozes := container.NewGridWithColumns(len(snoozeOptions))
		for _, dur := range snoozeOptions {
			snoozes.Add(widget.NewButton("😴 "+compactDuration(dur), func() {
				u.sched.Snooze(a.ID, dur)
				d.Hide()
			}))
		}
		content := container.NewVBox(big, msg, snoozes)

		d = dialog.NewCustom("¡Alarma!", "🔕 Parar", content, u.win)
		// Every way out of the dialog (Parar, a snooze button, Esc) lands
		// here, so the sound bookkeeping cannot be bypassed.
		d.SetOnClosed(func() {
			u.ringDialogs--
			if u.ringDialogs == 0 {
				u.stopRingSound()
			}
			u.rebuildList()
		})
		d.Show()
	})
}

// startRingSound plays the alarm sound unless one is already ringing and
// (re)arms the auto-silence cap. Main thread only.
func (u *UI) startRingSound(sound string) {
	if !u.soundOn {
		u.player.Play(sound)
		u.soundOn = true
	}
	if u.silence == nil {
		u.silence = time.AfterFunc(ringSilenceAfter, func() {
			fyne.Do(u.silenceUnattended)
		})
		return
	}
	u.silence.Reset(ringSilenceAfter)
}

func (u *UI) stopRingSound() {
	if u.silence != nil {
		u.silence.Stop()
	}
	if u.soundOn {
		u.player.Stop()
		u.soundOn = false
	}
}

// silenceUnattended stops the sound of a ring nobody dismissed, leaving the
// dialog up, and lets the user know they missed it.
func (u *UI) silenceUnattended() {
	if !u.soundOn || u.ringDialogs == 0 {
		return
	}
	u.player.Stop()
	u.soundOn = false
	u.app.SendNotification(fyne.NewNotification("Easy Alarms", "🔕 Alarma sin atender: sonido silenciado"))
}
