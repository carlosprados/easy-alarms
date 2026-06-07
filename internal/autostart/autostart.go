// Package autostart toggles launching the app on session login via the XDG
// autostart spec (~/.config/autostart/easy-alarms.desktop).
package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

const desktopEntry = `[Desktop Entry]
Type=Application
Name=Easy Alarms
Comment=Simple alarm clock with tray integration
Exec=%s --hidden
Terminal=false
X-GNOME-Autostart-enabled=true
`

func path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "autostart", "easy-alarms.desktop"), nil
}

func Enabled() bool {
	p, err := path()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

func Enable() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	p, err := path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, fmt.Appendf(nil, desktopEntry, exe), 0o644)
}

func Disable() error {
	p, err := path()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
