package ui

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"easy-alarms/internal/alarm"
	"easy-alarms/internal/control"
	"easy-alarms/internal/humanize"
	"easy-alarms/internal/store"
)

var (
	audioFilter = storage.NewExtensionFileFilter([]string{".mp3", ".wav", ".ogg", ".oga", ".flac"})
	// Display order Monday-first, mapped to time.Weekday indices.
	dayOrder = [7]struct {
		short string
		wd    time.Weekday
	}{
		{"L", time.Monday}, {"M", time.Tuesday}, {"X", time.Wednesday},
		{"J", time.Thursday}, {"V", time.Friday}, {"S", time.Saturday},
		{"D", time.Sunday},
	}
)

func (u *UI) showEditDialog(a *alarm.Alarm, isNew bool) {
	label := widget.NewEntry()
	label.SetText(a.Label)
	label.PlaceHolder = "Etiqueta (opcional)"

	sound := a.Sound
	soundLabel := widget.NewLabel(soundName(sound))
	soundLabel.Truncation = fyne.TextTruncateEllipsis // long file names must not stretch the dialog

	// Single play/stop toggle: tapping while a preview plays stops it.
	var playBtn *widget.Button
	playing := false
	idle := func() {
		playing = false
		playBtn.SetIcon(theme.MediaPlayIcon())
		playBtn.SetText("Probar")
	}
	playBtn = widget.NewButtonWithIcon("Probar", theme.MediaPlayIcon(), func() {
		if playing {
			u.player.Stop()
			idle()
			return
		}
		if err := u.player.Preview(sound); err != nil {
			dialog.ShowError(fmt.Errorf("no se pudo reproducir este audio (formato no soportado): %w", err), u.win)
			return
		}
		playing = true
		playBtn.SetIcon(theme.MediaStopIcon())
		playBtn.SetText("Parar")
	})

	pick := widget.NewButton("Elegir...", func() {
		fd := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
			if err != nil || rc == nil {
				return
			}
			rc.Close()
			sound = rc.URI().Path()
			soundLabel.SetText(soundName(sound))
			u.player.Stop()
			idle()
			u.rememberSoundDir(filepath.Dir(sound))
		}, u.win)
		fd.SetFilter(audioFilter)
		// Open where the user picked a sound last time (e.g. ~/Music) instead
		// of making them navigate from $HOME on every edit.
		if dir := u.settings.LastSoundDir; dir != "" {
			if lister, err := storage.ListerForURI(storage.NewFileURI(dir)); err == nil {
				fd.SetLocation(lister)
			}
		}
		fd.Show()
	})
	clear := widget.NewButton("Tono", func() {
		sound = ""
		soundLabel.SetText(soundName(sound))
		u.player.Stop()
		idle()
	})
	// Border keeps the buttons at a fixed size on the right; the label takes
	// the remaining width and ellipsizes instead of overflowing.
	soundRow := container.NewBorder(nil, nil, nil,
		container.NewHBox(pick, clear, playBtn), soundLabel)

	items := []*widget.FormItem{
		widget.NewFormItem("Etiqueta", label),
	}

	var (
		timeEntry *widget.Entry
		durEntry  *widget.Entry
		dayChecks [7]*widget.Check
	)
	switch a.Kind {
	case alarm.KindClock:
		timeEntry = widget.NewEntry()
		timeEntry.SetText(fmt.Sprintf("%02d:%02d", a.Hour, a.Minute))
		timeEntry.Validator = func(s string) error {
			if _, _, err := control.ParseClockTime(s); err != nil {
				return errors.New("formato HH:MM")
			}
			return nil
		}
		days := container.NewHBox()
		for i, d := range dayOrder {
			c := widget.NewCheck(d.short, nil)
			c.SetChecked(a.Repeat[d.wd])
			dayChecks[i] = c
			days.Add(c)
		}
		items = append(items,
			widget.NewFormItem("Hora (HH:MM)", timeEntry),
			widget.NewFormItem("Repetir", days),
		)
	case alarm.KindTimer:
		durEntry = widget.NewEntry()
		durEntry.PlaceHolder = "10m, 1h30m..."
		durEntry.Validator = func(s string) error {
			if _, err := control.ParseTimerDuration(s); err != nil {
				return errors.New("ej: 10m, 1h30m")
			}
			return nil
		}
		if a.Duration > 0 {
			// Compact form: "1h30m", not Go's noisy "1h30m0s".
			durEntry.SetText(humanize.Compact(a.Duration))
		}
		items = append(items, widget.NewFormItem("Duración", durEntry))
	}
	items = append(items, widget.NewFormItem("🎵 Sonido", soundRow))

	title := map[alarm.Kind]string{alarm.KindClock: "⏰ Alarma", alarm.KindTimer: "⏱ Timer"}[a.Kind]
	d := dialog.NewForm(title, "Guardar", "Cancelar", items, func(ok bool) {
		u.player.Stop() // silence any running preview when the dialog closes
		if !ok {
			return
		}
		if err := applyEdit(a, label.Text, sound, timeEntry, durEntry, dayChecks); err != nil {
			dialog.ShowError(err, u.win)
			return
		}
		// Saving always activates: alarms re-enable, timers (re)start. A
		// pending snooze belongs to the pre-edit schedule, so drop it.
		a.Enabled = true
		if a.Kind == alarm.KindTimer {
			a.FiresAt = time.Now().Add(a.Duration)
			a.Remaining = 0
		}
		u.sched.ClearSnooze(a.ID)
		if isNew {
			u.store.Add(a)
		}
		u.commit()
	}, u.win)

	// Enter in any text field saves the dialog (no-op while input is invalid,
	// since Submit respects the form's validation state).
	submit := func(string) { d.Submit() }
	label.OnSubmitted = submit
	if timeEntry != nil {
		timeEntry.OnSubmitted = submit
	}
	if durEntry != nil {
		durEntry.OnSubmitted = submit
	}
	d.Show()
}

// applyEdit validates and applies the dialog fields. Parsing is delegated to
// internal/control — the same rules the alarmctl API enforces — with the error
// messages kept in Spanish for the GUI.
func applyEdit(a *alarm.Alarm, label, sound string, timeEntry, durEntry *widget.Entry, dayChecks [7]*widget.Check) error {
	switch a.Kind {
	case alarm.KindClock:
		hour, minute, err := control.ParseClockTime(timeEntry.Text)
		if err != nil {
			return errors.New("hora inválida, formato HH:MM")
		}
		a.Hour, a.Minute = hour, minute
		for i, d := range dayOrder {
			a.Repeat[d.wd] = dayChecks[i].Checked
		}
	case alarm.KindTimer:
		d, err := control.ParseTimerDuration(durEntry.Text)
		if err != nil {
			return errors.New("duración inválida, ej: 10m, 1h30m")
		}
		a.Duration = d
	}
	a.Label = label
	a.Sound = sound
	return nil
}

// rememberSoundDir persists the directory of the last picked sound so the next
// file dialog opens there. Best-effort: a save failure is not worth a dialog.
func (u *UI) rememberSoundDir(dir string) {
	if dir == "" || dir == u.settings.LastSoundDir {
		return
	}
	u.settings.LastSoundDir = dir
	_ = store.SaveSettings(u.settings)
}

func soundName(path string) string {
	if path == "" {
		return "Tono integrado"
	}
	return filepath.Base(path)
}
