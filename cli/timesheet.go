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
	case "elapsed":
		return timesheet.PrintElapsedTime()
	case "start":
		return timesheet.Start()
	case "stop":
		return timesheet.AppendEntry()
	case "setTablePath":
		return timesheet.SetTablePath()
	}

	flag.PrintDefaults()
	return nil
}
