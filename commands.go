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
	LogPath   string
}

// Started returns true if timer is running. (Helper for timesheettray.go)
func Started() (bool, error) {
	data, err := readData()
	if err != nil {
		return false, err
	}

	return data.Started, nil
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

// Status prints the duration since start time.
func Status() error {
	d, err := readData()
	if err != nil {
		return err
	}

	if Verbose {
		fmt.Printf("parsed data: %+v\n", d)
	}

	if d.Started {
		fmt.Printf("Elapsed time: %s\n", time.Since(d.StartTime))
	} else {
		fmt.Println("Not started")
	}
	return nil
}

// Table prints a csv table of daily durations where the columns are: date, duration worked, weekly
// total.
func Table() error {
	d, err := readData()
	if err != nil {
		return err
	}

	if d.LogPath == "" {
		fmt.Fprintln(os.Stderr, "LogPath not set")
		return nil
	}

	logFile, err := os.OpenFile(d.LogPath, os.O_RDONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return errors.WithStack(err)
	}
	defer logFile.Close()

	records, err := csv.NewReader(logFile).ReadAll()
	if err != nil {
		return errors.WithStack(err)
	}

	type entry struct {
		date     time.Time
		duration time.Duration
	}

	entries := make([]entry, len(records))

	for i, record := range records {
		if err := entries[i].date.UnmarshalText([]byte(record[0])); err != nil {
			return errors.WithStack(err)
		}

		duration, err := time.ParseDuration(record[1])
		if err != nil {
			return errors.WithStack(err)
		}

		entries[i].duration = duration
	}

	if len(entries) == 0 {
		return nil
	}

	firstLastDiff := entries[len(entries)-1].date.Sub(entries[0].date)
	days := int(firstLastDiff.Hours()/24 + 0.5) // round up to include last day
	dailyDurations := make([]time.Duration, days)

	for _, entry := range entries {
		day := int(entry.date.Sub(entries[0].date).Hours() / 24)
		dailyDurations[day] += entry.duration // TODO out of range error; dailyDurations needs one more element
	}

	outputTable := make([][]string, len(dailyDurations))
	for i := range dailyDurations {
		outputTable[i] = make([]string, 3)
		outputTable[i][0] = entries[0].date.Add(time.Duration(int(time.Hour) * 24 * i)).Format("2006-01-02")
		outputTable[i][1] = dailyDurations[i].String()
	}

	var shiftNDays int
	if entries[0].date.Weekday() == time.Sunday {
		shiftNDays = 0
	} else {
		shiftNDays = int(7 - entries[0].date.Weekday())
	}

	for i := range dailyDurations {
		if i%7 == shiftNDays {
			var startFrom int
			if i < 7 {
				startFrom = 0
			} else {
				startFrom = i - 6
			}

			weeklyTotal := time.Duration(0)
			for day := startFrom; day < i+1; day++ {
				weeklyTotal += dailyDurations[day]
			}

			outputTable[i][2] = weeklyTotal.String()
		}
	}

	if err := csv.NewWriter(os.Stdout).WriteAll(outputTable); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

type log struct {
	*os.File
	path string
}

func openLog(logPath string) (*log, error) {
	file, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &log{file, logPath}, nil
}

func (t *log) appendEntry(duration time.Duration) error {
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

// Stop clears start time from data file and appends duration since start time to log.
func Stop() error {
	d, err := readData()
	if err != nil {
		return err
	}

	if !d.Started {
		fmt.Fprintln(os.Stderr, "timer not started")
		return nil
	}

	if d.LogPath == "" {
		fmt.Fprintln(os.Stderr, "LogPath not set")
		return nil
	}

	logFile, err := os.OpenFile(d.LogPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return errors.WithStack(err)
	}
	defer logFile.Close()

	currentTime, err := time.Now().MarshalText()
	if err != nil {
		return errors.WithStack(err)
	}

	newRecord := []string{string(currentTime), time.Since(d.StartTime).String()}
	if err := csv.NewWriter(logFile).WriteAll([][]string{newRecord}); err != nil {
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

// SetLogPath changes LogPath entry in data file to argument 1.
func SetLogPath() error {
	d, err := readData()
	if err != nil {
		return err
	}

	d.LogPath = flag.Arg(1)
	return d.write()
}
