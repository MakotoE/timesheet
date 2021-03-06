package main

import (
	"flag"
	"fmt"

	"github.com/MakotoE/timesheet"
)

func main() {
	v := flag.Bool("v", false, "verbose")
	flag.Parse()
	timesheet.Verbose = *v

	if err := runCommand(flag.Arg(0)); err != nil {
		panic(fmt.Sprintf("%+v\n", err))
	}
}

func runCommand(command string) error {
	switch command {
	case "status":
		return timesheet.Status()
	case "table":
		return timesheet.Table()
	case "start":
		return timesheet.Start()
	case "stop":
		return timesheet.Stop()
	case "setLogPath":
		return timesheet.SetLogPath()
	}

	flag.PrintDefaults()
	return nil
}
