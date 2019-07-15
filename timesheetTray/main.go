package main

import "github.com/getlantern/systray"

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
			timesheet.start()
		case stopItem.ClickedCh:
		case exitItem.ClickedCh:
			systray.Quit()
			break loop
		}
	}
}

func onExit() {

}
