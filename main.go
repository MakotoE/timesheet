package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	v := flag.Bool("v", false, "verbose")
	flag.Parse()
	verbose = *v

	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	dataPath = home + "/.config/timesheet/data.json"

	if err := runCommand(flag.Arg(0)); err != nil {
		panic(fmt.Sprintf("%+v\n", err))
	}
}

func runCommand(command string) error {

	switch command {
	case "elapsed":
		return printElapsedTime()
	case "start":
		return start()
	case "stop":
		return appendEntry()
	case "setTablePath":
		return setTablePath()
	}

	flag.PrintDefaults()
	return nil
}
