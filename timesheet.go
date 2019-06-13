package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

const dataFilepath = "/home/makoto/Documents/timesheet/data"

func main() {
	flag.Parse()
	//runCommand(flag.Arg(0))
	if err := runCommand("start"); err != nil {
		panic(fmt.Sprintf("%+v\n", err))
	}
}

func runCommand(command string) error {
	switch command {
	case "":
	case "start":
		return writeTime()
	case "stop":
	}

	flag.PrintDefaults()
	return nil
}

func writeTime() error {
	b, err := time.Now().MarshalText()
	if err != nil {
		return err
	}

	if err := os.Mkdir(dataFilepath, os.ModePerm); err != nil && os.IsNotExist(err) {
		return err
	}

	file, err := os.OpenFile(dataFilepath+"/startTime", os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return err
	}

	_, err = file.Write(b)
	return err
}
