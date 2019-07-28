package timesheet

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Verbose output.
var Verbose bool

var dataPath string

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	dataPath = home + "/.config/timesheet/data.json"
}

type data struct {
	Started   bool
	StartTime time.Time
	TablePath string
}

func readData() (*data, error) {
	if Verbose {
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

	if Verbose {
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

// Started returns true if timer is running.
func Started() (bool, error) {
	data, err := readData()
	if err != nil {
		return false, err
	}

	return data.Started, nil
}

// PrintElapsedTime prints the duration since start time.
func PrintElapsedTime() error {
	d, err := readData()
	if err != nil {
		return err
	}

	if Verbose {
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

func (t *table) appendEntry(duration time.Duration) error {
	currentTime, err := time.Now().MarshalText()
	if err != nil {
		return errors.WithStack(err)
	}

	newRecord := []string{string(currentTime), duration.String()}
	if err := csv.NewWriter(t.File).WriteAll([][]string{newRecord}); err != nil {
		return errors.WithStack(err)
	}

	if Verbose {
		fmt.Println("added new entry:", newRecord)
	}

	return nil
}

// Start writes start time to data file.
func Start() error {
	d, err := readData()
	if err != nil {
		return err
	}

	d.Started = true
	d.StartTime = time.Now()
	return d.write()
}

// Stop clears start time from data file and appends duration since start time to table.
func Stop() error {
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

	tableFile, err := os.OpenFile(d.TablePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return errors.WithStack(err)
	}
	defer tableFile.Close()

	currentTime, err := time.Now().MarshalText()
	if err != nil {
		return errors.WithStack(err)
	}

	newRecord := []string{string(currentTime), time.Since(d.StartTime).String()}
	if err := csv.NewWriter(tableFile).WriteAll([][]string{newRecord}); err != nil {
		return errors.WithStack(err)
	}

	if Verbose {
		fmt.Println("added new entry:", newRecord)
	}

	d.Started = false
	if err = d.write(); err != nil {
		return err
	}

	return nil
}

// SetTablePath writes argument 1 to TablePath entry in data file.
func SetTablePath() error {
	d, err := readData()
	if err != nil {
		return err
	}

	d.TablePath = flag.Arg(1)
	return d.write()
}
