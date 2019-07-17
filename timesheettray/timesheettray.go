package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/MakotoE/timesheet"
	"github.com/fsnotify/fsnotify"
	"github.com/getlantern/systray"
)

func main() {
	systray.Run(onReady, func() {})
}

func onReady() {
	systray.SetTooltip("timesheet")

	startItem := systray.AddMenuItem("Start", "Start timer")
	stopItem := systray.AddMenuItem("Stop", "Stop timer")
	exitItem := systray.AddMenuItem("Exit", "")

	if err := updateIcon(); err != nil {
		logErr(err)
		systray.Quit()
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logErr(err)
		systray.Quit()
		return
	}
	defer watcher.Close()

	home, err := os.UserHomeDir()
	if err != nil {
		logErr(err)
		systray.Quit()
		return
	}

	if err := watcher.Add(home + "/.config/timesheet/data.json"); err != nil {
		logErr(err)
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

			//systray.SetIcon(playIcon)
		case <-stopItem.ClickedCh:
			if err := timesheet.Stop(); err != nil {
				logErr(err)
				systray.Quit()
				break loop
			}

			//systray.SetIcon(pauseIcon)
		case <-exitItem.ClickedCh:
			systray.Quit()
			break loop
		case <-watcher.Events: // TODO only activate on Write event
			if err := updateIcon(); err != nil {
				logErr(err)
				systray.Quit()
				break loop
			}
		case err := <-watcher.Errors:
			logErr(err)
			systray.Quit()
			break loop
		}
	}
}

func updateIcon() error {
	executablePath, err := os.Executable()
	if err != nil {
		return err
	}

	iconDir := filepath.Dir(executablePath) + "/timesheettrayIcons"

	started, err := timesheet.Started()
	if err != nil {
		return err
	}

	iconMap := map[bool]string{
		false: "play.ico",
		true:  "pause.ico",
	}

	iconBytes, err := ioutil.ReadFile(iconDir + "/" + iconMap[started])
	if err != nil {
		return err
	}

	systray.SetIcon(iconBytes)
	return nil
}

func logErr(e error) {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	logPath := home + "/.config/timesheet/log.txt"
	// TODO add time and append to file
	if err := ioutil.WriteFile(logPath, []byte(fmt.Sprintf("%+v\n", e)), 0666); err != nil {
		panic(err)
	}
}
