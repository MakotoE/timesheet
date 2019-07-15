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

type data struct {
	Started   bool
	StartTime time.Time
	TablePath string
}

func readData() (*data, error) {
	if verbose {
		fmt.Println("reading", dataPath)
	}

	file, err := os.Open(dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &data{}, nil
		}

		return nil, errors.WithStack(err)
	}
	defer file.Close()

	d := &data{}
	if err := json.NewDecoder(file).Decode(d); err != nil {
		return nil, errors.WithStack(err)
	}

	return d, nil
}

func (d *data) write() error {
	text, err := json.Marshal(d)
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
	defer file.Close()

	if err := file.Truncate(0); err != nil {
		return errors.WithStack(err)
	}

	_, err = file.Write(text)
	return errors.WithStack(err)
}

func dataDir() string {
	index := strings.LastIndex(dataPath, "/")
	if index == -1 {
		return dataPath
	}

	return dataPath[:index]
}

func printElapsedTime() error {
	d, err := readData()
	if err != nil {
		return err
	}

	if verbose {
		fmt.Printf("parsed data: %+v\n", d)
	}

	if d.Started {
		fmt.Println(time.Since(d.StartTime))
	} else {
		fmt.Fprintln(os.Stderr, "timer not started")
	}
	return nil
}

type table struct {
	*os.File
	path string
}

func openTable(tablePath string) (*table, error) {
	file, err := os.OpenFile(tablePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &table{file, tablePath}, nil
}

func (t *table) readAll() ([][]string, error) {
	if _, err := t.File.Seek(0, io.SeekStart); err != nil {
		return nil, errors.WithStack(err)
	}

	if verbose {
		fmt.Println("reading", t.path)
	}

	records, err := csv.NewReader(t.File).ReadAll()
	return records, errors.WithStack(err)
}

func (t *table) appendEntry(duration time.Duration) error {
	currentTime, err := time.Now().Truncate(time.Hour * 24).MarshalText()
	if err != nil {
		return errors.WithStack(err)
	}

	newRecord := []string{string(currentTime), duration.String()}
	if err := csv.NewWriter(t.File).WriteAll([][]string{newRecord}); err != nil {
		return errors.WithStack(err)
	}

	if verbose {
		fmt.Println("added new entry:", newRecord)
	}

	return nil
}

func (t *table) deleteLastEntry() error {
	records, err := t.readAll()
	if err != nil {
		return errors.WithStack(err)
	}

	stat, err := t.File.Stat()
	if err != nil {
		return errors.WithStack(err)
	}

	tablePath := stat.Name()

	t.File.Close()
	t.File = nil

	// Workaround for access denied error with file.Truncate() bug in Windows
	if err := os.Truncate(tablePath, 0); err != nil {
		return errors.WithStack(err)
	}

	file, err := os.OpenFile(tablePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return errors.WithStack(err)
	}

	t.File = file

	if err := csv.NewWriter(t.File).WriteAll(records[:len(records)-1]); err != nil {
		return errors.WithStack(err)
	}

	if verbose {
		fmt.Println("deleted last entry")
	}

	return nil
}

func start() error {
	d, err := readData()
	if err != nil {
		return err
	}

	d.Started = true
	d.StartTime = time.Now()
	return d.write()
}

func appendEntry() error {
	d, err := readData()
	if err != nil {
		return err
	}

	if !d.Started {
		fmt.Fprintln(os.Stderr, "timer not started")
		return nil
	}

	if d.TablePath == "" {
		fmt.Fprintln(os.Stderr, "TablePath not set")
		return nil
	}

	table, err := openTable(d.TablePath)
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

		return table.appendEntry(time.Since(d.StartTime))
	}

	lastRecordedDate := time.Time{}
	if err := lastRecordedDate.UnmarshalText([]byte(records[len(records)-1][0])); err != nil {
		return errors.WithStack(err)
	}

	if verbose {
		fmt.Println("last entry:", records[len(records)-1])
	}

	if time.Since(lastRecordedDate) > time.Hour*24 {
		if err := table.appendEntry(time.Since(d.StartTime)); err != nil {
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

		sumDuration := recordedDuration + time.Since(d.StartTime)
		if err := table.appendEntry(sumDuration); err != nil {
			return err
		}
	}

	d.Started = false
	if err = d.write(); err != nil {
		return err
	}

	return nil
}

func setTablePath() error {
	d, err := readData()
	if err != nil {
		return err
	}

	d.TablePath = flag.Arg(1)
	return d.write()
}
