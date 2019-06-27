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

const dataPath = "./timesheetData/data.json"
const tablePath = "./timesheetData/timesheet.csv"

func main() {
	verbose := flag.Bool("v", false, "verbose")
	flag.Parse()
	if err := runCommand(flag.Arg(0), *verbose); err != nil {
		panic(fmt.Sprintf("%+v\n", err))
	}
}

func runCommand(command string, verbose bool) error {
	switch command {
	case "elapsed":
		return printElapsedTime(verbose)
	case "start":
		return (&Data{true, time.Now()}).write()
	case "stop":
		return appendEntry(verbose)
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

func printElapsedTime(verbose bool) error {
	if verbose {
		fmt.Println("reading", dataPath)
	}

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

func appendEntry(verbose bool) error {
	if verbose {
		fmt.Println("reading and resetting", dataPath)
	}

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
