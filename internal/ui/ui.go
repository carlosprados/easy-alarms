package ui

import (
	"fmt"
	"image/color"
	"sort"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"easy-alarms/internal/alarm"
	"easy-alarms/internal/audio"
	"easy-alarms/internal/store"
)

type UI struct {
	app    fyne.App
	win    fyne.Window
	store  *store.Store
	sched  *alarm.Scheduler
	player *audio.Player

	TrayEnabled bool

	list *fyne.Container
	rows []*row

	// ring state (main thread only): open ring dialogs, whether the alarm
	// sound is playing, and the auto-silence cap timer.
	ringDialogs int
	soundOn     bool
	silence     *time.Timer

	// tray state, set in setupTray (nil when the tray is disabled)
	desk     desktop.App
	trayMenu *fyne.Menu
	trayNext *fyne.MenuItem
	trayLast string // last rendered "next alarm" label, to skip no-op refreshes
}

type row struct {
	alarm *alarm.Alarm
	when  *canvas.Text
	last  string // last rendered text, to skip no-op refreshes
}

func New(a fyne.App, st *store.Store, sched *alarm.Scheduler, player *audio.Player) *UI {
	return &UI{
		app:    a,
		store:  st,
		sched:  sched,
		player: player,
		list:   container.NewVBox(),
	}
}

func (u *UI) Run(hidden bool) {
	u.app.SetIcon(appIcon)
	u.win = u.app.NewWindow("Easy Alarms")
	u.win.SetCloseIntercept(u.win.Hide) // close button minimizes to tray

	newAlarm := widget.NewButton("⏰  Nueva alarma", func() {
		u.showEditDialog(alarm.New(alarm.KindClock), true)
	})
	newAlarm.Importance = widget.HighImportance
	newTimer := widget.NewButton("⏱  Nuevo timer", func() {
		u.showEditDialog(alarm.New(alarm.KindTimer), true)
	})
	toolbar := container.NewGridWithColumns(2, newAlarm, newTimer)

	u.win.SetContent(container.NewBorder(
		container.NewPadded(toolbar), nil, nil, nil,
		container.NewVScroll(u.list),
	))
	// Resize AFTER SetContent (otherwise Fyne keeps the content's min size)
	// and center, so the window lands predictably on a multi-monitor desktop
	// instead of off on a secondary screen.
	u.win.Resize(fyne.NewSize(720, 560))
	u.win.CenterOnScreen()

	if u.TrayEnabled {
		u.setupTray()
	}
	u.rebuildList()
	go u.refreshLoop()

	if !hidden {
		u.win.Show()
	}
	u.app.Run()
}

// commit persists state, re-arms the scheduler and redraws the list. Call
// after any alarm mutation.
func (u *UI) commit() {
	if err := u.store.Save(); err != nil {
		dialog.ShowError(err, u.win)
	}
	u.sched.Reschedule()
	u.rebuildList()
	u.updateTrayNext()
}

func (u *UI) rebuildList() {
	u.list.Objects = nil
	u.rows = nil
	alarms := u.store.List()
	u.sortAlarms(alarms)
	if len(alarms) == 0 {
		empty := widget.NewLabel("🔕  Sin alarmas todavía.\nCrea una con los botones de arriba.")
		empty.Alignment = fyne.TextAlignCenter
		u.list.Add(container.NewPadded(container.NewCenter(empty)))
	}
	for _, a := range alarms {
		u.list.Add(u.newRow(a))
	}
	u.list.Refresh()
}

// sortAlarms orders the list: alarms that will actually ring (active) first,
// then the rest; within each group, the soonest on top. Inactive alarms are
// ordered by the time they *would* fire if running, so the ordering stays
// intuitive ("by time of day").
func (u *UI) sortAlarms(alarms []*alarm.Alarm) {
	now := time.Now()
	group := func(a *alarm.Alarm) (int, time.Time) {
		if next := u.sched.NextFor(a); !next.IsZero() {
			return 0, next
		}
		switch a.Kind {
		case alarm.KindTimer:
			d := a.Duration
			if a.Remaining > 0 {
				d = a.Remaining
			}
			return 1, now.Add(d)
		default:
			c := time.Date(now.Year(), now.Month(), now.Day(), a.Hour, a.Minute, 0, 0, now.Location())
			if !c.After(now) {
				c = c.AddDate(0, 0, 1)
			}
			return 1, c
		}
	}
	sort.SliceStable(alarms, func(i, j int) bool {
		gi, ti := group(alarms[i])
		gj, tj := group(alarms[j])
		if gi != gj {
			return gi < gj
		}
		return ti.Before(tj)
	})
}

