package timesheet

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
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
		fmt.Println("Reading", dataPath)
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
		fmt.Println("Writing to", dataPath)
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
		fmt.Printf("Parsed data: %+v\n", d)
	}

	if d.Started {
		fmt.Printf("Elapsed time: %s\n", time.Since(d.StartTime))
	} else {
		fmt.Println("Not started")
	}
	return nil
}

func weeklyTotal(durations []time.Duration) time.Duration {
	sum := time.Duration(0)
	for _, duration := range durations {
		sum += duration
	}
	return sum
}

func sundayIndex(entry0 entry) int {
	if entry0.date.Weekday() == time.Sunday {
		return 0
	}

	return int(7 - entry0.date.Weekday())
}

func max(a int, b int) int {
	if a > b {
		return a
	}

	return b
}

func formatDuration(d time.Duration) string {
	rounded := time.Duration(d).Round(time.Minute)
	remainder := time.Duration(d) - time.Duration(int(rounded.Hours())*int(time.Hour))
	return fmt.Sprintf("%02.0fh%02.0fm", rounded.Hours(), time.Duration(remainder).Minutes())
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

	entries, err := readLog(d.LogPath)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		return nil
	}

	durations := dailyDurations(entries)

	outputTable := make([][]string, len(durations))
	for i := range durations {
		outputTable[i] = make([]string, 3)
		date := entries[0].date.Add(time.Duration(int(time.Hour) * 24 * i))
		outputTable[i][0] = date.Format("2006-01-02")
		outputTable[i][1] = formatDuration(durations[i])

		if i%7 == sundayIndex(entries[0]) {
			outputTable[i][2] = formatDuration(weeklyTotal(durations[max(0, i-6) : i+1]))
		}
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
	csvWriter := csv.NewWriter(writer)
	csvWriter.Comma = '\t'
	if err := csvWriter.WriteAll(outputTable); err != nil {
		return errors.WithStack(err)
	}

	return errors.WithStack(writer.Flush())
}

// dailyDurations consolidates the table into a list of durations per day.
func dailyDurations(entries []entry) []time.Duration {
	if len(entries) == 0 {
		return nil
	}

	firstLastDiff := entries[len(entries)-1].date.Sub(entries[0].date)
	days := int(firstLastDiff.Hours()/24 + 1)
	dailyDurations := make([]time.Duration, days)

	for _, entry := range entries {
		day := int(entry.date.Sub(entries[0].date).Hours() / 24)
		dailyDurations[day] += entry.duration
	}

	return dailyDurations
}

type entry struct {
	date     time.Time
	duration time.Duration
}

func readLog(logPath string) ([]entry, error) {
	file, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer file.Close()

	var entries []entry

	reader := csv.NewReader(file)
	for {
		entry, err := nextLogRecord(reader)
		if err != nil {
			if errors.Cause(err) == io.EOF {
				return entries, nil
			}

			return nil, err
		}
		entries = append(entries, *entry)
	}
}

func nextLogRecord(reader *csv.Reader) (*entry, error) {
	record, err := reader.Read()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	result := entry{}
	if err := result.date.UnmarshalText([]byte(record[0])); err != nil {
		return nil, errors.WithStack(err)
	}

	duration, err := time.ParseDuration(record[1])
	if err != nil {
		return nil, errors.WithStack(err)
	}

	result.duration = duration
	return &result, nil
}

func appendLogEntry(logPath string, duration time.Duration) error {
	file, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return errors.WithStack(err)
	}
	defer file.Close()

	currentTime, err := time.Now().MarshalText()
	if err != nil {
		return errors.WithStack(err)
	}

	writer := csv.NewWriter(file)
	newRecord := []string{string(currentTime), duration.String()}
	if err := writer.Write(newRecord); err != nil {
		return errors.WithStack(err)
	}

	if Verbose {
		fmt.Println("Added new entry:", newRecord)
	}

	writer.Flush()
	return nil
}

// Start writes start time to data file.
func Start() error {
	d, err := readData()
	if err != nil {
		return err
	}

	if d.Started {
		fmt.Println("Timer already started; exiting without changing start time")
		return nil
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
		fmt.Fprintln(os.Stderr, "Timer not started")
		return nil
	}

	if d.LogPath == "" {
		fmt.Fprintln(os.Stderr, "LogPath not set")
		return nil
	}

	if err := appendLogEntry(d.LogPath, time.Since(d.StartTime)); err != nil {
		return err
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
