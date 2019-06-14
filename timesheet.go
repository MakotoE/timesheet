package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

func main() {
	flag.Parse()
	//runCommand(flag.Arg(0))
	if err := runCommand("elapsed"); err != nil {
		panic(fmt.Sprintf("%+v\n", err))
	}
}

func runCommand(command string) error {
	switch command {
	case "elapsed":
		return printElapsedTime()
	case "start":
		return writeTime()
	case "stop":
	}

	flag.PrintDefaults()
	return nil
}

func printElapsedTime() error {
	file, err := os.Open("./data")
	if err != nil {
		return err
	}

	duration, err := elapsedTime(file)
	if err != nil {
		return err
	}

	fmt.Println(duration.Seconds())
	return nil
}

type Data struct {
	Started   bool
	StartTime time.Time
}

func writeTime() error {
	text, err := json.Marshal(Data{true, time.Now()})
	if err != nil {
		return err
	}

	if err := os.Mkdir(".", os.ModePerm); err != nil && os.IsNotExist(err) {
		return err
	}

	return ioutil.WriteFile("./data", text, 0666)
}

func elapsedTime(dataFile *os.File) (time.Duration, error) {
	data := &Data{}
	if err := json.NewDecoder(dataFile).Decode(data); err != nil {
		return 0, err
	}

	return time.Since(data.StartTime), nil
}
