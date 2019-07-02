package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/pkg/errors"
)

/*
Stop time on sleep; do not start time on wake up
*/

var verbose bool

var dataPath string
var tablePath string

func main() {
	v := flag.Bool("v", false, "verbose")
	flag.Parse()
	verbose = *v

	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	dataPath = home + "/.config/timesheet/data.json"
	tablePath = home + "/.config/timesheet/timesheet.csv"

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

// Data .
type Data struct {
	Started   bool
	StartTime time.Time
}

func readData() (*Data, error) {
	if verbose {
		fmt.Println("reading", dataPath)
	}

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

	if verbose {
		fmt.Println("writing to", dataPath)
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

	if verbose {
		fmt.Printf("parsed data: %+v\n", data)
	}

	if data.Started {
		fmt.Println(time.Since(data.StartTime))
	} else {
		fmt.Fprintln(os.Stderr, "timer not started")
	}
	return nil
}

func appendEntry() error {
	data, err := readData()
	if err != nil {
		return err
	}

	if !data.Started {
		fmt.Fprintln(os.Stderr, "timer not started")
		return nil
	}

	if err = (&Data{Started: false}).write(); err != nil {
		return err
	}

	file, err := os.OpenFile(tablePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return errors.WithStack(err)
	}
	defer file.Close()

	if verbose {
		fmt.Println("reading", tablePath)
	}

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		return errors.WithStack(err)
	}

	lastRecordedDate := time.Time{}
	if len(records) > 0 {
		if err := lastRecordedDate.UnmarshalText([]byte(records[len(records)-1][0])); err != nil {
			return errors.WithStack(err)
		}

		if verbose {
			fmt.Println("last entry:", records[len(records)-1])
		}
	} else if verbose {
		fmt.Println("zero records in table")
	}

	if len(records) == 0 || time.Since(lastRecordedDate) > time.Hour*24 {
		currentTime, err := time.Now().MarshalText()
		if err != nil {
			return errors.WithStack(err)
		}

		newRecord := []string{string(currentTime), time.Since(data.StartTime).String()}
		writer := csv.NewWriter(file)
		if err := writer.Write(newRecord); err != nil {
			return errors.WithStack(err)
		}
		writer.Flush()

		if verbose {
			fmt.Println("added new entry:", newRecord)
		}
	} else {
		if err := file.Truncate(0); err != nil {
			return errors.WithStack(err)
		}

		writer := csv.NewWriter(file)
		if err := writer.WriteAll(records[:len(records)-1]); err != nil {
			return errors.WithStack(err)
		}

		if verbose {
			fmt.Println("removed last entry")
		}

		recordedDuration, err := time.ParseDuration(records[len(records)-1][1])
		if err != nil {
			return errors.WithStack(err)
		}

		currentTime, err := time.Now().MarshalText()
		if err != nil {
			return errors.WithStack(err)
		}

		sumDuration := recordedDuration + time.Since(data.StartTime)
		newRecord := []string{string(currentTime), sumDuration.String()}
		if err := writer.Write(newRecord); err != nil {
			return errors.WithStack(err)
		}
		writer.Flush()

		if verbose {
			fmt.Println("added new entry:", newRecord)
		}
	}

	return nil
}
