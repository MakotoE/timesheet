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

			//systray.SetIcon(playIcon)
		case <-stopItem.ClickedCh:
			if err := timesheet.Stop(); err != nil {
				logErr(err)
				break loop
			}

			//systray.SetIcon(pauseIcon)
		case <-exitItem.ClickedCh:
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
		// logErr(err)
		// systray.Quit()
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

	if err := ioutil.WriteFile(logPath, []byte(fmt.Sprintf("%+v\n", e)), 0666); err != nil {
		panic(err)
	}
}
