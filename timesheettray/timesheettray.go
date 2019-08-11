package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/MakotoE/timesheet"
	"github.com/fsnotify/fsnotify"
	"github.com/getlantern/systray"
	"github.com/pkg/errors"
)

func main() {
	systray.Run(onReady, func() {})
}

func onReady() {
	systray.SetTooltip("timesheet")

	startItem := systray.AddMenuItem("Start", "Start timer")
	stopItem := systray.AddMenuItem("Stop", "Stop timer")

	title := "Check error log (~/.config/timesheet/log.txt)"
	tooltip := "Click to dismiss; delete error log to clear notice"
	errorNoticeItem := systray.AddMenuItem(title, tooltip)
	if !errorLogExists() {
		errorNoticeItem.Hide()
	}

	exitItem := systray.AddMenuItem("Exit", "")

	if err := updateIcon(); err != nil {
		logErr(err)
		systray.Quit()
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logErr(errors.WithStack(err))
		systray.Quit()
		return
	}
	defer watcher.Close()

	home, err := os.UserHomeDir()
	if err != nil {
		logErr(errors.WithStack(err))
		systray.Quit()
		return
	}

	// Watching the data file allows concurrent processes—tray and cli—to update tray icon
	if err := watcher.Add(home + "/.config/timesheet/data.json"); err != nil {
		logErr(errors.WithStack(err))
		systray.Quit()
		return
	}

loop:
	for {
		select {
		case <-startItem.ClickedCh:
			if err := timesheet.Start(); err != nil {
				logErr(err)
				systray.Quit()
				break loop
			}
		case <-stopItem.ClickedCh:
			if err := timesheet.Stop(); err != nil {
				logErr(err)
				systray.Quit()
				break loop
			}
		case <-errorNoticeItem.ClickedCh:
			errorNoticeItem.Hide()
		case <-exitItem.ClickedCh:
			systray.Quit()
			break loop
		case event := <-watcher.Events:
			if event.Op == fsnotify.Write {
				if err := updateIcon(); err != nil {
					logErr(err)
					systray.Quit()
					break loop
				}
			}
		case err := <-watcher.Errors:
			logErr(errors.WithStack(err))
			systray.Quit()
			break loop
		}
	}
}

func updateIcon() error {
	executablePath, err := os.Executable()
	if err != nil {
		return errors.WithStack(err)
	}

	iconDir := filepath.Dir(executablePath) + "/timesheettrayIcons"

	started, err := timesheet.Started()
	if err != nil {
		return errors.WithStack(err)
	}

	iconMap := map[bool]string{
		false: "pause.ico",
		true:  "play.ico",
	}

	iconBytes, err := ioutil.ReadFile(iconDir + "/" + iconMap[started])
	if err != nil {
		return errors.WithStack(err)
	}

	systray.SetIcon(iconBytes)
	return nil
}

func logErr(e error) {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	file, err := os.OpenFile(home+"/.config/timesheet/log.txt", os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	if _, err := file.WriteString(fmt.Sprintf("%s\n%+v\n\n", time.Now(), e)); err != nil {
		panic(err)
	}
}

func errorLogExists() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	file, err := os.OpenFile(home+"/.config/timesheet/log.txt", os.O_RDONLY, 0666)
	if err != nil {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Size() > 0
}
