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
		return storeTime()
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

func storeTime() error {
	text, err := json.Marshal(Data{true, time.Now()})
	if err != nil {
		return errors.WithStack(err)
	}

	if err := os.Mkdir(".", os.ModePerm); err != nil && os.IsNotExist(err) {
		return errors.WithStack(err)
	}

	return ioutil.WriteFile("./data", text, 0666)
}

func elapsedTime() (time.Duration, error) {
	file, err := os.Open("./data")
	if err != nil {
		return 0, errors.WithStack(err)
	}

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

	if _, err := os.OpenFile("./data", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666); err != nil {
		return errors.WithStack(err)
	}

	// TODO
	_ = duration
	return nil
}
