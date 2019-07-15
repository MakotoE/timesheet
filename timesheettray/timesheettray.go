package main

import (
	"github.com/getlantern/systray"
	"github.com/MakotoE/timesheet"
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTooltip("timesheet")
	startItem := systray.AddMenuItem("Start", "Start timer")
	stopItem := systray.AddMenuItem("Stop", "Stop timer")
	exitItem := systray.AddMenuItem("Exit", "")

	loop: for {
		select {
		case startItem.ClickedCh:
			timesheet.Start()
		case stopItem.ClickedCh:
		case exitItem.ClickedCh:
			systray.Quit()
			break loop
		}
	}
}

func onExit() {

}
