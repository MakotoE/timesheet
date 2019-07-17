package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/MakotoE/timesheet"
	"github.com/getlantern/systray"
)

func main() {
	systray.Run(onReady, func() {})
}

func onReady() {
	executablePath, err := os.Executable()
	if err != nil {
		logErr(err)
		systray.Quit()
	}

	iconDir := filepath.Dir(executablePath) + "/timesheettrayIcons"
	pauseIcon, err := ioutil.ReadFile(iconDir + "/pause.ico")
	if err != nil {
		logErr(err)
		systray.Quit()
	}

	playIcon, err := ioutil.ReadFile(iconDir + "/play.ico")
	if err != nil {
		logErr(err)
		systray.Quit()
	}

	started, err := timesheet.Started()
	if err != nil {
		logErr(err)
		systray.Quit()
	}

	if started {
		systray.SetIcon(playIcon)
	} else {
		systray.SetIcon(pauseIcon)
	}
	systray.SetTooltip("timesheet")

	startItem := systray.AddMenuItem("Start", "Start timer")
	stopItem := systray.AddMenuItem("Stop", "Stop timer")
	exitItem := systray.AddMenuItem("Exit", "")

loop:
	for {
		select {
		case <-startItem.ClickedCh:
			if err := timesheet.Start(); err != nil {
				logErr(err)
				break loop
			}

			systray.SetIcon(playIcon)
		case <-stopItem.ClickedCh:
			if err := timesheet.Stop(); err != nil {
				logErr(err)
				break loop
			}

			systray.SetIcon(pauseIcon)
		case <-exitItem.ClickedCh:
			systray.Quit()
			break loop
		}
	}
}

func logErr(e error) {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	logPath := home + "/.config/timesheet/log.txt"

	if err := ioutil.WriteFile(logPath, []byte(fmt.Sprintf("%+v\n", e)), 0666); err != nil {
		panic(err)
	}
}
