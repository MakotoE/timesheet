package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/pkg/errors"
)

const dataPath = "./data.json"
const tablePath = "./timesheet.csv"

func main() {
	flag.Parse()
	if err := runCommand(flag.Arg(0)); err != nil {
		panic(fmt.Sprintf("%+v\n", err))
	}
}

func runCommand(command string) error {
	switch command {
	case "elapsed":
		return printElapsedTime()
	case "start":
		return (&Data{true, time.Now()}).write()
	case "stop":
		return appendEntry()
	}

	flag.PrintDefaults()
	return nil
}

type Data struct {
	Started   bool
	StartTime time.Time
}

func readData() (*Data, error) {
	file, err := os.Open(dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Data{}, nil
		}

		return nil, errors.WithStack(err)
	}
	defer file.Close()

	data := &Data{}
	if err := json.NewDecoder(file).Decode(data); err != nil {
		return nil, errors.WithStack(err)
	}

	return data, nil
}

func (data *Data) write() error {
	text, err := json.Marshal(data)
	if err != nil {
		return errors.WithStack(err)
	}

	if err := os.Mkdir(".", os.ModePerm); err != nil && os.IsNotExist(err) {
		return errors.WithStack(err)
	}

	return ioutil.WriteFile(dataPath, text, 0666)
}

func printElapsedTime() error {
	data, err := readData()
	if err != nil {
		return err
	}

	if data.Started {
		fmt.Println(time.Since(data.StartTime))
	} else {
		os.Stderr.WriteString("timer not started\n")
	}
	return nil
}

func appendEntry() error {
	/*
		stop should add entry but replace last if same date
	*/
	data, err := readData()
	if err != nil {
		return err
	}

	if err = (&Data{Started: false}).write(); err != nil {
		return err
	}

	file, err := os.OpenFile(tablePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return errors.WithStack(err)
	}
	defer file.Close()

	file.WriteString(time.Since(data.StartTime).String() + "\n")
	return nil
}