func (u *UI) newRow(a *alarm.Alarm) fyne.CanvasObject {
	// Inactive alarms (won't actually ring) are dimmed.
	active := !u.sched.NextFor(a).IsZero()
	fg := theme.Color(theme.ColorNameForeground)
	if !active {
		fg = theme.Color(theme.ColorNameDisabled)
	}

	title := rowText(rowTitle(a), 17, true, fg)

	whenText := u.rowWhen(a, time.Now())
	when := rowText(whenText, theme.TextSize(), false, fg)
	u.rows = append(u.rows, &row{alarm: a, when: when, last: whenText})

	left := container.NewVBox(title, when)
	if a.Kind == alarm.KindClock {
		if rep := repeatLabel(a.Repeat); rep != "" {
			left.Add(rowText("📅  "+rep, theme.TextSize(), false, fg))
		}
	}
	left.Add(rowText("🎵  "+soundName(a.Sound), theme.TextSize(), false, fg))

	dup := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		c := alarm.New(a.Kind)
		c.Label, c.Hour, c.Minute, c.Repeat = a.Label, a.Hour, a.Minute, a.Repeat
		c.Duration, c.Sound = a.Duration, a.Sound
		u.showEditDialog(c, true) // cancelling the editor discards the copy
	})
	edit := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {
		u.showEditDialog(a, false)
	})
	del := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		dialog.ShowConfirm("Eliminar", fmt.Sprintf("¿Eliminar %q?", rowTitle(a)), func(ok bool) {
			if !ok {
				return
			}
			u.store.Remove(a.ID)
			u.commit()
		}, u.win)
	})

	actions := container.NewHBox(u.stateControl(a), dup, edit, del)

	// Double-clicking the info area also opens the editor.
	info := newTappable(left, func() { u.showEditDialog(a, false) })
	body := container.NewBorder(nil, nil, nil, actions, info)
	return widget.NewCard("", "", container.NewPadded(body))
}

func rowText(s string, size float32, bold bool, col color.Color) *canvas.Text {
	t := canvas.NewText(s, col)
	t.TextSize = size
	t.TextStyle = fyne.TextStyle{Bold: bold}
	return t
}

// stateControl is the per-row enable toggle (alarms) or the
// start/pause/resume/stop buttons (timers).
func (u *UI) stateControl(a *alarm.Alarm) fyne.CanvasObject {
	switch a.Kind {
	case alarm.KindTimer:
		stop := widget.NewButtonWithIcon("", theme.MediaStopIcon(), func() {
			a.FiresAt = time.Time{}
			a.Remaining = 0
			u.sched.ClearSnooze(a.ID)
			u.commit()
		})
		switch {
		case !a.FiresAt.IsZero(): // running
			pause := widget.NewButtonWithIcon("", theme.MediaPauseIcon(), func() {
				if left := time.Until(a.FiresAt); left > 0 {
					a.Remaining = left
				}
				a.FiresAt = time.Time{}
				u.commit()
			})
			return container.NewHBox(pause, stop)
		case a.Remaining > 0: // paused
			resume := widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
				a.Enabled = true
				a.FiresAt = time.Now().Add(a.Remaining)
				a.Remaining = 0
				u.commit()
			})
			return container.NewHBox(resume, stop)
		default: // idle
			return widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
				a.Enabled = true
				a.FiresAt = time.Now().Add(a.Duration)
				u.commit()
			})
		}
	default:
		// Set OnChanged AFTER SetChecked: in Fyne SetChecked fires OnChanged,
		// and commit() rebuilds the list, which would recurse infinitely and
		// hang the UI thread. The guard is belt-and-braces.
		check := widget.NewCheck("", nil)
		check.SetChecked(a.Enabled)
		check.OnChanged = func(on bool) {
			if on == a.Enabled {
				return
			}
			a.Enabled = on
			if !on {
				u.sched.ClearSnooze(a.ID) // disabling also silences a pending snooze
			}
			u.commit()
		}
		return check
	}
}

// rowWhen renders a row's status line: the live countdown, or the paused
// state, which has no next trigger but should still show the time left.
func (u *UI) rowWhen(a *alarm.Alarm, now time.Time) string {
	if a.Kind == alarm.KindTimer && a.FiresAt.IsZero() && a.Remaining > 0 {
		if u.sched.NextFor(a).IsZero() { // not snoozed
			return "⏸ En pausa (quedan " + compactDuration(a.Remaining) + ")"
		}
	}
	return describeNext(u.sched.NextFor(a), now)
}

func rowTitle(a *alarm.Alarm) string {
	label := a.Label
	if label == "" {
		label = map[alarm.Kind]string{alarm.KindClock: "Alarma", alarm.KindTimer: "Timer"}[a.Kind]
	}
	switch a.Kind {
	case alarm.KindTimer:
		return fmt.Sprintf("⏱  %s · %s", label, compactDuration(a.Duration))
	default:
		// No time here: the "when" line below already shows it.
		return "⏰  " + label
	}
}

// refreshLoop keeps the "rings in..." labels ticking. It only touches labels
// whose text actually changed, so an idle window does no rendering work.
func (u *UI) refreshLoop() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for range t.C {
		fyne.Do(func() {
			now := time.Now()
			for _, r := range u.rows {
				next := u.rowWhen(r.alarm, now)
				if next != r.last {
					r.last = next
					r.when.Text = next
					r.when.Refresh()
				}
			}
			u.updateTrayNext()
		})
	}
}
