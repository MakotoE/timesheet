package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
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
		return storeData(Data{true, time.Now()})
	case "stop":
		return appendEntry()
	}

	flag.PrintDefaults()
	return nil
}

func printElapsedTime() error {
	duration, err := elapsedTime()
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

func storeData(data Data) error {
	text, err := json.Marshal(data)
	if err != nil {
		return errors.WithStack(err)
	}

	if err := os.Mkdir(".", os.ModePerm); err != nil && os.IsNotExist(err) {
		return errors.WithStack(err)
	}

	return ioutil.WriteFile(dataPath, text, 0666)
}

func elapsedTime() (time.Duration, error) {
	file, err := os.Open(dataPath)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	defer file.Close()

	data := &Data{}
	if err := json.NewDecoder(file).Decode(data); err != nil && err != io.EOF {
		return 0, errors.WithStack(err)
	}

	if !data.Started {
		return 0, errors.New("time not started")
	}

	return time.Since(data.StartTime), nil
}

func appendEntry() error {
	duration, err := elapsedTime()
	if err != nil {
		return err
	}

	if err = storeData(Data{Started: false}); err != nil {
		return err
	}

	file, err := os.OpenFile(tablePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return errors.WithStack(err)
	}
	defer file.Close()

	file.WriteString(duration.String() + "\n")
	return nil
}
