package main

import (
	"fmt"
	"io/ioutil"
	"os"

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
		case <-stopItem.ClickedCh:
			if err := timesheet.Stop(); err != nil {
				logErr(err)
				break loop
			}
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

	errorText := fmt.Sprintf("%+v\n", e)

	if err := ioutil.WriteFile(home+"/.config/log.txt", []byte(errorText), 0666); err != nil {
		panic(err)
	}
}
