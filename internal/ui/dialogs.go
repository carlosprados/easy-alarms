package ui

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"easy-alarms/internal/alarm"
)

var (
	timePattern = regexp.MustCompile(`^([01]?\d|2[0-3]):([0-5]\d)$`)
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
		}, u.win)
		fd.SetFilter(audioFilter)
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
		if a.Duration > 0 {
			durEntry.SetText(a.Duration.String())
		}
		items = append(items, widget.NewFormItem("Duración", durEntry))
	}
	items = append(items, widget.NewFormItem("🎵 Sonido", soundRow))

	title := map[alarm.Kind]string{alarm.KindClock: "⏰ Alarma", alarm.KindTimer: "⏱ Timer"}[a.Kind]
	dialog.ShowForm(title, "Guardar", "Cancelar", items, func(ok bool) {
		u.player.Stop() // silence any running preview when the dialog closes
		if !ok {
			return
		}
		if err := applyEdit(a, label.Text, sound, timeEntry, durEntry, dayChecks); err != nil {
			dialog.ShowError(err, u.win)
			return
		}
		if isNew {
			if a.Kind == alarm.KindTimer {
				a.FiresAt = time.Now().Add(a.Duration) // creating a timer starts it
			}
			u.store.Add(a)
		}
		u.commit()
	}, u.win)
}

func applyEdit(a *alarm.Alarm, label, sound string, timeEntry, durEntry *widget.Entry, dayChecks [7]*widget.Check) error {
	switch a.Kind {
	case alarm.KindClock:
		m := timePattern.FindStringSubmatch(timeEntry.Text)
		if m == nil {
			return errors.New("hora inválida, formato HH:MM")
		}
		fmt.Sscanf(timeEntry.Text, "%d:%d", &a.Hour, &a.Minute)
		for i, d := range dayOrder {
			a.Repeat[d.wd] = dayChecks[i].Checked
		}
	case alarm.KindTimer:
		d, err := time.ParseDuration(durEntry.Text)
		if err != nil || d <= 0 {
			return errors.New("duración inválida, ej: 10m, 1h30m")
		}
		a.Duration = d
		if !a.FiresAt.IsZero() {
			a.FiresAt = time.Now().Add(d) // restart running timer with new duration
		}
	}
	a.Label = label
	a.Sound = sound
	return nil
}

func soundName(path string) string {
	if path == "" {
		return "Tono integrado"
	}
	return filepath.Base(path)
}
