package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
)

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
	}

	flag.PrintDefaults()
	return nil
}

// Data .
type Data struct {
	Started   bool
	StartTime time.Time
	TablePath string
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

	if err := os.Mkdir(dataDir(), os.ModePerm); err != nil && !os.IsExist(err) {
		return errors.WithStack(err)
	}

	file, err := os.OpenFile(dataPath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return errors.WithStack(err)
	}

	if err := file.Truncate(0); err != nil {
		return errors.WithStack(err)
	}

	if _, err := file.Write(text); err != nil {
		return errors.WithStack(err)
	}

	return file.Sync()
}

func dataDir() string {
	index := strings.LastIndex(dataPath, "/")
	if index == -1 {
		return dataPath
	}

	return dataPath[:index]
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

// Table .
type Table struct {
	*os.File
}

func openTable(tablePath string) (*Table, error) {
	file, err := os.OpenFile(tablePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &Table{file}, nil
}

func (table *Table) readAll() ([][]string, error) {
	if _, err := table.File.Seek(0, io.SeekStart); err != nil {
		return nil, errors.WithStack(err)
	}

	if verbose {
		fmt.Println("reading", tablePath)
	}

	records, err := csv.NewReader(table.File).ReadAll()
	return records, errors.WithStack(err)
}

func (table *Table) appendEntry(duration time.Duration) error {
	currentTime, err := time.Now().Truncate(time.Hour * 24).MarshalText()
	if err != nil {
		return errors.WithStack(err)
	}

	newRecord := []string{string(currentTime), duration.String()}
	if err := csv.NewWriter(table.File).WriteAll([][]string{newRecord}); err != nil {
		return errors.WithStack(err)
	}

	if verbose {
		fmt.Println("added new entry:", newRecord)
	}

	return nil
}

func (table *Table) deleteLastEntry() error {
	records, err := table.readAll()
	if err != nil {
		return errors.WithStack(err)
	}

	if err := table.File.Truncate(0); err != nil {
		return errors.WithStack(err)
	}

	if err := csv.NewWriter(table.File).WriteAll(records[:len(records)-1]); err != nil {
		return errors.WithStack(err)
	}

	if verbose {
		fmt.Println("deleted last entry")
	}

	return nil
}

func start() error {
	data, err := readData()
	if err != nil {
		return err
	}

	data.Started = true
	data.StartTime = time.Now()

	return data.write()
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

	table, err := openTable(tablePath)
	if err != nil {
		return err
	}
	defer table.Close()

	records, err := table.readAll()
	if err != nil {
		return errors.WithStack(err)
	}

	if len(records) == 0 {
		if verbose {
			fmt.Println("0 records in table")
		}

		return table.appendEntry(time.Since(data.StartTime))
	}

	lastRecordedDate := time.Time{}
	if err := lastRecordedDate.UnmarshalText([]byte(records[len(records)-1][0])); err != nil {
		return errors.WithStack(err)
	}

	if verbose {
		fmt.Println("last entry:", records[len(records)-1])
	}

	if time.Since(lastRecordedDate) > time.Hour*24 {
		if err := table.appendEntry(time.Since(data.StartTime)); err != nil {
			return err
		}
	} else {
		recordedDuration, err := time.ParseDuration(records[len(records)-1][1])
		if err != nil {
			return errors.WithStack(err)
		}

		if err := table.deleteLastEntry(); err != nil {
			return err
		}

		sumDuration := recordedDuration + time.Since(data.StartTime)
		if err := table.appendEntry(sumDuration); err != nil {
			return err
		}
	}

	if err = (&Data{Started: false}).write(); err != nil {
		return err
	}

	return nil
}
